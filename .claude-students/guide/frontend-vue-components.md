---
source: docs/frontend-vue-components.md
source-hash: 8117e0df6d7abe8a
verified-at: c0092050a4
---

<!-- Derived from docs/frontend-vue-components.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Vue components, in plain English

**In one sentence:** the handful of genuinely interactive widgets in Gitea are Vue components, and
they do not start themselves — something else has to mount them onto a page.
**Come here when:** you are building an interactive widget, or deciding whether something needs Vue
at all.

## What this is

Gitea's Vue components: the Actions run log viewer, the diff file tree used in code review, the
merge form, the dashboard repository list, and the various repository charts.

Not everything in the folder is a component. Some files are plain TypeScript holding logic or shared
state that the components use.

## Why it exists

Most of Gitea's HTML is generated on the server. That is the right default: it is fast, it works
without JavaScript, and search engines can read it.

But some things genuinely cannot be done that way. A log viewer that streams output as a job runs. A
file tree you expand and collapse while the diff stays put. A form whose options change as you pick
them. These have *state that changes without a page load*, and expressing that in hand-written
DOM-manipulating JavaScript gets unmaintainable quickly.

So the project's rule is: **server templates for simple and search-visible pages, Vue for genuinely
interactive ones.** A page that needs one toggle does not need Vue — that is a few lines in a
`features/` file.

The other half of the design follows from the single-bundle problem in `frontend-js.md`. Every
component ships to every page. If components mounted themselves, every page would pay for all of
them. So mounting is explicit and guarded.

## Words you'll meet

- **component** — a reusable piece of UI with its own state and markup.
- **SFC** — "single-file component": markup, logic and styles in one `.vue` file.
- **mount** — attaching a component to a real element so it renders.
- **mount point** — that element, put in the server-rendered template.
- **props** — values passed into a component from outside.
- **store** — shared state several components read, kept outside any one of them.
- **presentational** — a component concerned with displaying, with logic moved elsewhere.
- **dead code** — code that ships but never runs.

## What's in these files

| Where | What it is for |
| --- | --- |
| `web_src/js/components/RepoActionView.vue` | The Actions run log viewer. |
| `web_src/js/components/DiffFileTree.vue`, `DiffFileTreeItem.vue` | The code review file tree. |
| `web_src/js/components/PullRequestMergeForm.vue` | The merge button and its options. |
| `web_src/js/components/DashboardRepoList.vue` | The dashboard repository list. |
| `web_src/js/components/ViewFileTreeStore.ts` | Shared state for the file-tree components. |
| `web_src/js/components/ActionRunView.ts`, `WorkflowGraph.utils.ts` | Logic pulled out of the neighbouring `.vue` files so it can be unit-tested. |
| `web_src/js/components/*.test.ts` | The tests, sitting beside what they test. |

## The rules, and why

**Filenames are PascalCase and match the component name.**

**Components do not mount themselves.** A module under `features/` does it, and only after checking
the element exists:

```ts
const el = document.querySelector('#repo-contributors-chart');
if (!el) return;
createApp(RepoContributors, {repoLink: el.getAttribute('data-repo-link')}).mount(el);
```

That guard is the whole point. The component ships to every page; the mount happens only where the
element is. So **adding a component always means touching the wiring in `frontend-js.md` too** —
the component alone does nothing.

**Props come from the server, not from an extra request.** Small values arrive as `data-`
attributes on the mount point; anything structured comes through the page data the handler set. Do
not fetch configuration the server already knew and could have handed you — that is a round trip for
information you already had.

**Show and hide with `v-if` and `v-show`.** The hidden class and helpers described in
`frontend-js.md` are the plain-JavaScript mechanism. Inside a component, Vue owns the DOM; mixing
the two produces elements that disagree about whether they are visible.

**Never mix Vue with Fomantic-UI's JavaScript.** Its CSS classes are fine. Its jQuery behaviour
mutates the DOM behind Vue's back, and Vue then overwrites it. The symptoms are bizarre and
intermittent.

**Single-file components only, no JSX.**

**Data fetching still goes through the shared wrappers.**

**Logic worth testing moves to a sibling `.ts` file.** The `.vue` file stays presentational. This is
why some files here are not components — testing a plain function is far easier than testing a
rendered component, so the logic is lifted out and tested directly.

## How to actually do it

**Add a component.** Four pieces, and all four are required:

1. `YourThing.vue`.
2. A mount block in the owning `features/` module, guarded by the element check.
3. A root element in the server template, carrying whatever `data-` attributes it needs.
4. If it has real logic, a sibling `.ts` file with a `.test.ts` beside it.

**Give a component data from the server.** Small values as `data-` attributes on the mount point.
Anything structured through the handler's page data, read from the global config object.

**Test one.** `pnpm exec vitest <name>`.

## Traps, and what they look like

**Nothing renders, and nothing errors.** This is the characteristic failure here, and it has two
causes that look identical. Either there is no mount call at all — in which case the component still
compiles, still ships in the bundle, and is simply dead code that the compiler will not warn you
about. Or the mount call exists but the selector does not match, because of a typo in the element's
id or class. The guard then does exactly what it is supposed to do and returns quietly.

When a component does not appear, check the mount call and the selector *before* looking inside the
component.

**You edit a `.ts` file in this folder expecting a component.** Not everything here is a `.vue`.
Some files are stores or extracted logic.

**Your component's visibility fights with something else.** Something is using the plain-JavaScript
hide mechanism on an element Vue controls. Inside a component, use `v-if` or `v-show`.

**Intermittent, inexplicable DOM behaviour.** Fomantic's JavaScript is operating on the same
elements as Vue.

## Where to go next

- `docs/frontend-vue-components.md` (in this folder) — the reference page this was written from
- `frontend-js.md` — the module that mounts your component
- `templates.md` — where the mount point and its data attributes are written
