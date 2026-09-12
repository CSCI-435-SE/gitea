---
source: docs/frontend-css.md
source-hash: e184d4cbebf3fcfd
verified-at: c0092050a4
---

<!-- Derived from docs/frontend-css.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Styling, in plain English

**In one sentence:** Gitea uses Tailwind with every class renamed to start with `tw-`, on top of a
CSS framework it is slowly replacing, and both facts will confuse you before someone explains them.
**Come here when:** you are styling something, or a class you wrote is not doing anything.

## What this is

All of Gitea's stylesheets, pulled together by one file that imports the rest. The build uses
Tailwind, configured to prefix everything.

The folders divide by purpose: replacements for the old framework's components, shared styles,
per-feature styles, and per-theme colour definitions. Alongside those sit one file per area of the
site — the repository pages, the admin pages, the dashboard, code review.

## Why it exists

Gitea is mid-migration, and understanding that explains almost everything odd here.

It was built on **Fomantic-UI**, a CSS-and-jQuery framework. That framework is now deprecated but
far too widely used to remove in one go. So Gitea is gradually replacing it: for each component, a
hand-written replacement is added, and the framework's version stops being used.

Meanwhile **Tailwind** was adopted for new work. Tailwind gives you tiny single-purpose classes you
combine in the markup, rather than writing new CSS files.

Those two collide: Tailwind and Fomantic both want to define a class called `hidden`. So Gitea
prefixes every Tailwind class with `tw-`, and the collision disappears.

The result is that you will meet four kinds of class — Tailwind's `tw-`, Gitea's own `gt-` and `g-`
helpers, Gitea's component replacements, and leftover Fomantic classes — and knowing which is which
is most of the skill.

## Words you'll meet

- **utility class** — a class doing one small thing, like `tw-flex`.
- **prefix** — the `tw-` at the front of every Tailwind class here.
- **purge** — Tailwind removing classes it cannot find used, to keep the file small.
- **specificity** — the rules deciding which CSS wins when two rules target the same element.
- **`!important`** — a sledgehammer that overrides specificity.
- **theme variable** — a named colour that changes with the user's chosen theme.
- **vendored** — a third-party library copied into the repository rather than installed.

## What's in these files

| Where | What it is for |
| --- | --- |
| `web_src/css/index.css` | The import list. **A stylesheet not imported here does nothing.** |
| `web_src/css/helpers.css` | Gitea's own `gt-` and `g-` helper classes. |
| `web_src/css/base.css` | Base element styles and shared variables. |
| `web_src/css/themes/` | Per-theme colour definitions. |
| `tailwind.config.ts` | The `tw-` prefix, which files are scanned for class names, and the blocklist. |

The subfolders: `modules/` holds Gitea's replacements for the old framework's components;
`shared/` styles used in several places; `features/` per-feature styling; and `repo/`, `markup/`,
`editor/` and `themes/` for those areas.

## The rules, and why

**Every Tailwind class starts with `tw-`.** `tw-hidden`, `tw-flex`. An unprefixed Tailwind class
does nothing at all — it is not a class Gitea has. This is the single most common confusion when
copying a Tailwind example from the internet.

**Prefer the flex layout helpers over margins on each child.** Setting a left or right margin on
every item to space them out breaks the moment one is hidden or the order changes. A flex container
with a gap does not.

**Gitea's own helpers are `gt-` and `g-`, and are a last resort.** Use them only where Tailwind has
no equivalent. Adding a `gt-` class that duplicates an existing `tw-` one means the next person has
two ways to do the same thing and no way to know which is current.

**Never edit a framework class to restyle something.** Changing what a framework class does changes
it *everywhere* — including in places you have never seen. Make a new class name and apply it
alongside.

**Avoid `!important`, and comment it when unavoidable.** It wins by brute force, which means the
next person cannot override it either. The existing `gt-` helpers do use it, deliberately, so they
beat framework rules — that is the exception, not the pattern.

**Names are kebab-case with two or three words describing the feature**, prefixed so they cannot
collide with a framework class.

**Write the class attribute as one readable string**, including any conditional parts:

```html
<div class="flex-text-inline {{if .IsFoo}}tw-hidden{{end}}"></div>
```

**`tw-hidden` is special.** It is the show/hide mechanism the JavaScript helpers use
(`frontend-js.md`). Do not reuse it as a general styling class — something else may toggle it.

## How to actually do it

**Style a new feature.** Create `web_src/css/features/<name>.css`, **import it from `index.css`**,
and prefix your class names with the feature name so they cannot collide.

**Restyle one of the framework's components.** Look in `web_src/css/modules/` first. Many components
already have a Gitea replacement, and that replacement is the file to change.

**Use a colour.** Always a theme variable, never a literal colour value. A hard-coded colour looks
right in whichever theme you were using and wrong or invisible in the others.

## Traps, and what they look like

**Your stylesheet has no effect whatsoever.** You did not add the `@import` to `index.css`. Nothing
warns you — the file simply is not part of the build.

**Your class works in development and vanishes in the built site.** Tailwind scans the files listed
in its config and removes classes it cannot find. A class assembled in JavaScript — sticking a
prefix and a variable together — is invisible to that scan and gets purged. Always write the full
class name literally, even if that means a few more lines.

**A Tailwind utility you know exists does not work.** Two possibilities: you left off the `tw-`
prefix, or it is in the blocklist in the config, which is deliberate.

**You change a colour and one theme looks broken.** You used a literal colour instead of a theme
variable.

**You edit something in `web_src/css/modules/` expecting the third-party framework.** That folder is
Gitea's *replacement* for it. The vendored original is elsewhere, under `web_src/fomantic/`.

**Your style is overridden and you cannot see why.** Before reaching for `!important`, check whether
you are fighting a `gt-` helper — those use it deliberately and will win.

## Where to go next

- `docs/frontend-css.md` (in this folder) — the reference page this was written from
- `templates.md` — where these classes are actually written
- `frontend-js.md` — the hide mechanism and the DOM helpers
