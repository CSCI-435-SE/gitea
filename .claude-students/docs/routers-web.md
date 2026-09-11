---
scope: routers/web, routers/common, modules/web
verified-at: c0092050a4
---

# routers/web — the HTML route table and page handlers

**Read when:** adding or changing a page, a form POST, or a guard on a web route.
**Not here:** the `ctx` value itself -> `services-context.md`; form structs -> `services-forms-and-convert.md`; JSON endpoints -> `routers-api-v1.md`.

## Responsibilities

- `routers/web/web.go` — `Routes` assembles the middleware chain; `registerWebRoutes` declares
  every HTML route in nested `m.Group` blocks.
- `routers/web/{repo,user,org,admin,auth,explore,feed,events,misc,healthcheck,devtest,shared}/` —
  the handlers themselves. `events` is the server-sent-events endpoint; `devtest` is the
  `/devtest` component gallery.
- `routers/common/` — middleware and helpers shared with the API layer.
- `modules/web/` — a thin chi wrapper: `NewRouter`, `BeforeRouting`, `AfterRouting`, `Mount`,
  `Bind`, `GetForm`.

## Key files

| Path | What it holds |
| --- | --- |
| `routers/web/web.go` | `Routes`, `registerWebRoutes`, `verifyAuthWithOptions` |
| `routers/web/repo/issue.go` | a representative handler set, and the `templates.TplName` constants |
| `routers/common/pagetmpl.go` | `PageGlobalData` — data every template can rely on |
| `routers/common/middleware.go` | `MustInitSessioner` |
| `routers/common/blockexpensive.go`, `qos.go` | `BlockExpensive`, `QoS` load shedding |
| `routers/common/errpage.go` | the panic/error page renderer |
| `modules/web/router.go` | `Bind`, `GetForm` |
| `services/context/context_response.go` | `HTML`, and the dev-only `ctx.Data["TemplateName"]` |
| `templates/base/footer_content.tmpl` | prints that template name in the page footer |

## Conventions & invariants

- A handler is `func Foo(ctx *context.Context)` from `gitea.dev/services/context`. It puts data in
  `ctx.Data`, then calls `ctx.HTML(http.StatusOK, tplFoo)`.
- `tplFoo` is a package-level `templates.TplName` constant whose value is a path under `templates/`
  without the extension — `tplIssues templates.TplName = "repo/issue/list"`.
- `ctx.Data` is the only channel from handler to template. A template cannot reach anything the
  handler did not put there.
- Sign-in guards are local variables built at the top of `registerWebRoutes` from
  `verifyAuthWithOptions`: `reqSignIn`, `reqSignOut`, `optSignIn`, `optExploreSignIn`. A new guard
  is declared there too, not in a subpackage.
- Repo pages get `context.RepoAssignment` (see `services-context.md`), which fills `ctx.Repo`;
  ref-scoped pages add `context.RepoRefByType`.
- A POST that takes a form is registered as
  `m.Post("/path", web.Bind(forms.XxxForm{}), handler)`; the handler reads it back with
  `web.GetForm(ctx)`.
- The middleware chain in `Routes` is, in order: `chi_middleware.GetHead`, optional gzip,
  `common.MustInitSessioner`, `context.Contexter`, the web auth middleware, `goGet`,
  `common.PageGlobalData`, then `common.BlockExpensive` and `common.QoS`.

## Recipes

**Add a page.**

1. Handler in the matching `routers/web/<area>/` file, signature `func(ctx *context.Context)`.
2. A `templates.TplName` constant beside the other constants in that file.
3. The template file at the matching path under `templates/`.
4. The route inside the right `m.Group` in `registerWebRoutes`, with the guards that group needs.
5. Any user-visible string as a key in `options/locale/locale_en-US.json` — never a literal.

**Add a form POST.** Do the above, plus a form struct in `services/forms/` and `web.Bind` on the
route. See `services-forms-and-convert.md`.

**Find the template behind a page you are looking at.** Outside production, `ctx.HTML` sets
`ctx.Data["TemplateName"]` and `templates/base/footer_content.tmpl` prints it in the page footer, so
a dev instance names its own template on every page. Failing that, grep the visible string in
`options/locale/locale_en-US.json` for its key, then grep that key under `templates/`. To go the
other way, from a template to its handler, grep for the `TplName` value — the template path without
the extension.

## Gotchas

- The static routes registered on `routes` *before* the `mid` slice is assembled (`/assets/*`,
  `/avatars/*`, `/favicon.ico`, `/api/healthz`) deliberately bypass session and auth middleware.
  Do not add an authenticated route there.
- `registerWebRoutes` is long. Find your group by URL prefix rather than reading top to bottom, and
  add the route to the group that already carries the guards you need instead of re-listing them.
- `ctx.HTML` after something has already written the response is a bug; `ctx.Written()` reports
  whether that happened.

## Related

- `services-context.md` — what `ctx` carries and the response helpers
- `routers-api-v1.md` — the JSON side of the same handlers
- `architecture.md` — where routing sits in the dependency direction
