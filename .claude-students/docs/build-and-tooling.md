---
scope: Makefile, package.json, vite.config.ts, .golangci.yml, eslint.config.ts, stylelint.config.ts
verified-at: c0092050a4
---

# build and tooling — targets, linters, generated files

**Read when:** a lint or build step fails, or you are unsure which command to run before pushing.
**Not here:** the test tiers -> `testing.md`; first-time setup -> `STUDENTS.md`.

## Responsibilities

The `Makefile` is the entry point for everything. `make help` lists every target; the ones below
are the ones a course task actually needs.

## Key files

| Path | What it holds |
| --- | --- |
| `Makefile` | every build, lint, test and generate target |
| `package.json`, `pnpm-lock.yaml` | frontend dependencies |
| `vite.config.ts` | the frontend build |
| `.golangci.yml` | Go linter configuration |
| `eslint.config.ts`, `stylelint.config.ts` | JS/TS and CSS linting |
| `.editorconfig` | whitespace rules enforced by `make lint-editorconfig` |
| `tools/lint-shell.sh` | shellcheck, run over `git ls-files '*.sh'` |
| `tools/lint-go-all.go` | `lintGoHeader` — the copyright-header check, plus the golangci-lint passes |

## Conventions & invariants

- **Before committing:** `make fmt`. **Before pushing:** `make lint-go` for Go changes,
  `make lint-js` for TypeScript (`AGENTS.md`).
- **After any `go.mod` change:** `make tidy`, and justify the dependency change in the PR
  description (`docs/guidelines-backend.md`).
- New `.go` files need a copyright header with the current year (`AGENTS.md`). `lintGoHeader` in
  `tools/lint-go-all.go` enforces the exact `// Copyright <year> The Gitea Authors...` +
  `// SPDX-License-Identifier:` form.
- `.golangci.yml` `depguard` **bans imports**: `encoding/json` (use `modules/json`), `io/ioutil`,
  `golang.org/x/exp`, `gopkg.in/ini.v1` (use the config system), and `modules/git/internal`.
- `gci` enforces import order: standard library, then `prefix(gitea.dev)`, then a blank line, then
  everything else. `gofumpt` with `extra-rules` formats on top of `gofmt`. `make fmt` applies both.
- `CONTRIBUTING.md` requires **"after" screenshots on any PR that changes the UI**, and "before"
  screenshots too unless it is a new feature.
- No trailing whitespace, LF endings, final newline — `.editorconfig`, checked by
  `make lint-editorconfig`.
- **Generated files are committed and verified in CI.** `templates/swagger/v1_json.tmpl` comes from
  `make generate-swagger`, and `make swagger-check` fails when it is stale
  (`routers-api-v1.md`). Never hand-edit a generated file.
- `make lint-md` only checks root-level `*.md`. `make lint-shell` shellchecks every tracked `.sh`,
  which includes `.claude-students/check.sh`.
- Conventional Commits for commit messages and PR titles; add an
  `Assisted-by: AGENT_NAME:MODEL_VERSION` trailer; never add `Co-Authored-By` or `Signed-off-by`
  (`AGENTS.md`).

## Recipes

**Build and run.** `make build` then `./gitea web`. `make watch` rebuilds on change during
development.

**The targets worth memorising.**

```sh
make fmt            # format Go
make lint-go        # Go linters
make lint-js        # TypeScript linters
make tidy           # after go.mod changes
make generate-swagger  # after editing API swagger comments
make help           # everything else
```

**Frontend only.** `pnpm install` then `pnpm exec vite build`; no Go rebuild needed for CSS or TS
changes if you are running `make watch`.

**The rebuild loop.** A `.go` change needs the binary rebuilt and restarted. A `.ts` or `.css`
change needs the frontend assets rebuilt. A `.tmpl` or locale change is picked up on reload in
development. `make watch` covers all of them (`docs/development.md`).

## Gotchas

- `make lint-go` is slow on a cold cache. Run it once before pushing rather than on every save.
- A swagger comment edit that is not followed by `make generate-swagger` passes locally and fails
  CI.
- Build artifacts (`gitea`, `data/`, `custom/`, `public/assets/*`) are gitignored — check
  `git status` before committing (`STUDENTS.md` §7).
- The repository expects LF endings; a CRLF commit shows as a whole-file diff.

## Related

- `testing.md` — the test targets
- `architecture.md` — what the layers are that you are linting
