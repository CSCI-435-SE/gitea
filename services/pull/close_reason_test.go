// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/references"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCloseCrossReferencesClosesAsCompleted(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // open, in repo 1
	require.NoError(t, pr.LoadIssue(t.Context()))
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// what "Fixes #1" in the pull request's description records on issue #1
	require.NoError(t, db.Insert(t.Context(), &issues_model.Comment{
		Type:       issues_model.CommentTypePullRef,
		PosterID:   doer.ID,
		IssueID:    1,
		RefRepoID:  pr.Issue.RepoID,
		RefIssueID: pr.Issue.ID,
		RefAction:  references.XRefActionCloses,
		RefIsPull:  true,
	}))

	require.NoError(t, handleCloseCrossReferences(t.Context(), pr, doer))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.True(t, issue.IsClosed)
	assert.Equal(t, issues_model.CloseReasonCompleted, issue.CloseReason)
}

func TestAdjustPullsCausedByBranchDeletedClosesAsNotPlanned(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // open, head branch "branch2" in repo 1
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pr.HeadRepoID})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	require.NoError(t, AdjustPullsCausedByBranchDeleted(t.Context(), doer, repo, pr.HeadBranch))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: pr.IssueID})
	assert.True(t, issue.IsClosed)
	assert.Equal(t, issues_model.CloseReasonNotPlanned, issue.CloseReason)
}
