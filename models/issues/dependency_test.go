// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIssueDependency(t *testing.T) {
	// Prepare
	assert.NoError(t, unittest.PrepareTestDatabase())

	user1, err := user_model.GetUserByID(t.Context(), 1)
	assert.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(t.Context(), 1)
	assert.NoError(t, err)

	issue2, err := issues_model.GetIssueByID(t.Context(), 2)
	assert.NoError(t, err)

	// Create a dependency and check if it was successful
	err = issues_model.CreateIssueDependency(t.Context(), user1, issue1, issue2)
	assert.NoError(t, err)

	// Do it again to see if it will check if the dependency already exists
	err = issues_model.CreateIssueDependency(t.Context(), user1, issue1, issue2)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrDependencyExists(err))

	// Check for circular dependencies
	err = issues_model.CreateIssueDependency(t.Context(), user1, issue2, issue1)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrCircularDependency(err))

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeAddDependency, PosterID: user1.ID, IssueID: issue1.ID})

	// Check if dependencies left is correct
	left, err := issues_model.IssueNoDependenciesLeft(t.Context(), issue1)
	assert.NoError(t, err)
	assert.False(t, left)

	// Close #2 and check again
	_, err = issues_model.CloseIssue(t.Context(), issue2, user1, issues_model.CloseReasonOptions{})
	assert.NoError(t, err)

	issue2Closed, err := issues_model.GetIssueByID(t.Context(), 2)
	assert.NoError(t, err)
	assert.True(t, issue2Closed.IsClosed)

	left, err = issues_model.IssueNoDependenciesLeft(t.Context(), issue1)
	assert.NoError(t, err)
	assert.True(t, left)

	// Test removing the dependency
	err = issues_model.RemoveIssueDependency(t.Context(), user1, issue1, issue2, issues_model.DependencyTypeBlockedBy)
	assert.NoError(t, err)

	_, err = issues_model.ReopenIssue(t.Context(), issue2, user1)
	assert.NoError(t, err)

	issue2Reopened, err := issues_model.GetIssueByID(t.Context(), 2)
	assert.NoError(t, err)
	assert.False(t, issue2Reopened.IsClosed)
}

func TestMergeBlockedByDependencies(t *testing.T) {
	// pull request 2 (issue 3, repo1) gets one dependency, depID, closed first when closeDep is set
	setup := func(t *testing.T, depID int64, closeDep bool) *issues_model.PullRequest {
		require.NoError(t, unittest.PrepareTestDatabase())
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		require.NoError(t, pr.LoadIssue(t.Context()))
		dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: depID})
		if closeDep {
			_, err := issues_model.CloseIssue(t.Context(), dep, doer, issues_model.CloseReasonOptions{})
			require.NoError(t, err)
		}
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), doer, pr.Issue, dep))
		return pr
	}
	assertBlocked := func(t *testing.T, pr *issues_model.PullRequest, expected bool) {
		blocked, err := issues_model.MergeBlockedByDependencies(t.Context(), pr)
		require.NoError(t, err)
		assert.Equal(t, expected, blocked)
	}

	t.Run("OpenIssue", func(t *testing.T) {
		assertBlocked(t, setup(t, 1, false), true)
	})
	t.Run("ClosedIssue", func(t *testing.T) {
		assertBlocked(t, setup(t, 1, true), false)
	})
	t.Run("OpenPull", func(t *testing.T) {
		assertBlocked(t, setup(t, 11, false), true) // pull request 5, not merged
	})
	t.Run("MergedPull", func(t *testing.T) {
		assertBlocked(t, setup(t, 2, true), false) // pull request 1 is merged; closed as a real merge would, the fixture leaves it open
	})
	t.Run("PullClosedWithoutMerging", func(t *testing.T) {
		pr := setup(t, 11, true)
		assertBlocked(t, pr, true)

		// closing pr itself still treats the closed pull request as done
		noDeps, err := issues_model.IssueNoDependenciesLeft(t.Context(), pr.Issue)
		require.NoError(t, err)
		assert.True(t, noDeps)
	})
}

func TestGetMergeBlockingDependencies(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // issue 3, repo 1
	require.NoError(t, pr.LoadIssue(t.Context()))

	// one dependency of each kind in the issue's table, closed first where needed
	for _, dep := range []struct {
		issueID int64
		close   bool
	}{
		{1, false},  // open issue
		{5, false},  // closed issue, closed in the fixtures
		{2, true},   // merged pull request 1; closed as a real merge would, the fixture leaves it open
		{11, true},  // pull request 5, closed without merging
		{12, false}, // open pull request 6, in repo 3
	} {
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: dep.issueID})
		if dep.close {
			_, err := issues_model.CloseIssue(t.Context(), issue, doer, issues_model.CloseReasonOptions{})
			require.NoError(t, err)
		}
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), doer, pr.Issue, issue))
	}

	deps, err := issues_model.GetMergeBlockingDependencies(t.Context(), pr)
	require.NoError(t, err)
	repoByIssue := map[int64]int64{}
	for _, dep := range deps {
		require.NotNil(t, dep.Issue.Repo, "issue %d has no repository loaded", dep.Issue.ID)
		repoByIssue[dep.Issue.ID] = dep.Issue.Repo.ID // the merge box links each dependency to its own repository
	}
	assert.Equal(t, map[int64]int64{1: 1, 11: 1, 12: 3}, repoByIssue)
}
