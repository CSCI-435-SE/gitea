// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"time"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/templates"
	"gitea.dev/modules/timeutil"
	shared_user "gitea.dev/routers/web/shared/user"
	"gitea.dev/services/context"
)

const (
	tplCalendar templates.TplName = "repo/calendar"

	calendarCellLimit   = 3                                // entries a day cell shows before "+N more"
	calendarMaxEntries  = 1000                             // bounds the query on very busy months
	milestoneNoDeadline = timeutil.TimeStamp(253370764800) // 9999-01-01 UTC, how older versions stored "no due date"
)

type calendarEntry struct {
	Kind  string // "milestone", "issue" or "pull"
	State string // "open", "closed" or "merged"
	Index int64  // 0 for milestones
	Title string
	Link  string
	unix  int64 // the deadline, places and orders the entry
}

type calendarDay struct {
	Date    time.Time
	InMonth bool
	IsToday bool
	Entries []calendarEntry
}

// Visible is what the cell shows directly, Overflow goes behind "+N more"
func (d calendarDay) Visible() []calendarEntry {
	n := min(len(d.Entries), calendarCellLimit)
	return d.Entries[:n:n] // clipped so an append can't overwrite Overflow()[0]
}

func (d calendarDay) Overflow() []calendarEntry {
	return d.Entries[min(len(d.Entries), calendarCellLimit):]
}

// parseCalendarMonth resolves the "month" query (YYYY-MM) to the first instant of that month in now's
// location. An invalid or out-of-range value falls back to now's month, so a stale link still renders.
func parseCalendarMonth(s string, now time.Time) time.Time {
	if t, err := time.ParseInLocation("2006-01", s, now.Location()); err == nil && t.Year() >= 1970 && t.Year() <= 9998 {
		return t
	}
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

func issueCalendarEntry(repoLink string, issue *issues_model.Issue) calendarEntry {
	kind, path, state := "issue", "issues", "open"
	if issue.IsPull {
		kind, path = "pull", "pulls"
	}
	if issue.IsClosed {
		state = "closed"
	}
	if issue.PullRequest != nil && issue.PullRequest.HasMerged {
		state = "merged"
	}
	return calendarEntry{
		Kind:  kind,
		State: state,
		Index: issue.Index,
		Title: issue.Title,
		Link:  fmt.Sprintf("%s/%s/%d", repoLink, path, issue.Index),
		unix:  int64(issue.DeadlineUnix),
	}
}

func milestoneCalendarEntry(repoLink string, m *issues_model.Milestone) calendarEntry {
	return calendarEntry{
		Kind:  "milestone",
		State: "open", // only open milestones are shown
		Title: m.Name,
		Link:  fmt.Sprintf("%s/milestone/%d", repoLink, m.ID),
		unix:  int64(m.DeadlineUnix),
	}
}

func compareCalendarEntries(a, b calendarEntry) int {
	if aMilestone, bMilestone := a.Kind == "milestone", b.Kind == "milestone"; aMilestone != bMilestone {
		if aMilestone {
			return -1
		}
		return 1
	}
	return cmp.Or(cmp.Compare(a.unix, b.unix), cmp.Compare(a.Index, b.Index), cmp.Compare(a.Title, b.Title))
}

// calendarWeekdays are locale key suffixes, Monday first like buildMonthGrid
var calendarWeekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// buildMonthGrid lays out the whole weeks (Monday first, as ISO 8601 and the "week" due filter) that
// cover monthStart's month, placing each entry on the day its deadline falls on in monthStart's location.
// Web forms pin deadlines to 23:59:59 there, but API-set ones may not be pinned, so they follow the same day rule.
func buildMonthGrid(monthStart, now time.Time, entries []calendarEntry) [][]calendarDay {
	loc := monthStart.Location()
	y, m, _ := monthStart.Date()
	lastDay := time.Date(y, m+1, 0, 0, 0, 0, 0, loc).Day()
	ny, nm, nd := now.In(loc).Date()

	byDate := make(map[string][]calendarEntry)
	for _, e := range entries {
		key := time.Unix(e.unix, 0).In(loc).Format(time.DateOnly)
		byDate[key] = append(byDate[key], e)
	}
	for _, es := range byDate {
		slices.SortStableFunc(es, compareCalendarEntries)
	}

	var weeks [][]calendarDay
	// day-of-month of the grid's first Monday, <= 0 when it falls in the previous month
	for d := 1 - (int(monthStart.Weekday())+6)%7; d <= lastDay; d += 7 {
		week := make([]calendarDay, 7)
		for i := range week {
			date := time.Date(y, m, d+i, 0, 0, 0, 0, loc) // normalised by time.Date, never by adding 24h, so DST can't shift it
			dy, dm, dd := date.Date()
			dayEntries := byDate[date.Format(time.DateOnly)]
			week[i] = calendarDay{Date: date, InMonth: dm == m, IsToday: dy == ny && dm == nm && dd == nd, Entries: dayEntries}
		}
		weeks = append(weeks, week)
	}
	return weeks
}

// calendarIsPull limits the query to the units the doer can read; the route guard ensures at least one
func calendarIsPull(canReadIssues, canReadPulls bool) optional.Option[bool] {
	switch {
	case canReadIssues && canReadPulls:
		return optional.None[bool]()
	case canReadPulls:
		return optional.Some(true)
	default:
		return optional.Some(false)
	}
}

// calendarMilestones keeps the open milestones due within [start, end]. milestoneID follows the milestone
// filter: 0 keeps all, -1 ("no milestone") keeps none, any other value keeps only that milestone.
func calendarMilestones(milestones []*issues_model.Milestone, start, end time.Time, milestoneID int64) []*issues_model.Milestone {
	var kept []*issues_model.Milestone
	for _, m := range milestones {
		if m.IsClosed || m.DeadlineUnix <= 0 || m.DeadlineUnix >= milestoneNoDeadline {
			continue
		}
		if int64(m.DeadlineUnix) < start.Unix() || int64(m.DeadlineUnix) > end.Unix() {
			continue
		}
		if milestoneID != 0 && milestoneID != m.ID {
			continue
		}
		kept = append(kept, m)
	}
	return kept
}

// calendarMonthLink formats a prev/next month link, or "" when parseCalendarMonth would reject it and fall back to today
func calendarMonthLink(t time.Time) string {
	if t.Year() < 1970 || t.Year() > 9998 {
		return ""
	}
	return t.Format("2006-01")
}

// Calendar renders a month grid of the repository's milestone due dates and issue and pull request deadlines
func Calendar(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.calendar")
	ctx.Data["PageIsIssueList"] = true

	now := timeutil.TimeStampNow().AsTime() // in setting.DefaultUILocation, where deadlines are pinned to 23:59:59
	monthStart := parseCalendarMonth(ctx.FormString("month"), now)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	milestoneID := ctx.FormInt64("milestone")
	if milestoneID < 0 && milestoneID != db.NoConditionID {
		milestoneID = 0 // keeps the issue and milestone filters in agreement
	}
	assigneeID := ctx.FormString("assignee")
	state := ctx.FormString("state")
	var isClosed optional.Option[bool]
	switch state {
	case "closed":
		isClosed = optional.Some(true)
	case "all":
	default:
		state = "open"
		isClosed = optional.Some(false)
	}

	opts := &issues_model.IssuesOptions{
		Paginator:          &db.ListOptions{Page: 1, PageSize: calendarMaxEntries + 1},
		RepoIDs:            []int64{ctx.Repo.Repository.ID},
		AssigneeID:         assigneeID,
		IsClosed:           isClosed,
		IsPull:             calendarIsPull(ctx.Repo.Permission.CanRead(unit.TypeIssues), ctx.Repo.Permission.CanRead(unit.TypePullRequests)),
		DeadlineAfterUnix:  monthStart.Unix(),
		DeadlineBeforeUnix: monthEnd.Unix(),
		HasDeadline:        optional.Some(true),
		SortType:           "oldest", // deterministic when the cap truncates; cells re-sort by deadline
	}
	if milestoneID > 0 || milestoneID == db.NoConditionID {
		opts.MilestoneIDs = []int64{milestoneID}
	}
	issues, err := issues_model.Issues(ctx, opts) // also loads the PRs that issueCalendarEntry needs for the merged state
	if err != nil {
		ctx.ServerError("Issues", err)
		return
	}
	truncated := len(issues) > calendarMaxEntries // the extra row only detects truncation
	if truncated {
		issues = issues[:calendarMaxEntries]
	}

	milestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{RepoID: ctx.Repo.Repository.ID})
	if err != nil {
		ctx.ServerError("FindMilestones", err)
		return
	}

	entries := make([]calendarEntry, 0, len(issues))
	for _, m := range calendarMilestones(milestones, monthStart, monthEnd, milestoneID) {
		entries = append(entries, milestoneCalendarEntry(ctx.Repo.RepoLink, m))
	}
	for _, issue := range issues {
		entries = append(entries, issueCalendarEntry(ctx.Repo.RepoLink, issue))
	}

	assignees, err := repo_model.GetRepoAssignees(ctx, ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("GetRepoAssignees", err)
		return
	}
	ctx.Data["Assignees"] = shared_user.MakeSelfOnTop(ctx.Doer, assignees)
	ctx.Data["OpenMilestones"], ctx.Data["ClosedMilestones"] = issues_model.MilestoneList(milestones).SplitByOpenClosed()

	ctx.Data["CalendarWeekdays"] = calendarWeekdays
	ctx.Data["CalendarHasEntries"] = len(entries) > 0
	ctx.Data["Weeks"] = buildMonthGrid(monthStart, now, entries)
	ctx.Data["MonthNum"] = int(monthStart.Month())
	ctx.Data["Year"] = monthStart.Year()
	ctx.Data["Month"] = monthStart.Format("2006-01")
	ctx.Data["PrevMonth"] = calendarMonthLink(monthStart.AddDate(0, -1, 0))
	ctx.Data["NextMonth"] = calendarMonthLink(monthStart.AddDate(0, 1, 0))
	ctx.Data["MilestoneID"] = milestoneID
	ctx.Data["AssigneeID"] = assigneeID
	ctx.Data["State"] = state
	ctx.Data["CalendarTruncated"] = truncated
	ctx.Data["CalendarMaxEntries"] = calendarMaxEntries
	ctx.HTML(http.StatusOK, tplCalendar)
}
