---
verified-at: c0092050a4
---

# Start here

You have a working Gitea (if not, `STUDENTS.md` gets you there) and now you have to change
something in a codebase with roughly 150 Go packages. This page is the on-ramp.

## Two layers of notes, and which one is for you

This folder has two sets of documentation covering exactly the same ground.

| | `docs/` | `guide/` |
| --- | --- | --- |
| Written for | an AI session | a person |
| Style | terse, every claim cites a file | explains, defines jargon, names symptoms |
| Length | ~80 lines | up to 200 |
| Status | **the source of truth** | a rendering of it |

**`guide/` is derived from `docs/`.** Every guide page is a plain-English rendering of one
reference page, and it stores a fingerprint of that page. Change the reference and forget to
update the guide, and `./.claude-students/check.sh` will tell you.

That has a consequence worth understanding: **if the guide and the reference disagree, the guide is
wrong.** Do not fix the guide. Fix the reference page and regenerate — `MAINTENANCE.md` rules 16-22
spell out how. A disagreement is a bug report, not an editing job.

Both layers have the same filenames, so `guide/models-db.md` explains `docs/models-db.md`.
`INDEX.md` is the map: find your task, read that page here first, then the reference page when you
want the detail.

## Reading order on day one

1. This page, to the end.
2. `guide/architecture.md` — where code lives and why, and how a web request actually flows.
3. `guide/testing.md` — because you will need to run one before you have written anything.
4. Then only the page for the thing you are touching. Do not read all 34.

## Two traps before you write any code

**The import paths are not upstream Gitea's.** This fork's Go module is `gitea.dev`, so imports
read `gitea.dev/services/context`, never `code.gitea.io/gitea/...`. Anything you copy from the
official Gitea docs, a GitHub search or an AI answer will use the upstream path and will not
compile. Copy imports from a neighbouring file in this repo instead.

**Most files you will touch are not Go.** A typical small change is one Go file, one `.tmpl`
template and one line of JSON. Which of those you touched decides what you have to rebuild.

## Go templates, in five minutes

Gitea's pages are Go `html/template` files under `templates/`. The syntax looks like nothing in
A Tour of Go, so here is the whole of what you need:

```html
{{.SomeValue}}                      the handler put SomeValue in ctx.Data
{{if .IsFoo}} ... {{end}}           conditional
{{range .Items}} {{.Name}} {{end}}  loop; inside it, "." is the current item
{{$outer := .}}                     save the outer "." so you can reach it inside a range
{{template "base/head" .}}          include another template, passing "." to it
{{ctx.Locale.Tr "repo.issues.new"}} a translated string, never a literal
```

The one that catches everyone: **`.` changes meaning inside `range`.** Inside a loop it is the
current item, not the page. `$` always means the top-level page data.

## Words you will meet everywhere

- **handler** — the Go function that answers one URL. Takes `ctx`, puts data in it, renders a
  template.
- **`ctx`** — the request. Carries the signed-in user, the repo being viewed, permissions, and the
  `ctx.Data` bag the template reads.
- **doer** — the user performing an action. Distinct from the user being *looked at*.
- **bean** — a Go struct that maps to a database table. Gitea's database library (XORM) calls them
  beans.
- **XORM** — the library that turns those structs into SQL.
- **migration** — a numbered script that changes the database shape. Existing installs have already
  run the old ones, so you add a new one rather than editing an old one.
- **fixture** — fake rows loaded into the test database so tests have something to work with.
- **unit** — a repository *feature* (code, issues, wiki, actions). Enabled per repo, so "does this
  repo have issues" is a real question with a real answer.
- **middleware** — code that runs before your handler: logging someone in, loading the repo,
  checking permissions.
- **notifier** — Gitea's way of reacting to events. One thing happens, and mail, webhooks and CI all
  hear about it.
- **queue** — how background work happens, so a slow job does not hold up a web request.
- **`tw-` class** — a Tailwind CSS utility. Gitea prefixes them all with `tw-`.

## The rebuild loop

What you changed decides what you rerun:

| You changed | Do this |
| --- | --- |
| a `.go` file | rebuild the binary, restart it |
| a `.tmpl` template | reload the page |
| a locale string | reload the page |
| `.ts` or `.css` | rebuild the frontend assets |

`make watch` does all of it for you on every save, and is worth the one-time setup.

## When you are stuck

In order: the `guide/` page for that area, then its `docs/` page, then the real code. Ask a
teammate before burning an afternoon — and if the answer was not in either layer, that is a gap
worth filling.

## Where to go next

- `INDEX.md` — the map from task to page
- `guide/architecture.md` — how the codebase is laid out
- `README.md` — the toolkit, and what to do when you find a page that is wrong
- `MAINTENANCE.md` — the 22 rules behind that, if you are editing these notes
- `STUDENTS.md` and `CONTRIBUTING.md` — setup, and what a PR must contain
