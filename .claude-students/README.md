# .claude-students — start here

Course-only documentation for the CSCI-435 Gitea fork. It exists so a person — or an AI session —
can read one or two short pages instead of exploring ~150 Go packages. That exploration is the
single largest avoidable cost on this project, in your time and in tokens.

This page is the quick start: what is here, what tools you have, and the three things you will
actually need to do.

## Two layers, same content

| | `docs/` | `guide/` |
| --- | --- | --- |
| Written for | an AI session | a person |
| Style | terse, every claim cites a file | explains, defines jargon, names symptoms |
| Length | ~80 lines | up to 200 |
| Status | **the source of truth** | a rendering of it |

Both have the same 34 filenames, so `guide/models-db.md` explains `docs/models-db.md`.

**`guide/` is derived from `docs/`.** Each guide stores a fingerprint of its source page, so if the
reference changes and the guide is not regenerated, the validator says so. **If the two ever
disagree, the guide is wrong** — fix the reference page, not the guide.

## What is in this folder

| File | What it is for |
| --- | --- |
| `INDEX.md` | Routing table: "working on X → read Y". Loaded into every AI session. |
| `guide/START-HERE.md` | **If you are new to this codebase, read this first.** |
| `guide/*.md` | A plain-English page per reference page. |
| `docs/*.md` | The reference pages. Terse, citation-heavy. |
| `MAINTENANCE.md` | The 22 rules. Read before editing anything here. |
| `_TEMPLATE.md`, `guide/_TEMPLATE.md` | The only permitted shapes for a new page. |
| `check.sh` | The validator. Run it before you push. |

## Your toolkit

One command:

```sh
./.claude-students/check.sh
```

It prints `check.sh: ok (34 docs, guide 34/34)` when everything is fine, or `file: message (rule N)`
for each problem. It is dependency-free and finishes in under a second.

It checks, among other things: that every page follows its template, stays under its line cap, cites
only paths that exist, and is listed in `INDEX.md`; that no two reference pages claim the same code;
that **every guide matches the fingerprint of its source page**; and that a guide cites no path its
source page does not — so a guide can add explanation, but never new facts.

## The three things you will actually do

### 1. You are new and want to understand the codebase

Read `guide/START-HERE.md`, then `guide/architecture.md`, then `guide/testing.md`. After that, read
only the page for whatever you are touching. Do not read all 34.

`INDEX.md` maps tasks to pages.

### 2. You found a page that is wrong or incomplete

This is expected, and fixing it is part of the work — not a favour.

1. Fix **`docs/<name>.md`**, never the guide. Every claim needs a repo path; no line numbers, only
   file plus symbol name.
2. Regenerate **`guide/<name>.md`** to match, and update its `source-hash` to the first 16
   characters of `sha256sum .claude-students/docs/<name>.md`.
3. Run `./.claude-students/check.sh`.
4. Ship both in the same pull request as the code change that taught you this, titled
   `docs(context): <subject>`.

If you skip step 2 the validator will tell you, by name, on the next run.

The easiest way to do step 2 is to ask an AI session: *"`docs/<name>.md` changed — regenerate
`guide/<name>.md` from it and update the hash."*

### 3. You are starting a fresh AI session

You do not have to do anything. `CLAUDE.md` imports `INDEX.md`, so every Claude session already has
the routing table, and `AGENTS.md` points other tools at it.

If a session starts exploring the tree instead of reading a page, say: *"Read
`.claude-students/INDEX.md` and use the page for this area."*

If you ask a session to change documentation here, point it at `MAINTENANCE.md` — rules 16-22 are
the ones about keeping the two layers in step.

## Not upstream

This folder is **ours**, not Gitea's, and is deliberately self-contained so it can be removed in one
step if this fork is ever merged upstream:

```sh
rm -rf .claude-students/
```

Then remove the two pointers: the `@.claude-students/INDEX.md` line in `CLAUDE.md`, and the
`.claude-students/INDEX.md` bullet in `AGENTS.md`. Nothing else refers to this folder — no
`Makefile` target, no CI job.

## Authority

These pages never override the project's own rules. `AGENTS.md`, `CONTRIBUTING.md` and
`docs/guidelines-*.md` at the repository root are authoritative; the pages here summarise them and
point at code. If a page here contradicts one of those, the page here is wrong.

Note that `docs/` means two different things depending on where you are standing: inside this
folder it is our reference pages, at the repository root it is Gitea's own documentation. The guides
say "in this folder" or "Gitea's own" wherever it could be ambiguous.
