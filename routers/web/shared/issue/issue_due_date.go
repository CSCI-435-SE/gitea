// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/optional"
)

// DueDateFilter is the "due" query parameter of the issue lists, resolved to inclusive deadline bounds
type DueDateFilter struct {
	Name        string // the query value, empty when absent or unknown
	AfterUnix   int64
	BeforeUnix  int64
	HasDeadline optional.Option[bool]
}

// ParseDueDateFilter resolves the "due" query parameter against now. Calendar options use now's
// location, so callers pass a time in setting.DefaultUILocation, where deadlines are pinned to the
// end of their day. An unknown value silently turns the filter off, so a stale link still renders.
func ParseDueDateFilter(due string, now time.Time) DueDateFilter {
	f := DueDateFilter{Name: due, HasDeadline: optional.Some(true)}
	y, m, d := now.Date()
	switch due {
	case "overdue":
		// same comparisons as the overdue and near-due badges
		f.BeforeUnix = now.Unix()
	case "soon":
		f.AfterUnix = now.Unix() + 1
		f.BeforeUnix = now.Add(issues_model.NearDueDuration).Unix()
	case "today":
		f.AfterUnix = time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Unix()
		f.BeforeUnix = time.Date(y, m, d+1, 0, 0, 0, 0, now.Location()).Unix() - 1
	case "week":
		monday := d - (int(now.Weekday())+6)%7 // weeks start on Monday, as in ISO 8601
		f.AfterUnix = time.Date(y, m, monday, 0, 0, 0, 0, now.Location()).Unix()
		f.BeforeUnix = time.Date(y, m, monday+7, 0, 0, 0, 0, now.Location()).Unix() - 1
	case "set":
	case "none":
		f.HasDeadline = optional.Some(false)
	default:
		return DueDateFilter{}
	}
	return f
}

// Apply narrows opts to the issues the filter matches
func (f DueDateFilter) Apply(opts *issues_model.IssuesOptions) {
	opts.DeadlineAfterUnix = f.AfterUnix
	opts.DeadlineBeforeUnix = f.BeforeUnix
	opts.HasDeadline = f.HasDeadline
}
