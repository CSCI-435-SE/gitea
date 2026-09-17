---
source: docs/routers-web.md
source-hash: eff7c884d160466a
verified-at: 69d66d604f
---

<!-- Derived from docs/routers-web.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Web pages and how URLs reach them, in plain English

**In one sentence:** every HTML page in Gitea is a URL listed in one big file, pointing at a Go
function that fills a bag of data and names a template.
**Come here when:** you are adding or changing a page, handling a form submission, or trying to find
which code produces something you can see in the browser.

## What this is

Three things working together:

- `routers/web/web.go` — the complete list of web URLs. It is long, but it is the index of the
  entire user interface.
- `routers/web/<area>/` — the handler functions themselves, grouped by area: `repo`, `user`, `org`,
  `admin`, `auth`, `explore`, and a few smaller ones.
- `routers/common/` — middleware shared with the JSON API.

Underneath sits `modules/web`, a thin wrapper over the router library, which gives you `Bind` and
`GetForm` for form handling.

## Why it exists

A web server has to answer one question very fast: *this URL just arrived — what code runs?*

Gitea answers it by declaring every route in one place, in nested groups. That matters because a
route is never just a path: it also carries **guards**, the checks that run first. A repository page
must load the repository and check you are allowed to see it, before the handler runs at all.

Grouping routes means those guards are declared once for a whole section rather than repeated on
every line — and it means that when you add a route to the right group, you inherit the correct
security checks for free. Add it to the wrong group and you have quietly built a page that skips
them.

## Words you'll meet

- **route** — a URL pattern plus the function that answers it.
- **handler** — that function. In this layer its signature is always
  `func Foo(ctx *context.Context)`.
- **middleware** — code that runs before the handler: starting a session, signing the user in,
  loading the repo, shedding load when the server is busy.
- **guard** — middleware whose job is to refuse. `reqSignIn` rejects anonymous visitors.
- **group** — a block of routes sharing a path prefix and a set of guards.
- **`ctx.Data`** — a map the handler fills and the template reads. The only channel between them.
- **`TplName`** — a template's name: its path under `templates/` with the extension removed.
- **bind** — parsing a submitted form into a Go struct before the handler runs.

## What's in these files

| Where | What it is for |
| --- | --- |
| `routers/web/web.go` | Two functions matter. `Routes` assembles the middleware chain; `registerWebRoutes` declares every URL. |
| `routers/web/repo/issue.go` | A good example to copy from: real handlers and the template-name constants beside them. |
| `routers/common/pagetmpl.go` | `PageGlobalData` — the values every template can rely on being present. |
| `routers/common/middleware.go` | Session setup. |
| `routers/common/blockexpensive.go`, `qos.go` | Load shedding, so expensive pages cannot take the site down. |
| `routers/common/errpage.go` | Renders the error page when a handler panics. |
| `modules/web/router.go` | `Bind` and `GetForm`. |
| `services/context/context_response.go` | `ctx.HTML`, and the dev-only trick described below. |
| `templates/base/footer_content.tmpl` | Prints the template name in the footer, outside production. |

The subfolders of `routers/web/` are worth knowing: `repo`, `user`, `org`, `admin`, `auth`,
`explore`, `feed`, `events` (server-sent events), `misc`, `healthcheck`, `devtest` and `shared`.

## The rules, and why

**A handler always looks the same.** It takes `ctx`, puts values in `ctx.Data`, and finishes with
`ctx.HTML(http.StatusOK, tplFoo)`. If yours does something else, check it against a neighbour.

**Except when the page itself is JavaScript-driven.** A handler backing a widget that fetches its
own data — an autocomplete list, a live search panel — skips the template and calls `ctx.JSON`
instead, the same way an API handler would. `routers/web/repo/issue_suggestions.go` is one.
This is still `routers/web`, not the `/api/v1` layer: the difference is who calls it (this page's
own JavaScript, not an external API client), not where the code lives.

**Template names are constants, not strings.** `tplIssues templates.TplName = "repo/issue/list"`
names the file at that path under `templates/`, with `.tmpl` added. Declaring it as a constant
beside the other constants in the file is what makes it findable later — it is the searchable link
between a template and the handler that renders it.

**`ctx.Data` is the only way to pass data to a template.** A template cannot reach into your handler
or call the database. If a value is not in `ctx.Data`, the template renders nothing for it — and
renders nothing *silently*, with no error.

**Sign-in guards are declared at the top of `registerWebRoutes`**, built by `verifyAuthWithOptions`:
`reqSignIn`, `reqSignOut`, `optSignIn`, `optExploreSignIn`. A new guard belongs there with them, not
buried in a subpackage where the next person will not find it.

**Repository pages need `context.RepoAssignment`.** That middleware is what loads the repo into
`ctx.Repo` and works out the visitor's permissions. Pages scoped to a branch or tag add
`context.RepoRefByType`. Without them `ctx.Repo` is empty and your handler will panic on a nil
pointer.

**A form POST is registered with its form type**, as
`m.Post("/path", web.Bind(forms.XxxForm{}), handler)`, and the handler reads it back with
`web.GetForm(ctx)`.

**The middleware order in `Routes` is deliberate**: request-method handling, optional compression,
session, context creation, authentication, then page data, then load shedding. Authentication has to
come after the session exists; page data has to come after authentication, because it includes who
you are.

## How to actually do it

**Add a page.**

1. Write the handler in the matching `routers/web/<area>/` file, signature
   `func(ctx *context.Context)`.
2. Add a `templates.TplName` constant beside the other constants in that file.
3. Create the template at the matching path under `templates/`.
4. Register the route inside the right group in `registerWebRoutes` — the group whose guards your
   page needs.
5. Put every user-visible string in `options/locale/locale_en-US.json` as a key. Never a literal.

**Add a form POST.** All of the above, plus a form struct in `services/forms/` and `web.Bind` on the
route.

**Find the code behind something you can see in the browser.** Three techniques, best first:

1. **Run in dev mode and read the page footer.** Outside production, `ctx.HTML` records
   `ctx.Data["TemplateName"]` and `templates/base/footer_content.tmpl` prints it. Every page tells
   you its own template.
2. **Search the visible text.** Grep the string in `options/locale/locale_en-US.json` to get its
   key, then grep that key under `templates/`. Pick the longest, most distinctive phrase — a common
   word matches many keys.
3. **Go from template to handler.** Grep for the `TplName` value — the template path without the
   extension — and you land on the constant, which sits beside the handler that uses it.

## Traps, and what they look like

**Your page shows a blank where a value should be.** The handler did not put that key in
`ctx.Data`, or spelled it differently. Templates do not complain about missing keys; they render
nothing.

**Your handler panics with a nil pointer on `ctx.Repo`.** The route is not in a group that runs
`context.RepoAssignment`, so no repository was ever loaded.

**Your new page is reachable without signing in.** You added the route to a group with weaker
guards than you assumed. Check which group you are inside — the guards come from the group, not
from the line you wrote.

**You cannot find where a route is declared** because `registerWebRoutes` is enormous. Do not read
it top to bottom. Search for a distinctive static piece of the URL, or for the handler's name. The
path parameters are placeholders like `{username}`, so grepping the literal URL never works.

**`ctx.HTML` after something already wrote the response.** That is a bug; `ctx.Written()` tells you
whether it happened. Return immediately after anything that writes a response.

One more piece of context: the static routes registered before the middleware chain is assembled —
assets, avatars, the favicon, the health check — deliberately skip session and authentication. Do
not add an authenticated route among them.

## Where to go next

- `docs/routers-web.md` (in this folder) — the reference page this was written from
- `services-context.md` — what `ctx` actually carries
- `services-forms-and-convert.md` — form structs and validation
- `templates.md` — writing the template your handler renders
