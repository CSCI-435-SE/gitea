// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueCloseReasonPersists(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	other := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issues_model.CloseReasonNone, other.CloseReason) // fixtures set no reason, so this is the column default
	assert.Empty(t, other.CloseReasonText)
	assert.Zero(t, other.CloseDuplicateIssueID)

	other.CloseReason = issues_model.CloseReasonOther
	other.CloseReasonText = "superseded by a new design"
	require.NoError(t, issues_model.UpdateIssueCols(t.Context(), other, "close_reason", "close_reason_text"))

	duplicate := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	duplicate.CloseReason = issues_model.CloseReasonDuplicate
	duplicate.CloseDuplicateIssueID = 1
	require.NoError(t, issues_model.UpdateIssueCols(t.Context(), duplicate, "close_reason", "close_duplicate_issue_id"))

	// reload and compare: bean conditions skip unmapped and zero-valued fields, so they can't catch a missing column
	other = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issues_model.CloseReasonOther, other.CloseReason)
	assert.Equal(t, "superseded by a new design", other.CloseReasonText)

	duplicate = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	assert.Equal(t, issues_model.CloseReasonDuplicate, duplicate.CloseReason)
	assert.EqualValues(t, 1, duplicate.CloseDuplicateIssueID)
}

func TestAllowedCloseReasons(t *testing.T) {
	assert.Equal(t, []issues_model.CloseReason{
		issues_model.CloseReasonCompleted,
		issues_model.CloseReasonNotPlanned,
		issues_model.CloseReasonDuplicate,
		issues_model.CloseReasonOther,
	}, issues_model.AllowedCloseReasons(false))
	assert.Equal(t, []issues_model.CloseReason{
		issues_model.CloseReasonNotPlanned,
		issues_model.CloseReasonDuplicate,
		issues_model.CloseReasonOther,
	}, issues_model.AllowedCloseReasons(true))
}

func TestDefaultCloseReason(t *testing.T) {
	assert.Equal(t, issues_model.CloseReasonCompleted, issues_model.DefaultCloseReason(false))
	assert.Equal(t, issues_model.CloseReasonNotPlanned, issues_model.DefaultCloseReason(true))
	for _, isPull := range []bool{false, true} {
		assert.Contains(t, issues_model.AllowedCloseReasons(isPull), issues_model.DefaultCloseReason(isPull), "isPull %v", isPull)
	}
}

func TestBulkCloseReasons(t *testing.T) {
	assert.Equal(t, []issues_model.CloseReason{issues_model.CloseReasonCompleted, issues_model.CloseReasonNotPlanned}, issues_model.BulkCloseReasons(false))
	assert.Equal(t, []issues_model.CloseReason{issues_model.CloseReasonNotPlanned}, issues_model.BulkCloseReasons(true))
	for _, isPull := range []bool{false, true} {
		assert.Contains(t, issues_model.BulkCloseReasons(isPull), issues_model.DefaultCloseReason(isPull), "isPull %v: the bulk button starts on it", isPull)
	}
	// it removes from the list it is built from, which must stay whole for the item page
	assert.Equal(t, []issues_model.CloseReason{issues_model.CloseReasonCompleted, issues_model.CloseReasonNotPlanned, issues_model.CloseReasonDuplicate, issues_model.CloseReasonOther}, issues_model.AllowedCloseReasons(false))
}

func TestCloseReasonNames(t *testing.T) {
	cases := []struct {
		name string
		want issues_model.CloseReason
	}{
		{name: "", want: issues_model.CloseReasonNone},
		{name: "completed", want: issues_model.CloseReasonCompleted},
		{name: "not_planned", want: issues_model.CloseReasonNotPlanned},
		{name: "duplicate", want: issues_model.CloseReasonDuplicate},
		{name: "other", want: issues_model.CloseReasonOther},
		{name: "2", want: issues_model.CloseReasonUnknown}, // the stored number is not accepted
		{name: "Completed", want: issues_model.CloseReasonUnknown},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, issues_model.AsCloseReason(c.name), "name %q", c.name)
	}

	for _, reason := range issues_model.AllowedCloseReasons(false) { // an issue allows every reason
		assert.NotEmpty(t, reason.String(), "reason %d", reason)
		assert.Equal(t, reason, issues_model.AsCloseReason(reason.String()), "reason %d", reason)
	}
	assert.Empty(t, issues_model.CloseReasonNone.String())
	assert.Empty(t, issues_model.CloseReasonUnknown.String())
}

func TestCloseReasonOptionsValidate(t *testing.T) {
	cases := []struct {
		name   string
		opts   issues_model.CloseReasonOptions
		isPull bool
		isErr  func(error) bool // nil when the options are valid
	}{
		{name: "no reason", opts: issues_model.CloseReasonOptions{}},
		{name: "no reason on a pull request", opts: issues_model.CloseReasonOptions{}, isPull: true},
		{name: "completed issue", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}},
		{name: "not planned pull request", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonNotPlanned}, isPull: true},
		{name: "duplicate with a number", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 3}},
		{name: "other with text", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: "superseded"}},
		{name: "other with 255 two-byte characters", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: strings.Repeat("é", 255)}},
		{name: "whitespace-only text with another reason", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted, Text: " "}},

		{name: "completed pull request", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}, isPull: true, isErr: issues_model.IsErrCloseReasonNotAllowed},
		{name: "out of range", opts: issues_model.CloseReasonOptions{Reason: 99}, isErr: issues_model.IsErrCloseReasonNotAllowed},
		{name: "other with no text", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "other with whitespace-only text", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: " \t\n"}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "other with 256 characters", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: strings.Repeat("a", 256)}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "text with another reason", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonNotPlanned, Text: "because"}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "text with no reason", opts: issues_model.CloseReasonOptions{Text: "because"}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "duplicate with no number", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate}, isErr: issues_model.IsErrInvalidCloseDuplicate},
		{name: "duplicate with a negative number", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: -1}, isErr: issues_model.IsErrInvalidCloseDuplicate},
		{name: "number with another reason", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonNotPlanned, DuplicateIndex: 3}, isErr: issues_model.IsErrInvalidCloseDuplicate},
		{name: "number with no reason", opts: issues_model.CloseReasonOptions{DuplicateIndex: 3}, isErr: issues_model.IsErrInvalidCloseDuplicate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.opts.Validate(c.isPull)
			if c.isErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.True(t, c.isErr(err), "unexpected error: %v", err)
			assert.ErrorIs(t, err, util.ErrInvalidArgument)
		})
	}
}

func TestCloseIssueRecordsReason(t *testing.T) {
	cases := []struct {
		name     string
		reason   issues_model.CloseReasonOptions
		wantText string
	}{
		{name: "no reason", reason: issues_model.CloseReasonOptions{}},
		{name: "completed", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}},
		{name: "other with text", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: "superseded"}, wantText: "superseded"},
		{name: "other text is stored as entered", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: "  superseded by #4  "}, wantText: "  superseded by #4  "},
		{name: "whitespace-only text is not stored", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonNotPlanned, Text: " "}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
			doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			comment, err := issues_model.CloseIssue(t.Context(), issue, doer, c.reason)
			require.NoError(t, err)

			want := issues_model.CloseReasonOptions{Reason: c.reason.Reason, Text: c.wantText}
			issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
			assert.True(t, issue.IsClosed)
			assert.Equal(t, want, issues_model.CloseReasonOptions{Reason: issue.CloseReason, Text: issue.CloseReasonText})

			comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID}) // reloaded, so the metadata went through JSON
			assert.Equal(t, issues_model.CommentTypeClose, comment.Type)
			assert.Equal(t, want, comment.MetaCloseReason())
		})
	}
}

func TestCloseIssueInvalidReasonLeavesItOpen(t *testing.T) {
	cases := []struct {
		name    string
		issueID int64
		reason  issues_model.CloseReasonOptions
		isErr   func(error) bool
	}{
		{name: "pull request closed as completed", issueID: 3, reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}, isErr: issues_model.IsErrCloseReasonNotAllowed}, // an open, unmerged pull request
		{name: "other with no text", issueID: 1, reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther}, isErr: issues_model.IsErrInvalidCloseReasonText},
		{name: "duplicate of itself", issueID: 1, reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 1}, isErr: issues_model.IsErrInvalidCloseDuplicate},
		{name: "duplicate of a number that does not exist", issueID: 1, reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 999}, isErr: issues_model.IsErrInvalidCloseDuplicate},
		{name: "duplicate of a number only in another repository", issueID: 6, reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 3}, isErr: issues_model.IsErrInvalidCloseDuplicate}, // repo 3 has no #3; repo 1 does
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			_, err := issues_model.CloseIssue(t.Context(), issue, doer, c.reason)
			assert.True(t, c.isErr(err), "unexpected error: %v", err)
			assert.False(t, issue.IsClosed, "the in-memory issue must not be changed either")

			issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			assert.False(t, issue.IsClosed)
			assert.Equal(t, issues_model.CloseReasonNone, issue.CloseReason)
			unittest.AssertNotExistsBean(t, &issues_model.Comment{IssueID: c.issueID, Type: issues_model.CommentTypeClose})
		})
	}
}

func TestCloseIssueAsDuplicate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	target := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: issue.RepoID, Index: 4}) // closed, which is allowed
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	comment, err := issues_model.CloseIssue(t.Context(), issue, doer, issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 4})
	require.NoError(t, err)

	issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issues_model.CloseReasonDuplicate, issue.CloseReason)
	assert.Equal(t, target.ID, issue.CloseDuplicateIssueID) // stored as the global ID, not the number

	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID})
	assert.Equal(t, issues_model.CloseReasonDuplicate, comment.MetaCloseReason().Reason)
	require.NoError(t, comment.LoadCloseDuplicateIssue(t.Context()))
	require.NotNil(t, comment.CloseDuplicateIssue)
	assert.Equal(t, target.ID, comment.CloseDuplicateIssue.ID)
	assert.Equal(t, int64(4), comment.CloseDuplicateIssue.Index)
	assert.Equal(t, target.Title, comment.CloseDuplicateIssue.Title)
	assert.True(t, strings.HasSuffix(comment.CloseDuplicateIssue.Link(), "/user2/repo1/issues/4"), comment.CloseDuplicateIssue.Link())
}

func TestLoadCloseDuplicateIssueDeletedTarget(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	comment := &issues_model.Comment{CommentMetaData: &issues_model.CommentMetaData{CloseReason: issues_model.CloseReasonDuplicate, CloseDuplicateIssueID: 999999}}
	assert.True(t, issues_model.IsErrIssueNotExist(comment.LoadCloseDuplicateIssue(t.Context())))
	assert.Nil(t, comment.CloseDuplicateIssue)
}

func TestReopenIssueClearsReason(t *testing.T) {
	cases := []struct {
		name   string
		reason issues_model.CloseReasonOptions
	}{
		{name: "other with text", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonOther, Text: "superseded"}},
		{name: "duplicate", reason: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
			doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			closeComment, err := issues_model.CloseIssue(t.Context(), issue, doer, c.reason)
			require.NoError(t, err)
			reopenComment, err := issues_model.ReopenIssue(t.Context(), issue, doer)
			require.NoError(t, err)

			issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
			assert.False(t, issue.IsClosed)
			assert.Equal(t, issues_model.CloseReasonNone, issue.CloseReason)
			assert.Empty(t, issue.CloseReasonText)
			assert.Zero(t, issue.CloseDuplicateIssueID)

			closeComment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: closeComment.ID})
			assert.Equal(t, c.reason.Reason, closeComment.MetaCloseReason().Reason, "the close comment keeps its reason")
			assert.Equal(t, c.reason.Text, closeComment.MetaCloseReason().Text)
			require.NoError(t, closeComment.LoadCloseDuplicateIssue(t.Context()))
			assert.Equal(t, c.reason.Reason == issues_model.CloseReasonDuplicate, closeComment.CloseDuplicateIssue != nil)

			reopenComment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: reopenComment.ID})
			assert.Equal(t, issues_model.CommentTypeReopen, reopenComment.Type)
			assert.Nil(t, reopenComment.CommentMetaData)
		})
	}
}

func TestCloseReasonOptionsDuplicateIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	target := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: 4})

	id, err := issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}.DuplicateIssueID(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Zero(t, id) // only the duplicate reason names a target

	dup := issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 4}
	id, err = dup.DuplicateIssueID(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, target.ID, id) // the global ID, not the number

	id, err = dup.DuplicateIssueID(t.Context(), 1, 0) // an item not created yet has no number of its own
	require.NoError(t, err)
	assert.Equal(t, target.ID, id)

	_, err = dup.DuplicateIssueID(t.Context(), 1, 4)
	assert.True(t, issues_model.IsErrInvalidCloseDuplicate(err), "an item cannot duplicate itself")

	_, err = dup.DuplicateIssueID(t.Context(), 2, 1) // repo 2 has no #4
	assert.True(t, issues_model.IsErrInvalidCloseDuplicate(err), "the target must be in the same repository")
}
