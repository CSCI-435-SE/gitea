---
scope: web_src/js/index.ts, web_src/js/features, web_src/js/modules, web_src/js/utils, web_src/js/webcomponents, web_src/js/markup, web_src/js/render
verified-at: 187c98fee9
---

# web_src/js — page features and how they get wired up

**Read when:** adding or changing browser behaviour on a page.
**Not here:** Vue components -> `frontend-vue-components.md`; styling -> `frontend-css.md`; the HTML -> `templates.md`.

## Responsibilities

| Directory | Owns |
| --- | --- |
| `features/` | one file per page or feature (`repo-issue.ts`, `repo-diff.ts`, `repo-code.ts`), plus `features/admin/` and `features/comp/` for shared widgets |
| `modules/` | cross-cutting infrastructure: `fetch.ts`, `init.ts`, `observer.ts`, `i18n.ts`, `toast.ts`, `tippy.ts`, `fomantic.ts`, `clipboard.ts`, `codeeditor/` |
| `utils/` | pure helpers with no DOM assumptions: `dom.ts`, `html.ts`, `url.ts`, `time.ts`, `color.ts`, `string.ts` |
| `markup/`, `render/` | enhancing already-rendered content (anchors, mermaid, math) |
| `webcomponents/` | custom elements such as `relative-time`, `overflow-menu` |

## Key files

| Path | What it holds |
| --- | --- |
| `web_src/js/index.ts` | the entry point: imports fomantic and `../css/index.css`, then passes every `initXxx` to `callInitFunctions` |
| `web_src/js/modules/init.ts` | `callInitFunctions`, `InitPerformanceTracer` |
| `web_src/js/modules/observer.ts` | `registerGlobalInitFunc`, `registerGlobalEventFunc`, `registerGlobalSelectorFunc`, `initGlobalSelectorObserver` |
| `web_src/js/modules/fetch.ts` | `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `web_src/js/utils/dom.ts` | `showElem`, `hideElem`, `toggleElem`, `queryElems`, `createElementFromHTML` |
| `web_src/js/modules/i18n.ts` | `trN` — the frontend side of the locale system |

## Conventions & invariants

**Two ways to wire a feature, and new code should prefer the second.**

1. *Page-load init:* export `initFoo()` from a `features/` file and add it to the array in
   `index.ts`. Every function in that array runs on **every** page, so it must find its elements
   and return immediately when they are absent.
2. *Declarative:* `registerGlobalInitFunc('initFoo', (el) => {...})` in the module, and
   `data-global-init="initFoo"` on the element in the template. For clicks,
   `registerGlobalEventFunc('click', 'onFoo', ...)` with `data-global-click="onFoo"`. This covers
   elements added to the DOM later and runs once per element (guarded by an internal
   `_giteaGlobalInited` flag), so re-adding a node does not re-initialise it.

`registerGlobalSelectorFunc(selector, handler)` also exists, but `observer.ts` marks it as "less
efficient and less maintainable" — it is for element types already spread across many pages, not
for new work.

- **Ordering rule:** every `registerGlobalInitFunc` must run *before* `initGlobalSelectorObserver`,
  which `index.ts` deliberately calls last. Registering after it throws.
- **One function per element.** `callGlobalInitFunc` in `observer.ts` looks the attribute up as a
  whole string, so `data-global-init` names exactly one function — space-separating two is an open
  TODO there, not a feature. To add behaviour to an element that already has one (the new-issue
  title in `templates/repo/issue/new_form.tmpl` carries `autoFocusEnd`), put the attribute on a
  wrapper and query for the inner element.
- Data fetching goes through the `fetch.ts` wrappers, never a raw `fetch`.
- Show and hide with `.tw-hidden` plus `showElem` / `hideElem` / `toggleElem`. In Vue use
  `v-if`/`v-show` instead (`frontend-vue-components.md`).
- Use `node.getAttribute`, not `node.dataset` (camel-casing surprises), and never bind
  user-provided data onto a DOM node.
- Custom DOM events are prefixed `ce-`, declared as an exported constant next to their emitter —
  see `EventEditorContentChanged` in `features/comp/EditorMarkdown.ts`.
- TypeScript, per `docs/guidelines-frontend.md`: `import type` for type-only imports,
  `@ts-expect-error` over `@ts-ignore`, `!` rather than `?.`/`??` when a value always exists, and
  `async` only on a function that actually awaits.
- One feature per file or directory; ids and classes are kebab-case with 2-3 feature keywords.

## Recipes

**Add behaviour to an element.** Register a global init func in the owning `features/` module and
put `data-global-init="yourFuncName"` on the element in its template. No change to `index.ts`.

**Add a whole-page feature.** New `features/<name>.ts` exporting `initName()`, an early return when
its root element is missing, then add the import and the array entry in `index.ts`.

**Profile a slow page.** Append `?_ui_performance_trace=1` to the URL to print per-init timings.
`index.ts` already logs an error when total init exceeds 500ms.

**Run one test file.** `pnpm exec vitest <path-filter>` (`testing.md`).

## Gotchas

- Forgetting the early return in a page-load init means that feature's cost is paid on every page
  in Gitea, which is what the 500ms warning is there to catch.
- Fomantic-UI (jQuery) is vendored and deprecated. Do not build new features on it, and do not mix
  it with Vue — Vue may use its CSS classes but not its JavaScript.
- `utils/` is for pure helpers. A function that reaches for `window` or a global selector belongs
  in `modules/` or `features/`.

## Related

- `frontend-vue-components.md` — when a feature should be a Vue app instead
- `templates.md` — where `data-global-init` attributes and `pageData` come from
- `frontend-css.md` — `tw-hidden` and the class conventions
