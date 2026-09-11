---
scope: models/migrations
verified-at: c0092050a4
---

# models/migrations — versioned schema changes

**Read when:** you changed a persisted struct under `models/`, or a migration test fails.
**Not here:** the runtime engine -> `models-db.md`; running the suite -> `testing.md`.

## Responsibilities

An ordered, append-only list of schema migrations. Each has an integer id; the database records how
far it has got, and startup applies whatever is missing. `services/versioned_migration` is what
invokes this at boot.

## Key files

| Path | What it holds |
| --- | --- |
| `models/migrations/migrations.go` | the ordered slice, `newMigration`, `minDBVersion`, `Migrate`, `EnsureUpToDate`, `GetCurrentDBVersion`, `ExpectedDBVersion` |
| `models/migrations/v1_27/v342.go` | the newest migration — the shape to copy |
| `models/migrations/base/db.go` | `RecreateTable`, `RecreateTables`, `DropTableColumns`, `ModifyColumn` |
| `models/migrations/migrationtest/tests.go` | `PrepareTestEnv`, `MainTest` for migration tests |
| `models/migrations/fixtures/Test_<FuncName>/` | per-test fixture directory, named after the test |

## Conventions & invariants

- A migration is one `newMigration(<id>, "<description>", v1_27.FuncName)` entry **appended at the
  bottom** of the slice in `migrations.go`. Order is the schema history; never insert in the
  middle, never renumber.
- The implementation lives in the current version subpackage as `v<id>.go` — migration 342 is
  `models/migrations/v1_27/v342.go` — with the function exported and named for what it does.
- Signature is `func(x db.EngineMigration) error`, or with a leading `context.Context` when the
  migration needs one. `newMigration` accepts either.
- The database version is *the last migration's id + 1*. `minDBVersion` is 70 (Gitea 1.5.3):
  installs older than that must upgrade through an intermediate release first.
- **A migration declares its own local copy of every struct it touches, inside the function**, with
  only the columns it cares about. `v342.go` declares a local `ActionRun` holding just the three
  new columns. Never import the live model from `models/` — the model keeps evolving, and a
  migration must describe the schema *as of its own moment* forever.
- Partial table changes use `x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true,
  IgnoreConstrains: true}, new(Foo))` rather than plain `Sync`, so unrelated indices and
  constraints survive (`docs/guidelines-backend.md`).
- New columns need `NOT NULL DEFAULT <value>` in the xorm tag, or existing rows cannot be migrated.
- New `.go` files carry a copyright header with the current year (`AGENTS.md`).

## Recipes

**Add a migration.**

1. Change the real struct in its `models/<domain>/` package.
2. Create `models/migrations/v1_27/v<next id>.go` with an exported function and a local struct copy
   holding only the affected columns.
3. Append `newMigration(<next id>, "<description>", v1_27.FuncName)` to the bottom of the slice in
   `migrations.go`.
4. Add `v<next id>_test.go` beside it, using `migrationtest.PrepareTestEnv`; put any fixtures in
   `models/migrations/fixtures/Test_<FuncName>/`.
5. `make test-migration`.

**Change a column type or drop columns.** Use `base.ModifyColumn` or `base.DropTableColumns` rather
than raw SQL — they handle the per-driver differences. `base.RecreateTable` is the last resort for
changes some drivers cannot express.

## Gotchas

- **Never edit a migration that has shipped.** Installs that already ran it will not run it again,
  so the edit silently applies to new installs only and the two diverge. Add a new migration.
- Registering a model with `db.RegisterModel` creates the table on a *fresh* install only. Existing
  installs need the migration — this is the most common way a change works locally and breaks on
  upgrade.
- Migration tests use their own harness (`migrationtest`), not `models/unittest`. The fixture
  directory name must match the test function name exactly or no fixtures load.
- `make test-migration` is a separate target from `make test-backend`; a migration can be broken
  while the backend suite is green.

## Related

- `models-db.md` — `db.RegisterModel` and the engine
- `testing.md` — the other four test tiers
