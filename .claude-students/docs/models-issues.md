---
scope: models/issues, models/pull
verified-at: c0092050a4
---

# models/issues — issues, pull requests, comments, reviews

**Read when:** querying or changing issue, PR, comment, label, milestone or review rows.
**Not here:** the business logic on top -> `services/issue` and `services/pull`; project boards -> `models/project`.

## Responsibilities

One package for the whole issue domain, because pull requests *are* issues in this schema.
`models/pull` is separate and tiny: automerge scheduling and per-user review state.

| Area | Files |
| --- | --- |
| Issue core | `issue.go`, `issue_list.go`, `issue_update.go`, `issue_search.go`, `issue_stats.go` |
| Pull requests | `pull.go`, `pull_list.go` |
| Comments | `comment.go`, `comment_list.go`, `comment_code.go`, `content_history.go` |
| Reviews | `review.go`, `review_list.go` |
| Labels / milestones | `issue_label.go`, `label.go`, `milestone.go`, `milestone_list.go` |
| People / time | `assignees.go`, `stopwatch.go`, `tracked_time.go` |
| Relations / state | `dependency.go`, `issue_xref.go`, `issue_watch.go`, `issue_pin.go`, `issue_lock.go`, `issue_user.go`, `issue_project.go`, `reaction.go` |

## Key files

| Path | What it holds |
| --- | --- |
| `models/issues/issue.go` | `Issue`, `ErrIssueNotExist`, `LoadRepo`, `LoadPullRequest`, `LoadAttributes` |
| `models/issues/issue_list.go` | `IssueList` and its batching `LoadAttributes` |
| `models/issues/pull.go` | `PullRequest`, `PullRequestType`, `PullRequestStatus` |
| `models/issues/comment.go` | `Comment` and the `CommentType` enum |
| `models/issues/issue_label.go` | `LoadLabels` — the idempotency pattern in miniature |
| `models/issues/issue_index.go` | `RecalculateIssueIndexForRepo` |

## Conventions & invariants

- **One `issue` table holds both issues and pull requests.** `Issue.IsPull` discriminates. The
  PR-only columns live in a separate `PullRequest` row linked by `IssueID`, reachable via
  `issue.LoadPullRequest(ctx)` then `issue.PullRequest`. A query that forgets `IsPull` returns both
  kinds.
- `Issue.Index` is the **per-repository** number users see in URLs (`UNIQUE(repo_index)` with
  `RepoID`); `Issue.ID` is the global primary key. Look-ups by URL go through `RepoID` + `Index`.
- Never assign `Index` yourself: it comes from
  `db.GetNextResourceIndex(ctx, "issue_index", repo.ID)` inside the inserting transaction, as
  `issue_update.go` and `pull.go` do.
- Fields tagged `xorm:"-"` (`Repo`, `Poster`, `Labels`, `Milestone`, `Assignee`, `Attachments`,
  `Comments`, `PullRequest`, ...) are in-memory only, filled by the matching `LoadXxx(ctx)` method.
  `LoadAttributes(ctx)` loads the common set.
- `LoadXxx` methods are **idempotent and will not refresh**: each is guarded by an `isXxxLoaded`
  flag, so a second call is free — but a value that changed in the database after the first call
  stays stale on that struct. Re-fetch the issue rather than re-calling the loader.
- Error types come in the three-part shape: `ErrIssueNotExist` struct, `IsErrIssueNotExist(err)`,
  and `Unwrap() error` returning a `modules/util` sentinel such as `util.ErrNotExist`, so
  `errors.Is` works across layers.
- Comment kinds are the `CommentType` iota in `comment.go` (`CommentTypeComment` is a plain
  comment, `CommentTypeReview` a review, plus many state-change kinds). Rendering branches on it —
  adding a value means handling it in the templates too.

## Recipes

**Load a list of issues for display.** Fetch into an `IssueList`, then call
`issues.LoadAttributes(ctx)` once. Calling `issue.LoadAttributes(ctx)` in a loop is the N+1 this
type exists to prevent.

**Add a column to `Issue`.** Add the field with its xorm tag, add a migration
(`models-migrations.md`), and if it is denormalised from elsewhere, extend the consistency check in
`models/main_test.go` and the fixtures (`testing.md`).

**Find a PR from an issue.** `issue.LoadPullRequest(ctx)`, then `issue.PullRequest`. From the other
direction, `pr.Issue` after the PR's own loader.

## Gotchas

- Counter columns on `Repository` (`NumIssues`, `NumClosedIssues`, `NumPulls`, ...) are
  denormalised. Changing issue state means keeping them in step, and fixture rows must match or
  `unittest.CheckConsistencyFor` fails.
- `Issue.Index` and `PullRequest.Index` are both present. They agree, but write through the issue.
- `models/pull` is not "pull requests" — those are here in `models/issues/pull.go`. `models/pull`
  is only automerge and review state.

## Related

- `models-db.md` — transactions and `GetNextResourceIndex`
- `models-repo-and-git.md` — the counter columns and repo units that gate issues
