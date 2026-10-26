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
(AC3, AC4).

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
   deadline in Go anyway.) If the result hits the cap, set
   `ctx.Data["CalendarTruncated"]`. Then `issues.LoadPullRequests(ctx)` once, for PR state.
4. Milestones: `db.Find[issues_model.Milestone](ctx, FindMilestoneOptions{RepoID, IsClosed: Some(false)})`,
   filtered in Go to `0 < DeadlineUnix < 253370764800` and inside `[start, end]`. With a milestone
   filter, keep only that milestone.
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
    Milestone ▾  Assignee ▾  State ▾        (same markup as repo/issue/filter_list)
  [truncated notice when CalendarTruncated]
  .repo-calendar-grid
    7 weekday header cells (Mon … Sun)
    .repo-calendar-day [.is-other-month] [.is-today]
      day number
      first 3 entries
      <details><summary>+N more</summary> remaining entries </details>
```

Each entry is an `<a>` to its issue, PR or milestone (AC2) with an icon, `#index` for issues and
PRs, and the title truncated with `tw-truncate`; the full title is in `title=`.

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
`repo.calendar.truncated`, `repo.calendar.state`, `repo.calendar.state_all`, plus
`repo.calendar.month_1`…`month_12` and `repo.calendar.weekday_mon`…`weekday_sun`. The locale has no
month or weekday names, and the existing date helpers only render whole dates, so the names are
added as keys.

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
   - Grid shape: Oct 2026 has 5 rows starting Mon 28 Sep; Feb 2027 has 4; a month starting on a
     Monday has no leading days.
   - A deadline at 23:59:59 in `America/New_York` lands on its own day.
   - Nov 2026 (DST ends in New York) puts every day on the right date.
   - Cell order: milestones first, then by deadline.
   - `IsToday` and `InMonth` flags.
   - `parseCalendarMonth`: valid values, invalid values and the year bounds.
2. Handler test with `contexttest.MockContext` and fixtures. Fixture issue 10 (repo 42, open) has
   `deadline_unix: 1019307200` (20 Apr 2002):
   - `?month=2002-04` shows it and `?month=2002-05` does not.
   - `?state=closed` and `?assignee=` filter it out.
   - Milestones with the `253370764800` value never appear.
   - A fixture PR given a deadline in the test appears with `Kind == "pull"`.
3. `tests/integration/repo_calendar_test.go`, aiming for under 2 seconds:
   - Anonymous user on a public repo gets 200 and `.repo-calendar-grid` is in the HTML.
   - Anonymous user on a private repo gets 404.

No Playwright test: the page has no JS behaviour. Some Go tests fail on native Windows, so run the
integration test on WSL2 or CI.

## Out of scope

`.ics` feeds (#B and #C), the cross-repository `/calendar` page (#D), drag to reschedule, week and
day views, and an exact-date `due=` filter on the issue list.
