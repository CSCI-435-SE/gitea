---
source: docs/services-repository.md
source-hash: a783b1663ae7f937
verified-at: c0092050a4
---

<!-- Derived from docs/services-repository.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Repositories, branches and what happens on push, in plain English

**In one sentence:** creating, forking, transferring and deleting repositories, editing files through
the web UI, and — most importantly — everything that fires the moment somebody runs `git push`.
**Come here when:** you are working on repository lifecycle, branch operations, the file editor, or
anything that should happen after a push.

## What this is

The business logic for repositories, grouped by job: lifecycle, branches, file editing, push
handling, settings and access, and various derived data like contributor graphs and archives.

Alongside it sits `services/git`, two small files that enrich commit data with signature
verification and CI status.

The centre of gravity is the push path, because that is where most of Gitea's automatic behaviour
starts.

## Why it exists

**Because a push is not just a file transfer.** When you run `git push`, the user expects git to
accept their commits. What they also expect, without thinking about it: the branch list updates, any
`fixes #12` in the commit message closes issue 12, open pull requests from that branch recheck
whether they still merge, webhooks fire, and CI starts.

None of that is git. It is all Gitea, reacting.

That reaction has to live somewhere shared, because a push can arrive over HTTP or over SSH, and
both must behave identically. If the reaction were wired into the HTTP handler, pushing over SSH
would silently skip it.

It also has to be **fast**, because the user is sitting there watching `git push` not finish. That
is why the slower parts are handed to background workers rather than done inline.

## Words you'll meet

- **push path** — everything that runs when a push arrives.
- **hook** — a script git runs at a particular moment. Gitea installs its own.
- **post-receive** — the hook that fires after a push is accepted.
- **ref** — a branch or a tag.
- **fan-out** — one event causing many separate reactions.
- **notifier** — a subscriber to those events. Webhooks and CI are both notifiers.
- **goroutine** — Go's lightweight background task, used here to avoid making the user wait.
- **temporary clone** — a scratch copy used for risky operations.
- **empty repository** — one created but never pushed to, so it has no commits at all.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/repository/create.go` | Creating a repository. |
| `services/repository/fork.go` | Forking, and converting a fork back to a normal repository. |
| `services/repository/transfer.go` | Renaming, and moving a repository to a new owner. |
| `services/repository/delete.go` | Deletion. |
| `services/repository/branch.go` | Creating, renaming and deleting branches, plus keeping the branch rows in step with git. |
| `services/repository/push.go` | **The fan-out after every push.** The most important file here. |
| `routers/private/hook_post_receive.go` | The caller, which runs when the git hook fires. |
| `services/repository/files/update.go` | The web file editor's commit path. |
| `services/git/compare.go` | Comparing two refs — what powers the compare and pull request views. |

Also here: adoption and generation from templates, migration from other hosts, settings,
collaborators, hooks, and the derived data — commit statuses, archives, commit graphs, contributor
statistics, licences and LFS.

## The rules, and why

**The push path is two steps, and the caller is a git hook handler.**
`routers/private/hook_post_receive.go` runs when git finishes accepting a push. It first calls
`SyncBranchesToDB` to bring the branch rows up to date, then `PushUpdates` for everything else.

`PushUpdates` is where the fan-out happens. It:

- scans commit messages for issue references and acts on them,
- queues rechecks for open pull requests — **in a goroutine**, deliberately, so the user's `git
  push` does not wait for it,
- fires the push and ref notifications, which is how webhooks and CI find out, since both are
  registered notifiers.

So **new "on push" behaviour belongs in `PushUpdates`**, not bolted onto whichever router you
happen to be reading. Put it in a router and it fires for pushes over one protocol and not the
other.

**Git on disk is the truth; the database rows are a cache.** When they disagree, git wins and the
rows need resyncing. There is a bulk resync for exactly that.

**File edits happen in a temporary clone**, the same pattern as pull request merges. Never modify a
live repository's working state — other people are cloning from it.

**A `*Directly` suffix means "no checks, no queueing, do it now".** `CreateRepositoryDirectly` and
`DeleteRepositoryDirectly` assume the caller already did the authorisation. Do not call them from a
router; that is how you get an endpoint that deletes repositories without asking whether you may.

**Renames and transfers leave a redirect behind**, so existing links keep working.

**Branch deletion is two calls.** `CanDeleteBranch` asks; `DeleteBranch` does. The action does not
perform the check for you.

## How to actually do it

**Make something happen on every push.** Add it to `PushUpdates`, guarded by whether it applies to
the kind of ref being pushed. Then keep it cheap, or hand it to a queue. This code runs while the
user waits for `git push` to return — which is precisely why the existing pull request recheck is
fired into a goroutine rather than awaited.

**Add a file-editing operation.** Follow `files/update.go`: build the temporary clone, apply the
change, commit, push back, and then let `PushUpdates` do all the downstream work. You do not have to
send the notifications yourself — pushing back triggers them.

**Create a new kind of repository creation.** The base is `CreateRepositoryDirectly`. Forking,
adopting an existing directory, and generating from a template are the three existing variations to
model yours on.

## Traps, and what they look like

**Your new push behaviour works over HTTP and not SSH, or the reverse.** It went into a router
instead of `PushUpdates`.

**`git push` became noticeably slower.** Something expensive was added to the push path and runs
inline. Hand it to a queue or a goroutine.

**A freshly created repository crashes your code.** An empty repository has no commits and therefore
no default branch commit. Anything assuming one panics on a repository nobody has pushed to yet —
and that is exactly the state a repository is in right after creation.

**Repository creation half-failed and left a mess.** Creation is not one transaction: it writes
database rows *and* creates a directory on disk, and those cannot be rolled back together. That is
why the existing code looks so defensive about cleanup. Match that care.

**You add git command-building to `services/git`.** That package is two small files of enrichment
helpers. The actual git plumbing belongs in `modules/git` and `modules/gitrepo`.

**You open a second handle on a repository that is already open.** `SyncBranchesToDB` takes a
callback for fetching commits precisely so it can reuse an open repository. Pass the one you have.

## Where to go next

- `docs/services-repository.md` (in this folder) — the reference page this was written from
- `models-repo-and-git.md` — the rows this package keeps up to date
- `services-pull-and-gitdiff.md` — what a push does to open pull requests
- `services-issue.md` — how `fixes #12` in a commit message closes an issue
