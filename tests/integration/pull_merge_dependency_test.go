// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	auth_model "gitea.dev/models/auth"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	pull_model "gitea.dev/models/pull"
	repo_model "gitea.dev/models/repo"
	unit_model "gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/commitstatus"
	"gitea.dev/modules/gitrepo"
	"gitea.dev/modules/queue"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/test"
	"gitea.dev/services/automerge"
	"gitea.dev/services/forms"
	issue_service "gitea.dev/services/issue"
	commitstatus_service "gitea.dev/services/repository/commitstatus"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setDependenciesEnabled(t *testing.T, repoID int64, enabled bool) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
	issuesUnit, err := repo.GetUnit(t.Context(), unit_model.TypeIssues)
	require.NoError(t, err)
	issuesUnit.IssuesConfig().EnableDependencies = enabled
	require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))
}

// preparePullBlockedByClosedPull makes pull request 2 (#3 in user2/repo1) depend on pull request 5 (#5),
// closes #5 without merging it, and waits until nothing but the dependency can stop #3 from merging
func preparePullBlockedByClosedPull(t *testing.T) (owner *user_model.User, pr *issues_model.PullRequest) {
	owner = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadIssue(t.Context()))
	dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 11})

	setDependenciesEnabled(t, pr.BaseRepoID, true) // fixture repos leave it unset, which reads as off
	require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, pr.Issue, dep))
	require.NoError(t, issue_service.CloseIssue(t.Context(), dep, owner, ""))

	// the server rechecks every pull request on start; until that ends a merge is refused as "not ready"
	require.Eventually(t, func() bool {
		return unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).Status == issues_model.PullRequestStatusMergeable
	}, 5*time.Second, 50*time.Millisecond)
	return owner, pr
}

func TestPullMergeBlockedByDependency(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	owner, pr := preparePullBlockedByClosedPull(t)

	session := loginUser(t, owner.Name)
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/pulls/%d/merge", pr.Index), map[string]string{"do": string(repo_model.MergeStyleMerge)})
	resp := session.MakeRequest(t, req, http.StatusBadRequest)
	assert.Equal(t, "This pull request can't be merged until the pull requests it depends on are merged and the issues it depends on are closed.", test.ParseJSONError(resp.Body.Bytes()).ErrorMessage)
	assert.False(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).HasMerged)
}

func TestAPIMergePullBlockedByDependency(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	owner, pr := preparePullBlockedByClosedPull(t)

	token := getUserToken(t, owner.Name, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/user2/repo1/pulls/%d/merge", pr.Index), &forms.MergePullRequestForm{
		Do: string(repo_model.MergeStyleMerge),
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusMethodNotAllowed)
	assert.Equal(t, "This pull request can't be merged until the pull requests it depends on are merged and the issues it depends on are closed.", DecodeJSON(t, resp, &api.APIError{}).Message)
	assert.False(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).HasMerged)
}

func TestPullAutoMergeBlockedByDependency(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		owner, pr := preparePullBlockedByClosedPull(t)
		scheduled, err := automerge.ScheduleAutoMerge(t.Context(), owner, pr, repo_model.MergeStyleMerge, "auto merge test", false)
		require.NoError(t, err)
		require.True(t, scheduled)

		baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pr.BaseRepoID})
		gitRepo, err := gitrepo.OpenRepository(t.Context(), baseRepo)
		require.NoError(t, err)
		sha, err := gitRepo.GetRefCommitID(pr.GetGitHeadRefName())
		gitRepo.Close()
		require.NoError(t, err)
		checksSucceed := func(statusContext string) { // then waits until the scheduled merge has run
			require.NoError(t, commitstatus_service.CreateCommitStatus(t.Context(), baseRepo, owner, sha, &git_model.CommitStatus{
				State:   commitstatus.CommitStatusSuccess,
				Context: statusContext,
			}))
			require.NoError(t, queue.GetManager().FlushAll(t.Context(), 5*time.Second))
		}

		checksSucceed("ci/first")
		assert.False(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).HasMerged)
		assert.False(t, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: pr.IssueID}).IsClosed)
		unittest.AssertExistsAndLoadBean(t, &pull_model.AutoMerge{PullID: pr.ID}) // still scheduled, so it is tried again on the next change

		// with dependencies turned off the same scheduled merge goes through, so only the dependency stopped it above
		setDependenciesEnabled(t, pr.BaseRepoID, false)
		checksSucceed("ci/second")
		assert.True(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).HasMerged)
	})
}

func TestPullCloseWithUnmergedDependency(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	owner, pr := preparePullBlockedByClosedPull(t)

	testIssueClose(t, loginUser(t, owner.Name), owner.Name, "repo1", strconv.FormatInt(pr.Index, 10))
	assert.True(t, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: pr.IssueID}).IsClosed)
}
