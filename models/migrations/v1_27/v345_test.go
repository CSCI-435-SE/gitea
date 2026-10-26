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

	var after []struct {
		notificationBeforeReason `xorm:"extends"`
		Reason                   int
	}
	require.NoError(t, x.Table("notification").Asc("id").Find(&after))
	require.Len(t, after, len(before))
	for i, n := range after {
		assert.Zero(t, n.Reason, "#52 AC3 notification %d is Other", n.ID)
		assert.Equal(t, before[i], n.notificationBeforeReason, "#52 AC3 nothing else about notification %d changes", n.ID)
	}
}
