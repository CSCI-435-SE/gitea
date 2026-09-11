---
source: docs/models-db.md
source-hash: bfa8b8439f8662bd
verified-at: c0092050a4
---

<!-- Derived from docs/models-db.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Talking to the database, in plain English

**In one sentence:** the one package that speaks SQL, so nothing else has to — you ask it for the
database connection, and it decides whether you get the normal one or the one inside your
transaction.
**Come here when:** you are writing any query, need several writes to succeed or fail together, or
are adding a new table.

## What this is

`models/db` is the only place in Gitea that talks to the database library directly. Every other
package under `models/` goes through it to get a connection, start a transaction, run a generic
lookup, or register a new table.

If you are writing a query, you are using this package, even if indirectly.

## Why it exists

Two reasons, and the second is the interesting one.

The obvious one: putting all the database machinery in one place means the rest of the code does
not repeat it, and swapping or upgrading the underlying library is one package's problem.

The real one: **transactions have to travel**. Suppose creating an issue writes three rows, and the
third fails. You need the first two undone — otherwise you have half an issue in the database
forever. That means all three writes must know they belong to the same transaction.

Gitea solves this by hiding the transaction inside `ctx`. Every database function takes `ctx` as its
first argument and asks `db.GetEngine(ctx)` for a connection. If that `ctx` came from a transaction,
it hands back the transaction; otherwise the normal connection. Nobody passes transaction objects
around, and any function that takes `ctx` automatically joins whatever transaction its caller
started.

That is elegant, and it has one sharp edge: **pass the wrong `ctx` and your write silently happens
outside the transaction.** No error, no warning. It just does not roll back.

## Words you'll meet

- **engine** — the database connection. `db.GetEngine(ctx)` gives you one.
- **transaction** — a group of writes that all succeed or all get undone.
- **roll back** — undoing a transaction's writes because something failed.
- **bean** — a Go struct mapping to a table. Gitea's database library calls them beans.
- **register a model** — telling the library that a struct is a table.
- **generic helper** — a function working for any bean type, written `db.GetByID[T]`.
- **atomic** — all-or-nothing.
- **`WHERE` clause** — the part of an SQL statement saying *which* rows. Leaving it off means all
  of them.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/db/context.go` | The core. `GetEngine`, `WithTx`, `WithTx2`, `Insert`, and the generic lookups. |
| `models/db/engine.go` | `RegisterModel`, plus table-level utilities and SQL logging. |
| `models/db/engine_init.go` | Building the connection at startup. |
| `models/db/list.go` | Pagination: `ListOptions`, `FindOptions`. |
| `models/db/index.go` | `GetNextResourceIndex` — per-parent numbering, which is how per-repo issue numbers work. |
| `models/db/iterate.go` | Streaming through a result set too large to hold in memory. |
| `models/db/error.go` | The database-level error types. |
| `models/db/install/` | Database setup during first-run installation. |

## The rules, and why

**Always take `ctx` as the first parameter, and get the engine from it.** Never keep an engine in a
struct field, and never accept a session object as a parameter. This is what makes transactions
propagate; break it and they stop working with no visible symptom.

**`db.GetEngine(ctx)` is the whole mechanism.** Inside a transaction it returns that transaction's
session; outside, the shared connection. So a helper that is handed the wrong `ctx` — a background
one, or one from an outer scope — quietly writes outside the transaction and cannot be rolled back.
When you read a function, checking which `ctx` it passes on is the single most useful thing to look
at.

**Writes that must succeed together go inside `db.WithTx`.** Use `db.WithTx2` when the block needs
to return a value as well as an error.

**Every stored struct registers itself.** In the file declaring it, an `init()` calls
`db.RegisterModel(new(Foo))`. 101 files under `models/` do this — copy whichever one you are next
to.

**Prefer the generic helpers** — `db.GetByID[T]`, `db.Exist[T]`, `db.DeleteByID[T]` — over
hand-written engine calls. They already apply the right conditions and error handling, and a
hand-rolled equivalent usually forgets one.

**Never update without a `WHERE`.** An update with no condition rewrites *every row in the table*.
This is the most destructive mistake available in this package, and the database will do it without
complaint.

**Fields tagged `xorm:"-"` are not columns.** They exist only in memory, filled in by Go code. They
are never read from or written to the database, so setting one and saving the struct changes
nothing.

## How to actually do it

**Make several writes atomic.**

```go
err := db.WithTx(ctx, func(ctx context.Context) error {
    if err := db.Insert(ctx, thing); err != nil {
        return err
    }
    return other(ctx) // must take ctx, or it escapes the transaction
})
```

Note the shadowing: the closure's parameter is also called `ctx`, deliberately. Inside the block you
want the *transaction's* `ctx`, and giving it the same name means you cannot accidentally use the
outer one. If you call a function that does not take `ctx`, its writes are outside the transaction.

**Add a table.** Declare the struct in the right `models/<domain>/` package, add
`db.RegisterModel(new(Foo))` to that file's `init()`, and then **write a migration**. Registration
alone creates the table on a *fresh* install only; existing databases need the migration.

**Number rows within a parent.** `db.GetNextResourceIndex(ctx, "<table>_index", groupID)`, called
inside the transaction that inserts the row. This is how issue numbers restart at 1 in each
repository.

## Traps, and what they look like

**Your transaction did not roll back.** Something inside it ran with the wrong `ctx`. Trace which
`ctx` each call receives — the bug is always that one function did not take it, or was handed an
outer one.

**A panic saying "using session context in an iterator would cause corrupted results".** That is
`contextSafetyCheck`, which runs outside production and in tests. It is reporting a real bug: a
transaction-bound connection used inside `Iterate` would give wrong results. Restructure the query
rather than trying to silence it.

**You cannot tell what SQL actually ran.** Set `GITEA_TEST_LOG_SQL=1` and every statement is
printed. This is also how you confirm a write really happened inside the transaction you intended.

**A bulk insert fails, but only sometimes or only on one database.** Batch inserts must respect
`db.MaxBatchInsertSize` for that bean; too large a slice in one call exceeds what some drivers
accept.

**Rows with IDs you chose yourself behave oddly.** Preset IDs need special handling on SQL Server
and PostgreSQL. Avoid them outside migrations and test fixtures.

**Your change works locally and breaks on upgrade.** You registered the model but wrote no
migration, so your development database has the table and nobody else's does.

## Where to go next

- `docs/models-db.md` (in this folder) — the reference page this was written from
- `models-migrations.md` — making a real database match your struct
- `models-issues.md` and `models-repo-and-git.md` — the packages that use all this
- `testing.md` — the test database and fixtures
