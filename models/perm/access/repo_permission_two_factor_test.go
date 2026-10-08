// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package access

import (
	"testing"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/organization"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// org3 fixtures: user4 is a member with write on private repo3 (team1), user10 an outside collaborator with
// write on public repo32, user2 an owner and user5 a stranger. None of them has 2FA.
func TestRepoPermissionTwoFactorPolicy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	org.RequireTwoFactor = true
	require.NoError(t, user_model.UpdateUserCols(ctx, org, "require_two_factor"))

	privateRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	publicRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 32})
	loadUser := func(id int64) *user_model.User {
		return unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: id})
	}
	member, collaborator, owner, stranger := loadUser(4), loadUser(10), loadUser(2), loadUser(5)
	getPerm := func(repo *repo_model.Repository, u *user_model.User) *Permission {
		perm, err := GetIndividualUserRepoPermission(ctx, repo, u)
		require.NoError(t, err)
		return &perm
	}
	assertSamePerm := func(expected, actual *Permission, msg string) {
		assert.Equal(t, expected.AccessMode, actual.AccessMode, msg)
		for _, u := range []unit.Type{unit.TypeCode, unit.TypeIssues, unit.TypePullRequests, unit.TypeWiki} {
			assert.Equal(t, expected.UnitAccessMode(u), actual.UnitAccessMode(u), "%s: unit %s", msg, u)
		}
	}

	t.Run("BlockedIsNonMember", func(t *testing.T) {
		assert.False(t, getPerm(privateRepo, member).HasAnyUnitAccess(), "member on a private repo")
		assert.False(t, getPerm(privateRepo, owner).HasAnyUnitAccess(), "owner on a private repo")
		assertSamePerm(getPerm(publicRepo, stranger), getPerm(publicRepo, collaborator), "collaborator on a public repo")
		assert.False(t, getPerm(publicRepo, collaborator).CanWrite(unit.TypeCode))
		assert.False(t, CheckRepoUnitUser(ctx, privateRepo, member, unit.TypeIssues), "picks notification and mail recipients")

		// restricted users get less from public repos; a blocked one must match a restricted stranger
		member.IsRestricted, stranger.IsRestricted = true, true
		defer func() { member.IsRestricted, stranger.IsRestricted = false, false }()
		assertSamePerm(getPerm(publicRepo, stranger), getPerm(publicRepo, member), "restricted member on a public repo")
	})

	t.Run("Exempt", func(t *testing.T) {
		assert.True(t, getPerm(privateRepo, loadUser(1)).IsOwner(), "site admin")
		assert.True(t, getPerm(privateRepo, org).IsOwner(), "the org itself, as a deploy key pushes")

		perm, err := GetIndividualUserRepoPermission(organization.IgnoreTwoFactorPolicy(ctx), privateRepo, member)
		require.NoError(t, err)
		assert.True(t, perm.CanWrite(unit.TypeCode), "IgnoreTwoFactorPolicy sees the real access")
	})

	t.Run("RepoAdmin", func(t *testing.T) {
		isAdmin, err := IsUserRepoAdmin(ctx, privateRepo, owner)
		require.NoError(t, err)
		assert.False(t, isAdmin)
		isAdmin, err = IsUserRealRepoAdmin(ctx, privateRepo, owner)
		require.NoError(t, err)
		assert.False(t, isAdmin)
	})

	t.Run("EnrollingRestoresAccess", func(t *testing.T) {
		tfa := &auth_model.TwoFactor{UID: member.ID}
		require.NoError(t, tfa.SetSecret("JBSWY3DPEHPK3PXP"))
		require.NoError(t, auth_model.NewTwoFactor(ctx, tfa))
		assert.True(t, getPerm(privateRepo, member).CanWrite(unit.TypeCode))
	})
}
