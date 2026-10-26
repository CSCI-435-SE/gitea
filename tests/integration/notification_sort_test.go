// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	activities_model "gitea.dev/models/activities"
	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/test"
	issue_service "gitea.dev/services/issue"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// the notification queue is immediate in tests, so each event has stored its notifications when the call returns
func TestNotificationReason(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user4 watches user2/repo1; user5 doesn't
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	reasonOf := func(userID, issueID int64) activities_model.NotificationReason {
		return unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: userID, IssueID: issueID}).Reason
	}

	t.Run("ReviewRequestedOutranksALaterComment", func(t *testing.T) {
		pull := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
		_, err := issue_service.ReviewRequest(t.Context(), pull, user2, nil, user4, true)
		require.NoError(t, err)
		assert.Equal(t, activities_model.NotificationReasonReviewRequested, reasonOf(4, pull.ID)) // #52 AC1

		_, err = issue_service.CreateIssueComment(t.Context(), user2, repo1, pull, "a comment every watcher hears about", nil)
		require.NoError(t, err)
		assert.Equal(t, activities_model.NotificationReasonReviewRequested, reasonOf(4, pull.ID)) // #52 AC2
	})

	t.Run("Assigned", func(t *testing.T) {
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 5})
		removed, _, err := issue_service.ToggleAssigneeWithNotify(t.Context(), issue, user2, user4.ID)
		require.NoError(t, err)
		require.False(t, removed)
		assert.Equal(t, activities_model.NotificationReasonAssigned, reasonOf(4, issue.ID)) // #52 AC1
	})

	t.Run("MentionedAndSubscribed", func(t *testing.T) {
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
		_, err := issue_service.CreateIssueComment(t.Context(), user2, repo1, issue, "cc @user5", nil)
		require.NoError(t, err)
		assert.Equal(t, activities_model.NotificationReasonMentioned, reasonOf(5, issue.ID))  // #52 AC1
		assert.Equal(t, activities_model.NotificationReasonSubscribed, reasonOf(4, issue.ID)) // #52 AC1 a watcher
	})
}

func TestNotificationSort(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2's notifications: #2 read, #3 pinned and #4 unread in user2/repo1, #5 unread in user2/repo2; #5 is the newest
	set := func(id int64, reason activities_model.NotificationReason, source activities_model.NotificationSource) {
		_, err := db.GetEngine(t.Context()).ID(id).Cols("reason", "source").Update(&activities_model.Notification{Reason: reason, Source: source})
		require.NoError(t, err)
	}
	set(2, activities_model.NotificationReasonSubscribed, activities_model.NotificationSourceIssue)
	set(3, activities_model.NotificationReasonReviewRequested, activities_model.NotificationSourcePullRequest)
	set(4, activities_model.NotificationReasonOther, activities_model.NotificationSourceIssue)
	set(5, activities_model.NotificationReasonMentioned, activities_model.NotificationSourceIssue)

	session := loginUser(t, "user2")
	type row struct{ ID, Reason string }
	rowsAt := func(t *testing.T, link string) ([]row, *HTMLDoc) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", link), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		var rows []row
		doc.doc.Find("#notification_table .notifications-item").Each(func(_ int, s *goquery.Selection) {
			rows = append(rows, row{s.AttrOr("id", ""), strings.TrimSpace(s.Find(".notification-reason").Text())})
		})
		return rows, doc
	}

	t.Run("Reason", func(t *testing.T) {
		rows, doc := rowsAt(t, "/notifications?sort=reason")
		assert.Equal(t, []row{
			{"notification_3", "Review requested"}, // #52 AC9 the pinned row sorts with the rest
			{"notification_5", "Mentioned"},
			{"notification_4", ""}, // #52 AC4 "Other" has no label
		}, rows)
		assert.Equal(t, "Sort: Reason", strings.TrimSpace(doc.doc.Find(`[data-test-id="notification-sort"] > .text`).Text()))

		rows, _ = rowsAt(t, "/notifications?type=read&sort=reason")
		assert.Equal(t, []row{{"notification_3", "Review requested"}, {"notification_2", "Subscribed"}}, rows)
	})

	t.Run("Type", func(t *testing.T) {
		rows, _ := rowsAt(t, "/notifications?sort=type")
		assert.Equal(t, []row{{"notification_5", "Mentioned"}, {"notification_4", ""}, {"notification_3", "Review requested"}}, rows) // #52 AC5 issues, then pull requests
	})

	t.Run("NewestByDefaultWithLabels", func(t *testing.T) {
		rows, doc := rowsAt(t, "/notifications?sort=bogus")
		assert.Equal(t, []row{{"notification_5", "Mentioned"}, {"notification_4", ""}, {"notification_3", "Review requested"}}, rows) // #52 AC6, and Decision 7's labels
		assert.Equal(t, "Sort", strings.TrimSpace(doc.doc.Find(`[data-test-id="notification-sort"] > .text`).Text()))
	})

	t.Run("InsideRepositoryGroups", func(t *testing.T) {
		rows, doc := rowsAt(t, "/notifications?group=repo&sort=reason")
		assert.Equal(t, []row{{"notification_3", "Review requested"}, {"notification_4", ""}, {"notification_5", "Mentioned"}}, rows) // #52 AC8
		assert.Equal(t, 2, doc.doc.Find(".notification-group").Length())
	})

	t.Run("LinksKeepSort", func(t *testing.T) {
		_, doc := rowsAt(t, "/notifications?sort=reason&group=repo&source=issue&perPage=1") // two pages, so there are pagination links
		for _, c := range []struct{ selector, attr string }{
			{".small-menu-items a.item", "href"}, // Unread and Read tabs
			{`[data-test-id="notification-filter-repo"] .menu a.item`, "href"},
			{`[data-test-id="notification-filter-type"] .menu a.item`, "href"},
			{`[data-test-id="notification-group-repo"]`, "href"},
			{".pagination a.item[href]", "href"},
			{"form.notifications-buttons", "action"}, // single-row status actions
			{"#notification_div", "data-bulk-url"},
		} {
			found := doc.doc.Find(c.selector)
			require.NotZero(t, found.Length(), c.selector)
			found.Each(func(_ int, s *goquery.Selection) {
				assert.Contains(t, s.AttrOr(c.attr, ""), "sort=reason", c.selector) // #52 AC7
			})
		}
		clearFilters := doc.doc.Find(".flex-text-block a.ui.button").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return strings.TrimSpace(s.Text()) == "Clear filters"
		})
		assert.Equal(t, "/notifications?group=repo&sort=reason", clearFilters.AttrOr("href", "")) // a view setting, not a filter

		var sortLinks []string
		doc.doc.Find(`[data-test-id="notification-sort"] .menu a.item`).Each(func(_ int, s *goquery.Selection) {
			sortLinks = append(sortLinks, s.AttrOr("href", ""))
		})
		assert.Equal(t, []string{
			"/notifications?source=issue&group=repo",
			"/notifications?source=issue&group=repo&sort=reason",
			"/notifications?source=issue&group=repo&sort=type",
		}, sortLinks)
	})

	t.Run("InPlaceRefreshKeepsSort", func(t *testing.T) {
		rows, _ := rowsAt(t, "/notifications?sort=reason&div-only=true")
		assert.Equal(t, "notification_3", rows[0].ID)

		resp := session.MakeRequest(t, NewRequestWithValues(t, "POST", "/notifications/status?type=unread&page=1&sort=reason", map[string]string{
			"notification_id":     "4",
			"notification_action": "pin",
		}), http.StatusOK)
		var ids []string
		NewHTMLParser(t, resp.Body).doc.Find("#notification_table .notifications-item").Each(func(_ int, s *goquery.Selection) {
			ids = append(ids, s.AttrOr("id", ""))
		})
		assert.Equal(t, []string{"notification_3", "notification_5", "notification_4"}, ids) // #52 AC7 still by reason
	})

	t.Run("RedirectsKeepSort", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequestWithValues(t, "POST", "/notifications/bulk?type=read&sort=type", map[string]string{
			"action":           "mark_as_unread",
			"notification_ids": "2",
		}), http.StatusOK)
		redirect := test.ParseJSONRedirect(resp.Body.Bytes()).Redirect
		require.NotNil(t, redirect)
		assert.Equal(t, "/notifications?sort=type&type=read", *redirect) // #52 AC7

		resp = session.MakeRequest(t, NewRequest(t, "POST", "/notifications/purge?sort=type"), http.StatusSeeOther)
		assert.Equal(t, "/notifications?sort=type", test.RedirectURL(resp))
	})
}
