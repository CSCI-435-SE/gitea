---
source: docs/modules-infra.md
source-hash: 96d6b43b61e99a0e
verified-at: c0092050a4
---

<!-- Derived from docs/modules-infra.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Background work, caching and storage, in plain English

**In one sentence:** the machinery for doing things later, remembering things, storing files,
searching, and stopping two servers doing the same job at once.
**Come here when:** something needs to happen in the background, be cached, be stored as a file, be
searchable, or happen only once across several servers.

## What this is

A set of small infrastructure packages the rest of Gitea builds on:

| Package | Job |
| --- | --- |
| `modules/queue` | Background work. |
| `modules/cache` | Remembering expensive results. |
| `modules/storage` | Storing files — attachments, avatars, artifacts. |
| `modules/indexer` | Making things searchable. |
| `modules/globallock` | Mutual exclusion across processes. |
| `modules/log` | Logging. |
| `modules/graceful` | Starting and stopping cleanly. |
| `modules/process` | Managing long-running external commands. |

## Why it exists

**Because the user is waiting.** Several things Gitea does in response to a request are slow —
recomputing whether pull requests still merge, delivering webhooks, reindexing. Doing them while the
browser waits makes the request slow for no benefit to the person who triggered it.

So there are queues: the request records *that* work is needed and returns, and a worker does it
shortly afterwards.

That introduces a problem queues solve directly. A busy repository might generate the same "recheck
these pull requests" work twenty times in a minute. Doing it twenty times is waste. So most of
Gitea's queues are **unique** — adding an item already queued does nothing.

This is why so much code elsewhere enqueues freely rather than trying to be clever about when. The
queue does the deduplicating, and being clever is how you get work that never happens.

**Why locks across processes?** Because a Gitea installation can run several servers against one
database. When they all start at once, exactly one must run the database migrations. A normal
in-memory lock cannot express that.

## Words you'll meet

- **queue** — a list of work items a worker processes later.
- **unique queue** — one that ignores an item already in it.
- **worker** — the background goroutine doing the processing.
- **handler** — the function a worker calls per item.
- **idempotent** — running it twice does the same as running it once.
- **serialisation** — turning a value into bytes so it can be stored and read back.
- **eventually consistent** — correct soon, not immediately.
- **backend** — the interchangeable implementation behind an interface, chosen by configuration.
- **request-scoped** — living for one request and then discarded.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/queue/manager.go` | Creating queues — the simple and the unique kind. |
| `modules/queue/workergroup.go` | The workers behind a queue. |
| `modules/queue/base_channel.go`, `base_levelqueue.go`, `base_redis.go` | The interchangeable backends. |
| `modules/cache/cache.go`, `string_cache.go`, `context.go` | The cache, including the per-request one. |
| `modules/globallock/globallock.go`, `locker.go` | Locking across processes. |
| `modules/indexer/issues/`, `code/`, `stats/` | One subtree per kind of search index. |
| `modules/storage/storage.go` | The file storage interface and its backends. |

## The rules, and why

**Queues are how background work happens.** There are two kinds, and the unique one is usually what
you want: it deduplicates, which is why pushing the same pull request twice is free
(`services-pull-and-gitdiff.md`) and why the Actions job emitter can enqueue freely
(`services-actions.md`).

**A queue handler must be safe to run twice.** Items can be retried after a restart, and a
persistent backend replays anything not acknowledged. A handler that assumes it runs exactly once
will eventually corrupt something.

**Push an id, not an object.** Queue items are serialised to bytes, so a loaded struct with
unexported fields or pointers into the database layer does not survive the trip. Push the id; the
handler loads what it needs, which also means it gets *current* data rather than a snapshot from
when it was queued.

**The global lock is across processes, not goroutines.** It is not a mutex. It exists for
"only one of our five servers should do this".

**Cached values must tolerate being absent.** A cache can be cold, cleared, or disabled. Code that
requires a cache hit to be correct is broken. The per-request cache is the right place for "load
this once while serving this page" — and being request-scoped, it cannot leak one user's data into
another's response.

**Search indexes are eventually consistent.** A new issue is in the database immediately and in the
index shortly after. So a list built from the index can lag, and an index must **never** be the
source of truth for a correctness decision. Use it for search; use the database for "does this
exist".

**Storage returns an interface, never a path.** Files may be on disk or in object storage,
depending on configuration. Code assuming a local filesystem path works in development and breaks in
production.

## How to actually do it

**Do work in the background.** Create a unique queue at startup whose handler takes ids, and push
from wherever the triggering change happens. Two good examples to copy:
`services/actions/job_emitter.go` and `services/pull/check.go`.

**Cache something expensive for one request.** Use the request-scoped cache, not a package-level
map. A package-level map is shared between every user on the server, which is a data-leak bug
waiting to happen.

**Make something happen only once across several servers.** Take the global lock by name, and
always release it through the returned function.

## Traps, and what they look like

**A queued item vanishes and the work never happens.** The handler panicked. Handle the error and
return instead — a panic loses the item.

**The server does the same work over and over under load.** A non-unique queue with a frequent
trigger. Use the unique one unless duplicates genuinely mean something.

**It works locally and behaves differently in production.** Redis-backed queues, caches and locks
are only active when configured. A local SQLite instance uses the simpler backends, so the code path
you tested is not the one that runs. Test the non-Redis path too — and remember that a persistent
queue survives a restart while an in-memory one does not.

**"The issue exists but does not show up in search."** Almost always indexing lag or a missing
reindex, not a query bug. Check the database first to confirm the row is there.

**Your file handling breaks on a real deployment.** Something assumed a local path instead of using
the storage interface.

## Where to go next

- `docs/modules-infra.md` (in this folder) — the reference page this was written from
- `modules-setting.md` — how each of these is configured
- `services-actions.md` and `services-pull-and-gitdiff.md` — the heaviest queue users, and good
  patterns to copy
