// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"cmp"
	"context"
	"slices"

	"gitea.dev/models/db"
	"gitea.dev/modules/timeutil"

	"xorm.io/builder"
)

// MilestoneItem is the current state of an issue or pull request that is, or once was, in a milestone.
// Pull requests are returned too; whether they count is the caller's choice.
type MilestoneItem struct {
	ID          int64
	Index       int64  // the number in the item's URL, for listing what changed
	Title       string `xorm:"name"`
	IsPull      bool
	MilestoneID int64
	IsClosed    bool
	CreatedUnix timeutil.TimeStamp
	ClosedUnix  timeutil.TimeStamp
}

// MilestoneEvent is a comment that moved an item into or out of a milestone, or closed or reopened it.
type MilestoneEvent struct {
	ID             int64 // comment id, the tie-breaker for events in the same second
	IssueID        int64
	Type           CommentType
	OldMilestoneID int64
	MilestoneID    int64
	CreatedUnix    timeutil.TimeStamp
}

// milestoneHistoryChunk bounds each IN list, well under the parameter limits of SQLite and MSSQL
const milestoneHistoryChunk = 500

const milestoneEventCols = "comment.id, comment.issue_id, comment.type, comment.old_milestone_id, comment.milestone_id, comment.created_unix"

// GetMilestoneScopeEvents returns every milestone change into or out of a milestone, oldest first.
// The milestone columns of comment are not indexed and comment has no repo_id, so the scan is bounded
// by the indexed type column and a join on the repository's issues.
func GetMilestoneScopeEvents(ctx context.Context, repoID, milestoneID int64) ([]MilestoneEvent, error) {
	events := make([]MilestoneEvent, 0, 16)
	return events, db.GetEngine(ctx).
		Table("comment").
		Select(milestoneEventCols).
		Join("INNER", "issue", "issue.id = comment.issue_id").
		Where(builder.Eq{"issue.repo_id": repoID, "comment.type": CommentTypeMilestone}).
		And(builder.Or(builder.Eq{"comment.milestone_id": milestoneID}, builder.Eq{"comment.old_milestone_id": milestoneID})).
		OrderBy("comment.created_unix ASC, comment.id ASC").
		Find(&events)
}

// GetMilestoneStateEvents returns the close, merge and reopen comments of the given issues, oldest first.
func GetMilestoneStateEvents(ctx context.Context, issueIDs []int64) ([]MilestoneEvent, error) {
	events := make([]MilestoneEvent, 0, len(issueIDs))
	for chunk := range slices.Chunk(issueIDs, milestoneHistoryChunk) {
		if err := db.GetEngine(ctx).
			Table("comment").
			Select(milestoneEventCols).
			Where(builder.In("comment.type", CommentTypeClose, CommentTypeMergePull, CommentTypeReopen)).
			And(builder.In("comment.issue_id", chunk)).
			Find(&events); err != nil {
			return nil, err
		}
	}
	// sort once in Go: an ORDER BY per chunk would still leave the chunks unordered with each other
	slices.SortFunc(events, CompareMilestoneEvents)
	return events, nil
}

// CompareMilestoneEvents orders events by time, then by comment id, which is insertion order
func CompareMilestoneEvents(a, b MilestoneEvent) int {
	return cmp.Or(cmp.Compare(a.CreatedUnix, b.CreatedUnix), cmp.Compare(a.ID, b.ID))
}

// GetMilestoneItems returns the issues and pull requests in a milestone now, plus extraIssueIDs, which
// are the items that have left it since. Items from another repository are never returned.
func GetMilestoneItems(ctx context.Context, repoID, milestoneID int64, extraIssueIDs []int64) ([]MilestoneItem, error) {
	items := make([]MilestoneItem, 0, 16)
	cols := "id, `index`, name, is_pull, milestone_id, is_closed, created_unix, closed_unix"
	if err := db.GetEngine(ctx).Table("issue").Select(cols).
		Where(builder.Eq{"repo_id": repoID, "milestone_id": milestoneID}).
		Find(&items); err != nil {
		return nil, err
	}
	for chunk := range slices.Chunk(extraIssueIDs, milestoneHistoryChunk) {
		if err := db.GetEngine(ctx).Table("issue").Select(cols).
			Where(builder.Eq{"repo_id": repoID}).
			And(builder.Neq{"milestone_id": milestoneID}). // already loaded above
			And(builder.In("id", chunk)).
			Find(&items); err != nil {
			return nil, err
		}
	}
	return items, nil
}
