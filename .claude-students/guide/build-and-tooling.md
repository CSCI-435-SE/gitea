---
source: docs/build-and-tooling.md
source-hash: 840cbd1e5af507af
verified-at: 187c98fee9
---

<!-- Derived from docs/build-and-tooling.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Building, linting and shipping, in plain English

**In one sentence:** a handful of `make` commands you should run before every push, and a few rules
that will bounce your pull request if you do not know them in advance.
**Come here when:** a lint or build step failed, you are unsure what to run, or you are about to
open a pull request.

## What this is

`make` is the front door to everything: building, running, testing, linting and generating. `make
help` lists every target; the ones that matter day to day are below.

## Why it exists

**Because a project with five people needs the machine to enforce consistency, not the reviewers.**

If formatting and import order were matters of taste, every pull request would collect comments
about them, and reviewers would spend their attention on whitespace instead of logic. So the tools
decide, `make fmt` applies their decision, and nobody argues.

The other half is that **some things cannot be checked by reading the code**. Whether a generated
file is up to date, whether a new file has its licence header, whether you imported a package the
project has banned — those are mechanical, and CI checks them. Knowing which ones exist means you
find out locally in seconds rather than from a failed build twenty minutes later.

## Words you'll meet

- **target** — a named command in the `Makefile`, run as `make <target>`.
- **linter** — a tool that reports suspicious or non-conforming code.
- **formatter** — a tool that rewrites code into a canonical shape.
- **import order** — the required grouping of imports, enforced automatically.
- **banned import** — a package the project forbids, with a replacement.
- **generated file** — one produced by a tool and committed, which must never be hand-edited.
- **Conventional Commits** — the required `type(scope): subject` format for commit and PR titles.

## What's in these files

| Where | What it is for |
| --- | --- |
| `Makefile` | Every build, lint, test and generate target. |
| `package.json`, `pnpm-lock.yaml` | Frontend dependencies. |
| `vite.config.ts` | The frontend build. |
| `.golangci.yml` | Which Go linters run, and their configuration. |
| `eslint.config.ts`, `stylelint.config.ts` | TypeScript and CSS linting. |
| `.editorconfig` | Whitespace rules. |
| `pyproject.toml`, `uv.lock` | The Python tools that lint templates and workflow files. |
| `tools/lint-shell.sh` | Shell script checking. |
| `tools/lint-go-all.go` | The copyright-header check and the Go lint passes. |

## The rules, and why

**Before committing, `make fmt`. Before pushing, `make lint-go`, `make lint-js` or
`make lint-templates`** depending on what you touched.

**`make fmt` rewrites the whole repository, not just your changes.** It covers Go and templates, so
it can reformat a file you never opened. Check `git status` after running it.

**After changing dependencies, `make tidy`** — and justify the change in the pull request. A
dependency is a long-term commitment for the whole project.

**New Go files need a copyright header with the current year**, in an exact form the linter checks.
Copy the header from a neighbouring file.

**Some imports are banned, with replacements.** The standard JSON package is banned in favour of the
project's own; so are a handful of others, each with a stated reason. This is not stylistic — it is
how the project keeps an implementation swappable and avoids known-bad packages.

**Import order is enforced**: standard library, then this project's packages, then a blank line,
then everything else. `make fmt` applies it, so you never have to arrange them by hand — but if you
skip `make fmt`, this is often what fails.

**A pull request that changes the UI must include "after" screenshots**, and "before" ones too
unless it is a new feature. Worth knowing *before* you finish the work, so you capture the "before"
while you still can.

**Whitespace is checked**: no trailing spaces, LF line endings, a final newline.

**Generated files are committed and verified.** The API specification is generated from the swagger
comments, and CI fails when the committed version does not match. Never hand-edit a generated file;
regenerate it.

**Commit messages and PR titles use Conventional Commits**, with an `Assisted-by:` trailer, and
never `Co-Authored-By` or `Signed-off-by`.

## How to actually do it

**Build and run.** `make build`, then `./gitea web`. `make watch` rebuilds as you save and is worth
setting up on day one.

**The commands worth memorising.**

```sh
make fmt               # format Go and templates — run before every commit
make lint-go           # Go linters
make lint-js           # TypeScript linters
make lint-templates    # template linters — needs uv
make lint-editorconfig # whitespace and final newlines, any file type
make tidy              # after changing dependencies
make generate-swagger  # after editing API swagger comments
make help              # everything else
```

**Know what to rebuild.** A `.go` change means rebuilding the binary and restarting it. A `.ts` or
`.css` change means rebuilding the frontend assets. A template or locale change is picked up on
reload in development. `make watch` handles all of them, which is why it is worth the setup.

## Traps, and what they look like

**CI fails on the swagger check and you never touched the specification.** You edited a swagger
comment. Run `make generate-swagger` and commit the regenerated file with your change.

**Lint complains about your imports and you cannot see the problem.** Import order. Run `make fmt`.

**Lint rejects an import that works fine.** It is on the banned list — the message names the
replacement.

**Your new Go file fails lint immediately.** Missing or malformed copyright header. Copy it from a
neighbour and set the current year.

**A reviewer asks for "before" screenshots you can no longer take.** The UI has already changed on
your machine. Capture them before you start.

**`make lint-go` feels painfully slow.** It is, on a cold cache. Run it once before pushing rather
than after every save.

**`make lint-templates` fails saying `uv: No such file or directory`.** The template linters are
Python tools installed into a local environment by `uv`, and you do not have `uv` yet. The error
names `uv` rather than saying "install this", which is why it reads as a broken target.

**A file you never opened shows up in `git status` after `make fmt`.** It formats the whole
repository, so it can pick up drift someone else left behind. Revert those hunks — they are not
yours, and they will confuse the review of your change.

**You commit something that should not be there.** Build artefacts — the binary, the data and custom
directories, the built assets — are ignored, but check `git status` before committing anyway.

**A whole file shows as changed and you only edited one line.** Line endings. The repository expects
LF.

## Where to go next

- `docs/build-and-tooling.md` (in this folder) — the reference page this was written from
- `testing.md` — the test commands
- `architecture.md` — what the layers are that you are linting
