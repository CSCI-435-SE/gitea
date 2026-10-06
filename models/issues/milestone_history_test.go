// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertComment writes a bare comment row with a fixed time; no fixture comment has these types
func insertComment(t *testing.T, c *issues_model.Comment) int64 {
	t.Helper()
	_, err := db.GetEngine(t.Context()).NoAutoTime().Insert(c)
	require.NoError(t, err)
	return c.ID
}

func eventIDs(events []issues_model.MilestoneEvent) []int64 {
	ids := make([]int64, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	return ids
}

func TestGetMilestoneScopeEvents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// issue 1 is in repo 1, issue 10 in repo 42; milestone 1 belongs to repo 1
	joined := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMilestone, IssueID: 1, MilestoneID: 1, CreatedUnix: 100})
	sameSecond := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMilestone, IssueID: 2, MilestoneID: 1, CreatedUnix: 100})
	left := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMilestone, IssueID: 1, OldMilestoneID: 1, MilestoneID: 2, CreatedUnix: 200})
	insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMilestone, IssueID: 1, OldMilestoneID: 2, CreatedUnix: 300})                   // another milestone
	insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMilestone, IssueID: 10, MilestoneID: 1, CreatedUnix: 150})                     // another repository
	insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeComment, IssueID: 1, MilestoneID: 1, CreatedUnix: 160, Content: "not a move"}) // another type

	events, err := issues_model.GetMilestoneScopeEvents(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, []int64{joined, sameSecond, left}, eventIDs(events))
	assert.Equal(t, issues_model.CommentTypeMilestone, events[2].Type)
	assert.EqualValues(t, 1, events[2].OldMilestoneID)
	assert.EqualValues(t, 2, events[2].MilestoneID)
	assert.Equal(t, timeutil.TimeStamp(200), events[2].CreatedUnix)

	events, err = issues_model.GetMilestoneScopeEvents(t.Context(), 1, unittest.NonexistentID)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetMilestoneStateEvents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	reopened := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeReopen, IssueID: 2, CreatedUnix: 60})
	closed := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeClose, IssueID: 2, CreatedUnix: 50})
	merged := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeMergePull, IssueID: 2, CreatedUnix: 70})
	otherIssue := insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeClose, IssueID: 3, CreatedUnix: 55})
	insertComment(t, &issues_model.Comment{Type: issues_model.CommentTypeComment, IssueID: 2, CreatedUnix: 65, Content: "not a state change"})

	events, err := issues_model.GetMilestoneStateEvents(t.Context(), []int64{2})
	require.NoError(t, err)
	assert.Equal(t, []int64{closed, reopened, merged}, eventIDs(events))

	t.Run("Chunked", func(t *testing.T) {
		// more ids than one IN list holds, with the two real issues in different chunks
		ids := make([]int64, 0, 1200)
		ids = append(ids, 2)
		for i := range int64(1200) {
			ids = append(ids, unittest.NonexistentID+i)
		}
		ids = append(ids, 3)

		events, err := issues_model.GetMilestoneStateEvents(t.Context(), ids)
		require.NoError(t, err)
		assert.Equal(t, []int64{closed, otherIssue, reopened, merged}, eventIDs(events))
	})

	t.Run("NoIssues", func(t *testing.T) {
		events, err := issues_model.GetMilestoneStateEvents(t.Context(), nil)
		require.NoError(t, err)
		assert.Empty(t, events)
	})
}

func TestGetMilestoneItems(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// issue 2 is a pull request in milestone 1 and is kept, as the milestone's counters keep it
	items, err := issues_model.GetMilestoneItems(t.Context(), 1, 1, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.EqualValues(t, 2, items[0].ID)
	assert.EqualValues(t, 1, items[0].MilestoneID)
	assert.False(t, items[0].IsClosed)

	// issue 1 left the milestone; issue 2 is listed once although it is also passed as extra;
	// issue 10 is in another repository and must not leak in
	items, err = issues_model.GetMilestoneItems(t.Context(), 1, 1, []int64{1, 2, 10})
	require.NoError(t, err)
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	assert.ElementsMatch(t, []int64{1, 2}, ids)
}
