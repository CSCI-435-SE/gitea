// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/perm"
	access_model "gitea.dev/models/perm/access"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCalendarMonth(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Date(2026, 10, 7, 15, 30, 0, 0, loc)
	thisMonth := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)

	cases := map[string]time.Time{
		"2026-11":    time.Date(2026, 11, 1, 0, 0, 0, 0, loc),
		"2002-04":    time.Date(2002, 4, 1, 0, 0, 0, 0, loc),
		"1970-01":    time.Date(1970, 1, 1, 0, 0, 0, 0, loc),
		"9998-12":    time.Date(9998, 12, 1, 0, 0, 0, 0, loc),
		"":           thisMonth,
		"abc":        thisMonth,
		"2026-13":    thisMonth,
		"2026-1":     thisMonth,
		"2026-10-05": thisMonth,
		"1969-12":    thisMonth, // before the range
		"9999-01":    thisMonth, // legacy "no due date" milestones live here
	}
	for in, want := range cases {
		got := parseCalendarMonth(in, now)
		assert.True(t, want.Equal(got), "%q: want %s, got %s", in, want, got)
		assert.Equal(t, loc, got.Location(), in)
	}
}

func TestCalendarEntries(t *testing.T) {
	t.Run("issue", func(t *testing.T) {
		e := issueCalendarEntry("/o/r", &issues_model.Issue{Index: 7, Title: "Fix it", DeadlineUnix: 100})
		assert.Equal(t, calendarEntry{Kind: "issue", State: "open", Index: 7, Title: "Fix it", Link: "/o/r/issues/7", unix: 100}, e)
	})
	t.Run("closed issue", func(t *testing.T) {
		e := issueCalendarEntry("/o/r", &issues_model.Issue{Index: 7, IsClosed: true})
		assert.Equal(t, "closed", e.State)
	})
	t.Run("pull request", func(t *testing.T) {
		e := issueCalendarEntry("/o/r", &issues_model.Issue{Index: 8, IsPull: true, PullRequest: &issues_model.PullRequest{}})
		assert.Equal(t, "pull", e.Kind)
		assert.Equal(t, "open", e.State)
		assert.Equal(t, "/o/r/pulls/8", e.Link)
	})
	t.Run("merged pull request", func(t *testing.T) {
		e := issueCalendarEntry("/o/r", &issues_model.Issue{Index: 8, IsPull: true, IsClosed: true, PullRequest: &issues_model.PullRequest{HasMerged: true}})
		assert.Equal(t, "merged", e.State)
	})
	t.Run("closed pull request", func(t *testing.T) {
		e := issueCalendarEntry("/o/r", &issues_model.Issue{Index: 8, IsPull: true, IsClosed: true, PullRequest: &issues_model.PullRequest{}})
		assert.Equal(t, "closed", e.State)
	})
	t.Run("milestone", func(t *testing.T) {
		e := milestoneCalendarEntry("/o/r", &issues_model.Milestone{ID: 3, Name: "Sprint 3", DeadlineUnix: timeutil.TimeStamp(200)})
		assert.Equal(t, calendarEntry{Kind: "milestone", State: "open", Title: "Sprint 3", Link: "/o/r/milestone/3", unix: 200}, e)
	})
}

func TestBuildMonthGrid(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	at := func(y int, m time.Month, d, h, mi, s int) int64 { return time.Date(y, m, d, h, mi, s, 0, loc).Unix() }
	firstCell := func(weeks [][]calendarDay) string { return weeks[0][0].Date.Format(time.DateOnly) }
	lastCell := func(weeks [][]calendarDay) string { return weeks[len(weeks)-1][6].Date.Format(time.DateOnly) }
	dayOf := func(weeks [][]calendarDay, date string) calendarDay {
		for _, week := range weeks {
			for _, day := range week {
				if day.Date.Format(time.DateOnly) == date {
					return day
				}
			}
		}
		t.Fatalf("%s is not in the grid", date)
		return calendarDay{}
	}

	t.Run("October 2026 starts on a Thursday", func(t *testing.T) {
		weeks := buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), nil)
		assert.Len(t, weeks, 5)
		assert.Equal(t, "2026-09-28", firstCell(weeks))
		assert.Equal(t, "2026-11-01", lastCell(weeks))
		assert.False(t, weeks[0][0].InMonth)
		assert.True(t, weeks[0][3].InMonth)
		assert.True(t, dayOf(weeks, "2026-10-07").IsToday)
		assert.False(t, dayOf(weeks, "2026-10-08").IsToday)
		for _, week := range weeks {
			assert.Len(t, week, 7)
			assert.Equal(t, time.Monday, week[0].Date.Weekday())
		}
	})

	t.Run("February 2027 starts on a Monday and fills four rows", func(t *testing.T) {
		weeks := buildMonthGrid(time.Date(2027, 2, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), nil)
		assert.Len(t, weeks, 4)
		assert.Equal(t, "2027-02-01", firstCell(weeks))
		assert.Equal(t, "2027-02-28", lastCell(weeks))
	})

	t.Run("November 2026 starts on a Sunday, and DST ends on the 1st", func(t *testing.T) {
		weeks := buildMonthGrid(time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), nil)
		assert.Len(t, weeks, 6)
		assert.Equal(t, "2026-10-26", firstCell(weeks))
		assert.Equal(t, "2026-12-06", lastCell(weeks))
		assert.Equal(t, "2026-11-01", weeks[0][6].Date.Format(time.DateOnly))
		assert.Equal(t, "2026-11-02", weeks[1][0].Date.Format(time.DateOnly))
		assert.Equal(t, "2026-11-03", weeks[1][1].Date.Format(time.DateOnly))
		assert.Equal(t, 0, dayOf(weeks, "2026-11-02").Date.Hour(), "midnight, not shifted by the DST hour")
	})

	t.Run("entries land on their deadline's day, milestones first", func(t *testing.T) {
		entries := []calendarEntry{
			{Kind: "issue", Index: 9, unix: at(2026, 10, 14, 23, 59, 59)},
			{Kind: "pull", Index: 2, unix: at(2026, 10, 14, 23, 59, 59)},
			{Kind: "milestone", Title: "Sprint 3", unix: at(2026, 10, 14, 23, 59, 59)},
			{Kind: "issue", Index: 1, unix: at(2026, 10, 15, 0, 0, 0)},
		}
		weeks := buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), entries)
		got := dayOf(weeks, "2026-10-14").Entries
		require.Len(t, got, 3)
		assert.Equal(t, "milestone", got[0].Kind)
		assert.Equal(t, int64(2), got[1].Index)
		assert.Equal(t, int64(9), got[2].Index)
		assert.Len(t, dayOf(weeks, "2026-10-15").Entries, 1)
	})

	t.Run("a day splits at the cell limit", func(t *testing.T) {
		var entries []calendarEntry
		for i := range 5 {
			entries = append(entries, calendarEntry{Kind: "issue", Index: int64(i + 1), unix: at(2026, 10, 20, 23, 59, 59)})
		}
		day := dayOf(buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), entries), "2026-10-20")
		assert.Len(t, day.Visible(), calendarCellLimit)
		assert.Len(t, day.Overflow(), 5-calendarCellLimit)
		assert.Empty(t, calendarDay{}.Overflow())
	})

	t.Run("IsToday follows the grid's location", func(t *testing.T) {
		now := time.Date(2026, 10, 8, 2, 0, 0, 0, time.UTC) // still Oct 7 in New York
		weeks := buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), now, nil)
		assert.True(t, dayOf(weeks, "2026-10-07").IsToday)
		assert.False(t, dayOf(weeks, "2026-10-08").IsToday)
	})

	t.Run("an entry can land in a trailing out-of-month cell", func(t *testing.T) {
		entries := []calendarEntry{{Kind: "issue", Index: 1, unix: at(2026, 11, 1, 23, 59, 59)}}
		day := dayOf(buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), entries), "2026-11-01")
		assert.False(t, day.InMonth)
		assert.Len(t, day.Entries, 1)
	})

	t.Run("Visible and Overflow at the edges", func(t *testing.T) {
		mk := func(n int) calendarDay {
			var d calendarDay
			for i := range n {
				d.Entries = append(d.Entries, calendarEntry{Index: int64(i + 1)})
			}
			return d
		}
		assert.Len(t, mk(2).Visible(), 2)
		assert.Empty(t, mk(2).Overflow())
		assert.Len(t, mk(3).Visible(), 3)
		assert.Empty(t, mk(3).Overflow())

		day := mk(5)
		_ = append(day.Visible(), calendarEntry{})
		assert.Equal(t, int64(4), day.Overflow()[0].Index, "append to Visible must not overwrite Overflow")
	})

	t.Run("milestones with the same deadline are ordered by title", func(t *testing.T) {
		entries := []calendarEntry{
			{Kind: "milestone", Title: "B", unix: at(2026, 10, 14, 23, 59, 59)},
			{Kind: "milestone", Title: "A", unix: at(2026, 10, 14, 23, 59, 59)},
		}
		got := dayOf(buildMonthGrid(time.Date(2026, 10, 1, 0, 0, 0, 0, loc), time.Date(2026, 10, 7, 12, 0, 0, 0, loc), entries), "2026-10-14").Entries
		require.Len(t, got, 2)
		assert.Equal(t, "A", got[0].Title)
		assert.Equal(t, "B", got[1].Title)
	})
}

func TestCalendarIsPull(t *testing.T) {
	assert.Equal(t, optional.None[bool](), calendarIsPull(true, true))
	assert.Equal(t, optional.Some(false), calendarIsPull(true, false))
	assert.Equal(t, optional.Some(true), calendarIsPull(false, true))
	assert.Equal(t, optional.Some(false), calendarIsPull(false, false)) // the route guard prevents this, issues only is the safe default
}

func TestCalendarMilestones(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)
	in := timeutil.TimeStamp(time.Date(2026, 10, 14, 23, 59, 59, 0, time.UTC).Unix())
	milestones := []*issues_model.Milestone{
		{ID: 1, DeadlineUnix: in},
		{ID: 2, DeadlineUnix: in, IsClosed: true},
		{ID: 3, DeadlineUnix: 0},
		{ID: 4, DeadlineUnix: milestoneNoDeadline},
		{ID: 5, DeadlineUnix: timeutil.TimeStamp(end.Unix() + 1)},
		{ID: 6, DeadlineUnix: timeutil.TimeStamp(start.Unix())},
	}
	ids := func(ms []*issues_model.Milestone) (out []int64) {
		for _, m := range ms {
			out = append(out, m.ID)
		}
		return out
	}

	assert.Equal(t, []int64{1, 6}, ids(calendarMilestones(milestones, start, end, 0)))
	assert.Equal(t, []int64{6}, ids(calendarMilestones(milestones, start, end, 6)))
	assert.Empty(t, calendarMilestones(milestones, start, end, -1), "the no-milestone filter shows no milestones")
	assert.Empty(t, calendarMilestones(milestones, start, end, 2), "closed milestones never show")
}

// calendarLinks lists the entry links of the days inside the shown month
func calendarLinks(weeks [][]calendarDay) []string {
	var links []string
	for _, week := range weeks {
		for _, day := range week {
			if !day.InMonth {
				continue
			}
			for _, e := range day.Entries {
				links = append(links, e.Link)
			}
		}
	}
	return links
}

func TestCalendarHandler(t *testing.T) {
	unittest.PrepareTestEnv(t)

	renderAs := func(t *testing.T, path string, repoID int64, units ...unit.Type) [][]calendarDay {
		t.Helper()
		ctx, _ := contexttest.MockContext(t, path)
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, repoID)
		if len(units) > 0 { // readable units only
			var repoUnits []*repo_model.RepoUnit
			for _, u := range units {
				repoUnits = append(repoUnits, &repo_model.RepoUnit{Type: u})
			}
			ctx.Repo.Permission = access_model.Permission{AccessMode: perm.AccessModeRead}
			ctx.Repo.Permission.SetUnitsWithDefaultAccessMode(repoUnits, perm.AccessModeRead)
		}
		Calendar(ctx)
		require.Equal(t, http.StatusOK, ctx.Resp.WrittenStatus())
		return ctx.Data["Weeks"].([][]calendarDay)
	}

	render := func(t *testing.T, path string, repoID int64) [][]calendarDay {
		t.Helper()
		return renderAs(t, path, repoID)
	}

	// fixture issue 10 is user2/glob#1, open, due 2002-04-20
	t.Run("shows an issue in its deadline's month only", func(t *testing.T) {
		assert.Equal(t, []string{"/user2/glob/issues/1"}, calendarLinks(render(t, "user2/glob/calendar?month=2002-04", 42)))
		assert.Empty(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-05", 42)))
	})

	t.Run("state and assignee filters", func(t *testing.T) {
		assert.Empty(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-04&state=closed", 42)))
		assert.Len(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-04&state=all", 42)), 1)
		assert.Empty(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-04&assignee=(any)", 42)))
	})

	t.Run("assignee filter matches an assigned user", func(t *testing.T) {
		unittest.PrepareTestEnv(t) // these write fixtures, so start clean
		require.NoError(t, db.Insert(t.Context(), &issues_model.IssueAssignees{IssueID: 10, AssigneeID: 2}))
		assert.Len(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-04&assignee=2", 42)), 1)
		assert.Empty(t, calendarLinks(render(t, "user2/glob/calendar?month=2002-04&assignee=(none)", 42)))
	})

	t.Run("milestone on its due day", func(t *testing.T) {
		unittest.PrepareTestEnv(t) // these write fixtures, so start clean
		defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()
		due := time.Date(2026, 10, 14, 23, 59, 59, 0, setting.DefaultUILocation).Unix()
		_, err := db.GetEngine(t.Context()).ID(1).Cols("deadline_unix", "is_closed").Update(&issues_model.Milestone{DeadlineUnix: timeutil.TimeStamp(due)})
		require.NoError(t, err)

		assert.Contains(t, calendarLinks(render(t, "user2/repo1/calendar", 1)), "/user2/repo1/milestone/1")
		assert.Contains(t, calendarLinks(render(t, "user2/repo1/calendar?milestone=1", 1)), "/user2/repo1/milestone/1")
		assert.NotContains(t, calendarLinks(render(t, "user2/repo1/calendar?milestone=-1", 1)), "/user2/repo1/milestone/1")
	})

	t.Run("unit permissions limit what shows", func(t *testing.T) {
		unittest.PrepareTestEnv(t) // these write fixtures, so start clean
		defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()
		due := time.Date(2026, 10, 20, 23, 59, 59, 0, setting.DefaultUILocation).Unix()
		for _, id := range []int64{1, 2} { // issue 1 and pull 2 of repo1
			_, err := db.GetEngine(t.Context()).ID(id).Cols("deadline_unix").Update(&issues_model.Issue{DeadlineUnix: timeutil.TimeStamp(due)})
			require.NoError(t, err)
		}
		links := func(units ...unit.Type) []string {
			return calendarLinks(renderAs(t, "user2/repo1/calendar", 1, units...))
		}
		assert.Equal(t, []string{"/user2/repo1/pulls/2"}, links(unit.TypePullRequests))
		assert.Equal(t, []string{"/user2/repo1/issues/1"}, links(unit.TypeIssues))
		assert.ElementsMatch(t, []string{"/user2/repo1/issues/1", "/user2/repo1/pulls/2"}, links(unit.TypeIssues, unit.TypePullRequests))
	})

	t.Run("milestone filter applies to issues and pulls", func(t *testing.T) {
		unittest.PrepareTestEnv(t) // these write fixtures, so start clean
		defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()
		due := time.Date(2026, 10, 20, 23, 59, 59, 0, setting.DefaultUILocation).Unix()
		for _, id := range []int64{1, 2} { // issue 1 and pull 2 of repo1
			_, err := db.GetEngine(t.Context()).ID(id).Cols("deadline_unix").Update(&issues_model.Issue{DeadlineUnix: timeutil.TimeStamp(due)})
			require.NoError(t, err)
		}
		_, err := db.GetEngine(t.Context()).ID(1).Cols("milestone_id").Update(&issues_model.Issue{MilestoneID: 1})
		require.NoError(t, err)
		_, err = db.GetEngine(t.Context()).ID(2).Cols("milestone_id").Update(&issues_model.Issue{MilestoneID: 0}) // the fixture has milestone 1
		require.NoError(t, err)
		links := func(path string) (out []string) {
			for _, l := range calendarLinks(render(t, path, 1)) {
				if strings.Contains(l, "/issues/") || strings.Contains(l, "/pulls/") { // skip milestone entries
					out = append(out, l)
				}
			}
			return out
		}
		assert.Equal(t, []string{"/user2/repo1/issues/1"}, links("user2/repo1/calendar?milestone=1"))
		assert.Equal(t, []string{"/user2/repo1/pulls/2"}, links("user2/repo1/calendar?milestone=-1"))
	})

	t.Run("pull requests appear with their own kind", func(t *testing.T) {
		unittest.PrepareTestEnv(t) // these write fixtures, so start clean
		defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()
		due := time.Date(2026, 10, 30, 23, 59, 59, 0, setting.DefaultUILocation).Unix()
		// issue 2 is user2/repo1's pull #2
		_, err := db.GetEngine(t.Context()).ID(2).Cols("deadline_unix").Update(&issues_model.Issue{DeadlineUnix: timeutil.TimeStamp(due)})
		require.NoError(t, err)

		weeks := render(t, "user2/repo1/calendar", 1) // no ?month: the mocked current month
		assert.Equal(t, []string{"/user2/repo1/pulls/2"}, calendarLinks(weeks))
		for _, week := range weeks {
			for _, day := range week {
				for _, e := range day.Entries {
					assert.Equal(t, "pull", e.Kind)
				}
			}
		}
	})
}

func TestCalendarMonthLink(t *testing.T) {
	at := func(y int, m time.Month) time.Time { return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) }
	assert.Empty(t, calendarMonthLink(at(1969, 12)), "before the first month parseCalendarMonth accepts")
	assert.Equal(t, "1970-01", calendarMonthLink(at(1970, 1)))
	assert.Equal(t, "9998-12", calendarMonthLink(at(9998, 12)))
	assert.Empty(t, calendarMonthLink(at(9999, 1)), "after the last month parseCalendarMonth accepts")
}
