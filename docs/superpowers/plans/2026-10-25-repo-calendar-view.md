# Repository Calendar View Implementation Plan (#98)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A server-rendered month calendar at `/{owner}/{repo}/calendar` showing open milestone due dates and issue and pull request deadlines, with milestone/assignee/state filters and month navigation kept in the URL.

**Architecture:** A new handler `repo.Calendar` in `routers/web/repo/calendar.go` runs one `issues_model.Issues` query bounded by the month's deadline range plus one milestone query. It converts rows into `calendarEntry` values and hands them to a pure `buildMonthGrid` function that lays out Monday-first weeks. A Go template renders the grid; CSS grid lays it out; native `<details>` handles "+N more". No JS, no new endpoint.

**Tech Stack:** Go (chi routes, xorm models), Go `html/template`, Fomantic UI + `tw-` Tailwind classes, plain CSS, testify, goquery for integration tests.

**Spec:** `docs/superpowers/specs/2026-10-25-repo-calendar-view-design.md`

---

## Ground rules for this repo (read before starting)

- `make` is not installed on this machine. Use the direct commands given in each step.
- **Never run `git commit` yourself.** The human commits each change after reviewing it
  (`AGENTS.md`). Each "Commit" step means: stop, show the human the command, wait for them to run it.
  Commit messages are Conventional Commits ending with `(#98)` and the trailer
  `Assisted-by: Claude Code:claude-opus-5-5`. Never add `Co-Authored-By` or `Signed-off-by`.
- New `.go` files start with `// Copyright 2026 The Gitea Authors. All rights reserved.` and
  `// SPDX-License-Identifier: MIT`.
- Keep comments short, same-line where possible, and explain *why*.
- No trailing whitespace in edited files.
- Some Go tests fail on native Windows for platform reasons. If a test you didn't touch fails,
  run the suite on WSL2 or the devcontainer before debugging it.

## File map

| File | Status | Responsibility |
| --- | --- | --- |
| `routers/web/repo/calendar.go` | create | month parsing, entry conversion, grid building, milestone filtering, the `Calendar` handler |
| `routers/web/repo/calendar_test.go` | create | pure unit tests for the above, plus DB-backed handler tests |
| `routers/web/web.go` | modify (~line 1317) | register `GET /{username}/{reponame}/calendar` |
| `options/locale/locale_en-US.json` | modify (near line 1200) | `repo.calendar.*` keys, month and weekday names |
| `templates/repo/calendar.tmpl` | create | page: header, navigation, filters, grid |
| `templates/repo/calendar_entry.tmpl` | create | one entry link (icon, `#index`, title) |
| `templates/repo/issue/list.tmpl` | modify (line 22) | `Calendar` button next to Milestones |
| `web_src/css/features/calendar.css` | create | grid layout, entry accents, mobile agenda layout |
| `web_src/css/index.css` | modify (line 52) | import the new stylesheet |
| `tests/integration/repo_calendar_test.go` | create | end-to-end HTTP check: entries, the list button, 404 for private repos |

---

### Task 1: Parse the `month` query parameter

**Files:**
- Create: `routers/web/repo/calendar.go`
- Create: `routers/web/repo/calendar_test.go`

- [ ] **Step 1: Write the failing test**

Create `routers/web/repo/calendar_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCalendarMonth(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Date(2026, 10, 7, 15, 30, 0, 0, loc)
	thisMonth := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)

	cases := map[string]time.Time{
		"2026-11": time.Date(2026, 11, 1, 0, 0, 0, 0, loc),
		"2002-04": time.Date(2002, 4, 1, 0, 0, 0, 0, loc),
		"1970-01": time.Date(1970, 1, 1, 0, 0, 0, 0, loc),
		"9998-12": time.Date(9998, 12, 1, 0, 0, 0, 0, loc),
		"":        thisMonth,
		"abc":     thisMonth,
		"2026-13": thisMonth,
		"2026-1":  thisMonth,
		"2026-10-05": thisMonth,
		"1969-12": thisMonth, // before the range
		"9999-01": thisMonth, // legacy "no due date" milestones live here
	}
	for in, want := range cases {
		got := parseCalendarMonth(in, now)
		assert.True(t, want.Equal(got), "%q: want %s, got %s", in, want, got)
		assert.Equal(t, loc, got.Location(), in)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run '^TestParseCalendarMonth$' ./routers/web/repo/`
Expected: FAIL to compile with `undefined: parseCalendarMonth`.

- [ ] **Step 3: Write the minimal implementation**

Create `routers/web/repo/calendar.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"time"
)

// parseCalendarMonth resolves the "month" query (YYYY-MM) to the first instant of that month in now's
// location. An invalid or out-of-range value falls back to now's month, so a stale link still renders.
func parseCalendarMonth(s string, now time.Time) time.Time {
	if t, err := time.ParseInLocation("2006-01", s, now.Location()); err == nil && t.Year() >= 1970 && t.Year() <= 9998 {
		return t
	}
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -run '^TestParseCalendarMonth$' ./routers/web/repo/`
Expected: `ok  gitea.dev/routers/web/repo`

(Do not commit yet: Tasks 1–3 form one commit once the pure logic is complete.)

---

### Task 2: Calendar entries and the month grid

**Files:**
- Modify: `routers/web/repo/calendar.go`
- Modify: `routers/web/repo/calendar_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `routers/web/repo/calendar_test.go`. Also add `issues_model "gitea.dev/models/issues"` and `"gitea.dev/modules/timeutil"` to its imports.

```go
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
		for i, day := range []string{"2026-11-01", "2026-11-02", "2026-11-03"} {
			assert.Equal(t, day, weeks[0][6].Date.AddDate(0, 0, i).Format(time.DateOnly))
		}
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
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run '^(TestCalendarEntries|TestBuildMonthGrid)$' ./routers/web/repo/`
Expected: FAIL to compile with `undefined: issueCalendarEntry`, `undefined: buildMonthGrid`, and similar.

- [ ] **Step 3: Write the implementation**

In `routers/web/repo/calendar.go`, replace the import block and add the code below `parseCalendarMonth`:

```go
import (
	"cmp"
	"fmt"
	"slices"
	"time"

	issues_model "gitea.dev/models/issues"
)

const calendarCellLimit = 3 // entries a day cell shows before "+N more"

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
	return d.Entries[:min(len(d.Entries), calendarCellLimit)]
}

func (d calendarDay) Overflow() []calendarEntry {
	return d.Entries[min(len(d.Entries), calendarCellLimit):]
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
	return cmp.Or(cmp.Compare(a.unix, b.unix), cmp.Compare(a.Index, b.Index))
}

// buildMonthGrid lays out the whole weeks (Monday first, as ISO 8601 and the "week" due filter) that
// cover monthStart's month, placing each entry on the day its deadline falls on in monthStart's location
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

	var weeks [][]calendarDay
	// day-of-month of the grid's first Monday, <= 0 when it falls in the previous month
	for d := 1 - (int(monthStart.Weekday())+6)%7; d <= lastDay; d += 7 {
		week := make([]calendarDay, 7)
		for i := range week {
			date := time.Date(y, m, d+i, 0, 0, 0, 0, loc) // normalised by time.Date, never by adding 24h, so DST can't shift it
			dy, dm, dd := date.Date()
			dayEntries := byDate[date.Format(time.DateOnly)]
			slices.SortStableFunc(dayEntries, compareCalendarEntries)
			week[i] = calendarDay{Date: date, InMonth: dm == m, IsToday: dy == ny && dm == nm && dd == nd, Entries: dayEntries}
		}
		weeks = append(weeks, week)
	}
	return weeks
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -run '^(TestParseCalendarMonth|TestCalendarEntries|TestBuildMonthGrid)$' ./routers/web/repo/`
Expected: `ok  gitea.dev/routers/web/repo`

---

### Task 3: Unit permissions and milestone selection

**Files:**
- Modify: `routers/web/repo/calendar.go`
- Modify: `routers/web/repo/calendar_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `routers/web/repo/calendar_test.go`. Add `"gitea.dev/modules/optional"` to its imports.

```go
func TestCalendarIsPull(t *testing.T) {
	assert.Equal(t, optional.None[bool](), calendarIsPull(true, true))
	assert.Equal(t, optional.Some(false), calendarIsPull(true, false))
	assert.Equal(t, optional.Some(true), calendarIsPull(false, true))
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run '^(TestCalendarIsPull|TestCalendarMilestones)$' ./routers/web/repo/`
Expected: FAIL to compile with `undefined: calendarIsPull`, `undefined: calendarMilestones`, `undefined: milestoneNoDeadline`.

- [ ] **Step 3: Write the implementation**

In `routers/web/repo/calendar.go`, add `"gitea.dev/modules/optional"` and `"gitea.dev/modules/timeutil"` to the imports. Change the single constant into a block:

```go
const (
	calendarCellLimit   = 3                                  // entries a day cell shows before "+N more"
	milestoneNoDeadline = timeutil.TimeStamp(253370764800) // 9999-01-01 UTC, how older versions stored "no due date"
)
```

Add the two functions:

```go
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
```

- [ ] **Step 4: Run all calendar unit tests**

Run: `go test -run '^(TestParseCalendarMonth|TestCalendarEntries|TestBuildMonthGrid|TestCalendarIsPull|TestCalendarMilestones)$' ./routers/web/repo/`
Expected: `ok  gitea.dev/routers/web/repo`

- [ ] **Step 5: Format, then hand the commit to the human**

Run: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 fmt`
Expected: no output; `git diff --stat` shows only the two calendar files changed.

Ask the human to review and run:

```bash
git add routers/web/repo/calendar.go routers/web/repo/calendar_test.go && git commit -m "feat(calendar): add month grid builder for the repository calendar (#98)" -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 4: The `Calendar` handler and route

**Files:**
- Modify: `routers/web/repo/calendar.go`
- Modify: `routers/web/web.go:1314-1322`
- Modify: `routers/web/repo/calendar_test.go`

- [ ] **Step 1: Write the failing handler test**

Append to `routers/web/repo/calendar_test.go`. Add these imports: `"net/http"`, `"gitea.dev/models/db"`, `"gitea.dev/models/unittest"`, `"gitea.dev/modules/setting"` and `"gitea.dev/services/contexttest"`.

```go
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

	render := func(t *testing.T, path string, repoID int64) [][]calendarDay {
		t.Helper()
		ctx, _ := contexttest.MockContext(t, path)
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, repoID)
		Calendar(ctx)
		require.Equal(t, http.StatusOK, ctx.Resp.WrittenStatus())
		return ctx.Data["Weeks"].([][]calendarDay)
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

	t.Run("pull requests appear with their own kind", func(t *testing.T) {
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run '^TestCalendarHandler$' ./routers/web/repo/`
Expected: FAIL to compile with `undefined: Calendar`.

- [ ] **Step 3: Write the handler**

In `routers/web/repo/calendar.go`, set the import block to:

```go
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
```

Extend the constant block:

```go
const (
	tplCalendar templates.TplName = "repo/calendar"

	calendarCellLimit   = 3                                  // entries a day cell shows before "+N more"
	calendarMaxEntries  = 1000                               // bounds the query on very busy months
	milestoneNoDeadline = timeutil.TimeStamp(253370764800) // 9999-01-01 UTC, how older versions stored "no due date"
)
```

Append the handler:

```go
// Calendar renders a month grid of the repository's milestone due dates and issue and pull request deadlines
func Calendar(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.calendar")
	ctx.Data["PageIsIssueList"] = true

	now := timeutil.TimeStampNow().AsTime() // in setting.DefaultUILocation, where deadlines are pinned to 23:59:59
	monthStart := parseCalendarMonth(ctx.FormString("month"), now)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	milestoneID := ctx.FormInt64("milestone")
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
		Paginator:          &db.ListOptions{Page: 1, PageSize: calendarMaxEntries},
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
	issues, err := issues_model.Issues(ctx, opts)
	if err != nil {
		ctx.ServerError("Issues", err)
		return
	}
	if err := issues.LoadPullRequests(ctx); err != nil {
		ctx.ServerError("LoadPullRequests", err)
		return
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

	ctx.Data["Weeks"] = buildMonthGrid(monthStart, now, entries)
	ctx.Data["MonthNum"] = int(monthStart.Month())
	ctx.Data["Year"] = monthStart.Year()
	ctx.Data["Month"] = monthStart.Format("2006-01")
	ctx.Data["PrevMonth"] = monthStart.AddDate(0, -1, 0).Format("2006-01")
	ctx.Data["NextMonth"] = monthStart.AddDate(0, 1, 0).Format("2006-01")
	ctx.Data["MilestoneID"] = milestoneID
	ctx.Data["AssigneeID"] = assigneeID
	ctx.Data["State"] = state
	ctx.Data["CalendarTruncated"] = len(issues) >= calendarMaxEntries
	ctx.Data["CalendarMaxEntries"] = calendarMaxEntries
	ctx.HTML(http.StatusOK, tplCalendar)
}
```

- [ ] **Step 4: Register the route**

In `routers/web/web.go`, inside the group at lines 1314-1322 (the one ending
`}, optSignIn, context.RepoAssignment, reqRepoIssuesOrPullsReader)`), add the line after
`m.Get("/milestone/{id}/burndown", repo.MilestoneBurndownData)`:

```go
		m.Get("/calendar", repo.Calendar)
```

- [ ] **Step 5: Run the handler and unit tests**

Run: `go test -run '^(TestParseCalendarMonth|TestCalendarEntries|TestBuildMonthGrid|TestCalendarIsPull|TestCalendarMilestones|TestCalendarHandler)$' ./routers/web/repo/`
Expected: `ok  gitea.dev/routers/web/repo`

Fixture issue 10 has no rows in `issue_assignees.yml`, so `assignee=(any)` must filter it out (`applyAssigneeCondition` in `models/issues/issue_search.go`).

- [ ] **Step 6: Build check**

Run: `go build ./...`
Expected: no output.

(Commit together with Tasks 5–7, once the page renders.)

---

### Task 5: Locale keys

**Files:**
- Modify: `options/locale/locale_en-US.json` (after `"repo.milestones": "Milestones",` near line 1200)

- [ ] **Step 1: Add the keys**

Insert after the `"repo.milestones"` line. Keep the file's two-space indentation and its trailing commas:

```json
  "repo.calendar": "Calendar",
  "repo.calendar.today": "Today",
  "repo.calendar.prev_month": "Previous month",
  "repo.calendar.next_month": "Next month",
  "repo.calendar.more": "+%d more",
  "repo.calendar.truncated": "This month has more entries than can be shown. Showing the first %d; use the filters to narrow it down.",
  "repo.calendar.state": "State",
  "repo.calendar.state_all": "All",
  "repo.calendar.month_1": "January",
  "repo.calendar.month_2": "February",
  "repo.calendar.month_3": "March",
  "repo.calendar.month_4": "April",
  "repo.calendar.month_5": "May",
  "repo.calendar.month_6": "June",
  "repo.calendar.month_7": "July",
  "repo.calendar.month_8": "August",
  "repo.calendar.month_9": "September",
  "repo.calendar.month_10": "October",
  "repo.calendar.month_11": "November",
  "repo.calendar.month_12": "December",
  "repo.calendar.weekday_mon": "Mon",
  "repo.calendar.weekday_tue": "Tue",
  "repo.calendar.weekday_wed": "Wed",
  "repo.calendar.weekday_thu": "Thu",
  "repo.calendar.weekday_fri": "Fri",
  "repo.calendar.weekday_sat": "Sat",
  "repo.calendar.weekday_sun": "Sun",
```

The locale has no month or weekday names yet (`grep -n '"January"' options/locale/locale_en-US.json` finds nothing), so they're added here. Only `locale_en-US.json` is edited; translators fill the rest (`docs/i18n.md`).

- [ ] **Step 2: Validate the JSON**

Run: `node -e "JSON.parse(require('fs').readFileSync('options/locale/locale_en-US.json','utf8'))"`
Expected: no output.

---

### Task 6: Templates

**Files:**
- Create: `templates/repo/calendar_entry.tmpl`
- Create: `templates/repo/calendar.tmpl`
- Modify: `templates/repo/issue/list.tmpl:22`

- [ ] **Step 1: The entry partial**

Create `templates/repo/calendar_entry.tmpl`:

```html
{{/* One calendar entry; the dot is a calendarEntry */}}
<a class="repo-calendar-entry flex-text-inline is-{{.Kind}} is-{{.State}}" href="{{.Link}}" title="{{.Title}}">
	{{if eq .Kind "milestone"}}
		{{svg "octicon-milestone" 14}}
	{{else if eq .Kind "pull"}}
		{{if eq .State "merged"}}{{svg "octicon-git-merge" 14}}{{else if eq .State "closed"}}{{svg "octicon-git-pull-request-closed" 14}}{{else}}{{svg "octicon-git-pull-request" 14}}{{end}}
	{{else}}
		{{if eq .State "closed"}}{{svg "octicon-issue-closed" 14}}{{else}}{{svg "octicon-issue-opened" 14}}{{end}}
	{{end}}
	{{if .Index}}<span class="repo-calendar-index">#{{.Index}}</span>{{end}}
	<span class="tw-truncate">{{.Title}}</span>
</a>
```

- [ ] **Step 2: The page**

Create `templates/repo/calendar.tmpl`:

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content repository repo-calendar">
	{{template "repo/header" .}}
	<div class="ui container">
		{{template "base/alert" .}}
		{{$queryLink := QueryBuild "?" "month" $.Month "milestone" $.MilestoneID "assignee" $.AssigneeID "state" $.State}}
		<div class="list-header flex-text-block tw-flex-wrap">
			<div class="flex-text-block">
				<a class="ui small icon button" href="{{QueryBuild $queryLink "month" $.PrevMonth}}" aria-label="{{ctx.Locale.Tr "repo.calendar.prev_month"}}" data-tooltip-content="{{ctx.Locale.Tr "repo.calendar.prev_month"}}">{{svg "octicon-chevron-left"}}</a>
				<h2 class="repo-calendar-title tw-m-0">{{ctx.Locale.Tr (printf "repo.calendar.month_%d" $.MonthNum)}} {{$.Year}}</h2>
				<a class="ui small icon button" href="{{QueryBuild $queryLink "month" $.NextMonth}}" aria-label="{{ctx.Locale.Tr "repo.calendar.next_month"}}" data-tooltip-content="{{ctx.Locale.Tr "repo.calendar.next_month"}}">{{svg "octicon-chevron-right"}}</a>
				<a class="ui small button" href="{{QueryBuild $queryLink "month" NIL}}">{{ctx.Locale.Tr "repo.calendar.today"}}</a>
			</div>
			<div class="tw-flex-1"></div>
			<div class="ui secondary filter menu labels">
				{{template "repo/issue/filter_item_milestone" dict
					"QueryLink" $queryLink
					"MilestoneID" $.MilestoneID
					"OpenMilestones" .OpenMilestones
					"ClosedMilestones" .ClosedMilestones
				}}
				{{template "repo/issue/filter_item_user_assign" dict
					"QueryParamKey" "assignee"
					"QueryLink" $queryLink
					"UserSearchList" $.Assignees
					"SelectedUserId" $.AssigneeID
					"TextFilterTitle" (ctx.Locale.Tr "repo.issues.filter_assignee")
					"TextFilterMatchNone" (ctx.Locale.Tr "repo.issues.filter_assignee_no_assignee")
					"TextFilterMatchAny" (ctx.Locale.Tr "repo.issues.filter_assignee_any_assignee")
				}}
			</div>
			<div class="small-menu-items ui compact tiny menu">
				<a class="{{if eq $.State "open"}}active {{end}}item" href="{{QueryBuild $queryLink "state" "open"}}">{{ctx.Locale.Tr "repo.issues.open_title"}}</a>
				<a class="{{if eq $.State "closed"}}active {{end}}item" href="{{QueryBuild $queryLink "state" "closed"}}">{{ctx.Locale.Tr "repo.issues.closed_title"}}</a>
				<a class="{{if eq $.State "all"}}active {{end}}item" href="{{QueryBuild $queryLink "state" "all"}}">{{ctx.Locale.Tr "repo.calendar.state_all"}}</a>
			</div>
		</div>

		{{if .CalendarTruncated}}
			<div class="ui warning message">{{ctx.Locale.Tr "repo.calendar.truncated" .CalendarMaxEntries}}</div>
		{{end}}

		<div class="repo-calendar-grid">
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_mon"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_tue"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_wed"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_thu"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_fri"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_sat"}}</div>
			<div class="repo-calendar-weekday">{{ctx.Locale.Tr "repo.calendar.weekday_sun"}}</div>
			{{range $week := .Weeks}}
				{{range $day := $week}}
					<div class="repo-calendar-day {{if not $day.InMonth}}is-other-month{{end}} {{if $day.IsToday}}is-today{{end}} {{if not $day.Entries}}is-empty{{end}}">
						<div class="repo-calendar-date">{{$day.Date.Day}}</div>
						{{range $day.Visible}}{{template "repo/calendar_entry" .}}{{end}}
						{{if $day.Overflow}}
							<details class="repo-calendar-more">
								<summary>{{ctx.Locale.Tr "repo.calendar.more" (len $day.Overflow)}}</summary>
								{{range $day.Overflow}}{{template "repo/calendar_entry" .}}{{end}}
							</details>
						{{end}}
					</div>
				{{end}}
			{{end}}
		</div>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 3: The list header button**

In `templates/repo/issue/list.tmpl`, after line 22
(`<a class="ui small button" href="{{.RepoLink}}/milestones">...`), add:

```html
			<a class="ui small button" href="{{.RepoLink}}/calendar">{{ctx.Locale.Tr "repo.calendar"}}</a>
```

- [ ] **Step 4: Lint the SVG usage in templates**

Run: `node tools/lint-templates-svg.ts`
Expected: exits 0 with no errors. (The `djlint` half of `make lint-templates` needs `uv` and can't run here, so CI covers it.)

---

### Task 7: Styles

**Files:**
- Create: `web_src/css/features/calendar.css`
- Modify: `web_src/css/index.css:52`

- [ ] **Step 1: Write the stylesheet**

Create `web_src/css/features/calendar.css`:

```css
.repo-calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  border-top: 1px solid var(--color-secondary);
  border-left: 1px solid var(--color-secondary);
  margin-top: 1rem;
}

.repo-calendar-weekday,
.repo-calendar-day {
  border-right: 1px solid var(--color-secondary);
  border-bottom: 1px solid var(--color-secondary);
  padding: 4px 6px;
}

.repo-calendar-weekday {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-light-2);
  text-align: center;
}

.repo-calendar-day {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 7rem;
  min-width: 0; /* lets entry titles truncate inside the grid track */
}

.repo-calendar-day.is-other-month {
  background: var(--color-box-header);
  color: var(--color-text-light-3);
}

.repo-calendar-day.is-today {
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}

.repo-calendar-date {
  font-size: 12px;
}

.repo-calendar-entry {
  border-left: 3px solid var(--color-green);
  padding-left: 4px;
  font-size: 12px;
  color: var(--color-text);
  min-width: 0;
}

.repo-calendar-entry.is-milestone {
  border-left-color: var(--color-text);
  font-weight: var(--font-weight-semibold);
}

.repo-calendar-entry.is-closed {
  border-left-color: var(--color-red);
}

.repo-calendar-entry.is-merged {
  border-left-color: var(--color-purple);
}

.repo-calendar-index {
  color: var(--color-text-light-2);
}

.repo-calendar-more summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text-light-2);
}

/* narrow screens: an agenda list of the days that have something due */
@media (max-width: 767.98px) {
  .repo-calendar-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .repo-calendar-weekday,
  .repo-calendar-day.is-other-month,
  .repo-calendar-day.is-empty {
    display: none;
  }
  .repo-calendar-day {
    min-height: 0;
  }
}
```

All variables used here exist in `web_src/css/themes/theme-gitea-light.css` or `web_src/css/base.css` (checked while planning). `--color-box-header` is used for out-of-month days because `--color-box-body` is the same white as the page.

- [ ] **Step 2: Import it**

In `web_src/css/index.css`, after `@import "./features/char-counter.css";` (line 52), add:

```css
@import "./features/calendar.css";
```

- [ ] **Step 3: Lint the CSS**

Run: `pnpm exec stylelint --color --max-warnings=0 web_src/css web_src/js/components/*.vue`
Expected: no output, exit 0.

- [ ] **Step 4: Build the frontend so the page picks up the CSS**

Run: `pnpm exec vite build`
Expected: completes without errors.

---

### Task 8: Integration test

**Files:**
- Create: `tests/integration/repo_calendar_test.go`

- [ ] **Step 1: Write the test**

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestRepoCalendar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()

	session := loginUser(t, "user2")
	setDeadline := func(index int64, date string) {
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/%d/deadline", index), map[string]string{"deadline": date})
		session.MakeRequest(t, req, http.StatusOK)
	}
	// user2/repo1 holds issue #1 and pull #2, neither with a deadline in the fixtures
	deadlines := map[int64]string{1: "2026-10-06", 2: "2026-10-30"}
	for index, date := range deadlines {
		setDeadline(index, date)
	}
	defer func() { // the issue indexer outlives the test database, so leave it as the fixtures had it
		for index := range deadlines {
			setDeadline(index, "")
		}
	}()

	t.Run("shows issues and pulls on their days", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		var links []string
		doc.Find(".repo-calendar-day:not(.is-other-month) .repo-calendar-entry").Each(func(_ int, a *goquery.Selection) {
			links = append(links, a.AttrOr("href", ""))
		})
		assert.ElementsMatch(t, []string{"/user2/repo1/issues/1", "/user2/repo1/pulls/2"}, links)
		assert.Equal(t, 1, doc.Find(".repo-calendar-entry.is-pull").Length())
		assert.Equal(t, 1, doc.Find(".repo-calendar-day.is-today").Length())
	})

	t.Run("month navigation keeps the filters", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar?month=2026-09&state=all"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		assert.Zero(t, doc.Find(".repo-calendar-day:not(.is-other-month) .repo-calendar-entry").Length())
		next, _ := doc.Find(`.list-header a[href*="month=2026-10"]`).First().Attr("href") // the next-month arrow
		assert.Contains(t, next, "month=2026-10")
		assert.Contains(t, next, "state=all")
	})

	t.Run("the issue list links to the calendar", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		assert.Equal(t, 1, doc.Find(`.list-header a[href="/user2/repo1/calendar"]`).Length())
	})

	t.Run("anonymous users", func(t *testing.T) {
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar"), http.StatusOK)
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo2/calendar"), http.StatusNotFound) // private
	})
}
```

- [ ] **Step 2: Run it**

Run (bash, on WSL2 or the devcontainer if native Windows misbehaves):
`GITEA_TEST_DATABASE=sqlite go test -run '^TestRepoCalendar$' gitea.dev/tests/integration`
Expected: `ok  gitea.dev/tests/integration`. Should finish in under 2 seconds once the server is up.

- [ ] **Step 3: Hand the feature commit to the human**

Run first:
- `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 fmt`
- `go run ./tools/lint-go-all.go`. Expected: no issues reported.

Then ask the human to review and run:

```bash
git add routers/web/repo/calendar.go routers/web/repo/calendar_test.go routers/web/web.go options/locale/locale_en-US.json templates/repo/calendar.tmpl templates/repo/calendar_entry.tmpl templates/repo/issue/list.tmpl web_src/css/features/calendar.css web_src/css/index.css tests/integration/repo_calendar_test.go && git commit -m "feat(calendar): add repository calendar view of milestone and issue deadlines (#98)" -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 9: Check it in the browser

**Files:** none

- [ ] **Step 1: Build and run the app**

Run: `go build -o gitea.exe` then start `./gitea.exe web` from the repo root (port 3000, config in `custom/conf/app.ini`). Use the preview tool rather than a raw shell if it's available.

- [ ] **Step 2: Set up data and look**

In a repository you own:
- Give two issues and one pull request deadlines this month, with four or more on the same day.
- Create a milestone due this month.

Open `/{owner}/{repo}/calendar` and check:
- The milestone appears first in its cell.
- Issues have a green accent, and a merged pull request has a purple one.
- "+N more" expands in place.
- The previous/next arrows change `?month=` and keep the filters.
- The milestone, assignee and state filters narrow the grid.
- The Calendar button appears on both the Issues and Pull Requests lists.

- [ ] **Step 3: Check dark theme and mobile**

- Switch to the dark theme in user settings. The accents and the dimmed out-of-month days should stay readable.
- Resize to 375px wide. Only days that have entries should show, one per row.

- [ ] **Step 4: Screenshot for the PR**

Take one desktop and one mobile screenshot to attach to the PR description.

---

### Task 10: Paperwork

**Files:**
- Modify: `docs/superpowers/specs/2026-10-25-repo-calendar-view-design.md` (only if implementation differed)

- [ ] **Step 1: Update issue #98's body** (the human posts this)

Draft the edits listed in the spec's "Changes to the issue body" section for the human to paste:
- Answered Open Questions
- AC1 and AC5 wording
- New AC7
- Consequences notes

- [ ] **Step 2: Draft the PR description**

Use the PR format in `.claude-students/COURSE-WORKFLOW.md`, including the AI assistance section from `.claude-students/AI-ASSISTANCE.md`. No AI log links go in the PR. Raise the 1,000-entry cap and the 19 new locale keys as review points.

- [ ] **Step 3: AI log reminders**

Remind the human to:
- run `specstory sync`
- file the log under `ai-logs/sprint2/vvchopra/` (copy and rename only)
- post the AI Assistance comment on #98
