// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"strings"
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/timeutil"

	ics "github.com/arran4/golang-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testICalLocation = time.FixedZone("UTC-5", -5*60*60)

func testICalOptions() ICalOptions {
	return ICalOptions{
		Name:         "octo/app issues",
		RepoFullName: "octo/app",
		RepoURL:      "https://gitea.example/octo/app",
		Domain:       "gitea.example",
		Location:     testICalLocation,
		Now:          time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
	}
}

// endOfDay mirrors how deadlines are stored: the last second of the day in the UI zone
func endOfDay(month time.Month, day int) timeutil.TimeStamp {
	return timeutil.TimeStamp(time.Date(2026, month, day, 23, 59, 59, 0, testICalLocation).Unix())
}

func roundTrip(t *testing.T, cal *ics.Calendar) (string, *ics.Calendar) {
	out := cal.Serialize(ics.WithNewLineWindows)
	parsed, err := ics.ParseCalendar(strings.NewReader(out))
	require.NoError(t, err)
	return out, parsed
}

func eventProp(e *ics.VEvent, p ics.ComponentProperty) string {
	if prop := e.GetProperty(p); prop != nil {
		return prop.Value
	}
	return ""
}

func TestIssuesICalUIDSurvivesDeadlineChange(t *testing.T) {
	issue := &issues_model.Issue{ID: 7, Index: 42, Title: "Fix login", DeadlineUnix: endOfDay(10, 14)}
	_, before := roundTrip(t, IssuesICal(testICalOptions(), issues_model.IssueList{issue}))
	issue.DeadlineUnix = endOfDay(10, 21)
	_, after := roundTrip(t, IssuesICal(testICalOptions(), issues_model.IssueList{issue}))

	require.Len(t, before.Events(), 1)
	require.Len(t, after.Events(), 1)
	assert.Equal(t, "issue-7@gitea.example", before.Events()[0].Id())
	assert.Equal(t, before.Events()[0].Id(), after.Events()[0].Id())
	assert.Equal(t, "20261014", eventProp(before.Events()[0], ics.ComponentPropertyDtStart))
	assert.Equal(t, "20261021", eventProp(after.Events()[0], ics.ComponentPropertyDtStart))
}

func TestIssuesICalAllDayDatesUseUILocation(t *testing.T) {
	// 23:59:59 on the 14th at UTC-5 is already the 15th in UTC, so a zone bug moves the event a day
	issue := &issues_model.Issue{ID: 1, Index: 1, Title: "t", DeadlineUnix: endOfDay(10, 14)}
	out, _ := roundTrip(t, IssuesICal(testICalOptions(), issues_model.IssueList{issue}))
	assert.Contains(t, out, "DTSTART;VALUE=DATE:20261014\r\n")
	assert.Contains(t, out, "DTEND;VALUE=DATE:20261015\r\n")
	assert.Contains(t, out, "DTSTAMP:20261007T120000Z\r\n")
}

func TestIssuesICalSummaryAndURL(t *testing.T) {
	issues := issues_model.IssueList{
		{ID: 1, Index: 42, Title: "Fix login", DeadlineUnix: endOfDay(10, 14)},
		{ID: 2, Index: 43, Title: "Add SSO", IsPull: true, DeadlineUnix: endOfDay(10, 15)},
		{ID: 3, Index: 44, Title: "No deadline"},
	}
	_, cal := roundTrip(t, IssuesICal(testICalOptions(), issues))
	events := cal.Events()
	require.Len(t, events, 2) // undated rows never become events

	assert.Equal(t, "Fix login (octo/app#42)", eventProp(events[0], ics.ComponentPropertySummary))
	assert.Equal(t, "https://gitea.example/octo/app/issues/42", eventProp(events[0], ics.ComponentPropertyUrl))
	assert.Equal(t, "https://gitea.example/octo/app/issues/42", eventProp(events[0], ics.ComponentPropertyDescription))
	assert.Equal(t, "issue-2@gitea.example", events[1].Id()) // pulls are issue rows, so they share the scheme
	assert.Equal(t, "https://gitea.example/octo/app/pulls/43", eventProp(events[1], ics.ComponentPropertyUrl))
}

func TestMilestonesICal(t *testing.T) {
	milestones := []*issues_model.Milestone{
		{ID: 1, Name: "Sprint 3", DeadlineUnix: endOfDay(10, 14)},
		{ID: 2, Name: "No date"},
		{ID: 3, Name: "Legacy no date", DeadlineUnix: 253370764800},
	}
	out, cal := roundTrip(t, MilestonesICal(testICalOptions(), milestones))
	events := cal.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "milestone-1@gitea.example", events[0].Id())
	assert.Equal(t, "Sprint 3 (octo/app)", eventProp(events[0], ics.ComponentPropertySummary))
	assert.Equal(t, "https://gitea.example/octo/app/milestone/1", eventProp(events[0], ics.ComponentPropertyUrl))
	assert.Contains(t, out, "DTSTART;VALUE=DATE:20261014\r\n")
}

func TestICalCalendarProperties(t *testing.T) {
	out, _ := roundTrip(t, IssuesICal(testICalOptions(), nil))
	assert.Contains(t, out, "BEGIN:VCALENDAR\r\n")
	assert.Contains(t, out, "VERSION:2.0\r\n")
	assert.Contains(t, out, "METHOD:PUBLISH\r\n")
	assert.Contains(t, out, "CALSCALE:GREGORIAN\r\n")
	assert.Contains(t, out, "X-WR-CALNAME:octo/app issues\r\n")
}

func TestICalEscapingAndFolding(t *testing.T) {
	title := "a, b; c\\d\nnext " + strings.Repeat("x", 150)
	issue := &issues_model.Issue{ID: 1, Index: 1, Title: title, DeadlineUnix: endOfDay(10, 14)}
	out, cal := roundTrip(t, IssuesICal(testICalOptions(), issues_model.IssueList{issue}))

	assert.Equal(t, strings.Count(out, "\n"), strings.Count(out, "\r\n"), "every line must end in CRLF")
	for line := range strings.SplitSeq(out, "\r\n") {
		assert.LessOrEqual(t, len(line), 75, "RFC 5545 folds lines at 75 octets: %q", line)
	}
	assert.Equal(t, title+" (octo/app#1)", eventProp(cal.Events()[0], ics.ComponentPropertySummary))
}
