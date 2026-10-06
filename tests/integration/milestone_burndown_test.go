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
	"gitea.dev/models/unittest"
	api "gitea.dev/modules/structs"
	issue_service "gitea.dev/services/issue"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMilestoneBurndown(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Data", func(t *testing.T) {
		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

		due := time.Now().AddDate(0, 0, 7)
		req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/milestones", api.CreateMilestoneOption{
			Title: "burndown", Deadline: &due,
		}).AddTokenAuth(token)
		milestone := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Milestone{})

		indexes := make([]int64, 0, 2)
		for _, title := range []string{"first", "second"} {
			req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/issues", api.CreateIssueOption{
				Title: title, Milestone: milestone.ID,
			}).AddTokenAuth(token)
			indexes = append(indexes, DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).Index)
		}
		closed := "closed"
		req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/user2/repo1/issues/%d", indexes[0]), api.EditIssueOption{
			State: &closed,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		// anyone who can read the repository's issues can read its burndown
		req = NewRequest(t, "GET", fmt.Sprintf("/user2/repo1/milestone/%d/burndown", milestone.ID))
		burndown := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &issue_service.MilestoneBurndown{})

		// one day of history, or two if the test ran across midnight; too few to project either way
		require.NotEmpty(t, burndown.Points)
		require.LessOrEqual(t, len(burndown.Points), 2)
		today := burndown.Points[len(burndown.Points)-1]
		assert.Equal(t, 1, today.Remaining)
		assert.Equal(t, 2, today.Scope)
		assert.Equal(t, issue_service.BurndownInsufficientData, burndown.Status)
		require.NotNil(t, burndown.Ideal)
		assert.Equal(t, burndown.Points[0].Date, burndown.Ideal.From)
		assert.Equal(t, 1, burndown.Ideal.FromValue)
		assert.Equal(t, burndown.Deadline, burndown.Ideal.To)

		// the chart and the progress bar above it must count the same thing
		row := unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: milestone.ID})
		assert.Equal(t, row.NumIssues-row.NumClosedIssues, today.Remaining)
		assert.Equal(t, row.NumIssues, today.Scope)
	})

	t.Run("RealLifecycle", func(t *testing.T) {
		// close, reopen and move through the API, so the history the chart replays is written by the
		// real code paths rather than built by hand as in the unit tests
		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
		createMilestone := func(title string) int64 {
			req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/milestones", api.CreateMilestoneOption{Title: title}).AddTokenAuth(token)
			return DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Milestone{}).ID
		}
		sprint, other := createMilestone("sprint"), createMilestone("other")

		issues := map[string]*api.Issue{}
		for _, title := range []string{"closed", "reopened", "moved", "open"} {
			req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/issues", api.CreateIssueOption{Title: title, Milestone: sprint}).AddTokenAuth(token)
			issues[title] = DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		}
		edit := func(title string, option api.EditIssueOption) {
			req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/user2/repo1/issues/%d", issues[title].Index), option).AddTokenAuth(token)
			MakeRequest(t, req, http.StatusCreated)
		}
		closed, open := "closed", "open"
		edit("closed", api.EditIssueOption{State: &closed})
		edit("reopened", api.EditIssueOption{State: &closed})
		edit("reopened", api.EditIssueOption{State: &open})
		edit("moved", api.EditIssueOption{Milestone: &other})

		// the reconciliation step would hide missing history in the final counts, so check the
		// history itself: four joins, one move out, two closes and one reopen
		scopeEvents, err := issues_model.GetMilestoneScopeEvents(t.Context(), 1, sprint)
		require.NoError(t, err)
		var joins, leaves int
		for _, e := range scopeEvents {
			if e.MilestoneID == sprint {
				joins++
			} else if e.OldMilestoneID == sprint {
				leaves++
				assert.Equal(t, issues["moved"].ID, e.IssueID)
			}
		}
		assert.Equal(t, 4, joins)
		assert.Equal(t, 1, leaves)

		ids := []int64{issues["closed"].ID, issues["reopened"].ID, issues["moved"].ID, issues["open"].ID}
		stateEvents, err := issues_model.GetMilestoneStateEvents(t.Context(), ids)
		require.NoError(t, err)
		kinds := make([]issues_model.CommentType, 0, len(stateEvents))
		for _, e := range stateEvents {
			kinds = append(kinds, e.Type)
		}
		assert.ElementsMatch(t, []issues_model.CommentType{issues_model.CommentTypeClose, issues_model.CommentTypeClose, issues_model.CommentTypeReopen}, kinds)

		read := func(milestoneID int64) issue_service.BurndownPoint {
			req := NewRequest(t, "GET", fmt.Sprintf("/user2/repo1/milestone/%d/burndown", milestoneID))
			burndown := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &issue_service.MilestoneBurndown{})
			require.NotEmpty(t, burndown.Points)
			return burndown.Points[len(burndown.Points)-1]
		}
		// "closed", "reopened" and "open" are left; only the latter two are open
		today := read(sprint)
		assert.Equal(t, 3, today.Scope)
		assert.Equal(t, 2, today.Remaining)
		row := unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: sprint})
		assert.Equal(t, row.NumIssues, today.Scope)
		assert.Equal(t, row.NumIssues-row.NumClosedIssues, today.Remaining)

		// the moved issue now counts in the milestone it went to
		today = read(other)
		assert.Equal(t, 1, today.Scope)
		assert.Equal(t, 1, today.Remaining)
	})

	t.Run("CountsPullRequests", func(t *testing.T) {
		// milestone 1 holds pull request #2 and nothing else, has no creation time, and is due in the
		// year 9999: the chart counts the pull request, as the progress bar does, and draws no ideal line
		req := NewRequest(t, "GET", "/user2/repo1/milestone/1/burndown")
		burndown := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &issue_service.MilestoneBurndown{})
		require.NotEmpty(t, burndown.Points)
		today := burndown.Points[len(burndown.Points)-1]
		assert.Equal(t, 1, today.Scope)
		assert.Equal(t, 1, today.Remaining)
		assert.Nil(t, burndown.Ideal)
		assert.Empty(t, burndown.Deadline)
	})

	t.Run("NotFound", func(t *testing.T) {
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/milestone/99999/burndown"), http.StatusNotFound)
		// milestone 4 belongs to repo 42, so it cannot be read through repo 1
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/milestone/4/burndown"), http.StatusNotFound)
	})

	t.Run("PrivateRepoHidden", func(t *testing.T) {
		// user2/repo2 is private, so a signed-out reader learns nothing about its milestones
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo2/milestone/1/burndown"), http.StatusNotFound)
	})
}
