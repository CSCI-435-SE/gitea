// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities_test

import (
	"context"
	"testing"

	activities_model "gitea.dev/models/activities"
	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/util"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdateIssueNotifications(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})

	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), issue.ID, 0, 2, 0))

	// User 9 is inactive, thus notifications for user 1 and 4 are created
	notf := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: 1, IssueID: issue.ID})
	assert.Equal(t, activities_model.NotificationStatusUnread, notf.Status)
	unittest.CheckConsistencyFor(t, &issues_model.Issue{ID: issue.ID})

	notf = unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: 4, IssueID: issue.ID})
	assert.Equal(t, activities_model.NotificationStatusUnread, notf.Status)
}

func TestNotificationsForUser(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	notfs, err := db.Find[activities_model.Notification](t.Context(), activities_model.FindNotificationOptions{
		UserID: user.ID,
		Status: []activities_model.NotificationStatus{
			activities_model.NotificationStatusRead,
			activities_model.NotificationStatusUnread,
		},
	})
	assert.NoError(t, err)
	if assert.Len(t, notfs, 3) {
		assert.EqualValues(t, 5, notfs[0].ID)
		assert.Equal(t, user.ID, notfs[0].UserID)
		assert.EqualValues(t, 4, notfs[1].ID)
		assert.Equal(t, user.ID, notfs[1].UserID)
		assert.EqualValues(t, 2, notfs[2].ID)
		assert.Equal(t, user.ID, notfs[2].UserID)
	}
}

func TestNotification_GetRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	notf := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{RepoID: 1})
	repo, err := notf.GetRepo(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, repo, notf.Repository)
	assert.Equal(t, notf.RepoID, repo.ID)
}

func TestNotification_GetIssue(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	notf := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{RepoID: 1})
	issue, err := notf.GetIssue(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, issue, notf.Issue)
	assert.Equal(t, notf.IssueID, issue.ID)
}

func TestGetNotificationCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	cnt, err := db.Count[activities_model.Notification](t.Context(), activities_model.FindNotificationOptions{
		UserID: user.ID,
		Status: []activities_model.NotificationStatus{
			activities_model.NotificationStatusRead,
		},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 0, cnt)

	cnt, err = db.Count[activities_model.Notification](t.Context(), activities_model.FindNotificationOptions{
		UserID: user.ID,
		Status: []activities_model.NotificationStatus{
			activities_model.NotificationStatusUnread,
		},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)
}

func TestSetNotificationStatus(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	notf := unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{UserID: user.ID, Status: activities_model.NotificationStatusRead})
	_, err := activities_model.SetNotificationStatus(t.Context(), notf.ID, user, activities_model.NotificationStatusPinned)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{ID: notf.ID, Status: activities_model.NotificationStatusPinned})

	_, err = activities_model.SetNotificationStatus(t.Context(), 1, user, activities_model.NotificationStatusRead)
	assert.Error(t, err)
	_, err = activities_model.SetNotificationStatus(t.Context(), unittest.NonexistentID, user, activities_model.NotificationStatusRead)
	assert.Error(t, err)
}

func TestUpdateNotificationStatuses(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	notfUnread := unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{UserID: user.ID, Status: activities_model.NotificationStatusUnread})
	notfRead := unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{UserID: user.ID, Status: activities_model.NotificationStatusRead})
	notfPinned := unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{UserID: user.ID, Status: activities_model.NotificationStatusPinned})
	assert.NoError(t, activities_model.UpdateNotificationStatuses(t.Context(), user, activities_model.NotificationStatusUnread, activities_model.NotificationStatusRead, activities_model.FindNotificationOptions{}))
	unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{ID: notfUnread.ID, Status: activities_model.NotificationStatusRead})
	unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{ID: notfRead.ID, Status: activities_model.NotificationStatusRead})
	unittest.AssertExistsAndLoadBean(t,
		&activities_model.Notification{ID: notfPinned.ID, Status: activities_model.NotificationStatusPinned})
}

func TestUpdateNotificationStatusesFiltered(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// notification 4 (repo 1) and 5 (repo 2) are both unread for user 2
	assert.NoError(t, activities_model.UpdateNotificationStatuses(t.Context(), user, activities_model.NotificationStatusUnread, activities_model.NotificationStatusRead,
		activities_model.FindNotificationOptions{RepoID: 1}))
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 4, Status: activities_model.NotificationStatusRead})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})

	// a source filter that matches nothing leaves everything unread
	assert.NoError(t, activities_model.UpdateNotificationStatuses(t.Context(), user, activities_model.NotificationStatusUnread, activities_model.NotificationStatusRead,
		activities_model.FindNotificationOptions{Source: []activities_model.NotificationSource{activities_model.NotificationSourcePullRequest}}))
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})

	// a filter's UserID is ignored, so user 2 cannot touch user 1's notification 1
	assert.NoError(t, activities_model.UpdateNotificationStatuses(t.Context(), user, activities_model.NotificationStatusUnread, activities_model.NotificationStatusRead,
		activities_model.FindNotificationOptions{UserID: 1}))
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 1, Status: activities_model.NotificationStatusUnread})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusRead})
}

func TestFindNotificationRepoIDs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	unread, err := activities_model.FindNotificationRepoIDs(t.Context(), activities_model.FindNotificationOptions{
		UserID: 2,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread},
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int64{1, 2}, unread)

	all, err := activities_model.FindNotificationRepoIDs(t.Context(), activities_model.FindNotificationOptions{UserID: 1})
	assert.NoError(t, err)
	assert.Equal(t, []int64{1}, all)

	none, err := activities_model.FindNotificationRepoIDs(t.Context(), activities_model.FindNotificationOptions{
		UserID: 2,
		Source: []activities_model.NotificationSource{activities_model.NotificationSourceCommit},
	})
	assert.NoError(t, err)
	assert.Empty(t, none)
}

func TestSetIssueReadBy(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.NoError(t, db.WithTx(t.Context(), func(ctx context.Context) error {
		return activities_model.SetIssueReadBy(ctx, issue.ID, user.ID)
	}))

	nt, err := activities_model.GetIssueNotification(t.Context(), user.ID, issue.ID)
	assert.NoError(t, err)
	assert.Equal(t, activities_model.NotificationStatusRead, nt.Status)
}

func TestSetNotificationsStatus(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// 3 is pinned, 4 is unread, 1 belongs to user 1 and must be skipped
	n, err := activities_model.SetNotificationsStatus(t.Context(), user, activities_model.FindNotificationOptions{IDs: []int64{1, 3, 4}}, activities_model.NotificationStatusRead)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 3, Status: activities_model.NotificationStatusRead})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 4, Status: activities_model.NotificationStatusRead})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 1, Status: activities_model.NotificationStatusUnread})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusUnread})

	// rows already in the target status are not counted
	n, err = activities_model.SetNotificationsStatus(t.Context(), user, activities_model.FindNotificationOptions{IDs: []int64{2}}, activities_model.NotificationStatusRead)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n)

	// a view filter without IDs works too
	n, err = activities_model.SetNotificationsStatus(t.Context(), user, activities_model.FindNotificationOptions{
		RepoID: 2,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread, activities_model.NotificationStatusPinned},
	}, activities_model.NotificationStatusPinned)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5, Status: activities_model.NotificationStatusPinned})

	_, err = activities_model.SetNotificationsStatus(t.Context(), user, activities_model.FindNotificationOptions{RepoID: 1}, activities_model.NotificationStatusRead)
	assert.ErrorIs(t, err, util.ErrInvalidArgument)
}

func TestDeleteNotifications(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	n, err := activities_model.DeleteNotifications(t.Context(), user, activities_model.FindNotificationOptions{IDs: []int64{1, 2, 4}})
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
	unittest.AssertNotExistsBean(t, &activities_model.Notification{ID: 2})
	unittest.AssertNotExistsBean(t, &activities_model.Notification{ID: 4})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 1})

	// the unread view of repo 1 still holds pinned 3, but not repo 2's 5
	n, err = activities_model.DeleteNotifications(t.Context(), user, activities_model.FindNotificationOptions{
		RepoID: 1,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread, activities_model.NotificationStatusPinned},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)
	unittest.AssertNotExistsBean(t, &activities_model.Notification{ID: 3})
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5})

	_, err = activities_model.DeleteNotifications(t.Context(), user, activities_model.FindNotificationOptions{})
	assert.ErrorIs(t, err, util.ErrInvalidArgument)
	unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 5})
}

func TestDeletedNotificationReturnsOnNewActivity(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	_, err := activities_model.DeleteNotifications(t.Context(), user, activities_model.FindNotificationOptions{IDs: []int64{1}})
	assert.NoError(t, err)
	unittest.AssertNotExistsBean(t, &activities_model.Notification{UserID: user.ID, IssueID: 1})

	// delete must not unsubscribe, so new activity on issue 1 notifies user 1 again
	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), 1, 0, 2, 0))
	notf := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: user.ID, IssueID: 1})
	assert.Equal(t, activities_model.NotificationStatusUnread, notf.Status)
}
