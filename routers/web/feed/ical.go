// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	"gitea.dev/models/unit"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/modules/translation"
	"gitea.dev/services/context"
	feed_service "gitea.dev/services/feed"

	ics "github.com/arran4/golang-ical"
)

// ICalMaxItems caps each calendar feed; a variable so tests can lower it
var ICalMaxItems = 1000

// anonymousRepoPermission is what calendar feeds serve: they never authenticate, so private repositories stay
// hidden even from members until a dedicated calendar token exists
func anonymousRepoPermission(ctx *context.Context, expensive bool) (access_model.Permission, error) {
	if setting.Service.RequireSignInViewStrict {
		return access_model.PermissionNoAccess(), nil // anonymous clients must sign in, which calendar apps cannot
	}
	if expensive && setting.Service.BlockAnonymousAccessExpensive { // blockexpensive.go matches the "/issues" prefix of issues.ics
		return access_model.PermissionNoAccess(), nil
	}
	return access_model.GetIndividualUserRepoPermission(ctx, ctx.Repo.Repository, nil)
}

// SetRepoICalFeedLink shows a list page's subscribe button only when an anonymous client could fetch the feed
func SetRepoICalFeedLink(ctx *context.Context, feedName string, expensive bool) {
	if !setting.Other.EnableFeed {
		return
	}
	perm, err := anonymousRepoPermission(ctx, expensive)
	if err != nil {
		ctx.ServerError("anonymousRepoPermission", err)
		return
	}
	if perm.CanReadAny(unit.TypeIssues, unit.TypePullRequests) {
		ctx.Data["ICalFeedLink"] = ctx.Repo.RepoLink + "/" + feedName
	}
}

// parseICalState maps ?state= to the IsClosed condition; anything unknown means open, the feed default
func parseICalState(state string) optional.Option[bool] {
	switch state {
	case "closed":
		return optional.Some(true)
	case "all":
		return optional.None[bool]()
	default:
		return optional.Some(false)
	}
}

// icalIsPull limits the rows to the units an anonymous client can read; callers ensure at least one
func icalIsPull(canReadIssues, canReadPulls bool) optional.Option[bool] {
	switch {
	case canReadIssues && canReadPulls:
		return optional.None[bool]()
	case canReadPulls:
		return optional.Some(true)
	default:
		return optional.Some(false)
	}
}

// readableICalPermission writes a 404 unless anonymous clients can read the repository's issues or pulls
func readableICalPermission(ctx *context.Context, expensive bool) (access_model.Permission, bool) {
	perm, err := anonymousRepoPermission(ctx, expensive)
	if err != nil {
		ctx.ServerError("anonymousRepoPermission", err)
		return perm, false
	}
	if !perm.CanReadAny(unit.TypeIssues, unit.TypePullRequests) {
		ctx.NotFound(nil)
		return perm, false
	}
	return perm, true
}

func repoICalOptions(ctx *context.Context, nameKey string, now timeutil.TimeStamp) feed_service.ICalOptions {
	repo := ctx.Repo.Repository
	return feed_service.ICalOptions{
		Name:         translation.NewLocale(setting.Langs[0]).TrString(nameKey, repo.FullName()), // public cache, so not the requester's language
		RepoFullName: repo.FullName(),
		RepoURL:      repo.HTMLURL(ctx),
		Domain:       setting.Domain,
		Location:     setting.DefaultUILocation,
		Now:          now.AsTime(),
	}
}

func writeICal(ctx *context.Context, cal *ics.Calendar) {
	ctx.Resp.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	ctx.Resp.Header().Set("Cache-Control", "public, max-age=300") // identical for every client, since it is anonymous-only
	if err := cal.SerializeTo(ctx.Resp, ics.WithNewLineWindows); err != nil {
		ctx.ServerError("Render iCalendar failed", err)
	}
}

// RepoMilestonesICal serves the repository's milestone due dates as an iCalendar feed
func RepoMilestonesICal(ctx *context.Context) {
	if _, ok := readableICalPermission(ctx, false); !ok {
		return
	}
	milestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: ICalMaxItems},
		RepoID:      ctx.Repo.Repository.ID,
		IsClosed:    parseICalState(ctx.FormString("state")),
		HasDeadline: true,
	})
	if err != nil {
		ctx.ServerError("FindMilestones", err)
		return
	}
	now := timeutil.TimeStampNow()
	writeICal(ctx, feed_service.MilestonesICal(repoICalOptions(ctx, "repo.ical.milestones_of", now), milestones))
}

// RepoIssuesICal serves the repository's issue and pull request deadlines as an iCalendar feed
func RepoIssuesICal(ctx *context.Context) {
	perm, ok := readableICalPermission(ctx, true)
	if !ok {
		return
	}
	now := timeutil.TimeStampNow()
	issues, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
		Paginator:             &db.ListOptions{Page: 1, PageSize: ICalMaxItems},
		RepoIDs:               []int64{ctx.Repo.Repository.ID},
		IsClosed:              parseICalState(ctx.FormString("state")),
		IsPull:                icalIsPull(perm.CanRead(unit.TypeIssues), perm.CanRead(unit.TypePullRequests)),
		HasDeadline:           optional.Some(true),
		SortType:              "closestdeadline",
		DeadlineReferenceUnix: int64(now),
	})
	if err != nil {
		ctx.ServerError("Issues", err)
		return
	}
	writeICal(ctx, feed_service.IssuesICal(repoICalOptions(ctx, "repo.ical.issues_of", now), issues))
}
