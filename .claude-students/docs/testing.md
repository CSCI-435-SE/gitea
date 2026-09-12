---
scope: docs/testing.md, models/unittest, models/fixtures, services/contexttest, tests
verified-at: c0092050a4
---

# testing — the five tiers and their harnesses

**Read when:** choosing what kind of test to write, or a test needs a database, a request context,
a git repo or a fixture.
**Not here:** lint and build targets -> `architecture.md` Gotchas.

## Responsibilities

| Tier | Lives in | Runner |
| --- | --- | --- |
| Go unit | `*_test.go` beside the code | `make test-backend` |
| Frontend unit | `*.test.ts` beside the source | `make test-frontend` |
| Integration | `tests/integration/` | `make test-integration` |
| E2E (browser) | `tests/e2e/*.test.ts` | `make test-e2e` |
| Migration | `models/migrations/` | `make test-migration` |

Prefer a unit test whenever the logic is testable in isolation (`AGENTS.md`). Integration and e2e
tests should stay under roughly 2s locally, so push setup-heavy cases down a tier where you can.
Fuzz targets live in `tests/fuzz/`.

## Key files

| Path | What it holds |
| --- | --- |
| `models/unittest/testdb.go` | `MainTest`, `TestOptions`, `PrepareTestDatabase` — the unit-test DB harness |
| `models/unittest/unit_tests.go` | `AssertExistsAndLoadBean`, and the other assertion helpers |
| `models/unittest/consistency.go` | `CheckConsistencyFor` — verifies counter columns match rows |
| `models/fixtures/*.yml` | 78 seed files, one per table (`issue.yml`, `repository.yml`, `user.yml`) |
| `services/contexttest/context_tests.go` | `MockContext`, `MockAPIContext`, `MockPrivateContext` for handler-level tests |
| `tests/test_utils.go` | `InitIntegrationTest`, `PrepareTestEnv`, `PrintCurrentTest`, the storage `Prepare*` helpers |
| `tests/integration/integration_test.go` | `TestMain` for the whole integration suite |
| `tests/gitea-repositories-meta/`, `tests/gitea-lfs-meta/`, `tests/testdata/` | pre-built git repos and LFS data |
| `tests/e2e/utils.ts` | shared Playwright helpers; config in `playwright.config.ts` |
| `web_src/js/vitest.setup.ts`, `web_src/js/utils/testhelper.ts` | Vitest setup and helpers; config in `vitest.config.ts` |
| `docs/testing.md` | authoritative test documentation; this doc summarises it |

## Conventions & invariants

- A Go package whose tests touch the database needs a `main_test.go` containing
  `func TestMain(m *testing.M) { unittest.MainTest(m) }`. Without it the test DB is never
  initialised.
- Every DB-backed test calls `unittest.PrepareTestDatabase()` first, which reloads the fixtures, so
  tests do not depend on each other's writes.
- Adding a row to `models/fixtures/*.yml` means keeping denormalised counter columns
  (`num_issues`, `num_closed_issues`, ...) consistent, or `CheckConsistencyFor` fails.
- Handler tests use `contexttest.MockContext` rather than constructing a `context.Context`
  by hand — see `services/context` for what the real one carries.
- Integration tests need Git LFS installed. The database comes from `GITEA_TEST_DATABASE`; empty
  defaults to SQLite, which is what you want locally.
- Frontend tests are Vitest, not Jest. Playwright is only for the e2e tier.

## Recipes

**Run one test.**

```sh
go test -run '^TestName$' ./modulepath/   # or: make test-backend#TestName
pnpm exec vitest <path-filter>
GITEA_TEST_E2E_FLAGS='<filepath>' make test-e2e
```

**Add a DB-backed unit test.** Put `*_test.go` beside the code; add `main_test.go` with
`unittest.MainTest(m)` if the package has none; call `unittest.PrepareTestDatabase()`; load rows
with `unittest.AssertExistsAndLoadBean`; add fixture rows only if no existing row fits.

**Add an integration test.** New `tests/integration/<feature>_test.go`; `defer tests.PrepareTestEnv(t)()`
at the top of the test; drive it through HTTP the way the neighbouring tests do. Do not add a
`TestMain` — `integration_test.go` already owns it.

**Add a migration test.** Alongside the migration in `models/migrations/`, with its own fixtures
under `models/migrations/fixtures/`; helpers in `models/migrations/migrationtest/`.

## Gotchas

- `GITEA_TEST_LOG_SQL=1` logs every SQL statement — the fastest way to see what a failing DB test
  actually queried.
- `GITEA_TEST_E2E_TIMEOUT_FACTOR` multiplies e2e timeouts (default 4 on CI, 1 locally); a test that
  only passes with a raised factor is a flaky test, not a slow machine.
- A handful of Go unit tests fail on native Windows for platform reasons (symlinks, temp-file
  cleanup). Run the suite on Linux, macOS, WSL2 or the `.devcontainer` (`STUDENTS.md` §6).
- `tests/integration` and `tests/e2e` share one Gitea instance per run, so a test that leaves state
  behind breaks later tests rather than itself.

## Related

- `architecture.md` — which layer the code under test belongs to
- `docs/testing.md` — full documentation, including non-SQLite database setup
