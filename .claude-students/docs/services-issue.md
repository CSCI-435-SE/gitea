---
scope: services/issue
verified-at: c0092050a4
---

# services/issue — issue business logic

**Read when:** changing what happens when an issue is created, edited, labelled, assigned, closed
or commented on.
**Not here:** the rows themselves -> `models-issues.md`; pull requests -> `services-pull-and-gitdiff.md`; project boards -> `models-project.md`.

## Responsibilities

Everything that must happen *around* an issue row change: permission and block checks, the
timeline `Comment` recording the change, counter updates, and the notification fan-out. Routers
call this package; they must not call `models/issues` directly.

## Key files

| Path | What it holds |
| --- | --- |
| `services/issue/issue.go` | `NewIssue`, `ChangeTitle`, `ChangeIssueRef`, `ChangeTimeEstimate`, `DeleteIssue` |
| `services/issue/label.go` | `AddLabel`, `AddLabels`, `RemoveLabel`, `ReplaceLabels`, `ClearLabels` |
| `services/issue/assignee.go` | `ToggleAssignee`, `AddAssignees`, `RemoveAssignees`, `UpdateAssignees` |
| `services/issue/content.go` | `ChangeContent` |
| `services/issue/status.go` | open/close transitions |
| `services/issue/comments.go`, `reaction.go`, `milestone.go` | comment, reaction and milestone operations |
| `services/issue/review_request.go` | `ReviewRequest`, `TeamReviewRequest`, `CanDoerChangeReviewRequests` |
| `services/issue/commit.go` | `UpdateIssuesCommit` — acts on `fixes #123` in pushed commit messages |
| `services/issue/template.go` | issue template parsing |
| `services/issue/suggestion.go` | `GetSuggestion` for the issue autocomplete |

## Conventions & invariants

- Signature shape is `(ctx, issue, doer, ...)`. `doer` is who is acting — it is required, because
  it lands on the timeline comment and on the notification. Do not add a function that mutates an
  issue without a doer.
- **Order of operations, and it matters:**
  1. Load what you need (`issue.LoadRepo(ctx)`, `LoadPoster`) — see `models-issues.md` on loaders.
  2. Refuse blocked users with `user_model.IsUserBlockedBy` *before* mutating.
  3. Mutate inside `db.WithTx`, delegating the row writes to `models/issues`.
  4. **After the transaction returns**, call `notify_service.*`.
- That last point is the invariant to protect: `notify_service` calls sit outside `db.WithTx` in
  `NewIssue` and everywhere else. Notifications send mail, fire webhooks and write UI
  notifications; doing that inside a transaction that later rolls back tells users about something
  that never happened. Never move a notify call inside the tx.
- The import alias is always `notify_service "gitea.dev/services/notify"`.
- A state change is recorded as a `Comment` row with the matching `CommentType`, not just a field
  update. The issue timeline *is* those comment rows, so a mutation that skips the comment leaves
  no history.
- `ChangeContent` takes a `contentVersion` for optimistic concurrency — pass the version the client
  submitted, do not invent one.

## Recipes

**Add a new issue mutation.** Write the row change in `models/issues` first, then a thin function
here that does the four steps above. Look at `ChangeTitle` as the smallest complete example: load,
block-check, model call, `notify_service.IssueChangeTitle`.

**Act on a new commit-message keyword.** `UpdateIssuesCommit` in `commit.go`, which runs from the
push path (`services-repository.md`). Reference parsing itself is in `modules/references`.

**Add a timeline entry for something new.** Add a `CommentType` value in `models/issues`, create
the comment in your service function, and handle the new type in the issue-view templates —
an unhandled type renders as nothing.

## Gotchas

- Calling a `models/issues` function directly from a router skips every check in this package:
  blocked users, the timeline comment and the notification. Routers call services, not models.
- `NewIssue` here differs from `issues_model.NewIssue` in `models/issues` by exactly this wrapper.
  Import-alias the two (`issue_service` / `issues_model`) so the call site says which one you mean.
- Counter columns on `Repository` are maintained by the model layer; if you write a mutation that
  bypasses it, `unittest.CheckConsistencyFor` will fail (`testing.md`).

## Related

- `models-issues.md` — the rows and their loaders
- `services-pull-and-gitdiff.md` — the PR-side equivalents, including reviews
- `routers-web.md`, `routers-api-v1.md` — the callers
