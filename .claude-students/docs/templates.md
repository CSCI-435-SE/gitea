---
scope: templates, modules/templates
verified-at: 7075389dd1
---

# templates — the Go HTML templates and their helpers

**Read when:** editing a page's markup, adding a template, or needing a template function.
**Not here:** the handler that renders it -> `routers-web.md`; classes -> `frontend-css.md`; behaviour -> `frontend-js.md`.

## Responsibilities

Server-rendered HTML, composed with Go's `html/template`. `modules/templates` owns the renderer and
every function templates can call.

| Directory | Owns |
| --- | --- |
| `base/` | the page skeleton: `head.tmpl`, `head_script.tmpl`, `head_navbar.tmpl`, `footer.tmpl`, `paginate.tmpl`, `alert.tmpl` |
| `shared/` | reusable partials: `issuelist.tmpl` and its row partial `issuelist_items.tmpl`, `combomarkdowneditor.tmpl`, `actions/`, `secrets/`, `webhook/`, `user/` |
| `repo/`, `user/`, `org/`, `admin/`, `explore/`, `projects/`, `package/` | one directory per page area |
| `mail/` | email bodies |
| `status/` | error pages |
| `devtest/` | the `/devtest` component gallery, also driven by the e2e tests |
| `custom/` | site-admin override hooks — intentionally near-empty |
| `swagger/` | the generated API spec (`v1_json.tmpl`), see `routers-api-v1.md` |

## Key files

| Path | What it holds |
| --- | --- |
| `modules/templates/helper.go` | `newFuncMapWebPage` — the whole set of page template functions, including `dict`, `Iif` and `Eval` |
| `modules/templates/util_render.go`, `util_render_comment.go` | markup and comment rendering helpers |
| `modules/templates/util_date.go`, `util_avatar.go`, `util_string.go`, `util_slice.go`, `util_dict.go` | the rest of the function set, grouped by kind |
| `modules/templates/htmlrenderer.go` | `TplName` and the renderer |
| `modules/templates/scopedtmpl/` | the scoped-template machinery behind partial rendering |
| `templates/base/head_script.tmpl` | emits `window.config`, including `pageData` |

## Conventions & invariants

- A template is addressed by its path under `templates/` without the extension, as a
  `templates.TplName` constant in the handler — `"repo/issue/list"` is
  `templates/repo/issue/list.tmpl` (`routers-web.md`).
- **The handler's `ctx.Data` is the only data channel.** A template cannot reach anything the
  handler did not put there, and `ctx.RootData` inside a template context is that same data.
- Data destined for JavaScript goes through `ctx.PageData`, which `base/head_script.tmpl` emits as
  `window.config.pageData` — a separate channel from `ctx.Data`.
- **Every user-visible string is a locale key**, written `{{ctx.Locale.Tr "repo.issues.new"}}` or
  `ctx.Locale.TrN` for plurals. Never a literal. Keys are added only to
  `options/locale/locale_en-US.json`; the other locales come from Crowdin.
- Composition is `{{template "base/head" .}}` style includes. New shared markup goes in `shared/`,
  not copied between areas.
- `dict` is lowercase for historical reasons; new template functions get uppercase names
  (`helper.go` says so at the definition).
- `dict` treats the key `"."` specially: `dictMerge` in `modules/templates/util_dict.go` flattens
  that argument's map into the new one. So `dict "." $ "Issues" $subset` hands a partial the whole
  root context with one key replaced — but `"." $` **must come first**, or the merge overwrites the
  replacement instead. `templates/repo/issue/list_grouped.tmpl` renders each group this way.
- Mail templates use a different, smaller function set — `mailBodyFuncMap` and
  `mailSubjectTextFuncMap` in `modules/templates/mail.go`. A helper added for pages is not
  available in `templates/mail/`.
- Class attributes are written as one readable unit including their conditionals
  (`frontend-css.md`).
- Hook frontend behaviour up declaratively with `data-global-init` / `data-global-click`
  attributes rather than inline script (`frontend-js.md`).
- `.tmpl` files are tab-indented (`.editorconfig`).

## Recipes

**Add a page's template.** Create the `.tmpl` under the matching area directory, include
`base/head` and `base/footer`, declare the `TplName` constant in the handler, and add every string
as a locale key.

**Share markup between pages.** Put the partial in `templates/shared/` and include it from both
places. Two near-identical partials in two area directories is the thing to avoid.

**Add a template function.** Add it to the map returned by `newFuncMapWebPage` in
`modules/templates/helper.go` and implement it in the matching `util_*.go`. Prefer passing prepared data from the handler — a function that
does database work at render time is a performance bug.

**See a component in isolation.** Run Gitea in development mode and open `/devtest`.

## Gotchas

- A misspelled key in `ctx.Data` renders as empty, not as an error. Check the handler when
  something is silently blank.
- A missing locale key renders the key itself — visible, but easy to miss in a rarely-hit branch.
- `templates/swagger/v1_json.tmpl` is generated. Edit the swagger comments and run
  `make generate-swagger` instead (`routers-api-v1.md`).
- `templates/custom/` is for deployment overrides, not a place to put course code.

## Related

- `routers-web.md` — `TplName` and `ctx.Data`
- `frontend-css.md`, `frontend-js.md` — the classes and attributes used here
