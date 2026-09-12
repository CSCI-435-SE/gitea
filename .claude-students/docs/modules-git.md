---
scope: modules/git, modules/gitrepo
verified-at: c0092050a4
---

# modules/git and modules/gitrepo — talking to git

**Read when:** running a git operation, reading commits/refs/blobs, or building a git command.
**Not here:** the DB rows caching git state -> `models-repo-and-git.md`; repo lifecycle -> `services-repository.md`; diff rendering -> `services-pull-and-gitdiff.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `modules/git` | the low-level git layer: command construction (`gitcmd/`), repository/commit/blob/tree/ref types, cat-file batch readers, attributes, grep, GPG, language stats |
| `modules/gitrepo` | the Gitea-aware layer: operations taking a `Repository` (something with a `RelativePath()`) and resolving it under `setting.RepoRootPath` |

`modules/git` is a leaf package by design — it is at the far right of the dependency direction
(`architecture.md`) and must not import models or services.

## Key files

| Path | What it holds |
| --- | --- |
| `modules/git/gitcmd/command.go` | `NewCommand` — every git invocation is built here |
| `modules/git/repo_base_nogogit.go`, `repo_base_gogit.go` | `OpenRepository` for each build mode |
| `modules/git/commit.go`, `blob.go`, `tree.go`, `diff.go`, `grep.go` | the object and operation types |
| `modules/git/catfile_batch.go`, `catfile_batch_reader.go` | the long-lived `cat-file --batch` readers |
| `modules/gitrepo/gitrepo.go` | the `Repository` interface and `OpenRepository` |
| `modules/gitrepo/commit.go`, `branch.go`, `ref.go`, `merge_tree.go`, `push.go`, `clone.go` | the repo-aware operations |

## Conventions & invariants

- **Prefer `modules/gitrepo` over `modules/git`.** Its functions take a `Repository` and resolve
  the on-disk path themselves. Code that builds a filesystem path by hand is doing
  `gitrepo`'s job and will break when the storage layout changes.
- The `Repository` interface is deliberately tiny — a single `RelativePath() string` returning a
  unix-style `"owner/name.git"`. `repo_model.Repository` satisfies it, which is how the layering
  stays one-directional.
- **Never build a git command as a string.** Use `gitcmd.NewCommand(...)` with its typed arguments;
  it is what keeps user input out of the argument list.
- **Two build modes.** Files ending `_gogit.go` and `_nogogit.go` carry `//go:build gogit` and
  `//go:build !gogit` tags. The default build is *nogogit* (shelling out to git); the `gogit` tag
  is used for the Windows release binary (see `Makefile`) and exercised in CI with
  `TAGS="bindata gogit"`. A function with a mode-specific implementation needs **both** files, or
  the build you are not running locally breaks.
- `cat-file --batch` readers are long-lived processes. They must be closed, and they must not be
  shared across goroutines.
- Git is the source of truth for refs and commits; the rows in `models/git` are a cache
  (`models-repo-and-git.md`).

## Recipes

**Run a git operation on a repo.** Look for it in `modules/gitrepo` first. If it is missing, add it
there taking a `Repository`, implemented with `gitcmd.NewCommand`.

**Read commits.** `gitrepo.OpenRepository(ctx, repo)` for a `*git.Repository`, then the commit
helpers, and close what you open. On a repo web route the handle already exists: `RepoAssignment`
fills `ctx.Repo.GitRepo` through `gitrepo.RepositoryFromRequestContextOrOpen`, which caches it on
the request — reuse `ctx.Repo.GitRepo` instead of opening a second one (`services-context.md`).

**Add a git subcommand.** Check `modules/git/cmdverb.go` for the allowed verb, then build it with
`gitcmd.NewCommand`.

## Gotchas

- Opening a second `git.Repository` for a repo the request already has open is a common and
  invisible performance bug — `RepositoryFromRequestContextOrOpen` exists precisely to avoid it.
- A function added only to the `_nogogit.go` file compiles locally and fails the `gogit` build.
- `modules/git` must stay import-light. Needing a model here means the function belongs in
  `modules/gitrepo` or in a service instead.
- Git operations shell out to the `git` binary, so `git` must be on `PATH` at runtime
  (`STUDENTS.md` §7) and the version matters for newer subcommands.

## Related

- `services-repository.md` — the callers that mutate repositories
- `models-repo-and-git.md` — the cached rows
- `architecture.md` — why this package may not import upward
