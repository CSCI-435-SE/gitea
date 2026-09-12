---
scope: modules/queue, modules/cache, modules/storage, modules/indexer, modules/globallock, modules/log, modules/graceful, modules/process
verified-at: c0092050a4
---

# modules/* infrastructure — queues, cache, storage, indexers, locks

**Read when:** work needs to happen asynchronously, be cached, be stored as a file, be searchable,
or be serialised across processes.
**Not here:** configuration -> `modules-setting.md`; the database -> `models-db.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `modules/queue` | persistent worker-pool queues (channel, leveldb, redis backends) |
| `modules/cache` | the key/value cache, including a request-scoped layer |
| `modules/storage` | blob storage for attachments, LFS, avatars, artifacts (local, minio, azure) |
| `modules/indexer` | code, issue and stats indexers (bleve, elasticsearch, meilisearch, db) |
| `modules/globallock` | cross-process locks (memory or redis) |
| `modules/log` | logging |
| `modules/graceful` | graceful startup, shutdown and restart |
| `modules/process` | the process manager behind long-running commands |

## Key files

| Path | What it holds |
| --- | --- |
| `modules/queue/manager.go` | `CreateSimpleQueue`, `CreateUniqueQueue` |
| `modules/queue/workergroup.go` | the worker pool behind `WorkerPoolQueue[T]` |
| `modules/queue/base_channel.go`, `base_levelqueue.go`, `base_redis.go` | the backends |
| `modules/cache/cache.go`, `string_cache.go`, `context.go` | the cache API and the request-scoped cache |
| `modules/globallock/globallock.go`, `locker.go` | `Lock` and the locker implementations |
| `modules/indexer/issues/`, `code/`, `stats/` | one subtree per index |
| `modules/storage/storage.go` | the storage interface and its backends |

## Conventions & invariants

- **Queues are how Gitea does background work.** `CreateSimpleQueue` and `CreateUniqueQueue` return
  a `WorkerPoolQueue[T]`; a *unique* queue deduplicates by item, which is why pushing the same PR
  id twice is free (`services-pull-and-gitdiff.md`) and why the Actions job emitter can enqueue
  freely (`services-actions.md`).
- A queue is created once at startup and its handler must be idempotent: items can be retried after
  a restart, and a persistent backend replays what was not acknowledged.
- **Queue payloads must survive serialisation** — push an id, not a loaded struct with unexported
  fields or pointers to models.
- `globallock.Lock` is for mutual exclusion *across processes*, not a `sync.Mutex`. It is what
  makes things like the versioned migration safe when several instances start at once.
- Cached values must tolerate being absent. The request-scoped cache in `modules/cache/context.go`
  lives for one request and is the right place for "load this once per request".
- Indexers are eventually consistent. A freshly created issue is in the database immediately and in
  the index shortly after, so a list built from the index can lag — never use an index as the
  source of truth for correctness.
- `modules/storage` returns an interface; never assume a local filesystem path, because minio and
  azure backends are supported.

## Recipes

**Do work in the background.** Create a unique queue at startup with a handler that takes ids, and
push from wherever the triggering change happens. Copy `services/actions/job_emitter.go` or
`services/pull/check.go`.

**Cache an expensive per-request value.** Use the request-scoped cache rather than a package-level
map, so it cannot leak between users.

**Serialise something across instances.** `globallock.Lock(ctx, "<name>")`, and always release
through the returned function.

## Gotchas

- A queue handler that panics loses the item; handle errors and return rather than panicking.
- A non-unique queue plus a frequent trigger produces unbounded duplicate work — pick
  `CreateUniqueQueue` unless duplicates are meaningful.
- Redis-backed queues, caches and locks are only active when configured; behaviour differs between
  a local SQLite dev instance and production, so test the non-redis path too.
- Search results come from the indexer, so "the issue exists but does not appear in search" is
  usually an indexing lag or a missing reindex, not a query bug.

## Related

- `modules-setting.md` — how each of these is configured
- `services-actions.md`, `services-pull-and-gitdiff.md` — the two heaviest queue users
