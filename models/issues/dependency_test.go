// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIssueDependencyLoop(t *testing.T) {
	// issues 1, 2 and 3 (repo1) have no dependencies in the fixtures
	create := func(t *testing.T, issueID, depID int64) error {
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issueID})
		dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: depID})
		return issues_model.CreateIssueDependency(t.Context(), doer, issue, dep)
	}
	assertRefused := func(t *testing.T, issueID, depID int64) {
		err := create(t, issueID, depID)
		assert.True(t, issues_model.IsErrCircularDependency(err), "expected a loop error, got %v", err)
		unittest.AssertNotExistsBean(t, &issues_model.IssueDependency{IssueID: issueID, DependencyID: depID})
	}
	assertAccepted := func(t *testing.T, issueID, depID int64) {
		require.NoError(t, create(t, issueID, depID))
		unittest.AssertExistsAndLoadBean(t, &issues_model.IssueDependency{IssueID: issueID, DependencyID: depID})
	}

	t.Run("LoopOfThree", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())
		assertAccepted(t, 1, 2)
		assertAccepted(t, 2, 3)
		assertRefused(t, 3, 1)
	})
	t.Run("LoopOfTwo", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())
		assertAccepted(t, 1, 2)
		assertRefused(t, 2, 1)
	})
	t.Run("SelfDependency", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())
		assertRefused(t, 1, 1)
	})
	t.Run("SharedDependency", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())
		assertAccepted(t, 1, 3)
		assertAccepted(t, 2, 3)
	})
	t.Run("Shortcut", func(t *testing.T) { // 1 already reaches 3 through 2, but 3 does not reach 1
		require.NoError(t, unittest.PrepareTestDatabase())
		assertAccepted(t, 1, 2)
		assertAccepted(t, 2, 3)
		assertAccepted(t, 1, 3)
	})
	t.Run("BesideOldLoop", func(t *testing.T) { // a loop added before this check existed, which 1 does not join
		require.NoError(t, unittest.PrepareTestDatabase())
		require.NoError(t, db.Insert(t.Context(), &issues_model.IssueDependency{UserID: 1, IssueID: 2, DependencyID: 3}))
		require.NoError(t, db.Insert(t.Context(), &issues_model.IssueDependency{UserID: 1, IssueID: 3, DependencyID: 2}))
		assertAccepted(t, 1, 2)
	})
	t.Run("LoopPastFirstBatch", func(t *testing.T) { // one step of the chain holds more issues than one query takes
		require.NoError(t, unittest.PrepareTestDatabase())
		// #2 depends on made-up issue IDs (the check reads only issue_dependency); only those past the first batch lead back to #1
		var rows []any
		for id := int64(1001); id <= 1000+db.DefaultMaxInSize+10; id++ {
			rows = append(rows, &issues_model.IssueDependency{UserID: 1, IssueID: 2, DependencyID: id})
			if id > 1000+db.DefaultMaxInSize {
				rows = append(rows, &issues_model.IssueDependency{UserID: 1, IssueID: id, DependencyID: 1})
			}
		}
		require.NoError(t, db.Insert(t.Context(), rows...))
		assertRefused(t, 1, 2)
	})
}

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
