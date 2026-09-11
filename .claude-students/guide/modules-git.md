---
source: docs/modules-git.md
source-hash: 5fa10b230ac38231
verified-at: c0092050a4
---

<!-- Derived from docs/modules-git.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Actually running git, in plain English

**In one sentence:** Gitea mostly works by running the real `git` program and reading its output,
and there are two packages for that — one low-level, one that knows where repositories live, and you
almost always want the second.
**Come here when:** you need to read commits, branches or file contents, or run any git operation.

## What this is

Two layers.

`modules/git` is the low-level one: building git command lines, running them, and parsing what comes
back into commits, trees, blobs and refs.

`modules/gitrepo` sits on top and knows where repositories are on disk. Its functions take a
repository and work the path out themselves.

Both are leaf packages: they depend on almost nothing, and nothing in `models/` or `services/` may
be imported by them (`architecture.md`).

## Why it exists

**Gitea does not reimplement git.** It runs the actual `git` command and reads the output. That is
why `git` must be installed on the server, not just on your machine while building.

This works well — git is fast and correct — but it means every operation is a subprocess with
arguments, and **arguments assembled from user input are a security problem**. A branch name is
user input. A file path is user input. Build a command by pasting strings together and you have
handed someone a way to pass extra arguments to git.

So commands are built through a typed helper rather than string concatenation, and that is not a
style preference.

**Why two packages?** Because knowing where a repository lives is Gitea's business, not git's. The
lower layer takes a path. The upper layer takes a *repository* and computes the path from
configuration. If you compute paths yourself, your code breaks the day the storage layout changes —
and it duplicates a decision that should exist in one place.

## Words you'll meet

- **shell out** — running an external program instead of doing the work in Go.
- **subprocess** — that external program while it runs.
- **ref** — a branch or tag name.
- **blob** — the contents of a file at a particular commit.
- **tree** — a directory listing at a particular commit.
- **build tag** — a marker making a file compile only in certain builds.
- **batch reader** — a long-lived git process kept open to answer many questions quickly.
- **handle** — an open connection to a repository, which must be closed.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/git/gitcmd/command.go` | Building a git command safely. Every git invocation goes through here. |
| `modules/git/repo_base_nogogit.go`, `repo_base_gogit.go` | Opening a repository, one per build mode. |
| `modules/git/commit.go`, `blob.go`, `tree.go`, `diff.go`, `grep.go` | The object types and operations. |
| `modules/git/catfile_batch.go`, `catfile_batch_reader.go` | The long-lived readers used when many objects are needed. |
| `modules/gitrepo/gitrepo.go` | The repository interface, and opening one by repository rather than path. |
| `modules/gitrepo/commit.go`, `branch.go`, `ref.go`, `merge_tree.go`, `push.go`, `clone.go` | The repository-aware operations. |

## The rules, and why

**Prefer `modules/gitrepo`.** Its functions take a repository and resolve the path themselves. Code
that builds a filesystem path by hand is doing that package's job, and will break when the layout
changes.

**The repository interface is deliberately tiny** — one method returning a relative path like
`owner/name.git`. The database's repository type satisfies it without this package knowing anything
about the database. That is how a leaf package can usefully take "a repository" without breaking the
dependency direction.

**Never build a git command as a string.** Use the command builder with its typed arguments. This is
what keeps user-supplied text out of the argument list.

**There are two build modes, and this one bites people.** Some files end `_gogit.go` and others
`_nogogit.go`, with build tags making each compile in only one mode. The default build is the
`nogogit` one, which shells out to git. The `gogit` mode is used for the Windows release binary and
is exercised in CI.

So a function with a mode-specific implementation needs **both** files. Write only the one your
local build uses and everything passes locally, then CI fails on a build you never ran.

**Batch readers are processes and must be closed**, and must not be shared between goroutines.
Forgetting to close one leaks a process per request.

**Git on disk is the truth; the database rows are a cache** (`models-repo-and-git.md`).

## How to actually do it

**Run a git operation.** Look in `modules/gitrepo` first. If it is not there, add it there, taking a
repository, implemented with the command builder.

**Read commits.** Open the repository through `modules/gitrepo`, use the commit helpers, and close
what you opened.

But on a repository web page, **the handle already exists**. The middleware that loads the
repository also opens it and caches it on the request, so use `ctx.Repo.GitRepo` rather than opening
a second one (`services-context.md`).

**Add a git subcommand.** Check that the verb is in the allowed list, then build the command with
the command builder.

## Traps, and what they look like

**The page is slower than it should be, with nothing obviously wrong.** You opened a second handle
on a repository the request already had open. There is a helper whose entire purpose is to avoid
this, and the cost is invisible in the code — it just costs a process each time.

**It compiles locally and fails CI with errors about missing functions.** You added a function to
only one of the two build-mode files.

**Git fails at runtime with something about an unknown option.** Gitea runs the `git` on the
server's `PATH`, and newer subcommands need a newer git. This is a runtime dependency, not a build
one — it works on your machine and fails on a server with an older git.

**A process leak under load.** A batch reader was not closed.

**You want to import something from `models/` into `modules/git`.** You cannot, and the urge means
the function belongs in `modules/gitrepo` or in a service instead.

## Where to go next

- `docs/modules-git.md` (in this folder) — the reference page this was written from
- `services-repository.md` — the code that changes repositories
- `models-repo-and-git.md` — the cached rows, and which copy is authoritative
- `architecture.md` — why this package may not import upward
