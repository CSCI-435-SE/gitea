---
source: docs/models-issues.md
source-hash: d7da7f01064e5b06
verified-at: c0092050a4
---

<!-- Derived from docs/models-issues.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Issues and pull requests in the database, in plain English

**In one sentence:** one table holds both issues and pull requests, most of what you think of as
"the issue" is loaded separately on demand, and the number in the URL is not the primary key.
**Come here when:** you are querying or changing issues, pull requests, comments, labels, milestones
or reviews.

## What this is

The database side of the whole issue domain — and in Gitea, **pull requests live here too**, because
a pull request is an issue with extra columns.

The files divide by concern: issue core, pull requests, comments, reviews, labels and milestones,
people and time tracking, and the relationships between issues.

There is also a separate small package, `models/pull`, which despite its name is not where pull
requests live. It holds only automerge scheduling and per-user review state.

## Why it exists

**Why one table for two things?** Because an issue and a pull request are the same thing to almost
all of Gitea. Both have a title, a description, comments, labels, a milestone, assignees, an
open/closed state and a number. If they were separate tables, every one of those features would need
implementing twice. Instead there is one `issue` row with a flag, and the handful of pull-request-only
columns live in a second row linked to it.

**Why is so much loaded separately?** An issue row does not contain its author, labels, milestone or
comments — those are other tables. Fetching all of them every time would make listing a hundred
issues enormously expensive when the list page needs only a few. So the row comes back with those
fields empty and you load what you actually need.

That is a good trade, and it produces the single most common bug in this area: **reading a field
nobody loaded and getting an empty value rather than an error.**

## Words you'll meet

- **discriminator** — a column saying which kind of thing a row is. Here, `IsPull`.
- **primary key** — the globally unique `ID`.
- **per-repository index** — the number users see (`#42`), which restarts at 1 in every repository.
- **loader** — a `LoadXxx(ctx)` method that fills in one of those empty fields.
- **idempotent** — calling it twice does the same as calling it once.
- **denormalised counter** — a count stored on another row for speed, which then has to be kept
  correct by hand.
- **N+1** — fetching *n* things and then running *n* more queries, one per item.
- **sentinel error** — a shared error value that code elsewhere can test for.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/issues/issue.go` | The `Issue` struct itself, its loaders, and its "not found" error. |
| `models/issues/issue_list.go` | `IssueList` — a slice of issues that can load everything for all of them at once. |
| `models/issues/pull.go` | The pull-request-only row. |
| `models/issues/comment.go` | Comments, and the enum of comment kinds. |
| `models/issues/issue_label.go` | Labels on issues — and the clearest small example of the loader pattern. |
| `models/issues/issue_index.go` | Recalculating per-repository numbering. |

The rest divide by area: `review.go` and `review_list.go`, `label.go`, `milestone.go`,
`assignees.go`, `stopwatch.go` and `tracked_time.go`, and the relationship files `dependency.go`,
`issue_xref.go`, `issue_watch.go`, `issue_pin.go`, `issue_lock.go`, `reaction.go`.

## The rules, and why

**`IsPull` decides what a row is.** A query that forgets it returns both issues and pull requests.
On an issues page that means pull requests appearing in the list; on a count, a number that is
always too high.

**`Index` is not `ID`.** `ID` is the global primary key. `Index` is the number in the URL, unique
only *within* a repository — so every repository has an issue `#1`. Looking something up from a URL
means matching on the repository **and** the index. Using the URL number as an `ID` finds a
completely unrelated issue in another repository, which is worse than an error because it looks like
it worked.

**Never set `Index` yourself.** It comes from the shared numbering helper, called inside the same
transaction that inserts the row. Assigning it by hand races with anyone else creating an issue in
that repository at the same moment.

**Fields tagged `xorm:"-"` are empty until you load them.** `Repo`, `Poster`, `Labels`,
`Milestone`, `Assignee`, `Attachments`, `Comments`, `PullRequest` and others are all filled in by a
matching `LoadXxx(ctx)` method. `LoadAttributes(ctx)` loads the usual set in one go.

**Loaders never refresh.** Each is guarded by an internal "already loaded" flag, so calling one
twice is free — but it also means a value that changed in the database after the first call **stays
stale on that struct**. If you need current data, re-fetch the issue; calling the loader again does
nothing.

**Errors come in three parts.** There is an `ErrIssueNotExist` struct, an `IsErrIssueNotExist(err)`
test, and an `Unwrap()` that returns a shared sentinel like `util.ErrNotExist`. That third part is
what lets a handler ask "is this a not-found error?" without importing every model package — and it
is what turns a missing issue into a 404 instead of a 500.

**Comment kinds are an enum**, and rendering branches on it. A plain comment and a review are both
comments with different kinds, as are all the "closed this", "added a label" timeline entries.
Adding a kind means handling it in the templates too, or it renders as nothing.

## How to actually do it

**Display a list of issues.** Fetch into an `IssueList`, then call `LoadAttributes(ctx)` **once, on
the list**. That type exists precisely to load everything for every issue in a handful of queries.
Looping and calling the single-issue loader is the N+1 it was built to prevent, and it is the
difference between a fast page and a slow one.

**Add a column to `Issue`.** Add the field with its tag, write a migration, and if the value
duplicates something stored elsewhere, extend the consistency check in `models/main_test.go` and the
fixtures.

**Get the pull request for an issue.** `issue.LoadPullRequest(ctx)`, then read `issue.PullRequest`.
Going the other way, use the pull request's own loader and then `pr.Issue`.

## Traps, and what they look like

**A field is empty and you are sure it has data.** You did not call its loader. There is no error
for this — the field is simply the zero value. Check for the `LoadXxx` call before you check the
database.

**You changed something and the struct still shows the old value.** Loaders do not refresh. Re-fetch
the issue.

**Pull requests show up on the issues page, or a count is too high.** A query missing the `IsPull`
condition.

**You looked up issue `#42` and got somebody else's issue.** You used the URL number as an `ID`.
Match repository plus index.

**A test you did not touch fails on a consistency check.** The repository row stores counts —
`NumIssues`, `NumClosedIssues`, `NumPulls` and friends — alongside the real issues. Changing issue
state, or adding a fixture row, means keeping those in step.

**You go looking for pull requests in `models/pull` and it is nearly empty.** Pull requests are in
`models/issues/pull.go`. `models/pull` only holds automerge and review state. Both `Issue` and
`PullRequest` also carry an `Index`; they agree, and you should write through the issue.

## Where to go next

- `docs/models-issues.md` (in this folder) — the reference page this was written from
- `models-db.md` — transactions and the numbering helper
- `models-repo-and-git.md` — those counter columns, and the per-repo feature switches
