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

	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), issue.ID, 0, 2, 0, activities_model.NotificationReasonSubscribed))

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

func TestFindNotificationsGroupByRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	ids := func(opts activities_model.FindNotificationOptions) []int64 {
		nls, err := db.Find[activities_model.Notification](t.Context(), opts)
		assert.NoError(t, err)
		ids := make([]int64, 0, len(nls))
		for _, n := range nls {
			ids = append(ids, n.ID)
		}
		return ids
	}

	// #5 (user2/repo2) is user2's newest, so it leads the ungrouped list but follows all of user2/repo1 when grouped
	assert.Equal(t, []int64{5, 4, 3, 2}, ids(activities_model.FindNotificationOptions{UserID: 2}))
	assert.Equal(t, []int64{4, 3, 2, 5}, ids(activities_model.FindNotificationOptions{UserID: 2, GroupByRepo: true}))

	page2 := ids(activities_model.FindNotificationOptions{UserID: 2, GroupByRepo: true, ListOptions: db.ListOptions{Page: 2, PageSize: 2}})
	assert.Equal(t, []int64{2, 5}, page2) // the order holds across pages
}

func TestCreateOrUpdateIssueNotificationsReason(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	reasonOf := func(userID int64) activities_model.NotificationReason {
		return unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: userID, IssueID: 1}).Reason
	}

	// #52 AC1 a notification stores the reason it was created for
	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), 1, 0, 2, 4, activities_model.NotificationReasonMentioned))
	assert.Equal(t, activities_model.NotificationReasonMentioned, reasonOf(4))

	// #52 AC2 a lower reason doesn't replace a higher one, but raises user1's older "Other" notification
	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), 1, 0, 2, 0, activities_model.NotificationReasonSubscribed))
	assert.Equal(t, activities_model.NotificationReasonMentioned, reasonOf(4))
	assert.Equal(t, activities_model.NotificationReasonSubscribed, reasonOf(1))

	// #52 AC2 a higher reason replaces a lower one
	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), 1, 0, 2, 4, activities_model.NotificationReasonReviewRequested))
	assert.Equal(t, activities_model.NotificationReasonReviewRequested, reasonOf(4))
}

func TestFindNotificationsSort(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// user2's notifications: #2 read, #3 pinned and #4 unread in user2/repo1, #5 unread in user2/repo2, newest last
	set := func(id int64, reason activities_model.NotificationReason, source activities_model.NotificationSource) {
		_, err := db.GetEngine(t.Context()).ID(id).Cols("reason", "source").Update(&activities_model.Notification{Reason: reason, Source: source})
		assert.NoError(t, err)
	}
	set(2, activities_model.NotificationReasonSubscribed, activities_model.NotificationSourceIssue)
	set(3, activities_model.NotificationReasonReviewRequested, activities_model.NotificationSourcePullRequest)
	set(4, activities_model.NotificationReasonOther, activities_model.NotificationSourceIssue)
	set(5, activities_model.NotificationReasonMentioned, activities_model.NotificationSourceIssue)

	ids := func(opts activities_model.FindNotificationOptions) []int64 {
		opts.UserID = 2
		nls, err := db.Find[activities_model.Notification](t.Context(), opts)
		assert.NoError(t, err)
		ids := make([]int64, 0, len(nls))
		for _, n := range nls {
			ids = append(ids, n.ID)
		}
		return ids
	}

	assert.Equal(t, []int64{5, 4, 3, 2}, ids(activities_model.FindNotificationOptions{}))                                              // #52 AC6 newest first
	assert.Equal(t, []int64{5, 4, 3, 2}, ids(activities_model.FindNotificationOptions{Sort: "bogus"}))                                 // #52 AC6
	assert.Equal(t, []int64{3, 5, 2, 4}, ids(activities_model.FindNotificationOptions{Sort: activities_model.NotificationSortReason})) // #52 AC4 review requested, mentioned, subscribed, other
	assert.Equal(t, []int64{5, 4, 2, 3}, ids(activities_model.FindNotificationOptions{Sort: activities_model.NotificationSortType}))   // #52 AC5 issues newest first, then pull requests

	// #52 AC8 grouped, the repositories stay A-Z and the sort applies inside each one
	grouped := activities_model.FindNotificationOptions{GroupByRepo: true, Sort: activities_model.NotificationSortReason}
	assert.Equal(t, []int64{3, 2, 4, 5}, ids(grouped))
	grouped.ListOptions = db.ListOptions{Page: 2, PageSize: 2}
	assert.Equal(t, []int64{4, 5}, ids(grouped)) // the order holds across pages
}

func TestCountNotificationsByRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	unread, err := activities_model.CountNotificationsByRepo(t.Context(), activities_model.FindNotificationOptions{
		UserID: 2,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread},
	})
	assert.NoError(t, err)
	assert.Equal(t, map[int64]int64{1: 1, 2: 1}, unread)

	filtered, err := activities_model.CountNotificationsByRepo(t.Context(), activities_model.FindNotificationOptions{UserID: 2, RepoID: 1})
	assert.NoError(t, err)
	assert.Equal(t, map[int64]int64{1: 3}, filtered)

	none, err := activities_model.CountNotificationsByRepo(t.Context(), activities_model.FindNotificationOptions{
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
	assert.NoError(t, activities_model.CreateOrUpdateIssueNotifications(t.Context(), 1, 0, 2, 0, activities_model.NotificationReasonSubscribed))
	notf := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{UserID: user.ID, IssueID: 1})
	assert.Equal(t, activities_model.NotificationStatusUnread, notf.Status)
}
