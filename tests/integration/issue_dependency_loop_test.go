// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebAddIssueDependencyLoop(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	enableRepoDependencies(t, 1) // fixture repos leave it unset, which reads as off
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	issue := func(index int64) *issues_model.Issue {
		return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: index})
	}
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
