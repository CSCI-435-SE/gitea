---
source: docs/testing.md
source-hash: 6395a51cf7c96def
verified-at: c0092050a4
---

<!-- Derived from docs/testing.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Testing, in plain English

**In one sentence:** Gitea has five kinds of test, they cost wildly different amounts to run, and
picking the cheapest one that can prove your change works is most of the skill.
**Come here when:** you have written something and need to prove it works, or a test is failing and
you do not understand the error.

## What this is

Five separate test suites, each with its own command:

| Kind | Where the files live | How to run it |
| --- | --- | --- |
| Go unit | `*_test.go` next to the code it tests | `make test-backend` |
| Frontend unit | `*.test.ts` next to the source | `make test-frontend` |
| Integration | `tests/integration/` | `make test-integration` |
| Browser (e2e) | `tests/e2e/*.test.ts` | `make test-e2e` |
| Migration | `models/migrations/` | `make test-migration` |

There are also fuzz targets in `tests/fuzz/`, which you are unlikely to touch.

## Why it exists

The tiers exist because they trade speed for realism, and you want the fastest one that still
proves your point.

A **unit test** calls your function directly. It runs in milliseconds and tells you exactly which
function is wrong. An **integration test** starts a real Gitea with a real database and makes real
HTTP requests; it proves the whole path works but takes seconds and, when it fails, only tells you
that *something* in that path is wrong. A **browser test** drives an actual browser, which is the
only way to test JavaScript behaviour and the slowest thing here by far.

So the guidance is: prefer a unit test whenever the logic can be tested on its own. Use an
integration test when the point *is* that the pieces fit together. Keep integration and browser
tests under roughly two seconds each — if one needs heavy setup, that is a sign the logic underneath
it should have been unit-tested instead.

## Words you'll meet

- **fixture** — fake rows loaded into the test database before a test runs, so there is something
  to query. They live in `models/fixtures/` as YAML, one file per table.
- **harness** — the setup code that builds a test database, a fake request, or a temporary git repo
  so your test does not have to.
- **`TestMain`** — a special Go function that runs once before all the tests in a package. Gitea
  uses it to create the test database.
- **mock** — a stand-in for something real. A mock request context looks enough like a real one to
  call a handler with.
- **counter column** — a number stored on a row that must match a count of other rows, like a
  repository's issue count. Fixtures have to keep these honest.
- **flaky test** — one that passes and fails on the same code. Always a real bug, never bad luck.
- **Vitest / Playwright** — the frontend test runner, and the browser automation tool.

## What's in these files

**For Go unit tests that touch the database:**

- `models/unittest/testdb.go` — builds the test database. `MainTest` is the one you call from
  `TestMain`; `PrepareTestDatabase` resets it before each test.
- `models/unittest/unit_tests.go` — assertion helpers. `AssertExistsAndLoadBean` is the workhorse:
  it fetches a row and fails the test if it is missing.
- `models/unittest/consistency.go` — `CheckConsistencyFor`, which verifies those counter columns.
- `models/fixtures/*.yml` — 78 files of seed data, one per table (`issue.yml`, `repository.yml`,
  `user.yml`).

**For testing a handler without a browser:** `services/contexttest/context_tests.go` gives you
`MockContext`, `MockAPIContext` and `MockPrivateContext`.

**For integration tests:** `tests/test_utils.go` has the setup helpers, chiefly `PrepareTestEnv`.
`tests/integration/integration_test.go` owns the suite-wide `TestMain`.
`tests/gitea-repositories-meta/`, `tests/gitea-lfs-meta/` and `tests/testdata/` are pre-built git
repositories so tests do not have to create them.

**For frontend and browser tests:** `web_src/js/vitest.setup.ts` and
`web_src/js/utils/testhelper.ts` for unit tests; `tests/e2e/utils.ts` for browser tests.

`docs/testing.md` is the project's own full documentation, including how to run against a database
other than SQLite.

## The rules, and why

**A package whose tests use the database needs a `main_test.go`.** It contains exactly:

```go
func TestMain(m *testing.M) { unittest.MainTest(m) }
```

Without it the test database is never created, and your tests fail with confusing connection or
nil errors rather than a clear "there is no database".

**Every database test starts with `unittest.PrepareTestDatabase()`.** This reloads the fixtures, so
each test starts from the same known state. Skip it and your test passes alone but fails when run
after a test that changed a row — the classic "works on my machine, fails in CI" bug.

**If you add a fixture row, keep the counters consistent.** Add an issue to `issue.yml` without
bumping the owning repository's issue count and `CheckConsistencyFor` fails, usually in a test you
did not touch.

**Test handlers with `contexttest.MockContext`, not a hand-built context.** The real context carries
the signed-in user, the current repository and its permissions; building one by hand gets it subtly
wrong.

**Integration tests need Git LFS installed.** The database comes from the `GITEA_TEST_DATABASE`
environment variable, and leaving it unset gives you SQLite, which is what you want locally.

**Frontend tests use Vitest, not Jest.** The APIs look similar enough that Jest examples copied from
the internet will half-work and then fail oddly. Playwright is only for browser tests.

## How to actually do it

**Run a single test** — do this constantly; running the whole suite to check one function wastes
minutes each time.

```sh
go test -run '^TestName$' ./modulepath/
make test-backend#TestName
pnpm exec vitest <path-filter>
GITEA_TEST_E2E_FLAGS='<filepath>' make test-e2e
```

**Write a Go unit test that needs the database.**

1. Create `yourthing_test.go` next to the code.
2. If the package has no `main_test.go`, add one with `unittest.MainTest(m)`.
3. Start the test with `unittest.PrepareTestDatabase()`.
4. Load rows with `unittest.AssertExistsAndLoadBean`.
5. Only add a fixture row if no existing one fits — reusing one is less to keep consistent.

**Write an integration test.**

1. Create `tests/integration/<feature>_test.go`.
2. First line of the test: `defer tests.PrepareTestEnv(t)()` — note it is called *and* deferred.
3. Drive it over HTTP the way the neighbouring tests do.
4. Do **not** add a `TestMain`. `integration_test.go` already has one for the whole suite.

**Write a migration test.** Put it next to the migration in `models/migrations/`, with its own
fixtures under `models/migrations/fixtures/` and helpers from `models/migrations/migrationtest/`.

## Traps, and what they look like

**A database test fails with something about a nil or closed connection.** The package is missing
its `main_test.go`, so no test database was ever built.

**Your test passes alone and fails in the full run.** Either it is missing
`unittest.PrepareTestDatabase()`, or an earlier test left state behind. Integration and browser
tests share one Gitea instance for the whole run, so a test that creates something and does not
clean up breaks a *later* test, not itself — the failure points at innocent code.

**`CheckConsistencyFor` fails in a test you did not write.** You added or changed a fixture row and
a counter column no longer matches.

**A browser test only passes when you raise `GITEA_TEST_E2E_TIMEOUT_FACTOR`.** That multiplier
defaults to 4 on CI and 1 locally. Needing more is a flaky test — usually waiting on the wrong
thing — not a slow laptop. Fix the wait, do not raise the number.

**A handful of Go tests fail on Windows** for symlink and temp-file reasons that are not your bug.
Run the suite on Linux, macOS, WSL2 or the provided `.devcontainer`.

**You cannot tell what a failing database test actually did.** Set `GITEA_TEST_LOG_SQL=1` and it
prints every SQL statement it ran. This is the fastest debugging tool here.

## Where to go next

- `docs/testing.md` (in this folder) — the reference page this was written from
- Gitea's own `docs/testing.md`, at the repository root — the project's full test documentation,
  including how to run against a database other than SQLite. Same filename, different file:
  everything in `.claude-students/` is ours, everything at the repository root is Gitea's.
- `architecture.md` — which layer the code you are testing belongs to
