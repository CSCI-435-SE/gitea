// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
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
		{name: "duplicate issue", opts: issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate}},
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
