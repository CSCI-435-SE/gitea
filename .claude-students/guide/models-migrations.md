---
source: docs/models-migrations.md
source-hash: 7c15a26d2e32bc1c
verified-at: bcda5126ec
---

<!-- Derived from docs/models-migrations.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Changing the database shape, in plain English

**In one sentence:** a numbered, append-only list of scripts that bring any existing Gitea database
up to the shape the current code expects.
**Come here when:** you changed a struct that is stored in the database, or a migration test is
failing.

## What this is

Every Gitea installation records how far through the list it has got. On startup, Gitea runs the
ones it has not run yet, in order. Your job when you change a stored struct is to add one entry to
the end of that list.

## Why it exists

Your development database is probably a few days old and you can delete it whenever you like. Real
installations cannot. Somebody is running Gitea with five years of issues in it, and they are about
to upgrade to the version containing your change.

If your code expects a column that their database does not have, Gitea breaks on startup for them.
A migration is the script that adds it.

This also explains the rule people find strangest: **you must never edit a migration that has
already shipped.** Installations that ran it will not run it again. Editing it changes what *new*
installations get, so the two slowly diverge until they are running provably different schemas from
identical code. If a shipped migration is wrong, the fix is another migration.

## Words you'll meet

- **migration** — one script that changes the database shape.
- **schema** — the shape itself: tables, columns, indexes.
- **migration id** — its number. Also its position in history.
- **database version** — how far an installation has got.
- **column tag** — the backtick annotation on a struct field describing its column.
- **`NOT NULL DEFAULT`** — "this column always has a value, and here is the one existing rows get".
- **nullable / NULL** — a column allowed to hold NULL, meaning "no value at all". NULL is not 0 and
  not an empty string.
- **registered model** — a struct handed to `db.RegisterModel`, which is how Gitea's tools know its
  table exists.
- **index** — an extra structure making lookups fast.
- **constraint** — a rule the database enforces, like uniqueness.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/migrations/migrations.go` | The list itself, and the functions that run it. |
| `models/migrations/v1_27/v343.go` | The smallest complete migration that adds columns — the shape to copy. |
| `models/migrations/base/db.go` | Helpers for awkward changes: modifying a column, dropping columns, recreating a table. |
| `models/migrations/migrationtest/tests.go` | The test harness for migrations, separate from the normal one. |
| `models/migrations/fixtures/Test_<FuncName>/` | Fixtures for one migration test, in a directory named after the test function. |

## The rules, and why

**Append to the bottom of the list. Never insert, never renumber.** The list is history. Inserting
in the middle means an installation that already passed that point never runs your migration.

**The implementation is a file named after the id**, in the current version's subfolder — migration
342 is `models/migrations/v1_27/v342.go`, with an exported function named for what it does.

**The function takes the migration engine**, optionally with a `context.Context` first. Both shapes
are accepted.

**The database version is the last migration's id plus one.** There is a floor, too: installations
older than Gitea 1.5.3 must upgrade through an intermediate release first, so migrations do not have
to cope with truly ancient schemas.

**A migration declares its own private copy of every struct it touches.** This is the rule that
surprises everyone, and it is the most important one here.

Inside the migration function you re-declare the struct — locally, with only the columns this
migration cares about. You do **not** import the real model from `models/`.

Why: the real model keeps changing. In six months it will have three more columns and a renamed
field. But this migration must keep doing, forever, exactly what it did the day it was written —
because installations run it at whatever version they upgrade from. Importing the live model means
your migration's behaviour silently changes every time somebody edits that struct, and it will
eventually try to add a column that a later migration already added.

A migration is a photograph of one moment, not a window onto the present.

**Partial table changes use the options-taking sync**, with `IgnoreDropIndices` and
`IgnoreConstrains` set, rather than a plain sync. A plain sync makes the table match your local
struct — and since your local struct only lists the columns you care about, it would helpfully
delete the indexes and constraints you left out.

**A column added to an existing table takes `NOT NULL DEFAULT <zero value>`.** Existing rows already
exist, and they need *some* value in the new column. The default gives them one, in the same step
that adds the column — `models/migrations/v1_27/v343.go` does this for three columns at once.

Leave the default off and one of two things happens. With `NOT NULL` still there, the old rows have
no legal value, and the migration fails on any database with data in it — which is every real one.
Without `NOT NULL`, the column is nullable and the old rows get NULL
(`models/migrations/v1_27/v332.go` adds a column like that). NULL is not zero, so a query asking for
`col = 0` silently skips every one of those old rows.

**New `.go` files need a copyright header with the current year.**

## How to actually do it

**Add a migration.**

1. Change the real struct in its `models/<domain>/` package.
2. Create `models/migrations/v1_27/v<next id>.go` with an exported function and a **local copy** of
   the struct, listing only the affected columns.
3. Append `newMigration(<next id>, "<description>", v1_27.FuncName)` to the bottom of the list in
   `models/migrations/migrations.go`.
4. Add `v<next id>_test.go` beside it using `migrationtest.PrepareTestEnv`, with any fixtures in
   `models/migrations/fixtures/Test_<FuncName>/`.
5. Run `make test-migration`.

**Change a column's type, or drop columns.** Use `base.ModifyColumn` or `base.DropTableColumns`
rather than writing SQL — different databases express these differently and the helpers handle it.
`base.RecreateTable` is the last resort for changes some databases cannot express at all.

## Traps, and what they look like

**It works on your machine and breaks for everyone else on upgrade.** You added the field and
registered the model, but wrote no migration. Your database got the column because you created it
fresh; theirs did not. This is the single most common way a database change goes wrong, and you
cannot catch it by testing locally — your database already has the column.

**The migration fails on a database with data in it, but passes on an empty one.** Your new column
is `NOT NULL` with no `DEFAULT`, so existing rows have nothing legal to put in it.

**Your migration test finds no fixtures.** There are two silent causes. The fixture directory name
must exactly match the test function name. And the migration harness loads fixtures through
`models/unittest/fixtures_loader.go`, which only fills tables whose model is registered and skips
every other file without a word. A migration test package imports very few models, so yours may not
be registered. Blank-import the package that owns the table, and check how many rows you read back,
so a skipped fixture fails the test instead of letting it pass by checking nothing.
`models/migrations/v1_27/v343_test.go` does both.

**`make test-backend` is green but the migration is broken.** Migrations have their own target,
`make test-migration`, and their own harness. Passing the backend suite says nothing about them.

**You spot a bug in a migration that has already shipped and fix it in place.** Do not. Installations
that ran it will not run it again. Write a new migration that corrects the result.

## Where to go next

- `docs/models-migrations.md` (in this folder) — the reference page this was written from
- `models-db.md` — registering a model and the engine
- `testing.md` — the other four test suites
