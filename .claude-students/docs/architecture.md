---
scope: main.go, docs/guidelines-backend.md, docs/guidelines-frontend.md, docs/guidelines-refactoring.md
verified-at: c0092050a4
---

# architecture — layers, dependency direction, request lifecycle

**Read when:** you are unsure which layer a change belongs in, which package to import, or how a
request reaches your handler.
**Not here:** per-package detail -> the doc for that path in `INDEX.md`; test tiers -> `testing.md`.

## Responsibilities

| Layer | Owns |
| --- | --- |
| `cmd/` | CLI subcommands (`web`, `serv`, `hook`, `doctor`, `admin *`) |
| `routers/` | HTTP handlers, split into `web`, `api`, `private`, `install`, plus shared `common` |
| `services/` | business logic tying routers to models; also the request context and form binding |
| `models/` | XORM beans and database queries; keeps external dependencies to a minimum |
| `modules/` | standalone leaf utilities with few dependencies |
| `templates/` | Go `html/template` sources for server-rendered pages and mail |
| `web_src/` | frontend sources (TypeScript, Vue, CSS), built by vite into `public/assets/` |
| `options/` | shipped data: locales, gitignores, licenses, labels, readmes |

## Key files

| Path | What it holds |
| --- | --- |
| `main.go` | binary entry point; wires the `cmd` subcommands |
| `routers/init.go` | `NormalRoutes` — mounts `/`, `/api/v1`, `/api/internal`, `/api/packages`, `/api/actions` |
| `routers/web/web.go` | `Routes` (middleware chain) and `registerWebRoutes` (the whole web route table) |
| `routers/api/v1/api.go` | `Routes` plus every API guard middleware |
| `services/context/context.go` | `Context`, the value every web handler receives |
| `modules/reqctx/datastore.go` | request-scoped key/value store underneath the context |
| `docs/guidelines-backend.md` | authoritative backend rules; this doc only summarises them |
| `docs/guidelines-refactoring.md` | process rules for when a refactor needs its own PR |

## Conventions & invariants

Dependencies flow one way only (`docs/guidelines-backend.md`):

```text
cmd → routers → services → models → modules
```

A package on the left may import a package on its right, but never the reverse. A backwards import
is the most common structural review rejection — move the code rather than adding the import.

- Top-level packages are plural (`services`, `models`, `routers`); subpackages are singular
  (`services/user`, `models/repo`).
- Cross-layer name collisions take a snake_case import alias: `issues_model`, `user_service`,
  `access_model`, and `api "gitea.dev/modules/structs"`. Match the aliases already used in the file
  you are editing.
- The module path in this fork is `gitea.dev`, **not** `code.gitea.io/gitea`. Copy import paths
  from a neighbouring file, not from upstream docs or search results.
- There is no `modules/context` in this tree. The request context lives in `services/context`
  because it has to depend on models and services.
- A function that participates in a transaction takes `context.Context` as its first parameter, so
  the transaction propagates. Never pass a `*xorm.Session` as a parameter.

## Recipes

**Decide where new code goes.** Does it read or write the database? `models/`. Does it coordinate
several model calls, or fan out to mail / notifications / webhooks? `services/`. Does it read the
request or write the response? `routers/`. Is it a pure helper with no Gitea dependencies?
`modules/`.

**Trace a web request.** `routers/init.go` (`NormalRoutes`) → `routers/web/web.go` (`Routes` builds
the middleware chain; `registerWebRoutes` matches the path) → a handler in
`routers/web/{repo,user,org,admin,auth,explore}/` with signature `func(ctx *context.Context)` →
services → models → the handler fills `ctx.Data[...]` and calls `ctx.HTML(http.StatusOK, tplFoo)`,
where `tplFoo` is a package-level `templates.TplName` naming a file under `templates/`.

**Trace an API request.** The same until `routers/api/v1/api.go`, then a handler
`func(ctx *context.APIContext)` that converts models to a `modules/structs` type and returns it
with `ctx.JSON`.

## Gotchas

- Run `make fmt` before committing, `make lint-go` / `make lint-js` before pushing, and `make tidy`
  after any `go.mod` change (`AGENTS.md`). `make help` lists every target.
- Changing a struct under `models/` that is persisted to the database almost always requires a new
  migration in `models/migrations/`.
- `templates/` and `web_src/` sit outside the dependency rule, but a template may only use data the
  handler actually put in `ctx.Data`.
- `docs/guidelines-refactoring.md` limits refactor scope: fix the root cause, keep the PR tight,
  and split large refactors across PRs.

## Related

- `testing.md` — which test tier to write and how to set it up
- `docs/guidelines-backend.md`, `docs/guidelines-frontend.md` — the authoritative rules
