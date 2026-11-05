// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	unit_model "gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/test"
	"gitea.dev/services/forms"
	issue_service "gitea.dev/services/issue"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// preparePullBlockedByClosedPull makes pull request 2 (#3 in user2/repo1) depend on pull request 5 (#5),
// closes #5 without merging it, and waits until nothing but the dependency can stop #3 from merging
func preparePullBlockedByClosedPull(t *testing.T) (owner *user_model.User, pr *issues_model.PullRequest) {
	owner = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadIssue(t.Context()))
	dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 11})

	issuesUnit, err := repo.GetUnit(t.Context(), unit_model.TypeIssues)
	require.NoError(t, err)
	issuesUnit.IssuesConfig().EnableDependencies = true // fixture repos leave it unset, which reads as off
	require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))
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
