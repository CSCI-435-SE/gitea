// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseIssueRecordsDefaultReason(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	cases := []struct {
		name    string
		issueID int64
		want    issues_model.CloseReason
	}{
		{"issue", 1, issues_model.CloseReasonCompleted},
		{"pull request", 3, issues_model.CloseReasonNotPlanned}, // pull request 2, open
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			require.NoError(t, CloseIssue(t.Context(), issue, doer, ""))

			issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			assert.True(t, issue.IsClosed)
			assert.Equal(t, c.want, issue.CloseReason)
			comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: c.issueID, Type: issues_model.CommentTypeClose})
			assert.Equal(t, c.want, comment.MetaCloseReason().Reason) // what the timeline shows
		})
	}
}
