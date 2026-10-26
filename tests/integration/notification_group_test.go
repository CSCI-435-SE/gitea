// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	activities_model "gitea.dev/models/activities"
	"gitea.dev/models/db"
	"gitea.dev/modules/test"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationGroupByRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 fixtures: #3 pinned (repo 1), #4 unread (repo 1), #5 unread (repo 2), #2 read (repo 1); #5 is the newest
	session := loginUser(t, "user2")

	type section struct {
		Header string
		IDs    []string
	}
	sectionsAt := func(t *testing.T, link string) ([]section, *HTMLDoc) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", link), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		var sections []section
		doc.doc.Find("#notification_table .notification-group").Each(func(_ int, g *goquery.Selection) {
			s := section{Header: strings.Join(strings.Fields(g.Find(".notification-group-header").Text()), " ")}
			g.Find(".notifications-item").Each(func(_ int, row *goquery.Selection) {
				s.IDs = append(s.IDs, row.AttrOr("id", ""))
			})
			sections = append(sections, s)
		})
		return sections, doc
	}

	t.Run("SectionsByRepoNewestFirst", func(t *testing.T) {
		sections, _ := sectionsAt(t, "/notifications?group=repo")
		assert.Equal(t, []section{
			{"user2/repo1 1 unread", []string{"notification_4", "notification_3"}}, // the pinned row stays in its repository
			{"user2/repo2 1 unread", []string{"notification_5"}},
		}, sections)
	})

	t.Run("OffIsUnchanged", func(t *testing.T) {
		sections, doc := sectionsAt(t, "/notifications")
		assert.Empty(t, sections)
		var ids []string
		doc.doc.Find("#notification_table .notifications-item").Each(func(_ int, s *goquery.Selection) {
			ids = append(ids, s.AttrOr("id", ""))
		})
		assert.Equal(t, []string{"notification_5", "notification_4", "notification_3"}, ids)
	})

	t.Run("UnreadCountCoversTheRepository", func(t *testing.T) {
		sections, _ := sectionsAt(t, "/notifications?type=read&group=repo")
		assert.Equal(t, []section{{"user2/repo1 1 unread", []string{"notification_3", "notification_2"}}}, sections) // #4 is unread, though listed on the other tab

		sections, _ = sectionsAt(t, "/notifications?group=repo&source=commit")
		assert.Empty(t, sections) // nothing matches, so no headers either
	})

	t.Run("ContinuedAcrossPages", func(t *testing.T) {
		// a third unread row makes three pages of one, since the pager counts unread rows only and pinned #3 is extra
		extra := &activities_model.Notification{UserID: 2, RepoID: 2, Status: activities_model.NotificationStatusUnread, Source: activities_model.NotificationSourceRepository, UpdatedBy: 1}
		require.NoError(t, db.Insert(t.Context(), extra))
		defer func() {
			_, err := db.DeleteByID[activities_model.Notification](t.Context(), extra.ID)
			assert.NoError(t, err)
		}()
		extraID := fmt.Sprintf("notification_%d", extra.ID) // the newest row, so it leads user2/repo2

		sections, _ := sectionsAt(t, "/notifications?group=repo&perPage=1&page=2")
		assert.Equal(t, []section{{"user2/repo1 (continued) 1 unread", []string{"notification_3"}}}, sections)

		sections, _ = sectionsAt(t, "/notifications?group=repo&perPage=1&page=3")
		assert.Equal(t, []section{{"user2/repo2 2 unread", []string{extraID}}}, sections) // the previous page ended in another repository
	})

	t.Run("LinksKeepGrouping", func(t *testing.T) {
		_, doc := sectionsAt(t, "/notifications?group=repo&source=issue&perPage=1") // two pages, so there are pagination links
		for _, c := range []struct{ selector, attr string }{
			{".small-menu-items a.item", "href"}, // Unread and Read tabs
			{`[data-test-id="notification-filter-repo"] .menu a.item`, "href"},
			{`[data-test-id="notification-filter-type"] .menu a.item`, "href"},
			{".pagination a.item[href]", "href"},
			{"form.notifications-buttons", "action"}, // single-row status actions
			{"#notification_div", "data-bulk-url"},
		} {
			found := doc.doc.Find(c.selector)
			require.NotZero(t, found.Length(), c.selector)
			found.Each(func(_ int, s *goquery.Selection) {
				assert.Contains(t, s.AttrOr(c.attr, ""), "group=repo", c.selector)
			})
		}
		clearFilters := doc.doc.Find(".flex-text-block a.ui.button").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return strings.TrimSpace(s.Text()) == "Clear filters"
		})
		assert.Equal(t, "/notifications?group=repo", clearFilters.AttrOr("href", ""))
		assert.Equal(t, "/notifications?source=issue", doc.doc.Find(`[data-test-id="notification-group-repo"]`).AttrOr("href", "")) // the toggle turns it off
	})

	t.Run("ToggleTurnsItOn", func(t *testing.T) {
		_, doc := sectionsAt(t, "/notifications?type=read&source=issue")
		assert.Equal(t, "/notifications?type=read&source=issue&group=repo", doc.doc.Find(`[data-test-id="notification-group-repo"]`).AttrOr("href", ""))
	})

	t.Run("InPlaceRefreshKeepsGrouping", func(t *testing.T) {
		sections, _ := sectionsAt(t, "/notifications?group=repo&div-only=true")
		assert.Len(t, sections, 2)

		resp := session.MakeRequest(t, NewRequestWithValues(t, "POST", "/notifications/status?type=unread&page=1&group=repo", map[string]string{
			"notification_id":     "4",
			"notification_action": "pin",
		}), http.StatusOK)
		assert.Equal(t, 2, NewHTMLParser(t, resp.Body).doc.Find("#notification_table .notification-group").Length())
	})

	t.Run("RedirectsKeepGrouping", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequestWithValues(t, "POST", "/notifications/bulk?type=read&group=repo", map[string]string{
			"action":           "mark_as_unread",
			"notification_ids": "2",
		}), http.StatusOK)
		redirect := test.ParseJSONRedirect(resp.Body.Bytes()).Redirect
		require.NotNil(t, redirect)
		assert.Equal(t, "/notifications?group=repo&type=read", *redirect)

		resp = session.MakeRequest(t, NewRequest(t, "POST", "/notifications/purge?group=repo"), http.StatusSeeOther)
		assert.Equal(t, "/notifications?group=repo", test.RedirectURL(resp))
	})
}
