---
scope: services/repository, services/git
verified-at: c0092050a4
---

# services/repository — repository lifecycle, branches, file edits, push handling

**Read when:** creating/forking/transferring/deleting a repo, editing files through the web UI,
changing branch operations, or working on what happens after a push.
**Not here:** the `Repository` row -> `models-repo-and-git.md`; the git CLI wrapper -> `modules/git`; PR effects of a push -> `services-pull-and-gitdiff.md`.

## Responsibilities

| Area | Files |
| --- | --- |
| Lifecycle | `create.go`, `fork.go`, `adopt.go`, `transfer.go`, `delete.go`, `generate.go`, `template.go`, `migrate.go` |
| Branches | `branch.go` — create, rename, delete, sync-to-DB, default branch |
| File editing | `files/` — `update.go`, `upload.go`, `content.go`, `tree.go`, `patch.go`, `cherry_pick.go`, `temp_repo.go` |
| Push handling | `push.go` — `PushUpdates`, `PushUpdateAddDeleteTags` |
| Settings / access | `setting.go`, `collaboration.go`, `repo_team.go`, `hooks.go` |
| Derived data | `commitstatus/`, `archiver/`, `gitgraph/`, `contributors_graph.go`, `license.go`, `lfs.go`, `avatar.go` |
| `services/git` | `commit.go` (signature and status enrichment), `compare.go` (`GetCompareInfo`) |

## Key files

| Path | What it holds |
| --- | --- |
| `services/repository/create.go` | `CreateRepositoryDirectly` |
| `services/repository/fork.go` | `ForkRepository`, `ConvertForkToNormalRepository`, `FindForks` |
| `services/repository/transfer.go` | `ChangeRepositoryName`, `StartRepositoryTransfer`, `AcceptTransferOwnership` |
| `services/repository/delete.go` | `DeleteRepositoryDirectly` |
| `services/repository/branch.go` | `CreateNewBranch`, `RenameBranch`, `DeleteBranch`, `CanDeleteBranch`, `SyncBranchesToDB`, `SetRepoDefaultBranch` |
| `services/repository/push.go` | `PushUpdates` — the fan-out after every push |
| `routers/private/hook_post_receive.go` | the caller: `SyncBranchesToDB` then `PushUpdates` |
| `services/repository/files/update.go` | the web file editor's commit path |
| `services/git/compare.go` | `GetCompareInfo` — powers compare and PR views |

## Conventions & invariants

- **The push path is two steps, both driven by `routers/private/hook_post_receive.go`:** it calls
  `SyncBranchesToDB` to update the branch rows, then `PushUpdates` for the fan-out.
  `PushUpdates` runs `issue_service.UpdateIssuesCommit` (`services-issue.md`), queues PR rechecks
  with `pull_service.AddTestPullRequestTask` in a goroutine
  (`services-pull-and-gitdiff.md`), and calls `notify_service.PushCommits` / `CreateRef` /
  `DeleteRef` — which is how webhooks and Actions get triggered, since both register themselves as
  notifiers. New "on push" behaviour belongs in `PushUpdates`, not bolted onto a router.
- **Two sources of truth, deliberately.** Git on disk is authoritative for refs and commits;
  `models/git` rows are a queryable cache. When they disagree, git wins and the rows need
  resyncing — `AddAllRepoBranchesToSyncQueue` does that in bulk.
- File edits and patch application run in a temporary clone (`files/temp_repo.go`), the same
  pattern as PR merges. Never mutate a live repository's working state.
- The `*Directly` suffix (`CreateRepositoryDirectly`, `DeleteRepositoryDirectly`) means "no
  permission checks, no queueing — do it now". Callers must have done the authorisation. Do not
  reach for these from a router.
- Renames and transfers insert a `Redirect` row so old URLs keep working
  (`models-repo-and-git.md`).
- Branch deletion has a separate check function (`CanDeleteBranch`) from the action (`DeleteBranch`);
  call both.

## Recipes

**Do something on every push.** Add it to `PushUpdates` in `push.go`, guarded by whether it applies
to the ref type being pushed. Keep it cheap or hand it to a queue — this path is in the critical
latency of `git push`, which is why the existing PR recheck is fired in a goroutine.

**Add a file-editing operation.** Model it on `files/update.go`: build the temp repo, apply the
change, commit, push back, then let `PushUpdates` do the downstream work.

**Create a repo variant.** `CreateRepositoryDirectly` is the base; `ForkRepository`,
`AdoptRepository` and `GenerateGitContent` are the three existing variations to copy from.

## Gotchas

- Repo creation is not one transaction end to end — it writes rows *and* creates a directory on
  disk. Failure handling has to clean up both, which is why the existing functions look defensive.
- `services/git` is two small files of enrichment helpers. The actual git plumbing is
  `modules/git` / `modules/gitrepo`; do not add git command construction here.
- `SyncBranchesToDB` takes a `getCommit` callback so it can avoid reopening the repository — pass
  the one you already have rather than opening another.
- A repo with `IsEmpty` true has no default branch commit; code that assumes one panics on a
  freshly created repo.

## Related

- `models-repo-and-git.md` — the rows this package maintains
- `services-pull-and-gitdiff.md` — what a push does to open PRs
- `services-issue.md` — commit-message issue references
