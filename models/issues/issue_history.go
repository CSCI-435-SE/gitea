// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/modules/timeutil"
)

// IssueLifetime records when an issue was opened and, if it is closed, when it was
// last closed. It is the minimum needed to chart a repository's open-issue count.
//
// Reopening overwrites ClosedUnix, so a reopened issue keeps only its most recent
// close and its earlier closed periods cannot be recovered from this table.
type IssueLifetime struct {
	CreatedUnix timeutil.TimeStamp
	ClosedUnix  timeutil.TimeStamp
	IsClosed    bool
}

// GetRepoIssueLifetimes returns the lifetimes of every issue in a repository, oldest
// first. Pull requests are excluded, so the result counts issues only.
func GetRepoIssueLifetimes(ctx context.Context, repoID int64) ([]IssueLifetime, error) {
	lifetimes := make([]IssueLifetime, 0, 64)
	return lifetimes, db.GetEngine(ctx).
		Table("issue").
		Select("created_unix, closed_unix, is_closed").
		Where("repo_id = ?", repoID).
		And("is_pull = ?", false).
		OrderBy("created_unix ASC").
		Find(&lifetimes)
}
