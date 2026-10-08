// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/organization"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/session"
	"gitea.dev/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgTwoFactorRequired(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	org.RequireTwoFactor = true
	require.NoError(t, user_model.UpdateUserCols(t.Context(), org, "require_two_factor"))
	for _, userID := range []int64{24, 32} {
		require.NoError(t, db.Insert(t.Context(), &organization.OrgUser{OrgID: org.ID, UID: userID}))
	}

	for _, tc := range []struct {
		name       string
		userID     int64
		session2FA bool
		want       bool
	}{
		{"anonymous", 0, false, false},
		{"member without 2FA", 4, false, true},
		{"stale session flag", 4, true, true},
		{"TOTP enrolled", 24, false, false},
		{"WebAuthn enrolled", 32, false, false},
		{"site admin", 1, false, false},
		{"unrelated user", 5, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := session.NewMockMemStore(t.Name())
			require.NoError(t, store.Set(session.KeyUserHasTwoFactorAuth, tc.session2FA))
			ctx, _ := contexttest.MockContext(t, "/", contexttest.MockContextOption{SessionStore: store})
			if tc.userID != 0 {
				ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: tc.userID})
			}
			assert.Equal(t, tc.want, orgTwoFactorRequired(ctx))
		})
	}
}
