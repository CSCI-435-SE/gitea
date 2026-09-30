// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"testing"

	"gitea.dev/models/migrations/migrationtest"

	_ "gitea.dev/models/issues" // registers the issue table; fixtures for unregistered tables are skipped

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddCloseReasonToIssue(t *testing.T) {
	type Issue struct {
		ID       int64 `xorm:"pk autoincr"`
		RepoID   int64 `xorm:"INDEX UNIQUE(repo_index)"`
		Index    int64 `xorm:"UNIQUE(repo_index)"`
		IsClosed bool  `xorm:"INDEX"`
		IsPull   bool  `xorm:"INDEX"`
	}

	x, deferable := migrationtest.PrepareTestEnv(t, 0, new(Issue))
	defer deferable()

	require.NoError(t, AddCloseReasonToIssue(x))

	var issues []struct {
		ID                    int64
		CloseReason           int
		CloseReasonText       string
		CloseDuplicateIssueID int64
	}
	require.NoError(t, x.Table("issue").Asc("id").Find(&issues))
	require.Len(t, issues, 4) // guards against the fixtures silently not loading
	for _, issue := range issues {
		assert.Zero(t, issue.CloseReason, "issue %d", issue.ID)
		assert.Empty(t, issue.CloseReasonText, "issue %d", issue.ID)
		assert.Zero(t, issue.CloseDuplicateIssueID, "issue %d", issue.ID)
	}
}
