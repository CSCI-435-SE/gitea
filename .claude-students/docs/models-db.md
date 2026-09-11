---
scope: models/db
verified-at: c0092050a4
---

# models/db — engine access, transactions and generic queries

**Read when:** writing any query, needing a transaction, or adding a table.
**Not here:** schema changes -> `models-migrations.md`; the test DB harness -> `testing.md`.

## Responsibilities

The only package that touches XORM directly. Everything else in `models/` goes through it for the
engine, transactions, generic CRUD, list pagination, per-group index allocation and bean
registration.

## Key files

| Path | What it holds |
| --- | --- |
| `models/db/context.go` | `GetEngine`, `TxContext`, `WithTx`, `WithTx2`, `Insert`, `Exec`, `InTransaction`, `contextSafetyCheck` |
| `models/db/context.go` (generics) | `Get[T]`, `GetByID[T]`, `Exist[T]`, `ExistByID[T]`, `Delete[T]`, `DeleteByID[T]`, `DeleteByIDs[T]` |
| `models/db/engine.go` | `RegisterModel`, `SyncAllTables`, `SetLogSQL`, `MaxBatchInsertSize` |
| `models/db/engine_init.go` | engine construction and startup wiring |
| `models/db/list.go` | `ListOptions`, `FindOptions`, `Paginator`, `SetSessionPagination` |
| `models/db/index.go` | `GetNextResourceIndex`, `SyncMaxResourceIndex`, `DeleteResourceIndex` |
| `models/db/iterate.go` | `Iterate` for streaming large result sets |
| `models/db/error.go` | the database-level error types |
| `models/db/install/` | installer-time database operations |

## Conventions & invariants

- **Always pass `context.Context` first** and get the engine with `db.GetEngine(ctx)`. Never store
  an `Engine` in a struct, and never accept a `*xorm.Session` as a parameter
  (`docs/guidelines-backend.md`).
- `db.GetEngine(ctx)` returns the *transaction session* when `ctx` came from `db.WithTx`, and the
  global engine otherwise. This is the whole mechanism — a helper called with the wrong `ctx`
  silently runs outside the transaction and cannot be rolled back.
- Operations that must roll back together go in `db.WithTx(ctx, func(ctx context.Context) error)`,
  or `db.WithTx2` when the closure must return a value.
- Every persisted struct registers itself in an `init()` with `db.RegisterModel(new(Foo))`, in the
  file that declares it. 101 files under `models/` do this; copy the neighbour.
- Prefer the generic helpers (`db.GetByID[T]`, `db.Exist[T]`, `db.DeleteByID[T]`) over hand-written
  engine calls — they carry the right conditions and error handling.
- **Never call `x.Update(exemplar)` without an explicit `WHERE`** — it updates every row
  (`docs/guidelines-backend.md`).
- Fields tagged `xorm:"-"` are in-memory only. They are never read from or written to the database.

## Recipes

**Run several writes atomically.**

```go
err := db.WithTx(ctx, func(ctx context.Context) error {
    if err := db.Insert(ctx, thing); err != nil {
        return err
    }
    return other(ctx) // must take ctx, or it escapes the transaction
})
```

**Add a table.** Declare the struct in the owning `models/<domain>/` package, add
`db.RegisterModel(new(Foo))` in that file's `init()`, then add a migration
(`models-migrations.md`) — registration alone does not create the table on an existing install.

**Allocate a per-parent sequence number.** `db.GetNextResourceIndex(ctx, "<table>_index", groupID)`
inside the transaction that inserts the row. This is how per-repo issue numbers work; see
`models-issues.md`.

## Gotchas

- `contextSafetyCheck` runs outside production (and in tests) and *panics* with "using session
  context in an iterator would cause corrupted results" when a session-bound engine is used inside
  `Iterate`. It is catching a real bug, not being fussy — restructure the query.
- `GITEA_TEST_LOG_SQL=1` (or `db.SetLogSQL`) prints every statement. Fastest way to see whether
  your query ran inside the transaction you expected.
- Batch inserts must respect `db.MaxBatchInsertSize` for the bean; a large slice inserted in one
  call fails on some drivers.
- Inserting rows with preset IDs needs `SET IDENTITY_INSERT` on MSSQL and a sequence update on
  PostgreSQL (`docs/guidelines-backend.md`). Avoid preset IDs outside migrations and fixtures.

## Related

- `models-migrations.md` — making the schema match your struct
- `models-issues.md`, `models-repo-and-git.md` — the packages that consume this one
- `testing.md` — `unittest.PrepareTestDatabase` and the fixture harness
