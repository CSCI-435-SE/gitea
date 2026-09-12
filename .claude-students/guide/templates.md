---
source: docs/templates.md
source-hash: adbcb2a65fa0f696
verified-at: c0092050a4
---

<!-- Derived from docs/templates.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Writing the HTML, in plain English

**In one sentence:** Gitea's pages are Go template files that can only see what the handler put in
`ctx.Data`, and every visible word in them is a translation key rather than English text.
**Come here when:** you are editing a page's markup, adding a template, or something on a page is
blank and you do not know why.

## What this is

The server-rendered HTML, plus the Go package that renders it and supplies the functions templates
can call.

The files divide by area. `base/` holds the page skeleton — the head, the navigation bar, the
footer, pagination, alerts. `shared/` holds partials used in several places. Then one folder per
area of the site: repository, user, organisation, admin, explore, projects, packages. Plus `mail/`
for emails, `status/` for error pages, `devtest/` for a component gallery, `custom/` for deployment
overrides, and `swagger/` for the generated API specification.

## Why it exists

Server-rendered HTML is Gitea's default for good reasons: the page arrives complete, it works before
any JavaScript runs, and search engines can read it.

The design has one central constraint. **A template cannot fetch anything.** It cannot query the
database, call a service, or make a request. It can only display what the handler already put in
`ctx.Data`.

That is a deliberate restriction, and it is what keeps pages fast and predictable: all the work
happens in the handler, in one place, where you can see it. A template that could query would let
someone accidentally write a loop that runs a query per row, and nobody would notice until the page
got slow.

The second constraint is translation. Gitea ships in 29 languages, so **no English text is ever
written in a template.** Every visible string is a key looked up at render time.

## Words you'll meet

- **template** — a file of HTML with placeholders, filled in per request.
- **partial** — a template included by others rather than rendered alone.
- **template function** — a function a template may call, like formatting a date.
- **`.` (dot)** — the current data. **Its meaning changes inside a loop**, where it becomes the
  current item.
- **`$`** — always the top-level data, so you can reach it from inside a loop.
- **locale key** — the dotted name standing in for a translated string.
- **pluralisation** — choosing between "1 issue" and "2 issues".
- **scoped template** — rendering one partial on its own rather than a whole page.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/templates/helper.go` | The list of functions templates can call. |
| `modules/templates/util_render.go`, `util_render_comment.go` | Rendering markup and comments. |
| `modules/templates/util_date.go`, `util_avatar.go`, `util_string.go`, `util_slice.go`, `util_dict.go` | The rest of those functions, grouped by kind. |
| `modules/templates/htmlrenderer.go` | The renderer, and the template-name type. |
| `modules/templates/scopedtmpl/` | Rendering a partial on its own. |
| `templates/base/head_script.tmpl` | Emits the global config object, including the data meant for JavaScript. |

## The rules, and why

**A template is named by its path without the extension**, declared as a constant in the handler.
That constant is the searchable link between the two — see `routers-web.md`.

**`ctx.Data` is the only channel.** Whatever the handler put there is available; nothing else is.
Inside a template, `ctx.RootData` is that same data.

**Data for JavaScript travels separately.** Anything the browser needs goes through the handler's
page data, which `base/head_script.tmpl` emits into a global config object. It is a different
channel from `ctx.Data` — put something in the wrong one and it silently does not arrive.

**Every user-visible string is a locale key.** Written `{{ctx.Locale.Tr "repo.issues.new"}}`, or the
plural form where a count is involved. Never write English directly. And keys are added **only** to
the English locale file — every other language comes from the translation service, so editing one by
hand is overwritten (`i18n.md`).

**Build pages by including partials**, and put anything shared in `shared/` rather than copying it
between areas. Two near-identical partials in two folders will drift apart, and the bug will be
fixed in one of them.

**New template functions get uppercase names.** There is a lowercase one, `dict`, for historical
reasons — the code says so at its definition. Do not copy the lowercase style.

**Mail templates have a smaller function set of their own.** A helper you add for pages is **not**
available in `templates/mail/`. They are rendered by a different function map.

**Class attributes are written as one readable string**, conditionals included (`frontend-css.md`).

**Hook up JavaScript with attributes, not inline script** — the declarative mechanism described in
`frontend-js.md`.

**`.tmpl` files are indented with tabs.**

## How to actually do it

**Add a page's template.** Create the `.tmpl` in the matching area folder, include the head and
footer partials, declare the template-name constant in the handler, and put every string in the
locale file as a key.

**Share markup between two pages.** Put the partial in `templates/shared/` and include it from both.
Resist copying.

**Add a template function.** Add it to the map in `modules/templates/helper.go` and implement it in
the matching `util_*.go`. But first ask whether the handler could prepare the value instead — **a
template function that touches the database runs once per render, and inside a loop, once per row.**
That is a performance bug that is very hard to spot from the template.

**Look at a component on its own.** Run Gitea in development mode and open `/devtest`, which is a
gallery of the standard components.

## Traps, and what they look like

**Something on the page is blank, and there is no error.** The handler did not put that key in
`ctx.Data`, or spelled it differently. Templates do not complain about missing values; they render
nothing. When something is unexpectedly empty, check the handler before the template.

**A raw key like `repo.issues.new` appears on the page.** That key is missing from the locale file.
It is visible — but easy to miss in a branch that rarely renders, so it can ship.

**Your value never reaches the JavaScript.** You put it in `ctx.Data` instead of the page data
channel. Templates can see the first; the browser gets the second.

**`.` is not what you expect.** Inside a `range` loop it is the current item, not the page. Use `$`
to reach the top level from inside a loop. This catches everyone once.

**Your new template function is undefined in an email.** Mail templates use a different, smaller
function map.

**You edit the generated API specification file.** `templates/swagger/v1_json.tmpl` is generated —
edit the swagger comments in the handlers and regenerate (`routers-api-v1.md`).

**You put course work in `templates/custom/`.** That folder is for site administrators to override
things at deployment. It is not a scratch area.

## Where to go next

- `docs/templates.md` (in this folder) — the reference page this was written from
- `routers-web.md` — the handler that fills `ctx.Data` and names your template
- `i18n.md` — adding the locale keys every string needs
- `frontend-css.md` and `frontend-js.md` — the classes and attributes you write here
