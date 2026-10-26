// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"testing"

	"gitea.dev/models/migrations/migrationtest"

	_ "gitea.dev/models/activities" // registers the notification table; fixtures for unregistered tables are skipped

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notificationBeforeReason struct {
	ID          int64 `xorm:"pk autoincr"`
	UserID      int64
	RepoID      int64
	Status      int
	Source      int
	IssueID     int64
	CommentID   int64
	UpdatedBy   int64
	CreatedUnix int64
	UpdatedUnix int64
}

func (notificationBeforeReason) TableName() string { return "notification" }

func Test_AddReasonToNotification(t *testing.T) {
	x, deferable := migrationtest.PrepareTestEnv(t, 0, new(notificationBeforeReason))
	defer deferable()

	var before []notificationBeforeReason
	require.NoError(t, x.Table("notification").Asc("id").Find(&before))
	require.Len(t, before, 4) // guards against the fixtures silently not loading

	require.NoError(t, AddReasonToNotification(x))

	var after []notificationBeforeReason
	require.NoError(t, x.Table("notification").Asc("id").Find(&after))
	assert.Equal(t, before, after) // #52 AC3 nothing else changes

	var reasons []int
	require.NoError(t, x.Table("notification").Asc("id").Cols("reason").Find(&reasons))
	assert.Equal(t, []int{0, 0, 0, 0}, reasons) // #52 AC3 every existing notification is "Other"
}
