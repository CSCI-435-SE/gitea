// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	stdCtx "context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	activities_model "gitea.dev/models/activities"
	"gitea.dev/models/db"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/base"
	"gitea.dev/modules/container"
	"gitea.dev/modules/log"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/structs"
	"gitea.dev/modules/templates"
	"gitea.dev/modules/util"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
	pull_service "gitea.dev/services/pull"
)

const (
	tplNotification              templates.TplName = "user/notification/notification"
	tplNotificationDiv           templates.TplName = "user/notification/notification_div"
	tplNotificationSubscriptions templates.TplName = "user/notification/notification_subscriptions"
)

// Notifications is the notification list page
func Notifications(ctx *context.Context) {
	prepareUserNotificationsData(ctx)
	if ctx.Written() {
		return
	}
	if ctx.FormBool("div-only") {
		ctx.HTML(http.StatusOK, tplNotificationDiv)
		return
	}
	ctx.HTML(http.StatusOK, tplNotification)
}

func prepareUserNotificationsData(ctx *context.Context) {
	pageType := ctx.FormString("type", ctx.FormString("q")) // "q" is the legacy query parameter for "page type"
	page := max(1, ctx.FormInt("page"))
	perPage := util.IfZero(ctx.FormInt("perPage"), 20) // this value is never used or exposed ....
	queryStatus := notificationViewStatus(pageType)
	filter := parseNotificationFilter(ctx)

	countOpts := filter.findOptions(ctx.Doer.ID)
	countOpts.Status = []activities_model.NotificationStatus{queryStatus}
	total, err := db.Count[activities_model.Notification](ctx, countOpts)
	if err != nil {
		ctx.ServerError("ErrGetNotificationCount", err)
		return
	}

	unreadOpts := filter.findOptions(ctx.Doer.ID)
	unreadOpts.Status = []activities_model.NotificationStatus{activities_model.NotificationStatusUnread}
	filteredUnreadCount, err := db.Count[activities_model.Notification](ctx, unreadOpts) // the navbar badge stays global
	if err != nil {
		ctx.ServerError("ErrGetNotificationCount", err)
		return
	}

	viewOpts := filter.findOptions(ctx.Doer.ID)
	viewOpts.Status = []activities_model.NotificationStatus{queryStatus, activities_model.NotificationStatusPinned}
	viewCount, err := db.Count[activities_model.Notification](ctx, viewOpts) // what "select all in this view" acts on
	if err != nil {
		ctx.ServerError("ErrGetNotificationCount", err)
		return
	}

	filterRepos, unreadRepoIDs, err := loadNotificationFilterRepos(ctx, ctx.Doer)
	if err != nil {
		ctx.ServerError("loadNotificationFilterRepos", err)
		return
	}

	pager := context.NewPagination(total, perPage, page, 5)
	if pager.Paginater.Current() < page {
		// use the last page if the requested page is more than total pages
		page = pager.Paginater.Current()
		pager = context.NewPagination(total, perPage, page, 5)
	}

	findOpts := filter.findOptions(ctx.Doer.ID) // pinned rows are filtered like the rest
	findOpts.ListOptions = db.ListOptions{
		PageSize: perPage,
		Page:     page,
	}
	findOpts.Status = []activities_model.NotificationStatus{queryStatus, activities_model.NotificationStatusPinned}
	nls, err := db.Find[activities_model.Notification](ctx, findOpts)
	if err != nil {
		ctx.ServerError("db.Find[activities_model.Notification]", err)
		return
	}

	notifications := activities_model.NotificationList(nls)

	failCount := 0

	repos, failures, err := notifications.LoadRepos(ctx)
	if err != nil {
		ctx.ServerError("LoadRepos", err)
		return
	}
	notifications = notifications.Without(failures)
	if err := repos.LoadAttributes(ctx); err != nil {
		ctx.ServerError("LoadAttributes", err)
		return
	}
	failCount += len(failures)
	notifications, failures, err = filterNotificationsByRepoAccess(ctx, ctx.Doer, notifications)
	if err != nil {
		ctx.ServerError("filterNotificationsByRepoAccess", err)
		return
	}
	failCount += len(failures)

	failures, err = notifications.LoadIssues(ctx)
	if err != nil {
		ctx.ServerError("LoadIssues", err)
		return
	}

	if err = notifications.LoadIssuePullRequests(ctx); err != nil {
		ctx.ServerError("LoadIssuePullRequests", err)
		return
	}

	notifications = notifications.Without(failures)
	failCount += len(failures)

	failures, err = notifications.LoadComments(ctx)
	if err != nil {
		ctx.ServerError("LoadComments", err)
		return
	}
	notifications = notifications.Without(failures)
	failCount += len(failures)

	if failCount > 0 {
		ctx.Flash.Error(fmt.Sprintf("ERROR: %d notifications were removed due to missing parts - check the logs", failCount))
	}

	ctx.Data["Title"] = ctx.Tr("notifications")
	ctx.Data["PageType"] = pageType
	ctx.Data["Notifications"] = notifications
	ctx.Data["Link"] = setting.AppSubURL + "/notifications"
	ctx.Data["SequenceNumber"] = ctx.FormString("sequence-number")
	ctx.Data["FilterRepoID"] = filter.RepoID
	ctx.Data["FilterSource"] = filter.Source
	ctx.Data["IsFiltered"] = filter.IsActive()
	ctx.Data["FilterRepos"] = filterRepos
	ctx.Data["FilterRepoUnread"] = unreadRepoIDs
	ctx.Data["FilteredUnreadCount"] = filteredUnreadCount
	ctx.Data["ViewCount"] = viewCount
	ctx.Data["ViewKey"] = notificationViewLink(pageType, 1, filter) // selections are kept per view, across its pages
	for _, repo := range filterRepos {
		if repo.ID == filter.RepoID {
			ctx.Data["FilterRepo"] = repo // only accessible repos are named, a guessed ID shows the generic label
			break
		}
	}

	pager.AddParamFromRequest(ctx.Req)
	pager.RemoveParam(container.SetOf("div-only", "sequence-number"))
	ctx.Data["Page"] = pager
}

func filterNotificationsByRepoAccess(ctx stdCtx.Context, doer *user_model.User, notifications activities_model.NotificationList) (activities_model.NotificationList, []int, error) {
	failures := make([]int, 0)
	for i, notification := range notifications {
		if notification.Repository == nil {
			continue
		}
		perm, err := access_model.GetIndividualUserRepoPermission(ctx, notification.Repository, doer)
		if err != nil {
			return nil, nil, err
		}
		if !perm.HasAnyUnitAccessOrPublicAccess() {
			failures = append(failures, i)
		}
	}
	return notifications.Without(failures), failures, nil
}

// notificationSourceParams maps the "source" query value to the notification source it selects
var notificationSourceParams = map[string]activities_model.NotificationSource{
	"issue":      activities_model.NotificationSourceIssue,
	"pull":       activities_model.NotificationSourcePullRequest,
	"commit":     activities_model.NotificationSourceCommit,
	"repository": activities_model.NotificationSourceRepository,
}

// notificationFilter is the repository and type filter of the notifications page
type notificationFilter struct {
	RepoID int64
	Source string // a key of notificationSourceParams, or "" for any
}

func parseNotificationFilter(ctx *context.Context) notificationFilter {
	filter := notificationFilter{RepoID: max(0, ctx.FormInt64("repo"))}
	if source := ctx.FormString("source"); notificationSourceParams[source] != 0 {
		filter.Source = source // unknown values are ignored rather than matching nothing
	}
	return filter
}

func (f notificationFilter) IsActive() bool {
	return f.RepoID != 0 || f.Source != ""
}

func (f notificationFilter) findOptions(userID int64) activities_model.FindNotificationOptions {
	opts := activities_model.FindNotificationOptions{UserID: userID, RepoID: f.RepoID}
	if f.Source != "" {
		opts.Source = []activities_model.NotificationSource{notificationSourceParams[f.Source]}
	}
	return opts
}

// queryValues is built from the parsed values, never the raw request, so redirects only carry validated input
func (f notificationFilter) queryValues() url.Values {
	q := url.Values{}
	if f.RepoID != 0 {
		q.Set("repo", strconv.FormatInt(f.RepoID, 10))
	}
	if f.Source != "" {
		q.Set("source", f.Source)
	}
	return q
}

func (f notificationFilter) queryString() string {
	return f.queryValues().Encode()
}

// notificationViewStatus is the status a tab lists; pinned notifications are listed on both tabs
func notificationViewStatus(pageType string) activities_model.NotificationStatus {
	return util.Iif(pageType == "read", activities_model.NotificationStatusRead, activities_model.NotificationStatusUnread)
}

// notificationViewLink links to a page of a tab with its filter, normalising the tab so equal views get equal links
func notificationViewLink(pageType string, page int, filter notificationFilter) string {
	q := filter.queryValues()
	if notificationViewStatus(pageType) == activities_model.NotificationStatusRead {
		q.Set("type", "read")
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	link := setting.AppSubURL + "/notifications"
	if len(q) > 0 {
		link += "?" + q.Encode()
	}
	return link
}

// loadNotificationFilterRepos returns the repositories offered by the repository filter, sorted by full name,
// and the IDs of those with unread notifications
func loadNotificationFilterRepos(ctx stdCtx.Context, doer *user_model.User) (repo_model.RepositoryList, container.Set[int64], error) {
	searchOpts := repo_model.SearchRepoOptions{
		Actor:   doer,
		OwnerID: doer.ID,
		Private: true,
	}
	repos, _, err := repo_model.SearchRepositoryByCondition(ctx, searchOpts, repo_model.SearchRepositoryCondition(searchOpts), false)
	if err != nil {
		return nil, nil, err
	}

	// watched repos the user neither owns nor collaborates on are missing from the search
	notifRepoIDs, err := activities_model.FindNotificationRepoIDs(ctx, activities_model.FindNotificationOptions{UserID: doer.ID})
	if err != nil {
		return nil, nil, err
	}
	known := container.SetOf(repos.IDs()...)
	missingIDs := slices.DeleteFunc(notifRepoIDs, func(id int64) bool { return known.Contains(id) })
	if len(missingIDs) > 0 {
		missing, err := repo_model.GetRepositoriesMapByIDs(ctx, missingIDs)
		if err != nil {
			return nil, nil, err
		}
		for _, repo := range missing {
			perm, err := access_model.GetIndividualUserRepoPermission(ctx, repo, doer)
			if err != nil {
				return nil, nil, err
			}
			if perm.HasAnyUnitAccessOrPublicAccess() {
				repos = append(repos, repo)
			}
		}
	}
	slices.SortFunc(repos, func(a, b *repo_model.Repository) int {
		return strings.Compare(strings.ToLower(a.FullName()), strings.ToLower(b.FullName()))
	})

	unreadIDs, err := activities_model.FindNotificationRepoIDs(ctx, activities_model.FindNotificationOptions{
		UserID: doer.ID,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread},
	})
	if err != nil {
		return nil, nil, err
	}
	return repos, container.SetOf(unreadIDs...), nil
}

// NotificationStatusPost is a route for changing the status of a notification
func NotificationStatusPost(ctx *context.Context) {
	notificationID := ctx.FormInt64("notification_id")
	var newStatus activities_model.NotificationStatus
	switch ctx.FormString("notification_action") {
	case "mark_as_read":
		newStatus = activities_model.NotificationStatusRead
	case "mark_as_unread":
		newStatus = activities_model.NotificationStatusUnread
	case "pin":
		newStatus = activities_model.NotificationStatusPinned
	default:
		return // ignore user's invalid input
	}
	if _, err := activities_model.SetNotificationStatus(ctx, notificationID, ctx.Doer, newStatus); err != nil {
		ctx.ServerError("SetNotificationStatus", err)
		return
	}

	prepareUserNotificationsData(ctx)
	if ctx.Written() {
		return
	}
	ctx.HTML(http.StatusOK, tplNotificationDiv)
}

// NotificationPurgePost is a route for 'purging' the list of notifications - marking all unread as read
func NotificationPurgePost(ctx *context.Context) {
	filter := parseNotificationFilter(ctx) // only the notifications the user is looking at are marked
	err := activities_model.UpdateNotificationStatuses(ctx, ctx.Doer, activities_model.NotificationStatusUnread, activities_model.NotificationStatusRead, filter.findOptions(ctx.Doer.ID))
	if err != nil {
		ctx.ServerError("UpdateNotificationStatuses", err)
		return
	}

	redirect := setting.AppSubURL + "/notifications"
	if q := filter.queryString(); q != "" {
		redirect += "?" + q
	}
	ctx.Redirect(redirect, http.StatusSeeOther)
}

// notificationBulkAction is the status a bulk action sets, and the flash keys reporting how many changed
type notificationBulkAction struct {
	status              activities_model.NotificationStatus // 0 deletes the notifications
	flashOne, flashMany string
}

var notificationBulkActions = map[string]notificationBulkAction{
	"mark_as_read":   {activities_model.NotificationStatusRead, "notification.bulk_marked_read_1", "notification.bulk_marked_read_n"},
	"mark_as_unread": {activities_model.NotificationStatusUnread, "notification.bulk_marked_unread_1", "notification.bulk_marked_unread_n"},
	"pin":            {activities_model.NotificationStatusPinned, "notification.bulk_pinned_1", "notification.bulk_pinned_n"},
	"delete":         {0, "notification.bulk_deleted_1", "notification.bulk_deleted_n"},
}

// parseNotificationBulkTarget returns the notifications a bulk request acts on: the listed IDs,
// or with all=true every notification in the view, re-applied here instead of trusting IDs from the browser
func parseNotificationBulkTarget(ctx *context.Context, filter notificationFilter) (activities_model.FindNotificationOptions, bool) {
	if ctx.FormBool("all") {
		opts := filter.findOptions(ctx.Doer.ID)
		opts.Status = []activities_model.NotificationStatus{notificationViewStatus(ctx.FormString("type")), activities_model.NotificationStatusPinned}
		return opts, true
	}
	ids, err := base.StringsToInt64s(strings.Split(ctx.FormString("notification_ids"), ","))
	if err != nil || len(ids) == 0 {
		return activities_model.FindNotificationOptions{}, false
	}
	return activities_model.FindNotificationOptions{IDs: ids}, true // the model limits them to the doer's own
}

// NotificationBulkPost applies one action to several notifications and redirects back to the view
func NotificationBulkPost(ctx *context.Context) {
	action, ok := notificationBulkActions[ctx.FormString("action")]
	if !ok {
		ctx.JSONError("unknown notification action")
		return
	}
	filter := parseNotificationFilter(ctx)
	opts, ok := parseNotificationBulkTarget(ctx, filter)
	if !ok {
		ctx.JSONError("no valid notifications selected")
		return
	}

	var changed int64
	var err error
	if action.status == 0 {
		changed, err = activities_model.DeleteNotifications(ctx, ctx.Doer, opts)
	} else {
		changed, err = activities_model.SetNotificationsStatus(ctx, ctx.Doer, opts, action.status)
	}
	if err != nil {
		ctx.ServerError("NotificationBulkPost", err)
		return
	}

	ctx.Flash.Success(ctx.Locale.TrN(changed, action.flashOne, action.flashMany, changed))
	ctx.JSONRedirect(notificationViewLink(ctx.FormString("type"), ctx.FormInt("page"), filter)) // the list page clamps a page left empty
}

// NotificationSubscriptions returns the list of subscribed issues
func NotificationSubscriptions(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)

	sortType := ctx.FormString("sort")
	ctx.Data["SortType"] = sortType

	state := ctx.FormString("state")
	if !util.SliceContainsString([]string{"all", "open", "closed"}, state, true) {
		state = "all"
	}

	ctx.Data["State"] = state
	// default state filter is "all"
	showClosed := optional.None[bool]()
	switch state {
	case "closed":
		showClosed = optional.Some(true)
	case "open":
		showClosed = optional.Some(false)
	}

	issueType := ctx.FormString("issueType")
	// default issue type is no filter
	issueTypeBool := optional.None[bool]()
	switch issueType {
	case "issues":
		issueTypeBool = optional.Some(false)
	case "pulls":
		issueTypeBool = optional.Some(true)
	}
	ctx.Data["IssueType"] = issueType

	var labelIDs []int64
	selectedLabels := ctx.FormString("labels")
	ctx.Data["Labels"] = selectedLabels
	if len(selectedLabels) > 0 && selectedLabels != "0" {
		var err error
		labelIDs, err = base.StringsToInt64s(strings.Split(selectedLabels, ","))
		if err != nil {
			ctx.Flash.Error(ctx.Tr("invalid_data", selectedLabels), true)
		}
	}

	count, err := issues_model.CountIssues(ctx, &issues_model.IssuesOptions{
		SubscriberID: ctx.Doer.ID,
		IsClosed:     showClosed,
		IsPull:       issueTypeBool,
		LabelIDs:     labelIDs,
	})
	if err != nil {
		ctx.ServerError("CountIssues", err)
		return
	}
	issues, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
		Paginator: &db.ListOptions{
			PageSize: setting.UI.IssuePagingNum,
			Page:     page,
		},
		SubscriberID: ctx.Doer.ID,
		SortType:     sortType,
		IsClosed:     showClosed,
		IsPull:       issueTypeBool,
		LabelIDs:     labelIDs,
	})
	if err != nil {
		ctx.ServerError("Issues", err)
		return
	}

	commitStatuses, lastStatus, err := pull_service.GetIssuesAllCommitStatus(ctx, issues)
	if err != nil {
		ctx.ServerError("GetIssuesAllCommitStatus", err)
		return
	}
	if !ctx.Repo.Permission.CanRead(unit.TypeActions) {
		for key := range commitStatuses {
			git_model.CommitStatusesHideActionsURL(ctx, commitStatuses[key])
		}
	}
	ctx.Data["CommitLastStatus"] = lastStatus
	ctx.Data["CommitStatuses"] = commitStatuses
	ctx.Data["Issues"] = issues
	ctx.Data["IssueRefEndNames"], ctx.Data["IssueRefURLs"] = issue_service.GetRefEndNamesAndURLs(issues, "")

	approvalCounts, err := issues.GetApprovalCounts(ctx)
	if err != nil {
		ctx.ServerError("ApprovalCounts", err)
		return
	}
	ctx.Data["ApprovalCounts"] = func(issueID int64, typ string) int64 {
		counts, ok := approvalCounts[issueID]
		if !ok || len(counts) == 0 {
			return 0
		}
		reviewTyp := issues_model.ReviewTypeApprove
		switch typ {
		case "reject":
			reviewTyp = issues_model.ReviewTypeReject
		case "waiting":
			reviewTyp = issues_model.ReviewTypeRequest
		}
		for _, count := range counts {
			if count.Type == reviewTyp {
				return count.Count
			}
		}
		return 0
	}

	ctx.Data["Status"] = 1
	ctx.Data["Title"] = ctx.Tr("notification.subscriptions")

	// redirect to last page if request page is more than total pages
	pager := context.NewPagination(count, setting.UI.IssuePagingNum, page, 5)
	if pager.Paginater.Current() < page {
		ctx.Redirect(fmt.Sprintf("/notifications/subscriptions?page=%d", pager.Paginater.Current()))
		return
	}
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplNotificationSubscriptions)
}

// NotificationWatching returns the list of watching repos
func NotificationWatching(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)

	keyword := ctx.FormTrim("q")
	ctx.Data["Keyword"] = keyword

	var orderBy db.SearchOrderBy
	ctx.Data["SortType"] = ctx.FormString("sort")
	switch ctx.FormString("sort") {
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "recentupdate":
		orderBy = db.SearchOrderByRecentUpdated
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
	case "moststars":
		orderBy = db.SearchOrderByStarsReverse
	case "feweststars":
		orderBy = db.SearchOrderByStars
	case "mostforks":
		orderBy = db.SearchOrderByForksReverse
	case "fewestforks":
		orderBy = db.SearchOrderByForks
	default:
		ctx.Data["SortType"] = "recentupdate"
		orderBy = db.SearchOrderByRecentUpdated
	}

	archived := ctx.FormOptionalBool("archived")
	ctx.Data["IsArchived"] = archived

	fork := ctx.FormOptionalBool("fork")
	ctx.Data["IsFork"] = fork

	mirror := ctx.FormOptionalBool("mirror")
	ctx.Data["IsMirror"] = mirror

	template := ctx.FormOptionalBool("template")
	ctx.Data["IsTemplate"] = template

	private := ctx.FormOptionalBool("private")
	ctx.Data["IsPrivate"] = private

	repos, count, err := repo_model.SearchRepository(ctx, repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			PageSize: setting.UI.User.RepoPagingNum,
			Page:     page,
		},
		Actor:              ctx.Doer,
		Keyword:            keyword,
		OrderBy:            orderBy,
		Private:            ctx.IsSigned,
		WatchedByID:        ctx.Doer.ID,
		Collaborate:        optional.Some(false),
		TopicOnly:          ctx.FormBool("topic"),
		IncludeDescription: setting.UI.SearchRepoDescription,
		Archived:           archived,
		Fork:               fork,
		Mirror:             mirror,
		Template:           template,
		IsPrivate:          private,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}
	ctx.Data["Total"] = count
	ctx.Data["Repos"] = repos

	// redirect to last page if request page is more than total pages
	pager := context.NewPagination(count, setting.UI.User.RepoPagingNum, page, 5)
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	ctx.Data["Status"] = 2
	ctx.Data["Title"] = ctx.Tr("notification.watching")

	ctx.HTML(http.StatusOK, tplNotificationSubscriptions)
}

// NewAvailable returns the notification counts
func NewAvailable(ctx *context.Context) {
	total, err := db.Count[activities_model.Notification](ctx, activities_model.FindNotificationOptions{
		UserID: ctx.Doer.ID,
		Status: []activities_model.NotificationStatus{activities_model.NotificationStatusUnread},
	})
	if err != nil {
		log.Error("db.Count[activities_model.Notification]", err)
		ctx.JSON(http.StatusOK, structs.NotificationCount{New: 0})
		return
	}

	ctx.JSON(http.StatusOK, structs.NotificationCount{New: total})
}
