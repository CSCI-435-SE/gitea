---
source: docs/modules-markup.md
source-hash: 9bb2144416bcd7ea
verified-at: c0092050a4
---

<!-- Derived from docs/modules-markup.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Turning user text into safe HTML, in plain English

**In one sentence:** Markdown becomes HTML, then a second pass turns `#123` and `@someone` into
links, and everything goes through a filter whose job is to make sure user content cannot inject
markup.
**Come here when:** you are changing how comments, READMEs, code or references are displayed.

## What this is

Four related packages. `modules/markup` does the rendering and the safety filtering.
`modules/highlight` colours code. `modules/emoji` turns `:tada:` into an emoji. `modules/references`
recognises `#123` and similar in text.

## Why it exists

**Because every issue comment is untrusted input that Gitea then displays as HTML.**

That is the most dangerous thing a web application can do. If a comment containing a script tag were
rendered as-is, the script would run in the browser of everyone who reads that issue, with their
session. Preventing that is the entire reason this package is shaped the way it is.

Hence two rules that are not negotiable: everything passes through a filter with an explicit list of
permitted tags and attributes, and that list is a security boundary rather than a formatting
preference.

**Why two stages of rendering?** Because `#123` is not Markdown. Markdown knows about headings and
emphasis; it has no idea that `#123` means an issue in *this repository*, or that `@someone` is a
user. So Markdown runs first, producing ordinary HTML, and a second pass walks that HTML adding the
Gitea-specific links.

That separation is why "make issue numbers link" is a post-processing change, not a Markdown change.

## Words you'll meet

- **render** — turn source text into HTML.
- **post-process** — walk already-rendered HTML and modify it.
- **sanitise** — strip anything not on the permitted list.
- **allow-list** — the explicit set of permitted tags and attributes. Anything absent is removed.
- **XSS** — cross-site scripting: injecting script into a page other people view.
- **linkify** — turn a plain reference into a hyperlink.
- **render helper** — the object supplying context (which repository, which branch) so links can be
  built.
- **profile** — one of several allow-lists, used in different places.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/markup/renderer.go`, `render.go` | The registry of renderers and the way in. |
| `modules/markup/markdown/` | The Markdown renderer. |
| `modules/markup/orgmode/`, `csv/`, `console/`, `jupyter/`, `external/` | The other formats. |
| `modules/markup/html.go` and the `html_*.go` files | Post-processing: links, mentions, issue references, emoji, commit hashes. |
| `modules/markup/sanitizer.go`, `sanitizer_default.go`, `sanitizer_custom.go` | The allow-lists. |
| `modules/markup/render_helper.go` | Supplies the repository context links need. |
| `modules/references/` | Recognising `#123` and friends. |

## The rules, and why

**Rendering is two stages, and new features usually belong in the second.** A renderer turns source
into HTML; the post-processors then rewrite that HTML. "Link issue numbers", "show emoji", "linkify
commit hashes" are all post-processing.

**Everything passes through the sanitiser, and nothing bypasses it.** Output needing a new tag or
attribute must have it added to the allow-list explicitly. There is no "just this once" — the
package's entire purpose is that user content cannot inject markup.

**Rendering needs context, which is why it takes more than a string.** To turn `#123` into a link,
the renderer must know which repository it is in. That context arrives through a helper object,
which is also how this package avoids importing the database: the helper is provided by code that
*can* reach models, and this package only calls it (`architecture.md`).

**The reference parser is shared, and that has consequences.** The same code that linkifies `#123`
in a comment is what closes an issue when a commit message says `fixes #123`
(`services-issue.md`). Change the grammar and **both** behaviours change — including the one you
were not thinking about.

## How to actually do it

**Make a new kind of reference into a link.** Three steps: add the pattern to the reference parser,
add a post-processor, and allow any new markup in the sanitiser. Forgetting the third means your
links are generated and then stripped.

**Support a new file type in the file view.** Add a renderer and register it. Formats handled by an
external tool go through the external renderer.

**Change code highlighting.** The Go side is in `modules/highlight`; the colours themselves are CSS
(`frontend-css.md`).

## Traps, and what they look like

**You add a tag to the allow-list to make something display.** Stop and treat this as a security
change, not a formatting one. Widening that list widens it for *every* piece of user content in
Gitea — every comment, every README, every wiki page, written by anyone. The question is not "does
my content need this tag" but "is this tag safe in text written by a stranger".

**Your markup is generated and then vanishes.** The sanitiser stripped it. That is the system
working; add the tag deliberately or produce different markup.

**Every issue page gets slower.** Rendering happens on *read*, every time anyone views the page.
Anything expensive added to post-processing is paid on every view of every comment, forever.

**Your change works in one place and not another.** There are several sanitiser profiles for
different contexts. Check which one the call site actually uses.

**Changing the reference grammar breaks issue-closing.** The parser is shared with the commit
message path. Test both.

## Where to go next

- `docs/modules-markup.md` (in this folder) — the reference page this was written from
- `templates.md` — the template functions that trigger rendering
- `services-issue.md` — the other user of the reference parser
