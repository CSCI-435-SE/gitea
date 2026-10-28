# Repository calendar view: design (#98)

Part of #36. Issue: https://github.com/CSCI-435-SE/gitea/issues/98

## Goal

A month calendar at `/{owner}/{repo}/calendar` showing the repository's open milestone due dates
and the deadlines of its issues **and pull requests**, so users can see what is due when without
scanning the lists. Server-rendered, no JS required.

## Decisions taken during brainstorming

| Question | Decision |
| --- | --- |
| Include pull requests? | Yes. Issues and PRs both appear, styled by kind and state. |
| Where is the entry point? | A `Calendar` button next to Milestones in the list header (`templates/repo/issue/list.tmpl`), so it shows on both the Issues and Pull Requests lists. No new repo tab. |
| Milestone entries under filters | They follow the milestone filter (only the selected milestone is shown). Assignee and state filters do not hide milestones. Milestones are always open-only. |
| Busy days | A cell shows 3 entries; the rest sit in a native `<details>` ("+N more") that expands in place. |
| Structure | A new handler plus a pure `buildMonthGrid` function (not reusing `prepareIssueFilterAndList`, not Vue). |

## Changes to the issue body

- Open Questions: answered as in the table above.
- AC1: "every issue **and pull request** with a deadline in that month".
- AC5: 404 only when the user can read **neither** issues nor pull requests; when they can read
  one unit, only that unit's entries appear.
- New AC7: given a day with more than 3 entries, the cell shows 3 and a "+N more" control that
  reveals the rest on the same page.
- Consequences: old milestones store "no due date" as `253370764800` and are skipped; the query is
  capped at 1,000 entries with a visible notice.

## Route

`m.Get("/calendar", repo.Calendar)` in the existing group in `routers/web/web.go` that already holds
`/labels`, `/milestones` and `/milestone/{id}`. That group applies `optSignIn`,
`context.RepoAssignment` and `reqRepoIssuesOrPullsReader`, which gives AC5 with no new guard.

## Query parameters

| Param | Meaning | Missing or invalid |
| --- | --- | --- |
| `month` | `YYYY-MM`, year 1970–9998 | current month in `setting.DefaultUILocation`, no error |
| `milestone` | milestone ID in this repo | no milestone filter |
| `assignee` | user ID | no assignee filter |
| `state` | `open`, `closed`, `all` | `open` |

Every navigation and filter link carries the whole query string, so month and filters persist
(AC3, AC4). The prev and next month links come from `calendarMonthLink`, which returns "" at the
1970 and 9998 bounds (where `parseCalendarMonth` would fall back to today); the template hides a
link whose value is empty.

## Handler: `routers/web/repo/calendar.go`

1. `parseCalendarMonth(ctx.FormString("month"), now)` returns the first instant of the month in
   `setting.DefaultUILocation`. `end` is the last second of the month's final day.
2. Pick `IsPull` from unit permissions: both readable → `None`; issues only → `Some(false)`;
   pull requests only → `Some(true)`. An external issue tracker counts as issues not readable.
3. One `issues_model.Issues(ctx, &IssuesOptions{...})` call:
   `RepoIDs = [repo.ID]`, `DeadlineAfterUnix = start`, `DeadlineBeforeUnix = end`,
   `HasDeadline = Some(true)` (AC6), `MilestoneIDs`, `AssigneeID`, `IsClosed`, `IsPull`,
   `SortType: "oldest"`, `Paginator` capped at 1,000. (The `nearduedate` sort prefers a
   milestone's deadline over the issue's own, so it would truncate the wrong rows; cells re-sort by
   deadline in Go anyway.) `Issues` loads each PR through `LoadAttributes`, so there is no separate
   `LoadPullRequests` call. The query asks for `PageSize: calendarMaxEntries + 1`; the extra row only
   detects truncation and is dropped, and `ctx.Data["CalendarTruncated"]` is set when it is present.
   A negative `?milestone` other than -1 ("no milestone") is normalised to 0, so the issue and
   milestone filters agree.
4. Milestones: `db.Find[issues_model.Milestone](ctx, FindMilestoneOptions{RepoID})` with no
   `IsClosed` filter, because the dropdown needs the closed milestones too. `calendarMilestones` then
   keeps the open ones in Go, with `0 < DeadlineUnix < 253370764800` and inside `[start, end]`. With
   a milestone filter, keep only that milestone (`-1` keeps none).
5. Convert issues and milestones to `calendarEntry` values, then
   `ctx.Data["Weeks"] = buildMonthGrid(start, now, entries)`.
6. Fill the milestone and assignee dropdowns with the helpers the issue list already uses.
7. Render `repo/calendar`.

### Grid builder (no side effects, no DB)

```go
type calendarEntry struct {
	Kind  string // "milestone" | "issue" | "pull"
	State string // "open" | "closed" | "merged"
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

func (d calendarDay) Visible() []calendarEntry  // first 3 entries
func (d calendarDay) Overflow() []calendarEntry // the rest, behind "+N more"

func buildMonthGrid(monthStart, now time.Time, entries []calendarEntry) [][]calendarDay
```

- Weeks start on Monday (ISO 8601, as #34's `week` filter). The grid holds whole weeks, padding
  with days from the neighbouring months marked `InMonth=false`.
- Day boundaries come from `time.Date(y, m, d, ..., loc)`, never from adding `24h`, so DST
  transitions cannot shift a date.
- An entry lands on the date of its deadline in `monthStart.Location()`. Deadlines are pinned to
  23:59:59 so they never spill into the next day.
- Inside a cell: milestones first, then issues and PRs by deadline, then by index.

## Template: `templates/repo/calendar.tmpl`

```
repo/header
.ui.container
  .list-header.flex-text-block
    ‹ prev · "October 2026" · next › · Today
    .list-header-filters: Milestone ▾  Assignee ▾   (the issue list's filter_item_* partials)
    State menu (Open · Closed · All), role="group" with an aria-label
  [truncated notice when CalendarTruncated]
  .repo-calendar-grid
    7 weekday header cells (Mon … Sun)
    .repo-calendar-day [.is-other-month] [.is-today]
      <time datetime="YYYY-MM-DD"> weekday label (narrow screens) + day number </time>
      first 3 entries
      <details><summary>+N more</summary> remaining entries </details>
  [empty-month message when the month has no entries]
```

The per-day weekday label is hidden on wide screens, where the header row names the columns, and
shown under 768px, where the grid collapses to an agenda list. The empty-month message is
`repo.calendar.empty`.

Each entry is an `<a>` to its issue, PR or milestone (AC2) with an icon, `#index` for issues and
PRs, and the title truncated with `tw-truncate`; the full title is in `title=`, prefixed with the
state word ("Closed: " or "Merged: ") for closed issues and PRs and merged PRs, since the icon colour
alone does not say it.

| Kind | Icon | Accent |
| --- | --- | --- |
| Milestone | `octicon-milestone` | `--color-text`, bold title |
| Issue | `octicon-issue-opened` / `octicon-issue-closed` | `--color-green` open, `--color-red` closed |
| Pull request | `octicon-git-pull-request` / `-merged` / `-closed` | `--color-green` open, `--color-purple` merged, `--color-red` closed |

Entry point: a `Calendar` button after the Milestones button in `templates/repo/issue/list.tmpl`.

## CSS: `web_src/css/features/calendar.css`

Imported from `web_src/css/index.css`.

- `.repo-calendar-grid`: CSS grid, `repeat(7, minmax(0, 1fr))`; day cells at least `7rem` tall.
- `.is-other-month`: `--color-text-light-3` text on `--color-box-header`. `.is-today`: a
  `--color-primary` outline.
- Under 768px wide: one column, out-of-month and empty days hidden, so it reads as an agenda list.
  CSS only.
- Use `flex-*` helpers for layout inside cells, not per-child margins.

## Locale

Keys in `options/locale/locale_en-US.json` only: `repo.calendar`, `repo.calendar.today`,
`repo.calendar.prev_month`, `repo.calendar.next_month`, `repo.calendar.more` (`+%d more`),
`repo.calendar.truncated`, `repo.calendar.state`, `repo.calendar.state_all`,
`repo.calendar.empty` ("Nothing is due this month."), plus
`repo.calendar.month_1`…`month_12` and `repo.calendar.weekday_mon`…`weekday_sun`. The locale has no
month or weekday names, and the existing date helpers only render whole dates, so the names are
added as keys. Both families are built with `printf` / `print` in `templates/repo/calendar.tmpl`, so
a grep for the full key finds no use; see `.claude-students/docs/i18n.md`.

## Edge cases

| Case | Handling |
| --- | --- |
| Invalid or out-of-range `?month=` | Current month, no error |
| Milestone with `DeadlineUnix` of `0` or `253370764800` | Not shown |
| DST change inside the month | Correct dates, via `time.Date` boundaries |
| `?milestone=` from another repo | Matches nothing; empty grid, no leak |
| More than 1,000 issues and PRs in the month | First 1,000 by creation date, plus a notice suggesting filters |
| Issues disabled, pull requests enabled | Page allowed; PRs and milestones shown |

## Testing

1. `routers/web/repo/calendar_test.go`, no DB:
   - `TestParseCalendarMonth`: valid values, invalid values and the year bounds.
   - `TestCalendarEntries`: kind and state of issues, PRs (open, closed, merged) and milestones.
   - `TestBuildMonthGrid`: grid shape (Oct 2026 starts Thursday, Feb 2027 fills four rows, Nov 2026
     with DST ending on the 1st), entries on their deadline's day with milestones first, the cell
     limit split, `IsToday` in the grid's location, trailing out-of-month cells, `Visible` and
     `Overflow` edges, and ordering by title on equal deadlines.
   - `TestCalendarIsPull`, `TestCalendarMilestones` and `TestCalendarMonthLink`: the helpers.
2. `TestCalendarHandler` in the same file, with `contexttest.MockContext` and fixtures: an issue
   shows only in its deadline's month, the state and assignee filters, a milestone on its due day,
   unit permissions limiting what shows, and PRs appearing with `Kind == "pull"`.
3. `tests/integration/repo_calendar_test.go`, `TestRepoCalendar`, aiming for under 2 seconds: issues
   and PRs on their days, month navigation keeping the filters, the issue list linking to the
   calendar, and anonymous access (200 on a public repo, 404 on a private one).

No Playwright test: the page has no JS behaviour. Some Go tests fail on native Windows, so run the
integration test on WSL2 or CI.

## Out of scope

`.ics` feeds (#B and #C), the cross-repository `/calendar` page (#D), drag to reschedule, week and
day views, and an exact-date `due=` filter on the issue list.
