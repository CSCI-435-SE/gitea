---
scope: services/pull, services/gitdiff, services/automerge, services/automergequeue, services/agit
verified-at: c0092050a4
---

# services/pull and services/gitdiff — pull requests, review, diffs

**Read when:** touching PR creation, mergeability, merging, code review comments, or the diff
viewer.
**Not here:** the `PullRequest` row -> `models-issues.md`; issue-side logic -> `services-issue.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `services/pull` | PR lifecycle: create, mergeability checking, merge, review, code comments, branch protection enforcement |
| `services/gitdiff` | parsing a git patch into the structures the diff viewer renders |
| `services/automerge`, `services/automergequeue` | "merge when checks pass" |
| `services/agit` | AGit flow — creating a PR by pushing a magic ref |

## Key files

| Path | What it holds |
| --- | --- |
| `services/pull/pull.go` | `NewPullRequest`, `ChangeTargetBranch`, `AddTestPullRequestTask`, `PushToBaseRepo`, `UpdateRef` |
| `services/pull/merge.go` | `Merge`, `SetMerged`, `MergedManually`, `IsUserAllowedToMerge`, `CheckPullBranchProtections`, `GetDefaultMergeMessage` |
| `services/pull/merge_merge.go`, `merge_squash.go`, `merge_rebase.go`, `merge_ff_only.go` | one file per merge style |
| `services/pull/check.go` | the mergeability queue: `AddPullRequestToCheckQueue`, `InitializePullRequests` |
| `services/pull/merge_tree.go` | conflict detection via `git merge-tree` |
| `services/pull/temp_repo.go` | `createTemporaryRepoForPR` — the scratch clone merges run in |
| `services/pull/patch.go` | `DownloadDiffOrPatch`, `AttemptThreeWayMerge`, `CheckFileProtection` |
| `services/pull/review.go` | `CreateCodeComment`, `SubmitReview`, `DismissReview`, `InvalidateCodeComments` |
| `services/pull/protected_branch.go` | `CreateOrUpdateProtectedBranch` |
| `services/gitdiff/gitdiff.go` | `Diff`, `DiffFile`, `DiffSection`, `DiffLine`, `ParsePatch`, `GetDiffForRender`, `GetDiffForAPI` |
| `services/gitdiff/gitdiff_excerpt.go` | expanding context lines in the viewer |
| `services/gitdiff/highlightdiff.go` | intra-line highlighting |

## Conventions & invariants

- **Mergeability is computed asynchronously.** Nothing recomputes it inline: a change pushes the PR
  id onto the worker queue in `check.go` via `AddPullRequestToCheckQueue`, and the result is
  written back to `PullRequest.Status`. The queue is unique-keyed, so pushing the same id twice is
  free. Code that needs a fresh answer must handle "not computed yet" rather than blocking.
- Merges do not run in the live repository. `createTemporaryRepoForPR` builds a scratch clone and
  the merge-style files operate there; only on success is the result pushed. A merge path that
  writes to the real repo directly is a bug.
- Merge styles are `repo_model.MergeStyle` values (`MergeStyleMerge`, `MergeStyleRebase`,
  `MergeStyleRebaseMerge`, `MergeStyleSquash`, `MergeStyleFastForwardOnly`) and each must be
  allowed by the repo's PR unit config — check with the unit accessors
  (`models-repo-and-git.md`), never by string comparison.
- `Merge` is not the permission check. `IsUserAllowedToMerge` and `CheckPullBranchProtections` sit
  beside it in `merge.go` but are separate calls the caller must make itself.
- **Review comments are anchored to a commit, not to a line of the current head.** `CreateCodeComment`
  takes `latestCommitID` and a `treePath`; when the branch moves, `InvalidateCodeComments` marks
  the ones that no longer apply. A comment feature that stores only a line number will drift.
- `GetDiffForRender` and `GetDiffForAPI` are deliberately separate — the render path carries
  highlighting, escaping and a repo link for building URLs. Do not use the API one to build HTML.
- `notify_service.*` is called after the transaction, exactly as in `services-issue.md`.

## Recipes

**Add something to the diff viewer.** Extend `DiffFile`/`DiffSection`/`DiffLine` in
`services/gitdiff/gitdiff.go`, populate it in `ParsePatch` or `GetDiffForRender`, then render it in
the repo diff templates and wire the frontend behaviour (`frontend-js.md`).

**Add a merge style.** A `MergeStyle` constant and its unit-config flag in `models/repo`, a
`merge_<style>.go` implementation, a branch in `Merge`, the repo settings UI, and the locale keys.

**Change when a PR becomes mergeable.** The check itself is in `check.go` / `merge_tree.go`. Add
the trigger by calling `AddPullRequestToCheckQueue`, not by computing inline.

## Gotchas

- `services/pull` is the logic; the `PullRequest` *rows* are in `models/issues/pull.go`. The
  package named `models/pull` is only automerge and review state (`models-issues.md`).
- A PR's issue fields (title, labels, comments) go through `services/issue`. Only PR-specific
  behaviour lives here, so a "PR change" often touches both packages.
- `AddTestPullRequestTask` reads as a test helper. It is production code that queues PR rechecks
  after a push.
- Diff size is capped by the `maxLines` / `maxFiles` arguments to `ParsePatch`; a large diff is
  truncated rather than failing, so check the `Diff` flags before assuming completeness.

## Related

- `services-issue.md` — the shared issue behaviour and the notify convention
- `models-issues.md` — `PullRequest`, `Review`, `Comment`
- `services-repository.md` — the push path that triggers PR rechecks
