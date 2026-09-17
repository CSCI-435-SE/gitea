---
scope: web_src/css, tailwind.config.ts
verified-at: a50a52cbc8
---

# web_src/css — styling conventions

**Read when:** adding or changing styles, or deciding between a Tailwind utility and a new class.
**Not here:** the markup -> `templates.md`; show/hide behaviour -> `frontend-js.md`.

## Responsibilities

All stylesheets, aggregated by `index.css`, which imports the module replacements, the helpers and
then the per-area files. The build is vite + Tailwind, configured in `tailwind.config.ts`.

| Directory | Owns |
| --- | --- |
| `modules/` | replacements for Fomantic-UI's components (`button.css`, `dropdown.css`, `form.css`, `modal.css`, ...) |
| `shared/` | styles shared across several areas |
| `features/` | per-feature styles (`heatmap.css`, `imagediff.css`, `projects.css`, `gitgraph.css`) |
| `repo/`, `markup/`, `editor/`, `themes/` | repo sub-areas, rendered markup, the editors, and the theme variables |
| top-level files | one per page area: `repo.css`, `admin.css`, `user.css`, `org.css`, `explore.css`, `dashboard.css`, `review.css`, `actions.css`, `install.css` |

## Key files

| Path | What it holds |
| --- | --- |
| `web_src/css/index.css` | the import list — a new stylesheet must be added here |
| `web_src/css/helpers.css` | the `gt-` and `g-` custom helpers |
| `web_src/css/base.css` | base element styles and shared variables |
| `web_src/css/themes/` | per-theme variable definitions |
| `tailwind.config.ts` | the `tw-` prefix, the content globs and the blocklist |

## Conventions & invariants

- **Tailwind utilities are prefixed `tw-`** (set in `tailwind.config.ts`), so `tw-hidden`,
  `tw-flex`. An unprefixed Tailwind class does nothing.
- Prefer the `flex-*` layout helpers over per-child `tw-ml-*` / `tw-mr-*` margins (`AGENTS.md`).
  Fall back to `tw-*` utilities where specificity forces `!important`.
- Gitea's own helpers live in `helpers.css`: `gt-` for general helpers, `g-` for framework-level
  ones. Use them **only where Tailwind has no utility** — do not add a `gt-` class that duplicates
  a `tw-` one.
- **Never edit a framework class to restyle something.** Create a new class name and apply it
  alongside (`docs/guidelines-frontend.md`).
- Avoid `!important`; where it is unavoidable, leave a comment saying why. The existing `gt-`
  helpers use it deliberately, which is why they win over framework rules.
- Class and id names are kebab-case with two or three feature keywords, prefixed to avoid
  collisions between frameworks.
- In a template, write the class attribute as one readable unit rather than concatenating:

  ```html
  <div class="flex-text-inline {{if .IsFoo}}tw-hidden{{end}}"></div>
  ```

- `tw-hidden` is also the show/hide mechanism used by `showElem`/`hideElem` in
  `frontend-js.md` — do not reuse it as a generic styling class.

## Recipes

**Style a new feature.** Add `web_src/css/features/<name>.css`, import it from `index.css`, and use
a `.<feature>-` class prefix so the names cannot collide.

**Restyle a Fomantic component.** Look in `web_src/css/modules/` first — many components are
already replaced there, and the replacement is the file to change.

**Add a theme-aware colour.** Use the variables from `themes/`, never a literal colour, or the
other themes break.

**Add a colour theme.** Drop a `web_src/css/themes/theme-<name>.css`. The glob in
`vite.config.ts` makes every file there its own build entry and `services/webtheme` discovers it by
filename, so there is nothing else to register — no Go, template or `app.ini` change. The file must
carry its own `gitea-theme-meta-info` block, because the backend parses the *last* one in the built
output; the cheapest shape imports the matching stock theme and overrides only what the palette
changes:

```css
@import "./theme-gitea-dark.css";

gitea-theme-meta-info {
  --theme-display-name: "Nord Dark";
  --theme-color-scheme: "dark";
}

:root { --color-primary: #81a1c1; }
```

Import the theme matching your scheme: it supplies `--is-dark-theme`, `color-scheme` and the
`var()`-derived colours, which is why `--color-primary-hover` and friends must be left alone.
`web_src/js/utils/theme-files.test.ts` then checks every theme's meta block and its contrast
against the stock theme it extends.

## Gotchas

- Forgetting the `@import` in `index.css` means your file compiles to nothing, with no error.
- The Tailwind `content` globs in `tailwind.config.ts` decide which classes survive the build. A
  class assembled dynamically in JavaScript will be purged — write the full class name literally.
- `tailwind.config.ts` has a `blocklist`; a utility that mysteriously does not exist may be listed
  there on purpose.
- `web_src/css/modules/` is Gitea's *replacement* for Fomantic, not Fomantic itself — the vendored
  original is under `web_src/fomantic/`.
- A theme file must not introduce a *new* `--color-*` name. `tailwind.config.ts` and
  `stylelint.config.ts` read only `base.css` and the two `theme-gitea-*` files, so an unknown name
  fails `make lint-css` and never gets a `tw-` utility.

## Related

- `templates.md` — where these classes are written
- `frontend-js.md` — `tw-hidden` and the DOM helpers
- `build-and-tooling.md` — `make lint-css` and the vite build
