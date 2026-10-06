---
source: docs/models-issues.md
source-hash: b9b6d7cb557c0756
verified-at: 94dfae067e
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
- **JSON field** — one column holding a small bundle of named values, saved as JSON text.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/issues/issue.go` | The `Issue` struct itself, its loaders, and its "not found" error. |
| `models/issues/issue_list.go` | `IssueList` — a slice of issues that can load everything for all of them at once. |
| `models/issues/pull.go` | The pull-request-only row. |
| `models/issues/comment.go` | Comments, and the enum of comment kinds. |
| `models/issues/issue_close_reason.go` | Why an issue or pull request was closed: the `CloseReason` values and their names, which ones each kind may use, which ones the list's bulk Close offers, which one the close button starts on, and the check that refuses a bad one, plus the lookup of the issue a duplicate points at (the API uses it too). Saved in three columns on `Issue`. |
| `models/issues/issue_label.go` | Labels on issues — and the clearest small example of the loader pattern. |
| `models/issues/issue_index.go` | Recalculating per-repository numbering. |
| `models/issues/issue_group.go` | Splitting a list of issues into category groups, for the grouped ("folder") list view. |

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

**Extra data on a comment goes in its metadata, not a new column.** `Comment` has a JSON field,
`CommentMetaData`, and its code comment in `comment.go` asks for any new non-index data to go there.
It is written once, when the comment is created, and never changed — which suits a timeline, where
each entry records what was true at that moment. The cost is that SQL cannot search, sort or index
on anything inside it.

**A close reason travels as a word, not a number.** The database stores each reason as a number,
but forms and pages send its name instead: `not_planned`, not `2`. `CloseReason.String` turns a
reason into its name and `AsCloseReason` turns a name back, the same way `CommentType.String` and
`AsCommentType` work for comment kinds. Because pages send these names, a name can no more be
changed than its number can.

**A close comment keeps its own copy of the reason.** The issue's reason describes the *current*
close; the comment's copy, in `CommentMetaData`, describes *that* close and never changes. Read it
with `Comment.MetaCloseReason()` rather than reaching into the metadata yourself: close comments
written before this feature existed have no metadata at all, and the method turns that into "no
reason" instead of an error.

**A duplicate is typed as a number but stored as an ID.** Closing as a duplicate of `#12` looks
`#12` up in the issue's own repository (`GetIssueByIndex`, inside `SetIssueAsClosed`), so a number
from another repository is refused. What gets stored is that issue's global ID. `Comment.LoadCloseDuplicateIssue` turns the ID back into the
issue, repository included; if the target has since been deleted, it reports "not found" instead
of breaking the page.

**Reopening wipes the issue's reason, not the comment's.** `setIssueAsReopen` is the only code that
reopens anything, and it clears the reason, text and duplicate target on the issue. The close
comment keeps what was true when it was written.

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

**You grouped the list by category and a group appears twice, or in a different place, on page 2.**
The grouping is worked out in two places that must agree: `applyGroupByLabelScope` in
`models/issues/issue_search.go` sorts the rows in SQL so that one page's issues of the same category
sit together, and `GroupByExclusiveLabelScope` in `models/issues/issue_group.go` then cuts that page
into groups in Go. If you change the order in one, change it in the other. In particular both settle
ties on the label's `id`, not its name, because the database and Go do not sort text the same way.

**`Test_MigrateFromGiteaToGitea` fails with "missing fixture" after you add a column to `Issue`.** The
migration code fetches issues in pages sized by `db.MaxBatchInsertSize` (`models/db/engine.go`): 999
divided by the number of columns. One more column changes the page size, the request asks for a
different `limit`, and the recorded response in
`tests/integration/_mock_data/Test_MigrateFromGiteaToGitea/` no longer matches its name. Rename that
file to the new `limit`; its contents stay valid while the repository has fewer issues than the page size.

**Building an issue's link crashes with a nil pointer.** `Issue.Link()` reads `issue.Repo`, and
`GetIssueByID` does not fill it in. Call `LoadRepo(ctx)` first.

**You go looking for pull requests in `models/pull` and it is nearly empty.** Pull requests are in
`models/issues/pull.go`. `models/pull` only holds automerge and review state. Both `Issue` and
`PullRequest` also carry an `Index`; they agree, and you should write through the issue.

## Where to go next

- `docs/models-issues.md` (in this folder) — the reference page this was written from
- `models-db.md` — transactions and the numbering helper
- `models-repo-and-git.md` — those counter columns, and the per-repo feature switches
