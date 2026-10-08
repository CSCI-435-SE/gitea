// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	api "gitea.dev/modules/structs"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPICloseReason(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue, auth_model.AccessTokenScopeWriteRepository)
	issuesURL := fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name)
	pullsURL := fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner.Name, repo.Name)
	closed := "closed"
	open := "open"

	newIssue := func(t *testing.T, title string) int64 {
		req := NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: title}).AddTokenAuth(token)
		return DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).Index
	}
	load := func(t *testing.T, index int64) *issues_model.Issue {
		return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: index})
	}

	t.Run("IssueWithReason", func(t *testing.T) { // AC4, AC7
		index := newIssue(t, "close as other")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{
			State: &closed, CloseReason: "other", CloseReasonText: "moved to the forum",
		}).AddTokenAuth(token)
		apiIssue := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		assert.Equal(t, "other", apiIssue.CloseReason)
		assert.Equal(t, "moved to the forum", apiIssue.CloseReasonText)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d", issuesURL, index)).AddTokenAuth(token)
		apiIssue = DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &api.Issue{})
		assert.Equal(t, "other", apiIssue.CloseReason) // read back
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: apiIssue.ID, Type: issues_model.CommentTypeClose})
		assert.Equal(t, issues_model.CloseReasonOther, comment.MetaCloseReason().Reason) // the timeline shows it
	})

	t.Run("IssueDuplicate", func(t *testing.T) {
		index := newIssue(t, "close as duplicate")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{
			State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 1,
		}).AddTokenAuth(token)
		apiIssue := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		assert.Equal(t, "duplicate", apiIssue.CloseReason)
		assert.Equal(t, int64(1), apiIssue.CloseDuplicateOf)
	})

	t.Run("IssueInvalid", func(t *testing.T) { // AC5
		index := newIssue(t, "stays open")
		cases := map[string]struct {
			opts api.EditIssueOption
			msg  string
		}{
			"other without text":   {api.EditIssueOption{State: &closed, CloseReason: "other"}, "close_reason_text: text is empty"},
			"other too long":       {api.EditIssueOption{State: &closed, CloseReason: "other", CloseReasonText: strings.Repeat("a", 256)}, "close_reason_text: text is longer than 255"},
			"duplicate missing":    {api.EditIssueOption{State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 999}, "invalid duplicate target"},
			"duplicate of itself":  {api.EditIssueOption{State: &closed, CloseReason: "duplicate", CloseDuplicateOf: index}, "invalid duplicate target"},
			"unknown name":         {api.EditIssueOption{State: &closed, CloseReason: "wontfix"}, "is not allowed for an issue"},
			"text with completed":  {api.EditIssueOption{State: &closed, CloseReason: "completed", CloseReasonText: "why"}, "close_reason_text: text is only allowed with the other reason"},
			"reason without state": {api.EditIssueOption{CloseReason: "completed"}, "only allowed when the request closes"},
			"reason while opening": {api.EditIssueOption{State: &open, CloseReason: "completed"}, "only allowed when the request closes"},
		}
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				opts := c.opts
				opts.Title = "changed by a rejected request"
				req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &opts).AddTokenAuth(token)
				resp := MakeRequest(t, req, http.StatusUnprocessableEntity)
				assert.Contains(t, resp.Body.String(), c.msg)
				issue := load(t, index)
				assert.False(t, issue.IsClosed)
				assert.Equal(t, "stays open", issue.Title) // nothing in the request was applied
			})
		}
	})

	t.Run("AlreadyClosed", func(t *testing.T) {
		req := NewRequestWithJSON(t, "PATCH", issuesURL+"/4", &api.EditIssueOption{State: &closed, CloseReason: "not_planned"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		assert.Equal(t, issues_model.CloseReasonNone, load(t, 4).CloseReason) // the fixture's closed issue keeps no reason
	})

	t.Run("Pull", func(t *testing.T) { // AC4, AC5 on a pull request
		req := NewRequestWithJSON(t, "PATCH", pullsURL+"/3", &api.EditPullRequestOption{State: &closed, CloseReason: "completed"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		assert.False(t, load(t, 3).IsClosed)

		req = NewRequestWithJSON(t, "PATCH", pullsURL+"/3", &api.EditPullRequestOption{
			State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 1,
		}).AddTokenAuth(token)
		apiPull := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.PullRequest{})
		assert.Equal(t, "duplicate", apiPull.CloseReason)
		assert.Equal(t, int64(1), apiPull.CloseDuplicateOf)
	})

	t.Run("Defaults", func(t *testing.T) { // AC6
		index := newIssue(t, "closed by an old script")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{State: &closed}).AddTokenAuth(token)
		assert.Equal(t, "completed", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "PATCH", pullsURL+"/5", &api.EditPullRequestOption{State: &closed}).AddTokenAuth(token)
		assert.Equal(t, "not_planned", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.PullRequest{}).CloseReason)
	})

	t.Run("CreateClosed", func(t *testing.T) { // AC5, AC6 on create
		req := NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "starts closed", Closed: true}).AddTokenAuth(token)
		assert.Equal(t, "completed", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "starts not planned", Closed: true, CloseReason: "not_planned"}).AddTokenAuth(token)
		assert.Equal(t, "not_planned", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "rejected create", Closed: true, CloseReason: "other"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "rejected create", CloseReason: "completed"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Title: "rejected create"}) // nothing was created
	})

	t.Run("NoReasonReadsEmpty", func(t *testing.T) { // AC7
		for _, index := range []int64{1, 4} { // open, and closed before close reasons existed
			req := NewRequest(t, "GET", fmt.Sprintf("%s/%d", issuesURL, index)).AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Body.String(), `"close_reason":""`)
		}
	})
}
