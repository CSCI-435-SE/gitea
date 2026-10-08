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
