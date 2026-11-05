// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"strconv"
	"testing"
	"time"

	"gitea.dev/models/db"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	"gitea.dev/models/pull"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/graceful"
	"gitea.dev/modules/queue"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/test"
	"gitea.dev/services/automergequeue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequest_AddToTaskQueue(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	idChan := make(chan int64, 10)
	testHandler := func(items ...string) []string {
		for _, s := range items {
			id, _ := strconv.ParseInt(s, 10, 64)
			idChan <- id
		}
		return nil
	}

	cfg, err := setting.GetQueueSettings(setting.CfgProvider, "pr_patch_checker")
	assert.NoError(t, err)
	prPatchCheckerQueue, err = queue.NewWorkerPoolQueueWithContext(t.Context(), "pr_patch_checker", cfg, testHandler, true)
	assert.NoError(t, err)

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	StartPullRequestCheckImmediately(t.Context(), pr)

	assert.Eventually(t, func() bool {
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		return pr.Status == issues_model.PullRequestStatusChecking
	}, 1*time.Second, 100*time.Millisecond)

	has, err := prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.True(t, has)
	assert.NoError(t, err)

	go prPatchCheckerQueue.Run()

	select {
	case id := <-idChan:
		assert.Equal(t, pr.ID, id)
	case <-time.After(time.Second):
		assert.FailNow(t, "Timeout: nothing was added to pullRequestQueue")
	}

	has, err = prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.False(t, has)
	assert.NoError(t, err)

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	assert.Equal(t, issues_model.PullRequestStatusChecking, pr.Status)

	prPatchCheckerQueue.ShutdownWait(time.Second)
	prPatchCheckerQueue = nil
}

func TestCheckSigningRequirementsHeadCommits(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadBaseRepo(ctx))
	require.NoError(t, pr.LoadHeadRepo(ctx))

	check := func() error {
		return checkSigningRequirements(ctx, pr, nil, repo_model.MergeStyleFastForwardOnly)
	}

	// No protected branch rule on the base branch: the check must pass.
	require.NoError(t, check())

	// Protected branch without RequireSignedCommits: the check must still pass.
	require.NoError(t, git_model.UpdateProtectBranch(ctx, pr.BaseRepo, &git_model.ProtectedBranch{
		RepoID:               pr.BaseRepoID,
		RuleName:             pr.BaseBranch,
		RequireSignedCommits: false,
	}, git_model.WhitelistOptions{}))
	require.NoError(t, check())

	// With RequireSignedCommits enabled: the test fixture commits have no signatures,
	// so the check must report ErrHeadCommitsNotAllVerified.
	pb, err := git_model.GetFirstMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	require.NoError(t, err)
	require.NotNil(t, pb)
	pb.RequireSignedCommits = true
	require.NoError(t, git_model.UpdateProtectBranch(ctx, pr.BaseRepo, pb, git_model.WhitelistOptions{}))
	require.ErrorIs(t, check(), ErrHeadCommitsNotAllVerified)
}

func TestMarkPullRequestAsMergeable(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	prPatchCheckerQueue = queue.CreateUniqueQueue(graceful.GetManager().ShutdownContext(), "pr_patch_checker", func(items ...string) []string { return nil })
	go prPatchCheckerQueue.Run()
	defer func() {
		prPatchCheckerQueue.ShutdownWait(time.Second)
		prPatchCheckerQueue = nil
	}()

	addToQueueShaChan := make(chan string, 1)
	defer test.MockVariableValue(&automergequeue.AddToQueue, func(pr *issues_model.PullRequest, sha string) {
		addToQueueShaChan <- sha
	})()
	ctx := t.Context()
	_, _ = db.GetEngine(ctx).ID(2).Update(&issues_model.PullRequest{Status: issues_model.PullRequestStatusChecking})
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.False(t, pr.HasMerged)
	require.Equal(t, issues_model.PullRequestStatusChecking, pr.Status)

	err := pull.ScheduleAutoMerge(ctx, &user_model.User{ID: 99999}, pr.ID, repo_model.MergeStyleMerge, "test msg", true)
	require.NoError(t, err)

	exist, scheduleMerge, err := pull.GetScheduledMergeByPullID(ctx, pr.ID)
	require.NoError(t, err)
	assert.True(t, exist)
	assert.True(t, scheduleMerge.Doer.IsGhost())

	markPullRequestAsMergeable(ctx, pr)
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.Equal(t, issues_model.PullRequestStatusMergeable, pr.Status)

	select {
	case sha := <-addToQueueShaChan:
		assert.Equal(t, "985f0301dba5e7b34be866819cd15ad3d8f508ee", sha) // ref: refs/pull/3/head
	case <-time.After(1 * time.Second):
		assert.FailNow(t, "Timeout: nothing was added to automergequeue")
	}
}

func TestCheckPullMergeableDependencies(t *testing.T) {
	// user2 owns repo1 and could merge pull request 2, which gets one dependency, depID, closed first when closeDep is set
	check := func(t *testing.T, depID int64, closeDep, dependenciesEnabled bool) error {
		require.NoError(t, unittest.PrepareTestDatabase())
		ctx := t.Context()
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		require.NoError(t, pr.LoadIssue(ctx))
		require.NoError(t, pr.LoadBaseRepo(ctx))

		dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: depID})
		if closeDep {
			_, err := issues_model.CloseIssue(ctx, dep, doer, issues_model.CloseReasonOptions{})
			require.NoError(t, err)
		}
		require.NoError(t, issues_model.CreateIssueDependency(ctx, doer, pr.Issue, dep))

		issuesUnit, err := pr.BaseRepo.GetUnit(ctx, unit.TypeIssues)
		require.NoError(t, err)
		issuesUnit.IssuesConfig().EnableDependencies = dependenciesEnabled // fixture repos leave it unset, which reads as off
		require.NoError(t, repo_model.UpdateRepoUnitConfig(ctx, issuesUnit))

		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // reload, so the setting comes from the database
		require.NoError(t, pr.LoadBaseRepo(ctx))
		perm, err := access_model.GetDoerRepoPermission(ctx, pr.BaseRepo, doer)
		require.NoError(t, err)
		return CheckPullMergeable(ctx, doer, &perm, pr, MergeCheckTypeGeneral, repo_model.MergeStyleMerge, false)
	}

	t.Run("OpenIssue", func(t *testing.T) {
		assert.ErrorIs(t, check(t, 1, false, true), ErrDependenciesLeft)
	})
	t.Run("ClosedIssue", func(t *testing.T) {
		assert.NoError(t, check(t, 1, true, true))
	})
	t.Run("PullClosedWithoutMerging", func(t *testing.T) {
		assert.ErrorIs(t, check(t, 11, true, true), ErrDependenciesLeft) // pull request 5
	})
	t.Run("MergedPull", func(t *testing.T) {
		assert.NoError(t, check(t, 2, true, true)) // pull request 1 is merged; closed as a real merge would, the fixture leaves it open
	})
	t.Run("DependenciesTurnedOff", func(t *testing.T) {
		assert.NoError(t, check(t, 11, true, false))
	})
}
