---
scope: modules/markup, modules/highlight, modules/emoji, modules/references
verified-at: c0092050a4
---

# modules/markup — rendering user content safely

**Read when:** changing how Markdown, code, mentions, issue references or emoji are rendered.
**Not here:** the page templates and their functions -> `templates.md`; diff rendering -> `services-pull-and-gitdiff.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `modules/markup` | the render pipeline: markdown, orgmode, csv, console, jupyter and external renderers, plus post-processing and sanitisation |
| `modules/highlight` | syntax highlighting for code blocks and file views |
| `modules/emoji` | the emoji table and `:shortcode:` replacement |
| `modules/references` | parsing `#123`, `owner/repo#123` and `fixes #123` out of text |

## Key files

| Path | What it holds |
| --- | --- |
| `modules/markup/renderer.go`, `render.go` | the renderer registry and entry points |
| `modules/markup/markdown/` | the Markdown renderer (goldmark) |
| `modules/markup/orgmode/`, `csv/`, `console/`, `jupyter/`, `external/` | the other renderers |
| `modules/markup/html.go` and `html_*.go` | post-processing: links, mentions, issue refs, emoji, commits, code previews |
| `modules/markup/sanitizer.go`, `sanitizer_default.go`, `sanitizer_custom.go` | the HTML allow-list |
| `modules/markup/render_helper.go` | the per-context helper supplying repo-aware link building |
| `modules/references/` | the reference parsers used by both rendering and `services/issue` |

## Conventions & invariants

- **Rendering is two stages:** a renderer turns source into HTML, then the `html_*.go`
  post-processors rewrite that HTML (linkifying references, mentions, emoji, commit SHAs). A
  feature like "link issue numbers" belongs in post-processing, not in the Markdown renderer.
- **Everything passes through the sanitiser.** `sanitizer.go` holds the allow-list; output that
  needs a new tag or attribute must be allowed there explicitly. Never bypass it — this package's
  entire job is that user content cannot inject markup.
- Rendering needs context to build links (which repo, which branch), supplied by a render helper.
  `models/renderhelper` provides the concrete ones for repo, comment and wiki scope — that is why
  rendering a comment needs more than the raw string.
- `modules/references` is shared: the same parser that linkifies `#123` in a comment is what
  `services/issue`'s `UpdateIssuesCommit` uses to close issues from commit messages
  (`services-issue.md`). Change the grammar and both behaviours move.
- These packages are leaves in the dependency direction (`architecture.md`) — the helper indirection
  exists precisely so `modules/markup` need not import models.

## Recipes

**Linkify a new kind of reference.** Add the pattern to `modules/references`, then a post-processor
in `modules/markup/html_*.go`, then allow any new markup in the sanitiser.

**Support a new file type in the file view.** Add a renderer under `modules/markup/` and register
it in `renderer.go`; external tools go through `modules/markup/external/`.

**Change code highlighting.** `modules/highlight` — and note the CSS side lives in
`web_src/css/modules/chroma.css` (`frontend-css.md`).

## Gotchas

- Adding a tag to the sanitiser widens the XSS surface for every piece of user content in Gitea.
  Treat it as a security change, not a formatting one.
- Rendering happens on read, on every view. Anything expensive added to post-processing is paid on
  every issue page load.
- There are several sanitiser profiles (`default`, `description`, `custom`); make sure you are
  changing the one the call site uses.

## Related

- `templates.md` — the template functions that invoke rendering
- `services-issue.md` — commit-message references
