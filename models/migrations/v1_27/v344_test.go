// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"testing"

	"gitea.dev/models/migrations/migrationtest"

	_ "gitea.dev/models/user" // registers the user table; fixtures for unregistered tables are skipped

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddRequireTwoFactorToUser(t *testing.T) {
	type User struct {
		ID        int64  `xorm:"pk autoincr"`
		LowerName string `xorm:"UNIQUE NOT NULL"`
		Name      string `xorm:"UNIQUE NOT NULL"`
		Type      int
	}

	x, deferable := migrationtest.PrepareTestEnv(t, 0, new(User))
	defer deferable()

	require.NoError(t, AddRequireTwoFactorToUser(x))

	var users []struct {
		ID               int64
		RequireTwoFactor bool
	}
	require.NoError(t, x.Table("user").Asc("id").Find(&users))
	require.Len(t, users, 2) // guards against the fixtures silently not loading
	for _, u := range users {
		assert.False(t, u.RequireTwoFactor, "user %d", u.ID)
	}
}
