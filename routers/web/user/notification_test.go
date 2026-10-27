// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	activities_model "gitea.dev/models/activities"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterNotificationsByRepoAccess(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 40})
	inaccessibleRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	require.True(t, inaccessibleRepo.IsPrivate)
	accessibleRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	notifications := activities_model.NotificationList{
		{ID: 1, Repository: inaccessibleRepo},
		{ID: 2, Repository: accessibleRepo},
	}

	filtered, failures, err := filterNotificationsByRepoAccess(t.Context(), doer, notifications)
	require.NoError(t, err)

	assert.Equal(t, []int{0}, failures)
	require.Len(t, filtered, 1)
	assert.EqualValues(t, 2, filtered[0].ID)
}

func TestParseNotificationFilter(t *testing.T) {
	cases := []struct {
		query    string
		expected notificationFilter
		active   bool
		qs       string
	}{
		{"", notificationFilter{}, false, ""},
		{"?repo=2&source=pull", notificationFilter{RepoID: 2, Source: "pull"}, true, "repo=2&source=pull"},
		{"?source=unknown", notificationFilter{}, false, ""},
		{"?repo=abc", notificationFilter{}, false, ""},
		{"?repo=-5&source=commit", notificationFilter{Source: "commit"}, true, "source=commit"},
		{"?group=repo", notificationFilter{GroupByRepo: true}, false, "group=repo"}, // grouping is not a filter
		{"?group=repo&repo=2", notificationFilter{RepoID: 2, GroupByRepo: true}, true, "group=repo&repo=2"},
		{"?group=bogus", notificationFilter{}, false, ""},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			ctx, _ := contexttest.MockContext(t, "notifications"+c.query)
			filter := parseNotificationFilter(ctx)
			assert.Equal(t, c.expected, filter)
			assert.Equal(t, c.active, filter.IsActive())
			assert.Equal(t, c.qs, filter.queryString())
		})
	}

	opts := notificationFilter{RepoID: 2, Source: "pull"}.findOptions(7)
	assert.EqualValues(t, 7, opts.UserID)
	assert.EqualValues(t, 2, opts.RepoID)
	assert.Equal(t, []activities_model.NotificationSource{activities_model.NotificationSourcePullRequest}, opts.Source)
}

func TestNotificationViewLink(t *testing.T) {
	filter := notificationFilter{RepoID: 2, Source: "pull"}
	assert.Equal(t, "/notifications", notificationViewLink("", 1, notificationFilter{}))
	assert.Equal(t, "/notifications", notificationViewLink("unread", 0, notificationFilter{})) // both spellings of the unread tab are one view
	assert.Equal(t, "/notifications?type=read", notificationViewLink("read", 1, notificationFilter{}))
	assert.Equal(t, "/notifications?page=3&repo=2&source=pull&type=read", notificationViewLink("read", 3, filter))
	assert.Equal(t, "/notifications?repo=2&source=pull", notificationViewLink("bogus", 1, filter))
	assert.Equal(t, "/notifications?group=repo&page=2&type=read", notificationViewLink("read", 2, notificationFilter{GroupByRepo: true}))
}

func TestGroupNotificationsByRepo(t *testing.T) {
	repo1 := &repo_model.Repository{ID: 1}
	repo2 := &repo_model.Repository{ID: 2}
	notifications := activities_model.NotificationList{
		{ID: 4, RepoID: 1, Repository: repo1, Status: activities_model.NotificationStatusUnread},
		{ID: 3, RepoID: 1, Repository: repo1, Status: activities_model.NotificationStatusPinned},
		{ID: 5, RepoID: 2, Repository: repo2, Status: activities_model.NotificationStatusUnread},
	}
	unread := map[int64]int64{1: 7, 2: 1} // counted over every page, so larger than this page's rows

	groups := groupNotificationsByRepo(notifications, 0, unread)
	require.Len(t, groups, 2)
	assert.Same(t, repo1, groups[0].Repo)
	assert.Equal(t, activities_model.NotificationList{notifications[0], notifications[1]}, groups[0].Notifications) // the pinned row stays in its repository
	assert.EqualValues(t, 7, groups[0].UnreadCount)
	assert.False(t, groups[0].Continued)
	assert.Same(t, repo2, groups[1].Repo)
	assert.Equal(t, activities_model.NotificationList{notifications[2]}, groups[1].Notifications)

	groups = groupNotificationsByRepo(notifications, 1, unread) // the previous page ended in repo 1
	assert.True(t, groups[0].Continued)
	assert.False(t, groups[1].Continued) // only the first section can continue

	groups = groupNotificationsByRepo(notifications[2:], 1, unread)
	assert.False(t, groups[0].Continued) // the previous page's repository has no rows here

	assert.Empty(t, groupNotificationsByRepo(nil, 1, unread))
}

func TestParseNotificationBulkTarget(t *testing.T) {
	parse := func(query string) (activities_model.FindNotificationOptions, bool) {
		ctx, _ := contexttest.MockContext(t, "notifications/bulk"+query)
		ctx.Doer = &user_model.User{ID: 2}
		return parseNotificationBulkTarget(ctx, parseNotificationFilter(ctx))
	}

	opts, ok := parse("?notification_ids=4,1,4")
	assert.True(t, ok)
	assert.Equal(t, []int64{4, 1, 4}, opts.IDs)
	assert.Zero(t, opts.UserID) // the model sets it

	opts, ok = parse("?all=true&type=read&repo=2&notification_ids=9") // listed IDs are ignored for a whole view
	assert.True(t, ok)
	assert.Empty(t, opts.IDs)
	assert.EqualValues(t, 2, opts.UserID)
	assert.EqualValues(t, 2, opts.RepoID)
	assert.Equal(t, []activities_model.NotificationStatus{activities_model.NotificationStatusRead, activities_model.NotificationStatusPinned}, opts.Status)

	for _, query := range []string{"", "?notification_ids=", "?notification_ids=1,x"} {
		_, ok = parse(query)
		assert.False(t, ok, query)
	}
}
