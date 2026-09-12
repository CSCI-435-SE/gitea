---
source: docs/services-issue.md
source-hash: fe02972eafd8c2c7
verified-at: c0092050a4
---

<!-- Derived from docs/services-issue.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# What happens when an issue changes, in plain English

**In one sentence:** the layer between the web page and the database that does everything *around* an
issue change — permission checks, the timeline entry, and telling everyone who cares.
**Come here when:** you are changing what happens when an issue is created, edited, labelled,
assigned, closed or commented on.

## What this is

A thin layer of functions, one per thing a user can do to an issue. Each one is short, because the
actual row writing is delegated to `models/issues`. What lives here is everything *else* that has to
happen.

Take renaming an issue. The database change is one field. But you also have to check the user is
allowed, record "so-and-so changed the title from X to Y" on the timeline, and notify the watchers,
the webhooks, and anyone subscribed by email.

That "everything else" is this package.

## Why it exists

**Because a database write is never the whole job**, and if each caller had to remember the rest,
each one would forget something different. The web page would send notifications but skip the
timeline entry; the API would do both but skip the block check.

So there is exactly one place per operation, and every caller goes through it. That is the rule
underneath the whole layer: **routers call services, services call models.** A router reaching
straight into `models/issues` gets the row change and silently skips the permission check, the
timeline entry and the notification.

## Words you'll meet

- **doer** — the user performing the action. Needed for the timeline entry and the notification.
- **timeline** — the chronological list on an issue page: comments, plus every state change.
- **notification fan-out** — one event causing email, webhooks, in-app notifications and CI.
- **blocked user** — someone a repository owner has blocked from interacting.
- **transaction** — a group of database writes that all succeed or all get undone.
- **optimistic concurrency** — letting two people edit at once and detecting the clash at save time,
  using a version number.
- **comment type** — what kind of timeline entry a row is: a real comment, or "closed this", or
  "added a label".

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/issue/issue.go` | Creating an issue, changing its title or reference, deleting it. |
| `services/issue/label.go` | Adding, removing and replacing labels. |
| `services/issue/assignee.go` | Assigning and unassigning people. |
| `services/issue/content.go` | Editing the issue body. |
| `services/issue/status.go` | Opening and closing. |
| `services/issue/comments.go`, `reaction.go`, `milestone.go` | Comments, reactions, milestones. |
| `services/issue/review_request.go` | Requesting a review from a person or a team. |
| `services/issue/commit.go` | Acting on `fixes #123` in a pushed commit message. |
| `services/issue/template.go` | Issue templates. |
| `services/issue/suggestion.go` | The autocomplete when you type `#`. |

## The rules, and why

**Every mutating function takes the doer.** The signature is `(ctx, issue, doer, ...)`. The doer is
not optional — it goes onto the timeline entry and onto the notification, so an operation without
one produces history saying nobody did it. Do not add a function that changes an issue without one.

**There is an order, and it matters.** Every function here follows the same four steps:

1. **Load what you need** — `issue.LoadRepo(ctx)`, `LoadPoster` and friends. Remember those fields
   are empty until loaded (`models-issues.md`).
2. **Refuse blocked users, before changing anything.** Checking afterwards means you already made
   the change.
3. **Make the change inside a transaction**, delegating the row writes to `models/issues`.
4. **Notify — after the transaction has returned.**

**Step 4 is the invariant to protect.** Notification calls sit *outside* the transaction, always.

Here is why. Notifications send email, fire webhooks and trigger CI. Those things leave the server
and cannot be recalled. If you notify inside a transaction and the transaction then rolls back, you
have emailed people about an issue that does not exist, called somebody's webhook about a change
that never happened, and possibly started a CI run against nothing.

The database can be rolled back. Sent email cannot. So the change commits first, and only then does
anyone get told. **Never move a notify call inside the transaction** — it will look tidier and it
will be wrong.

**A state change is a comment row, not just a field update.** The issue timeline *is* those rows.
Closing an issue by flipping the field and skipping the comment leaves no history — the issue is
closed and nothing says who did it or when.

**Content edits carry a version number.** `ChangeContent` takes the version the client submitted, so
that if someone else edited in the meantime the clash is detected. Pass through what the client
sent; do not invent one.

## How to actually do it

**Add a new issue operation.** Write the row change in `models/issues` first. Then write a short
function here doing the four steps. `ChangeTitle` is the smallest complete example: load,
block-check, model call, notify.

**React to a new commit-message keyword.** That is `UpdateIssuesCommit`, which runs as part of the
push path. The parsing of `fixes #123` itself lives in `modules/references`.

**Add a new kind of timeline entry.** Three places: a new comment type in `models/issues`, creating
that comment in your function here, and handling the new type in the issue-view templates. Miss the
last one and the entry exists in the database and renders as nothing on the page.

## Traps, and what they look like

**The change works but nobody is notified, and nothing appears in the timeline.** A router called
`models/issues` directly instead of going through this layer. The row changed; every check and side
effect was skipped.

**You called the wrong `NewIssue`.** There is one here and one in `models/issues`, and they differ
by exactly this wrapper. This is what the import aliases are for — `issue_service` and
`issues_model` — so the call site says which you meant.

**Users are emailed about something that did not happen.** A notify call ended up inside the
transaction and the transaction rolled back.

**A consistency test fails after your change.** The repository's issue counters are maintained by
the model layer. A mutation that bypasses it leaves the counts wrong, and a test you did not touch
catches it.

**Your new timeline entry does not appear.** The comment row is being created, but the templates do
not handle that comment type, so it renders as nothing.

## Where to go next

- `docs/services-issue.md` (in this folder) — the reference page this was written from
- `models-issues.md` — the rows underneath, and their loaders
- `services-pull-and-gitdiff.md` — the pull-request equivalents, including reviews
- `routers-web.md` and `routers-api-v1.md` — the callers
