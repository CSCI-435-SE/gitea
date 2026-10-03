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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationBulkActions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 fixtures: #2 read (repo 1), #3 pinned (repo 1), #4 unread (repo 1), #5 unread (repo 2); #1 is user1's
	session := loginUser(t, "user2")

	bulkAs := func(t *testing.T, s *TestSession, query string, values map[string]string, status int) string {
		resp := s.MakeRequest(t, NewRequestWithValues(t, "POST", "/notifications/bulk"+query, values), status)
		if status != http.StatusOK {
			return ""
		}
		redirect := test.ParseJSONRedirect(resp.Body.Bytes()).Redirect
		require.NotNil(t, redirect)
		return *redirect
	}
	bulk := func(t *testing.T, query string, values map[string]string, status int) string {
		return bulkAs(t, session, query, values, status)
	}
	flashAt := func(t *testing.T, link string) string {
		resp := session.MakeRequest(t, NewRequest(t, "GET", link), http.StatusOK)
		return strings.TrimSpace(NewHTMLParser(t, resp.Body).doc.Find(".ui.message.flash-message").Text())
	}

	t.Run("ByIDsChangesOnlyOwnSelection", func(t *testing.T) {
		link := bulk(t, "?type=unread", map[string]string{"action": "mark_as_read", "notification_ids": "4,1"}, http.StatusOK)
		assert.Equal(t, "/notifications", link)
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 4, Status: activities_model.NotificationStatusRead})
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 1, Status: activities_model.NotificationStatusUnread}) // user1's
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})
		assert.Equal(t, "Marked 1 notification as read.", flashAt(t, link))
	})

	t.Run("AllInViewRespectsTabAndFilter", func(t *testing.T) {
		// the unread view of repo 1 now only holds pinned #3; repo 2's #5 is outside it
		link := bulk(t, "?type=unread&repo=1&page=2", map[string]string{"action": "mark_as_read", "all": "true"}, http.StatusOK)
		assert.Equal(t, "/notifications?page=2&repo=1", link)
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 3, Status: activities_model.NotificationStatusRead})
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})
	})

	t.Run("DeleteRemovesFromBothTabs", func(t *testing.T) {
		link := bulk(t, "?type=read", map[string]string{"action": "delete", "notification_ids": "2,4"}, http.StatusOK)
		assert.Equal(t, "/notifications?type=read", link)
		unittest.AssertNotExistsBean(t, &activities_model.Notification{ID: 2})
		unittest.AssertNotExistsBean(t, &activities_model.Notification{ID: 4})
		assert.Equal(t, "Deleted 2 notifications.", flashAt(t, link))
		for _, tab := range []string{"/notifications", "/notifications?type=read"} {
			resp := session.MakeRequest(t, NewRequest(t, "GET", tab), http.StatusOK)
			doc := NewHTMLParser(t, resp.Body)
			assert.Zero(t, doc.doc.Find("#notification_2, #notification_4").Length())
		}
	})

	t.Run("OtherUsersNotificationsAreUntouched", func(t *testing.T) {
		other := loginUser(t, "user5")
		link := bulkAs(t, other, "", map[string]string{"action": "delete", "notification_ids": "5"}, http.StatusOK)
		assert.Equal(t, "/notifications", link)
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5})
	})

	t.Run("InvalidRequests", func(t *testing.T) {
		bulk(t, "", map[string]string{"action": "archive", "notification_ids": "5"}, http.StatusBadRequest)
		bulk(t, "", map[string]string{"action": "mark_as_read"}, http.StatusBadRequest)
		bulk(t, "", map[string]string{"action": "mark_as_read", "notification_ids": "5,x"}, http.StatusBadRequest)
		unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})
	})
}
