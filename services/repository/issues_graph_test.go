// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
)

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, value)
	assert.NoError(t, err)
	return parsed
}

func openIssue(t *testing.T, created string) issues_model.IssueLifetime {
	t.Helper()
	return issues_model.IssueLifetime{CreatedUnix: timeutil.TimeStamp(mustDate(t, created).Unix())}
}

func closedIssue(t *testing.T, created, closed string) issues_model.IssueLifetime {
	t.Helper()
	return issues_model.IssueLifetime{
		CreatedUnix: timeutil.TimeStamp(mustDate(t, created).Unix()),
		ClosedUnix:  timeutil.TimeStamp(mustDate(t, closed).Unix()),
		IsClosed:    true,
	}
}

func TestWeekStart(t *testing.T) {
	// 2026-01-04 is a Sunday, so every day up to the following Saturday shares its bucket
	sunday := mustDate(t, "2026-01-04").Unix()
	for _, day := range []string{"2026-01-04", "2026-01-05", "2026-01-08", "2026-01-10"} {
		assert.Equal(t, sunday, weekStart(mustDate(t, day).Unix()), day)
	}
	assert.Equal(t, mustDate(t, "2026-01-11").Unix(), weekStart(mustDate(t, "2026-01-11").Unix()))
}

func TestCalcIssueWeeks(t *testing.T) {
	// three whole weeks: 2026-01-04, 2026-01-11 and 2026-01-18, all Sundays
	start, end := mustDate(t, "2026-01-04"), mustDate(t, "2026-01-24")
	weekMillis := func(day string) int64 { return mustDate(t, day).Unix() * 1000 }

	cases := []struct {
		name      string
		lifetimes []issues_model.IssueLifetime
		expected  []IssueWeek
	}{
		{
			name:      "no issues",
			lifetimes: nil,
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04")},
				{Week: weekMillis("2026-01-11")},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			name:      "issue stays open",
			lifetimes: []issues_model.IssueLifetime{openIssue(t, "2026-01-06")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Opened: 1, Open: 1},
				{Week: weekMillis("2026-01-11"), Open: 1},
				{Week: weekMillis("2026-01-18"), Open: 1},
			},
		},
		{
			name:      "issue closed inside the range",
			lifetimes: []issues_model.IssueLifetime{closedIssue(t, "2026-01-06", "2026-01-13")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Opened: 1, Open: 1},
				{Week: weekMillis("2026-01-11"), Closed: 1},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			name:      "issue closed after the range stays open throughout",
			lifetimes: []issues_model.IssueLifetime{closedIssue(t, "2026-01-06", "2026-03-02")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Opened: 1, Open: 1},
				{Week: weekMillis("2026-01-11"), Open: 1},
				{Week: weekMillis("2026-01-18"), Open: 1},
			},
		},
		{
			name:      "issue opened before the range counts as open without being opened in it",
			lifetimes: []issues_model.IssueLifetime{closedIssue(t, "2025-11-03", "2026-01-14")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Open: 1},
				{Week: weekMillis("2026-01-11"), Closed: 1},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			name:      "issue opened and closed before the range is invisible",
			lifetimes: []issues_model.IssueLifetime{closedIssue(t, "2025-11-03", "2025-11-10")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04")},
				{Week: weekMillis("2026-01-11")},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			name:      "issue created after the range is ignored",
			lifetimes: []issues_model.IssueLifetime{openIssue(t, "2026-03-02")},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04")},
				{Week: weekMillis("2026-01-11")},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			// a fixture or an import can leave a closed issue without a close time
			name:      "close time before creation is treated as closed at creation",
			lifetimes: []issues_model.IssueLifetime{{CreatedUnix: timeutil.TimeStamp(mustDate(t, "2026-01-06").Unix()), IsClosed: true}},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Opened: 1, Closed: 1},
				{Week: weekMillis("2026-01-11")},
				{Week: weekMillis("2026-01-18")},
			},
		},
		{
			name: "several issues accumulate",
			lifetimes: []issues_model.IssueLifetime{
				openIssue(t, "2026-01-05"),
				openIssue(t, "2026-01-06"),
				closedIssue(t, "2026-01-07", "2026-01-12"),
				closedIssue(t, "2026-01-13", "2026-01-19"),
			},
			expected: []IssueWeek{
				{Week: weekMillis("2026-01-04"), Opened: 3, Open: 3},
				{Week: weekMillis("2026-01-11"), Opened: 1, Closed: 1, Open: 3},
				{Week: weekMillis("2026-01-18"), Closed: 1, Open: 2},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, CalcIssueWeeks(c.lifetimes, start, end))
		})
	}
}

func TestCalcIssueWeeksEmptyRange(t *testing.T) {
	// a repository whose only issue is newer than the end of the range has nothing to chart
	weeks := CalcIssueWeeks(nil, mustDate(t, "2026-01-18"), mustDate(t, "2026-01-04"))
	assert.Empty(t, weeks)
}
