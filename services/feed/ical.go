// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"fmt"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"

	ics "github.com/arran4/golang-ical"
)

// ICalOptions carries everything a calendar needs besides its rows, so the builders stay free of request state
type ICalOptions struct {
	Name         string         // X-WR-CALNAME, what calendar apps call the subscription
	RepoFullName string         // "owner/repo", shown in every event title
	RepoURL      string         // absolute repository URL
	Domain       string         // host part of every UID
	Location     *time.Location // zone the deadlines are pinned in, setting.DefaultUILocation in production
	Now          time.Time      // DTSTAMP
}

// ICalUID uses the row ID, never a date or title, so clients move events instead of duplicating them;
// changing this format duplicates every subscriber's events
func ICalUID(kind string, id int64, domain string) string {
	return fmt.Sprintf("%s-%d@%s", kind, id, domain)
}

func newICalendar(opts ICalOptions) *ics.Calendar {
	cal := ics.NewCalendar()
	cal.SetProductId("-//Gitea//Gitea " + setting.AppVer + "//EN")
	cal.SetMethod(ics.MethodPublish)
	cal.SetCalscale("GREGORIAN")
	cal.SetXWRCalName(opts.Name)
	return cal
}

func addAllDayEvent(cal *ics.Calendar, opts ICalOptions, uid, summary, url string, deadline, updated timeutil.TimeStamp) {
	day := deadline.AsTimeInLocation(opts.Location) // all-day dates carry no zone, so pick the day where the deadline was set
	event := cal.AddEvent(uid)
	event.SetDtStampTime(opts.Now)
	event.SetModifiedAt(updated.AsTime())
	event.SetAllDayStartAt(day)
	event.SetAllDayEndAt(day.AddDate(0, 0, 1)) // DTEND is exclusive
	event.SetSummary(summary)
	event.SetURL(url)
	event.SetDescription(url)
}

// MilestonesICal lists milestones that have a due date as all-day events
func MilestonesICal(opts ICalOptions, milestones []*issues_model.Milestone) *ics.Calendar {
	cal := newICalendar(opts)
	for _, m := range milestones {
		if !m.HasDeadline() {
			continue
		}
		addAllDayEvent(cal, opts, ICalUID("milestone", m.ID, opts.Domain),
			fmt.Sprintf("%s (%s)", m.Name, opts.RepoFullName),
			fmt.Sprintf("%s/milestone/%d", opts.RepoURL, m.ID),
			m.DeadlineUnix, m.UpdatedUnix)
	}
	return cal
}

// IssuesICal lists issues and pull requests that have a deadline as all-day events
func IssuesICal(opts ICalOptions, issues issues_model.IssueList) *ics.Calendar {
	cal := newICalendar(opts)
	for _, issue := range issues {
		if issue.DeadlineUnix == 0 {
			continue
		}
		path := "issues"
		if issue.IsPull {
			path = "pulls"
		}
		addAllDayEvent(cal, opts, ICalUID("issue", issue.ID, opts.Domain),
			fmt.Sprintf("%s (%s#%d)", issue.Title, opts.RepoFullName, issue.Index),
			fmt.Sprintf("%s/%s/%d", opts.RepoURL, path, issue.Index),
			issue.DeadlineUnix, issue.UpdatedUnix)
	}
	return cal
}
