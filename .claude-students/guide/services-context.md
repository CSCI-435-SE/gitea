---
source: docs/services-context.md
source-hash: fb1e6f4c44a5e8f4
verified-at: c0092050a4
---

<!-- Derived from docs/services-context.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# `ctx`, the thing every handler is handed, in plain English

**In one sentence:** `ctx` is the request — who is asking, what repository they are looking at, what
they are allowed to do, and the bag of data the template will read.
**Come here when:** you need something off `ctx`, you need to fail a request properly, or your
handler crashed on something being nil.

## What this is

Every handler you write takes one argument, and this is it. There are three flavours, all built on
a shared base:

| Type | Used by |
| --- | --- |
| `Context` | web pages that render HTML |
| `APIContext` | `/api/v1` endpoints returning JSON |
| `PrivateContext` | the internal API that git hooks call |

The same package also owns the middleware that turns a URL into loaded objects — taking
`/gitea/gitea/issues` and producing an actual repository in `ctx.Repo`, with the current visitor's
permissions worked out.

## Why it exists

Every handler needs the same half-dozen things: who is signed in, which repo this is, whether they
may see it, how to render a page, how to fail properly. Without somewhere shared, every handler
would redo all of it — and each would get the permission check subtly wrong in its own way.

So the work happens once, in middleware, before your handler runs. By the time you are called, the
repository is loaded and the permissions are computed. Your handler gets to be about your feature.

That is also why this lives in `services/` rather than `modules/`: it has to reach the database to
load users and repositories, and `modules/` is not allowed to (see `architecture.md`).

## Words you'll meet

- **request context** — one object holding everything about the request currently being served.
  Thrown away when the response is sent.
- **doer** — the person *doing* the thing. The signed-in user.
- **context user** — the person being *looked at*. On someone else's profile page, that is them,
  not you.
- **assignment middleware** — code that reads the URL and loads the matching object into `ctx`.
- **permission** — the computed answer to "what may this user do in this repository", per feature.
- **helper** — a method on `ctx` that renders a standard response, like a 404 page.
- **pagination** — splitting a long list across pages, and the object describing that.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/context/context.go` | Defines `Context` and the middleware that creates it. |
| `services/context/base.go` | The shared basics: returning JSON, plain text, redirects, HTTP errors, translation, and whether a response has already been written. |
| `services/context/context_response.go` | How you finish a request: `HTML`, `NotFound`, `ServerError`, `NotFoundOrServerError`, `RedirectToCurrentSite`. |
| `services/context/repo.go` | The repository side: the `Repository` struct that becomes `ctx.Repo`, and `RepoAssignment`, which fills it. |
| `services/context/permission.go` | Permission guards: `RequireRepoAdmin`, `RequireUnitWriter`, `RequireUnitReader`, `CanWriteToBranch`. |
| `services/context/pagination.go` | `NewPagination`. |
| `services/context/org.go`, `user.go`, `package.go` | The same assignment idea for organisations, users and packages. |
| `services/context/upload/` | File upload handling. |
| `modules/reqctx/datastore.go` | The key/value store underneath `ctx.Data`. |

## The rules, and why

**There is no `modules/context` in this codebase.** It is `services/context`. Upstream Gitea moved
it, so any import path you copy from older documentation or an AI answer will be wrong. This is the
single most common broken import here.

**`ctx.Data` and `ctx.PageData` are different bags.** `ctx.Data` is read by templates. `ctx.PageData`
becomes `window.config.pageData` in the browser, for JavaScript. Putting something in the wrong one
means it silently does not arrive.

**`ctx.Doer` is not `ctx.ContextUser`.** The doer is signed in; the context user is being viewed. On
profile and organisation pages they are different people. Confusing them is a recurring source of
permission bugs — you end up checking whether the *viewed* user may do something.

**`ctx.Repo` is not the database row.** It is a wrapper. The row is `ctx.Repo.Repository`,
permissions are `ctx.Repo.Permission`, and the opened git repository is `ctx.Repo.GitRepo`. The
branch and commit fields — `RefFullName`, `BranchName`, `TreePath`, `Commit`, `CommitID` — are only
filled in when a `RepoRef*` middleware ran on that route. On a route without one they are empty,
and code that assumes otherwise breaks.

**Fail through the helpers, never by writing the response yourself.** `ctx.NotFound`,
`ctx.ServerError`, `ctx.NotFoundOrServerError`, `ctx.HTTPError`. They log consistently and render
the proper error page. A handler that writes its own 404 produces something that looks nothing like
the rest of Gitea and logs nothing.

**Everything on `ctx` lasts exactly one request.** It is not a cache and not a place to keep state
between requests.

## How to actually do it

**Get the current user and repository.** `ctx.Doer` and `ctx.Repo.Repository` — but only if the
route carries the matching middleware. `optSignIn` or `reqSignIn` for the doer,
`context.RepoAssignment` for the repository. Without them these are nil, which is the most common
handler panic.

**Fail correctly.**

- A row that does not exist → `ctx.NotFound(err)`
- Something genuinely went wrong → `ctx.ServerError("what failed", err)`
- Could be either, depending on the error → `ctx.NotFoundOrServerError`, passing the model's
  `IsErrXxxNotExist` so it can decide

Getting this right is what makes a missing issue show a 404 rather than a 500.

**Paginate a list.** Build `context.NewPagination(total, pageSize, page, numPages)`, put it in
`ctx.Data["Page"]` — `routers/web/explore/repo.go` does exactly this — and include
`templates/base/paginate.tmpl` in your template.

## Traps, and what they look like

**Nil pointer panic on `ctx.Repo` or `ctx.Doer`.** The route is missing its middleware. Look at
where the route is registered, not at your handler.

**Your permission check passes for the wrong person.** You used `ctx.ContextUser` where you meant
`ctx.Doer`, or the reverse. On your own profile page both are you, so it appears to work — and then
fails on somebody else's.

**Branch or commit fields are empty on a repo page.** That route has `RepoAssignment` but no
`RepoRef*` middleware, so nothing resolved which branch you are on.

**Your text renders as escaped HTML, showing the tags.** `ctx.Tr` already returns escaped HTML.
Escaping it again shows the markup to the user.

**Two responses get written and the output is garbled.** Something wrote a response and the handler
carried on. Always return immediately after a helper that writes; `ctx.Written()` tells you whether
one already did.

**You are tempted to add a field to `Context`.** That field then costs memory on every request in
the whole server, for a value one handler needs. Put it in `ctx.Data` instead. This is almost always
the right answer.

## Where to go next

- `docs/services-context.md` (in this folder) — the reference page this was written from
- `routers-web.md` and `routers-api-v1.md` — where the middleware gets attached to routes
- `testing.md` — how to build a fake `ctx` for a handler test
