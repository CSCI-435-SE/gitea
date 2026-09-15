---
scope: services/context, modules/reqctx
verified-at: c0092050a4
---

# services/context — the request context every handler receives

**Read when:** you need something off `ctx`, need to fail a request correctly, or a handler needs
repo/org/user assignment.
**Not here:** route registration -> `routers-web.md` and `routers-api-v1.md`; mocking it in tests -> `testing.md`.

## Responsibilities

Three context types, all built on the shared `Base`:

| Type | File | Used by |
| --- | --- | --- |
| `Context` | `services/context/context.go` | web/HTML handlers |
| `APIContext` | `services/context/api.go` | `/api/v1` handlers |
| `PrivateContext` | `services/context/private.go` | the internal API `routers/private` |

It also owns the assignment middleware that resolves the URL into loaded models
(`ctx.Repo`, `ctx.Org`, `ctx.Package`, `ctx.ContextUser`) and the permission checks over them.

## Key files

| Path | What it holds |
| --- | --- |
| `services/context/context.go` | `Context`, `Contexter` (the middleware that creates it), `JSONError` |
| `services/context/base.go` | `Base` — `JSON`, `HTTPError`, `PlainText`, `Redirect`, `SetTotalCountHeader`, `Tr`, `TrN`, `Written` |
| `services/context/context_response.go` | `HTML`, `NotFound`, `ServerError`, `NotFoundOrServerError`, `RedirectToCurrentSite` |
| `services/context/repo.go` | `Repository` struct, `RepoAssignment`, `RepoRefByType`, `RepoRefByDefaultBranch` |
| `services/context/permission.go` | `RequireRepoAdmin`, `RequireUnitWriter`, `RequireUnitReader`, `CanWriteToBranch`, token-scope checks |
| `services/context/pagination.go` | `NewPagination` |
| `services/context/org.go`, `user.go`, `package.go` | the other assignment middlewares |
| `services/context/upload/` | multipart upload handling |
| `modules/reqctx/datastore.go` | the request-scoped key/value store `ctx.Data` is built on |

## Conventions & invariants

- There is **no `modules/context`** in this tree. The context lives in `services/` because it
  depends on models and services. Import paths copied from upstream Gitea docs will be wrong.
- `ctx.Data` (type `reqctx.ContextData`) carries values to templates. `ctx.PageData` is separate:
  it becomes `window.config.pageData` for frontend modules.
- `ctx.Doer` is the signed-in user; `ctx.ContextUser` is the user being *viewed*. They differ on
  most profile and org pages — conflating them is a recurring permission bug.
- `ctx.Repo` is a `context.Repository`, not a `repo_model.Repository`. The model is
  `ctx.Repo.Repository`; permissions are `ctx.Repo.Permission`; the opened git repo is
  `ctx.Repo.GitRepo`. `RefFullName`, `BranchName`, `TreePath`, `Commit` and `CommitID` are only
  populated when a `RepoRef*` middleware ran.
- Fail a request through the helpers, never by writing the response yourself: `ctx.NotFound`,
  `ctx.ServerError`, `ctx.NotFoundOrServerError` (which picks between them from an error check),
  `ctx.HTTPError` for a bare status. They log and render consistently.
- Anything a handler puts on `ctx` lives for one request only.

## Recipes

**Get the current user and repo.** `ctx.Doer` and `ctx.Repo.Repository`, but only after the route
carries the matching guard — `optSignIn`/`reqSignIn` for the doer (`routers-web.md`),
`context.RepoAssignment` for the repo.

**Return an error correctly.** A missing row is `ctx.NotFound(err)`. An unexpected failure is
`ctx.ServerError("what failed", err)`. When the same error could be either, pass the model's
`IsErrXxxNotExist` to `ctx.NotFoundOrServerError`.

**Paginate a list page.** Build a `context.NewPagination(total, pageSize, page, numPages)`, put it
in `ctx.Data["Page"]` (as `routers/web/explore/repo.go` does), and include
`templates/base/paginate.tmpl` from your template.

## Gotchas

- Adding a field to `Context` costs memory on every request and is almost never right — put it in
  `ctx.Data` instead.
- `ctx.Tr` returns `template.HTML`, so it is already escaped. Do not escape it again.
- After any helper that writes the response, return immediately. `ctx.Written()` exists for the
  cases where you cannot tell.

## Related

- `routers-web.md`, `routers-api-v1.md` — where the assignment middleware is attached
- `testing.md` — `contexttest.MockContext` for handler-level tests
