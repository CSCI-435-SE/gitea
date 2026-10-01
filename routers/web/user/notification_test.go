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
