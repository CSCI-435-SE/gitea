// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization_test

import (
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/organization"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequireTwoFactor(t *testing.T, org *organization.Organization, require2FA bool) {
	org.RequireTwoFactor = require2FA
	require.NoError(t, user_model.UpdateUserCols(t.Context(), org.AsUser(), "require_two_factor"))
}

// org3's members are user2 (owner), user4 and user28, and user10 is an outside collaborator on its
// repo21; none of them has 2FA. user24 has TOTP and user32 has WebAuthn.
func TestIsTwoFactorBlocked(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	loadUser := func(id int64) *user_model.User {
		return unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: id})
	}

	blocked, err := org.IsTwoFactorBlocked(t.Context(), loadUser(4))
	require.NoError(t, err)
	assert.False(t, blocked, "policy off")

	org.RequireTwoFactor = true
	for _, tc := range []struct {
		name    string
		user    *user_model.User
		blocked bool
	}{
		{"member without 2FA", loadUser(4), true},
		{"site admin without 2FA", loadUser(1), false},
		{"TOTP", loadUser(24), false},
		{"WebAuthn", loadUser(32), false},
		{"the org itself, as a deploy key pushes", org.AsUser(), false},
		{"ghost", user_model.NewGhostUser(), false},
		{"anonymous", nil, false},
	} {
		blocked, err := org.IsTwoFactorBlocked(t.Context(), tc.user)
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.blocked, blocked, tc.name)
	}

	blocked, err = org.IsTwoFactorBlocked(organization.IgnoreTwoFactorPolicy(t.Context()), loadUser(4))
	require.NoError(t, err)
	assert.False(t, blocked, "IgnoreTwoFactorPolicy")
}

func TestTwoFactorPolicyQueries(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})

	members, err := organization.CountOrgMembersWithoutTwoFactor(t.Context(), org.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, members)
	collaborators, err := organization.CountOrgOutsideCollaboratorsWithoutTwoFactor(t.Context(), org.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, collaborators, "user10 only; user2 collaborates on repo3 but is a member")

	canCreate := func() bool {
		ok, err := organization.CanCreateOrgRepo(t.Context(), org.ID, 2)
		require.NoError(t, err)
		return ok
	}
	createOrgIDs := func() (ids []int64) {
		orgs, err := organization.GetOrgsCanCreateRepoByUserID(t.Context(), 2)
		require.NoError(t, err)
		for _, o := range orgs {
			ids = append(ids, o.ID)
		}
		return ids
	}
	blockedByAnyOrg := func(id int64) bool {
		blocked, err := organization.IsTwoFactorBlockedByAnyOrg(t.Context(), unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: id}))
		require.NoError(t, err)
		return blocked
	}

	assert.True(t, canCreate())
	assert.Contains(t, createOrgIDs(), org.ID)
	assert.False(t, blockedByAnyOrg(4))

	setRequireTwoFactor(t, org, true)
	defer setRequireTwoFactor(t, org, false)

	assert.False(t, canCreate(), "an owner without 2FA can't create repos in the org")
	assert.NotContains(t, createOrgIDs(), org.ID)
	assert.True(t, blockedByAnyOrg(4), "member")
	assert.True(t, blockedByAnyOrg(10), "outside collaborator")
	assert.False(t, blockedByAnyOrg(5), "unrelated user")
	assert.False(t, blockedByAnyOrg(1), "site admin")
}

func TestCountOrgMembersWithoutTwoFactorExcludesAdmins(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()
	for _, userID := range []int64{24, 32} {
		require.NoError(t, db.Insert(ctx, &organization.OrgUser{OrgID: 3, UID: userID}))
	}

	count, err := organization.CountOrgMembersWithoutTwoFactor(ctx, 3)
	require.NoError(t, err)
	assert.EqualValues(t, 3, count, "TOTP and WebAuthn members are excluded")

	for i, userID := range []int64{4, 2} {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
		user.IsAdmin = true
		require.NoError(t, user_model.UpdateUserCols(ctx, user, "is_admin"))
		count, err = organization.CountOrgMembersWithoutTwoFactor(ctx, 3)
		require.NoError(t, err)
		assert.EqualValues(t, 2-i, count, "site admins are excluded, including owners")
	}
}

func TestCountOrgOutsideCollaboratorsWithoutTwoFactorExcludesAdmins(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	count, err := organization.CountOrgOutsideCollaboratorsWithoutTwoFactor(ctx, 3)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "user10 on repo21")

	collaborator := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	collaborator.IsAdmin = true
	require.NoError(t, user_model.UpdateUserCols(ctx, collaborator, "is_admin"))
	count, err = organization.CountOrgOutsideCollaboratorsWithoutTwoFactor(ctx, 3)
	require.NoError(t, err)
	assert.Zero(t, count, "a site admin is exempt, so not counted")
}

func TestGetOwnedOrgsRequiringTwoFactor(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	ownedIDs := func(uid int64) (ids []int64) {
		orgs, err := organization.GetOwnedOrgsRequiringTwoFactor(t.Context(), uid)
		require.NoError(t, err)
		for _, o := range orgs {
			ids = append(ids, o.ID)
		}
		return ids
	}

	assert.Empty(t, ownedIDs(2), "policy off")
	setRequireTwoFactor(t, org, true)
	assert.Equal(t, []int64{3}, ownedIDs(2), "user2 owns org3")
	assert.Empty(t, ownedIDs(4), "user4 is a member, not an owner")
}

func TestHasOrgOrUserVisibleTwoFactorPolicy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	org.Visibility = structs.VisibleTypePrivate
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})

	assert.True(t, organization.HasOrgOrUserVisible(t.Context(), org.AsUser(), member))
	org.RequireTwoFactor = true
	assert.False(t, organization.HasOrgOrUserVisible(t.Context(), org.AsUser(), member), "a blocked member can't see a private org")

	org.Visibility = structs.VisibleTypePublic
	assert.True(t, organization.HasOrgOrUserVisible(t.Context(), org.AsUser(), member), "anyone can see a public org")
}
