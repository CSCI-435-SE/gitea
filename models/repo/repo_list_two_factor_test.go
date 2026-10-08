// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/db"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// user4 reaches org3's private repo3 through team1, and the public repo32 like anyone else
func TestAccessibleReposTwoFactorPolicy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})

	accessibleIDs := func() []int64 {
		ids, err := repo_model.SearchRepositoryIDsByCondition(ctx, repo_model.AccessibleRepositoryCondition(member, unit.TypeInvalid))
		require.NoError(t, err)
		return ids
	}
	myRepoIDs := func() (ids []int64) { // the "my repositories" branch, which skips AccessibleRepositoryCondition
		repos, _, err := repo_model.SearchRepository(ctx, repo_model.SearchRepoOptions{
			ListOptions: db.ListOptionsAll,
			Actor:       member,
			OwnerID:     member.ID,
			Private:     true,
		})
		require.NoError(t, err)
		for _, r := range repos {
			ids = append(ids, r.ID)
		}
		return ids
	}

	assert.Contains(t, accessibleIDs(), int64(3))
	assert.Contains(t, myRepoIDs(), int64(3))

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	org.RequireTwoFactor = true
	require.NoError(t, user_model.UpdateUserCols(ctx, org, "require_two_factor"))

	assert.NotContains(t, accessibleIDs(), int64(3))
	assert.Contains(t, accessibleIDs(), int64(32), "public repos stay visible, as for any non-member")
	assert.NotContains(t, myRepoIDs(), int64(3))

	tfa := &auth_model.TwoFactor{UID: member.ID}
	require.NoError(t, tfa.SetSecret("JBSWY3DPEHPK3PXP"))
	require.NoError(t, auth_model.NewTwoFactor(ctx, tfa))

	assert.Contains(t, accessibleIDs(), int64(3))
	assert.Contains(t, myRepoIDs(), int64(3))
}

// The dashboard's "issues I created / was assigned / was mentioned in" conditions only match public repos.
// Anyone may read those, so the policy must only drop them when the owner org is private.
func TestPublicRepoIssueCondsTwoFactorPolicy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()
	const poster = 15 // posted issue17 in org3's public repo32, has no 2FA

	createdIssueRepoIDs := func() []int64 {
		ids, err := repo_model.SearchRepositoryIDsByCondition(ctx, repo_model.UserCreateIssueRepoCond("`repository`.id", poster, false))
		require.NoError(t, err)
		return ids
	}

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	org.RequireTwoFactor = true
	require.NoError(t, user_model.UpdateUserCols(ctx, org, "require_two_factor"))
	assert.Contains(t, createdIssueRepoIDs(), int64(32), "a public repo of a public org stays visible to users without 2FA")

	org.Visibility = structs.VisibleTypePrivate
	require.NoError(t, user_model.UpdateUserCols(ctx, org, "visibility"))
	assert.NotContains(t, createdIssueRepoIDs(), int64(32), "a private org's repos are hidden from blocked users")
}
