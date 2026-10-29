// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	unit_model "gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	api "gitea.dev/modules/structs"
	issue_service "gitea.dev/services/issue"
	repo_service "gitea.dev/services/repository"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// backdatedMilestone creates a milestone in user2/repo1, due today, whose three issues joined six days ago
// and two of which closed four and two days ago. API-created history is all from today, too little to
// project, so the timestamps are moved back in the database.
func backdatedMilestone(t *testing.T) int64 {
	token := getTokenForLoggedInUser(t, loginUser(t, "user2"), auth_model.AccessTokenScopeWriteIssue)
	due := time.Now()
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/milestones", api.CreateMilestoneOption{
		Title: "projected", Deadline: &due,
	}).AddTokenAuth(token)
	milestoneID := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Milestone{}).ID

	ids := make([]int64, 0, 3)
	indexes := make([]int64, 0, 3)
	for _, title := range []string{"closed first", "closed second", "still open"} {
		req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/issues", api.CreateIssueOption{
			Title: title, Milestone: milestoneID,
		}).AddTokenAuth(token)
		created := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		ids = append(ids, created.ID)
		indexes = append(indexes, created.Index)
	}
	closed := "closed"
	for _, index := range indexes[:2] {
		req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/user2/repo1/issues/%d", index), api.EditIssueOption{
			State: &closed,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	}

	daysAgo := func(n int) int64 { return time.Now().AddDate(0, 0, -n).Unix() }
	exec := func(query string, args ...any) {
		_, err := db.GetEngine(t.Context()).Exec(append([]any{query}, args...)...)
		require.NoError(t, err)
	}
	exec("UPDATE milestone SET created_unix=? WHERE id=?", daysAgo(6), milestoneID)
	for i, id := range ids {
		exec("UPDATE issue SET created_unix=? WHERE id=?", daysAgo(6), id)
		exec("UPDATE comment SET created_unix=? WHERE issue_id=? AND type=?", daysAgo(6), id, issues_model.CommentTypeMilestone)
		if i < 2 {
			closedAt := daysAgo(4 - 2*i)
			exec("UPDATE issue SET closed_unix=? WHERE id=?", closedAt, id)
			exec("UPDATE comment SET created_unix=? WHERE issue_id=? AND type=?", closedAt, id, issues_model.CommentTypeClose)
		}
	}
	return milestoneID
}

func milestoneRow(t *testing.T, url string, milestoneID int64) *goquery.Selection {
	doc := NewHTMLParser(t, MakeRequest(t, NewRequest(t, "GET", url), http.StatusOK).Body)
	row := doc.Find(".milestone-list > .item").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return s.Find(fmt.Sprintf(`a[href$="/milestone/%d"]`, milestoneID)).Length() > 0
	})
	require.Equal(t, 1, row.Length(), "milestone %d on %s", milestoneID, url)
	return row
}

func TestMilestoneListProjection(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	milestoneID := backdatedMilestone(t)

	t.Run("Projected", func(t *testing.T) {
		// the list must say what the chart says
		burndown := readBurndown(t, fmt.Sprintf("/user2/repo1/milestone/%d/burndown", milestoneID))
		require.Equal(t, issue_service.BurndownProjected, burndown.Status)
		require.NotNil(t, burndown.DaysLate)
		require.Positive(t, *burndown.DaysLate, "due today, so any projection is late")

		projection := milestoneRow(t, "/user2/repo1/milestones", milestoneID).Find(".milestone-projection")
		require.Equal(t, 1, projection.Length())
		assert.True(t, projection.HasClass("tw-text-red"))
		datetime, _ := projection.Find("relative-time").Attr("datetime")
		assert.True(t, strings.HasPrefix(datetime, burndown.Projected), "%q starts with %q", datetime, burndown.Projected)
		want := fmt.Sprintf("%d days after the due date", *burndown.DaysLate)
		if *burndown.DaysLate == 1 {
			want = "1 day after the due date"
		}
		assert.Contains(t, projection.Text(), want)
	})

	t.Run("InsufficientData", func(t *testing.T) {
		token := getTokenForLoggedInUser(t, loginUser(t, "user2"), auth_model.AccessTokenScopeWriteIssue)
		req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/milestones", api.CreateMilestoneOption{Title: "new"}).AddTokenAuth(token)
		newID := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Milestone{}).ID
		req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/issues", api.CreateIssueOption{Title: "today", Milestone: newID}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		assert.Equal(t, 0, milestoneRow(t, "/user2/repo1/milestones", newID).Find(".milestone-projection").Length())
	})

	t.Run("ClosedTab", func(t *testing.T) {
		token := getTokenForLoggedInUser(t, loginUser(t, "user2"), auth_model.AccessTokenScopeWriteIssue)
		closed := "closed"
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/user2/repo1/milestones/%d", milestoneID), api.EditMilestoneOption{
			State: &closed,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, 0, milestoneRow(t, "/user2/repo1/milestones?state=closed", milestoneID).Find(".milestone-projection").Length())
	})
}

func TestMilestoneListProjectionWithoutPullAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	milestoneID := backdatedMilestone(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	require.NoError(t, repo_service.UpdateRepositoryUnits(t.Context(), repo, nil, []unit_model.Type{unit_model.TypePullRequests}))

	// the milestone holds issues only, so an issue reader still gets its projection
	assert.Equal(t, 1, milestoneRow(t, "/user2/repo1/milestones", milestoneID).Find(".milestone-projection").Length())
}

func TestMilestoneListProjectionWithoutIssueAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	milestoneID := backdatedMilestone(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	require.NoError(t, repo_service.UpdateRepositoryUnits(t.Context(), repo, nil, []unit_model.Type{unit_model.TypeIssues}))

	// a pull request reader counts pull requests only, of which the milestone has none, so the
	// issues' projection must not show
	burndown := readBurndown(t, fmt.Sprintf("/user2/repo1/milestone/%d/burndown", milestoneID))
	assert.Equal(t, issue_service.BurndownEmpty, burndown.Status)
	assert.Equal(t, 0, milestoneRow(t, "/user2/repo1/milestones", milestoneID).Find(".milestone-projection").Length())
}
