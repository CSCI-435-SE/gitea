---
scope: web_src/js/components
verified-at: c0092050a4
---

# web_src/js/components — the Vue 3 components

**Read when:** building or changing an interactive widget complex enough to need state.
**Not here:** how a component gets mounted -> `frontend-js.md`; styling -> `frontend-css.md`.

## Responsibilities

Vue 3 single-file components for the parts of the UI too stateful for a Go template plus a little
JavaScript: the Actions run viewer, the diff file tree, the dashboard repo list, the merge form,
the various repo charts.

`docs/guidelines-frontend.md` draws the line as: **Go templates for simple and SEO-relevant pages,
Vue for complex interactive pages.** A page that only needs a toggle does not need Vue.

## Key files

| Path | What it holds |
| --- | --- |
| `web_src/js/components/RepoActionView.vue` | the Actions run log viewer |
| `web_src/js/components/DiffFileTree.vue`, `DiffFileTreeItem.vue` | the code-review file tree |
| `web_src/js/components/PullRequestMergeForm.vue` | the merge button and its options |
| `web_src/js/components/DashboardRepoList.vue` | the dashboard repo list |
| `web_src/js/components/ViewFileTreeStore.ts` | a plain-TS store shared by the file-tree components |
| `web_src/js/components/ActionRunView.ts`, `WorkflowGraph.utils.ts` | logic extracted out of the neighbouring `.vue` files so it is unit-testable |
| `web_src/js/components/*.test.ts` | Vitest tests, colocated |

## Conventions & invariants

- Filenames are PascalCase and match the component name.
- **Components do not mount themselves.** A `features/` module does it, guarded by the presence of
  its root element:

  ```ts
  const el = document.querySelector('#repo-contributors-chart');
  if (!el) return;
  createApp(RepoContributors, {repoLink: el.getAttribute('data-repo-link')}).mount(el);
  ```

  So adding a component always means touching `frontend-js.md`'s wiring too.
- Props come from `data-*` attributes read with `getAttribute`, or from `window.config.pageData`
  (set by the handler via `ctx.PageData`, see `templates.md`). Do not fetch configuration the
  server could have handed you.
- Show and hide with `v-if` / `v-show` — not `.tw-hidden`, which is the plain-JS mechanism.
- **Do not mix Vue with Fomantic-UI JavaScript.** Fomantic's CSS classes are fine; its jQuery
  behaviours fight Vue's ownership of the DOM.
- No JSX — SFCs only.
- Data fetching still goes through `web_src/js/modules/fetch.ts`.
- Logic worth testing is pulled into a sibling `.ts` file and tested there; the `.vue` file stays
  presentational. `ActionRunView.ts` beside `RepoActionView.vue` is the pattern.

## Recipes

**Add a component.** Create `YourThing.vue`, a mount block in the owning `features/` module,
a root element with the data attributes it needs in the template, and — if it has real logic — a
sibling `.ts` with a `.test.ts`.

**Give a component server data.** Small values as `data-*` attributes on the mount point; anything
structured via `ctx.PageData` in the handler, read as `window.config.pageData`.

**Test one.** `pnpm exec vitest <name>` (`testing.md`).

## Gotchas

- A component with no mount call is dead code that still ships in the bundle — the compiler will
  not tell you.
- `.ts` files in this directory are not components; they are stores and extracted logic. Do not
  assume everything here is a `.vue`.
- Because mounting is element-guarded, a typo in the root element's id or class fails silently:
  nothing renders and nothing errors.

## Related

- `frontend-js.md` — the `features/` module that mounts the component
- `templates.md` — the mount point and `pageData`
