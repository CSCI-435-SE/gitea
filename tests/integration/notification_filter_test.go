// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	activities_model "gitea.dev/models/activities"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/test"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestNotificationFilters(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 fixtures: #3 pinned (repo 1), #4 unread (repo 1), #5 unread (repo 2), #2 read (repo 1)
	session := loginUser(t, "user2")

	listedIDs := func(t *testing.T, query string) (ids []string, doc *HTMLDoc) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/notifications"+query), http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		doc.doc.Find("#notification_table .notifications-item").Each(func(_ int, s *goquery.Selection) {
			ids = append(ids, s.AttrOr("id", ""))
		})
		return ids, doc
	}

	t.Run("ByRepo", func(t *testing.T) {
		ids, _ := listedIDs(t, "?repo=2")
		assert.Equal(t, []string{"notification_5"}, ids)
	})

	t.Run("ByRepoOnReadTab", func(t *testing.T) {
		ids, _ := listedIDs(t, "?type=read&repo=1")
		assert.ElementsMatch(t, []string{"notification_2", "notification_3"}, ids) // pinned rows are filtered too
	})

	t.Run("NoMatchShowsEmptyState", func(t *testing.T) {
		ids, doc := listedIDs(t, "?source=pull")
		assert.Empty(t, ids)
		assert.Contains(t, doc.doc.Find(".empty-placeholder").Text(), "No notifications match")
		assert.Equal(t, "/notifications", doc.doc.Find(".empty-placeholder a").AttrOr("href", "")) // no stray "?"
	})

	t.Run("RepoDropdownMarksUnread", func(t *testing.T) {
		_, doc := listedIDs(t, "")
		item := doc.doc.Find(`[data-test-id="notification-filter-repo"] .menu a.item`).FilterFunction(func(_ int, s *goquery.Selection) bool {
			return strings.TrimSpace(s.Text()) == "user2/repo2"
		})
		assert.Equal(t, 1, item.Length())
		assert.Equal(t, 1, item.Find(`[role="img"]`).Length())

		allRepos := doc.doc.Find(`[data-test-id="notification-filter-repo"] .menu a.item`).First()
		assert.Equal(t, "/notifications", allRepos.AttrOr("href", ""))
	})

	t.Run("InaccessibleRepoIsNotExposed", func(t *testing.T) {
		other := loginUser(t, "user5") // repo 2 is user2's private repo
		resp := other.MakeRequest(t, NewRequest(t, "GET", "/notifications?repo=2"), http.StatusOK)
		assert.NotContains(t, resp.Body.String(), "user2/repo2")
		assert.Empty(t, NewHTMLParser(t, resp.Body).doc.Find("#notification_table .notifications-item").Nodes)
	})

	t.Run("MarkAllRespectsFilter", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "POST", "/notifications/purge?repo=1"), http.StatusSeeOther)
		assert.Equal(t, "/notifications?repo=1", test.RedirectURL(resp))
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 4, Status: activities_model.NotificationStatusRead})
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})
	})
}
