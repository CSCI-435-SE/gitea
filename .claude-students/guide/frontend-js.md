---
source: docs/frontend-js.md
source-hash: 2d5a7fdb8fea93a0
verified-at: c0092050a4
---

<!-- Derived from docs/frontend-js.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Browser behaviour, in plain English

**In one sentence:** Gitea ships one JavaScript bundle to every page, so the interesting question is
not "what does my code do" but "how does it know it is on the right page".
**Come here when:** you are making something on a page respond to a click, load data, or show and
hide.

## What this is

All the browser-side code except the Vue components, organised by how reusable it is:

- `features/` — one file per page or feature, about eighty of them. This is where most work happens.
- `modules/` — shared infrastructure: fetching, initialisation, tooltips, toasts, the code editor.
- `utils/` — pure helpers with no assumptions about the page: strings, URLs, dates, colours.
- `markup/` and `render/` — enhancing content that has already been rendered.
- `webcomponents/` — custom HTML elements like the relative-time display.

## Why it exists

Gitea builds **one bundle for the whole site**. Every page downloads the same JavaScript.

That is a deliberate trade — one file caches well and there is no per-page build — but it creates
the central problem this code has to solve. The issue-list code is loaded on the settings page. The
diff viewer is loaded on your profile. All of it runs on all of it.

So every feature must cheaply determine "am I on a page that needs me?" and get out of the way if
not. Everything below follows from that.

There is a second problem. Some elements appear *after* the page loads — a comment you just posted,
a dropdown rendered on demand. Code that ran once at page load never sees them, which is why the
newer of the two wiring mechanisms exists.

## Words you'll meet

- **bundle** — the single compiled JavaScript file served to every page.
- **init function** — a function that sets one feature up.
- **early return** — leaving a function immediately when there is nothing to do.
- **selector** — a CSS pattern for finding elements, like `.issue-list`.
- **data attribute** — a `data-something="value"` attribute on an HTML element.
- **observer** — code that watches for elements being added to the page.
- **event delegation** — one listener on the document handling clicks on many elements, including
  ones added later.
- **custom event** — an event your own code invents and dispatches.

## What's in these files

| Where | What it is for |
| --- | --- |
| `web_src/js/index.ts` | The entry point. Imports everything and runs the init functions. |
| `web_src/js/modules/init.ts` | Runs that list, and can time each entry. |
| `web_src/js/modules/observer.ts` | The declarative wiring: the register functions and the observer. |
| `web_src/js/modules/fetch.ts` | The HTTP wrappers every request must go through. |
| `web_src/js/utils/dom.ts` | Finding elements, showing and hiding them, building them. |
| `web_src/js/modules/i18n.ts` | Translated strings on the frontend. |

## The rules, and why

### Two ways to wire something up

**The old way — page-load init.** Export `initFoo()` from a file in `features/` and add it to the
list in `index.ts`. It runs on every page in Gitea, so it **must** look for its elements and return
immediately when they are not there.

**The new way, and what to prefer.** Register a named function once:
`registerGlobalInitFunc('initFoo', (el) => {...})`, then in the template put
`data-global-init="initFoo"` on the element. For clicks,
`registerGlobalEventFunc('click', 'onFoo', ...)` with `data-global-click="onFoo"`.

This is better for two concrete reasons. It runs only when an element that asks for it actually
exists — no per-page cost anywhere else. And it also catches elements added to the page later, which
the old way cannot. It runs once per element, guarded internally, so a node removed and re-added is
not initialised twice.

There is a third function that registers by CSS selector. The code that defines it calls it "less
efficient and less maintainable" in its own comment. It exists for element types already scattered
across many pages. Do not reach for it in new work.

**Registration must happen before the observer starts.** `index.ts` deliberately starts the observer
last, after everything has registered. Registering afterwards throws an error rather than failing
quietly — which is the kind thing for it to do.

### Everything else

**All HTTP goes through the fetch wrappers.** They handle the cross-site request token and error
conventions. A raw `fetch` skips both, and will fail on anything that writes.

**Show and hide with the helper functions and the hidden class**, not by setting styles directly.
Vue components use `v-if` and `v-show` instead — different mechanism, do not mix them.

**Read attributes with `getAttribute`, not `dataset`.** The `dataset` property silently renames
things: `data-some-value` becomes `someValue`. The rename is invisible and the bug it causes is
confusing. And never attach user-provided data to a DOM node.

**Custom events are prefixed `ce-`** and declared as an exported constant next to whatever emits
them, so the name is defined once instead of typed as a string in two places that can drift apart.

**TypeScript conventions:** `import type` for type-only imports, `@ts-expect-error` rather than
`@ts-ignore`, `!` rather than `?.` when a value genuinely always exists, and `async` only on
functions that actually await something.

**One feature per file.** Ids and classes are kebab-case with two or three words naming the feature.

## How to actually do it

**Make an element do something.** Register a global init function in the relevant `features/` file
and add `data-global-init="yourFuncName"` to the element in its template. `index.ts` does not
change.

**Add a whole-page feature.** Create `features/<name>.ts` exporting `initName()`. Find your root
element first and **return immediately if it is missing**. Then add the import and the list entry in
`index.ts`.

**Find out why a page feels slow.** Add `?_ui_performance_trace=1` to the URL and the timings for
every init function are printed. `index.ts` already logs an error if the total exceeds 500
milliseconds.

**Run one test file.** `pnpm exec vitest <path-filter>`.

## Traps, and what they look like

**The whole site gets slower after your change.** You added a page-load init without an early
return, so your feature's setup cost is now paid on every page in Gitea. The 500ms warning in the
console exists to catch exactly this.

**Your code works on load and not on new elements.** A comment posted without a page reload does not
get your behaviour. You used page-load init; use the declarative mechanism, which sees later
elements.

**Your `data-` attribute reads as undefined.** You used `dataset` and the property name was
silently camel-cased. Use `getAttribute` with the literal attribute name.

**Your request is rejected, usually as a 403.** You used a raw `fetch` and skipped the wrappers,
which is what adds the request token.

**Registering a global init function throws.** You registered it after the observer started.
Registration happens at module load; the observer starts last, on purpose.

**You build on Fomantic-UI because the existing code does.** It is the vendored jQuery library, and
it is deprecated. Do not start new features on it, and never mix its JavaScript with Vue — Vue may
use its CSS classes, but the two fight over who owns the DOM.

**You put a helper in `utils/` and it breaks in tests.** That folder is for pure helpers. Anything
reaching for `window` or searching the page belongs in `modules/` or `features/`.

## Where to go next

- `docs/frontend-js.md` (in this folder) — the reference page this was written from
- `frontend-vue-components.md` — when a feature is complex enough to deserve Vue
- `templates.md` — where those `data-global-init` attributes are written
- `frontend-css.md` — the hidden class and the naming conventions
