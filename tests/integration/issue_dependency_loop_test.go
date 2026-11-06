// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	api "gitea.dev/modules/structs"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repo1Issue(t *testing.T, index int64) *issues_model.Issue {
	return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: index})
}

func TestWebAddIssueDependencyLoop(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	enableRepoDependencies(t, 1) // fixture repos leave it unset, which reads as off
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	issue := func(index int64) *issues_model.Issue { return repo1Issue(t, index) }
	// a new session for each case, so a message left from an earlier case can't pass for this one
	assertRefusedOnWeb := func(t *testing.T, target, dep *issues_model.Issue) {
		session := loginUser(t, owner.Name)
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/%d/dependency/add", target.Index), map[string]string{
			"newDependency": strconv.FormatInt(dep.ID, 10),
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		flash := session.GetCookieFlashMessage()
		require.NotNil(t, flash)
		assert.Equal(t, "You cannot add this dependency, because the issues would block each other in a loop.", flash.ErrorMsg)
		unittest.AssertNotExistsBean(t, &issues_model.IssueDependency{IssueID: target.ID, DependencyID: dep.ID})
	}

	t.Run("LoopOfThree", func(t *testing.T) {
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, issue(1), issue(2)))
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, issue(2), issue(3)))
		assertRefusedOnWeb(t, issue(3), issue(1))
	})
	t.Run("LoopOfTwo", func(t *testing.T) {
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, issue(4), issue(5)))
		assertRefusedOnWeb(t, issue(5), issue(4))
	})
}

func TestAPIAddIssueDependencyLoop(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	enableRepoDependencies(t, 1)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// #1 on #2 and #2 on #3 for the loop of three, #4 on #5 alone for the loop of two
	for _, dep := range [][2]int64{{1, 2}, {2, 3}, {4, 5}} {
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, repo1Issue(t, dep[0]), repo1Issue(t, dep[1])))
	}
	token := getUserToken(t, owner.Name, auth_model.AccessTokenScopeWriteIssue)
	add := func(t *testing.T, endpoint string, pathIndex, bodyIndex int64, expectedStatus int) {
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/user2/repo1/issues/%d/%s", pathIndex, endpoint),
			&api.IssueMeta{Owner: "user2", Name: "repo1", Index: bodyIndex}).AddTokenAuth(token)
		MakeRequest(t, req, expectedStatus)
	}

	// "dependencies" makes the path issue depend on the body issue; "blocks" makes the body issue depend on the path issue
	for _, c := range []struct {
		name                 string
		endpoint             string
		pathIndex, bodyIndex int64
		expectedStatus       int
	}{
		{"LoopOfThree", "dependencies", 3, 1, http.StatusUnprocessableEntity},
		{"LoopOfThreeBlocks", "blocks", 1, 3, http.StatusUnprocessableEntity},
		{"LoopOfTwo", "dependencies", 5, 4, http.StatusUnprocessableEntity},
		{"LoopOfTwoBlocks", "blocks", 4, 5, http.StatusUnprocessableEntity},
		{"SelfDependency", "dependencies", 1, 1, http.StatusUnprocessableEntity},
		{"SelfDependencyBlocks", "blocks", 1, 1, http.StatusUnprocessableEntity},
		{"Duplicate", "dependencies", 1, 2, http.StatusConflict},
		{"DuplicateBlocks", "blocks", 2, 1, http.StatusConflict},
	} {
		t.Run(c.name, func(t *testing.T) {
			add(t, c.endpoint, c.pathIndex, c.bodyIndex, c.expectedStatus)
			unittest.AssertCount(t, &issues_model.IssueDependency{}, 3) // only the three from the setup
		})
	}

	t.Run("NoLoop", func(t *testing.T) { // #2 and #4 both depend on #3
		add(t, "dependencies", 4, 3, http.StatusCreated)
		unittest.AssertExistsAndLoadBean(t, &issues_model.IssueDependency{IssueID: repo1Issue(t, 4).ID, DependencyID: repo1Issue(t, 3).ID})
	})
}
