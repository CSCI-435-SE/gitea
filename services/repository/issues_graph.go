// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"time"

	issues_model "gitea.dev/models/issues"
)

// IssueWeek is one week of issue activity in a repository.
type IssueWeek struct {
	Week   int64 `json:"week"`   // Sunday 00:00 UTC that starts the week, in Unix milliseconds to match the other activity charts
	Open   int   `json:"open"`   // Issues still open when the week ended
	Opened int   `json:"opened"` // Issues created during the week
	Closed int   `json:"closed"` // Issues closed during the week
}

const weekSeconds int64 = 7 * 24 * 60 * 60

// epochToSunday is the distance from the Unix epoch back to the Sunday before it. The
// epoch fell on a Thursday, so weeks only align to Sunday after shifting by four days.
const epochToSunday int64 = 4 * 24 * 60 * 60

// weekStart rounds a Unix timestamp in seconds down to the Sunday that starts its week.
func weekStart(unix int64) int64 {
	return (unix+epochToSunday)/weekSeconds*weekSeconds - epochToSunday
}

// CalcIssueWeeks buckets issue lifetimes into the weeks spanning start and end.
//
// An issue counts as open in a week when it was created before that week ended and was
// either never closed or closed after the week ended, which is how a maintainer reads a
// backlog: an issue closed next month was still open this month. Issues created after
// end are ignored, and one closed after end stays open for every week in range.
func CalcIssueWeeks(lifetimes []issues_model.IssueLifetime, start, end time.Time) []IssueWeek {
	first, last := weekStart(start.Unix()), weekStart(end.Unix())
	if last < first {
		return []IssueWeek{}
	}

	weeks := make([]IssueWeek, 0, (last-first)/weekSeconds+1)
	index := make(map[int64]int, cap(weeks))
	for w := first; w <= last; w += weekSeconds {
		index[w] = len(weeks)
		weeks = append(weeks, IssueWeek{Week: w * 1000})
	}

	// issues that opened or closed before the range still shift the running open count
	var openBeforeRange int
	for _, lifetime := range lifetimes {
		created := weekStart(int64(lifetime.CreatedUnix))
		if created < first {
			openBeforeRange++
		} else if i, ok := index[created]; ok {
			weeks[i].Opened++
		}

		if !lifetime.IsClosed {
			continue
		}
		// a close time of zero predates the repository, so clamp it to keep the count monotone
		closedUnix := max(int64(lifetime.ClosedUnix), int64(lifetime.CreatedUnix))
		closed := weekStart(closedUnix)
		if closed < first {
			openBeforeRange--
		} else if i, ok := index[closed]; ok {
			weeks[i].Closed++
		}
	}

	running := openBeforeRange
	for i := range weeks {
		running += weeks[i].Opened - weeks[i].Closed
		weeks[i].Open = running
	}
	return weeks
}

// GetRepoIssueWeeks charts a repository's issues from the week its first issue was
// opened through the current week. It returns nil when the repository has no issues,
// which the page renders as an empty state.
func GetRepoIssueWeeks(ctx context.Context, repoID int64) ([]IssueWeek, error) {
	lifetimes, err := issues_model.GetRepoIssueLifetimes(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if len(lifetimes) == 0 {
		return nil, nil
	}

	// the query orders by creation, so the first row opens the range
	start := time.Unix(int64(lifetimes[0].CreatedUnix), 0).UTC()
	return CalcIssueWeeks(lifetimes, start, time.Now().UTC()), nil
}
