// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	auth_model "gitea.dev/models/auth"
	access_model "gitea.dev/models/perm/access"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// user10 is an outside collaborator on org3's public repo32, without 2FA. The policy keeps their allowlist
// entries while they are blocked, so the entries must not let them push or merge.
func TestProtectedBranchAllowlistTwoFactorPolicy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()
	collaborator := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	newRule := func() *ProtectedBranch { // fresh each time, so the repo owner is loaded with the current policy
		return &ProtectedBranch{
			RepoID: 32, CanPush: true,
			EnableWhitelist: true, WhitelistUserIDs: []int64{collaborator.ID},
			EnableMergeWhitelist: true, MergeWhitelistUserIDs: []int64{collaborator.ID},
		}
	}
	allowed := func() (push, merge bool) {
		pb := newRule()
		return pb.CanUserPush(ctx, collaborator), IsUserMergeWhitelisted(ctx, pb, collaborator.ID, access_model.Permission{})
	}

	push, merge := allowed()
	assert.True(t, push)
	assert.True(t, merge)

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	org.RequireTwoFactor = true
	require.NoError(t, user_model.UpdateUserCols(ctx, org, "require_two_factor"))
	push, merge = allowed()
	assert.False(t, push, "a blocked user's push allowlist entry grants nothing")
	assert.False(t, merge, "a blocked user's merge allowlist entry grants nothing")

	tfa := &auth_model.TwoFactor{UID: collaborator.ID}
	require.NoError(t, tfa.SetSecret("JBSWY3DPEHPK3PXP"))
	require.NoError(t, auth_model.NewTwoFactor(ctx, tfa))
	push, merge = allowed()
	assert.True(t, push)
	assert.True(t, merge)
}
