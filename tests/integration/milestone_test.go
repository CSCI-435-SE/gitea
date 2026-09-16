// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
)

// TestMilestoneDueDateHighlight also serves as the regression check for the
// milestone_issues.tmpl context bug fix: before that fix the detail page never
// rendered a milestone's due date in red, even when overdue.
func TestMilestoneDueDateHighlight(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	session := loginUser(t, owner.Name)

	createMilestone := func(t *testing.T, name, deadline string) int64 {
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/milestones/new", owner.Name, repo.Name), map[string]string{
			"title":    name,
			"deadline": deadline,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		milestone := unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{RepoID: repo.ID, Name: name})
		return milestone.ID
	}

	// assertListDueDateClass scopes to the specific milestone's row, since the
	// list page also renders the repo's other (fixture) milestones.
	assertListDueDateClass := func(t *testing.T, id int64, wantClass string) {
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/milestones", owner.Name, repo.Name))
		resp := session.MakeRequest(t, req, http.StatusOK)
		row := NewHTMLParser(t, resp.Body).Find(fmt.Sprintf(`a[href$="/milestone/%d"]`, id)).Closest(".item")
		assert.True(t, row.Find(".due-date").HasClass(wantClass))
	}

	// assertDetailDueDateClass checks the milestone's own detail page, where it is
	// the only milestone rendered.
	assertDetailDueDateClass := func(t *testing.T, id int64, wantClass string) {
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/milestone/%d", owner.Name, repo.Name, id))
		resp := session.MakeRequest(t, req, http.StatusOK)
		assert.True(t, NewHTMLParser(t, resp.Body).Find(".due-date").HasClass(wantClass))
	}

	t.Run("near due on milestone list", func(t *testing.T) {
		tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		id := createMilestone(t, "e2e-near-due-list", tomorrow)
		assertListDueDateClass(t, id, "tw-text-warning-text")
	})

	// This is the regression check for the milestone_issues.tmpl context bug fix:
	// before that fix .IsOverdue read the page context instead of .Milestone, so an
	// overdue milestone never rendered red on its own detail page.
	t.Run("overdue on milestone detail page", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		id := createMilestone(t, "e2e-overdue-detail", yesterday)
		assertDetailDueDateClass(t, id, "tw-text-red")
	})
}
