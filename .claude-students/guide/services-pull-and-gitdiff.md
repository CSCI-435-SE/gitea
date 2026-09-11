---
source: docs/services-pull-and-gitdiff.md
source-hash: 8b2526b516737ae9
verified-at: c0092050a4
---

<!-- Derived from docs/services-pull-and-gitdiff.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Pull requests, merging and the diff viewer, in plain English

**In one sentence:** everything that makes a pull request work — deciding whether it can merge,
actually merging it, and turning a patch into the side-by-side view you leave comments on.
**Come here when:** you are touching PR creation, merging, code review, or the diff display.

## What this is

Four packages doing related jobs:

- `services/pull` — the pull request lifecycle: creating, checking whether it can merge, merging,
  reviews and code comments.
- `services/gitdiff` — turning a raw git patch into the structure the diff viewer renders.
- `services/automerge` and `services/automergequeue` — "merge this once the checks pass".
- `services/agit` — creating a pull request by pushing to a special ref, without a web UI.

## Why it exists

Three hard problems live here, and each one explains a design decision you will otherwise find
strange.

**Can this pull request merge?** Answering it means actually attempting the merge. That is far too
slow to do while somebody waits for a page. So Gitea never computes it inline: a change puts the
pull request on a queue, a worker figures it out, and the answer is stored. The page reads the
stored answer, which may be "not worked out yet".

**Merging without breaking the repository.** A merge can conflict or fail halfway. Doing that in the
real repository could leave it in a broken state that everyone else then clones. So merges happen
in a scratch copy, and only a successful result gets pushed.

**Review comments outlive the code they point at.** You comment on line 40. The author pushes a fix
and line 40 is now something else entirely. If the comment only remembered "line 40" it would now be
attached to unrelated code. So comments remember which *commit* they were made against.

## Words you'll meet

- **mergeable** — whether this PR can merge cleanly right now.
- **queue** — a list of work a background worker processes, so slow jobs do not block a page.
- **unique queue** — one that ignores a duplicate entry, so adding the same item twice is free.
- **merge style** — how the merge is performed: a merge commit, rebase, squash, fast-forward.
- **conflict** — two changes to the same lines that git cannot combine.
- **temporary / scratch repository** — a throwaway clone used to try something risky.
- **branch protection** — rules about who may merge and what must pass first.
- **patch** — the text description of a change, which the diff viewer parses.
- **hunk / section** — one contiguous block of changed lines.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/pull/pull.go` | Creating a PR, changing its target branch, and queueing rechecks after a push. |
| `services/pull/merge.go` | The merge itself, plus the permission and branch-protection checks that sit beside it. |
| `services/pull/merge_merge.go`, `merge_squash.go`, `merge_rebase.go`, `merge_ff_only.go` | One file per merge style. |
| `services/pull/check.go` | The mergeability queue. |
| `services/pull/merge_tree.go` | Detecting conflicts without a full merge. |
| `services/pull/temp_repo.go` | Building the scratch clone merges run in. |
| `services/pull/patch.go` | Producing diffs and patches, and three-way merging. |
| `services/pull/review.go` | Code comments, submitting and dismissing reviews. |
| `services/pull/protected_branch.go` | Creating and updating protection rules. |
| `services/gitdiff/gitdiff.go` | The diff structures and the parser. |
| `services/gitdiff/gitdiff_excerpt.go` | Expanding the context lines around a change. |
| `services/gitdiff/highlightdiff.go` | Highlighting what changed *within* a line. |

## The rules, and why

**Mergeability is never computed inline.** A change pushes the pull request's id onto a queue; a
worker computes the answer and writes it back onto the row. The queue ignores duplicates, so pushing
the same id twice costs nothing — which means you should enqueue freely rather than trying to be
clever about when.

The consequence for your code: **there may be no answer yet.** Anything that reads mergeability has
to cope with "not computed", not block waiting for it.

**Merges run in a scratch clone.** A temporary repository is built, the merge is attempted there,
and only a successful result is pushed. A merge path that writes to the real repository directly is
a bug, however much simpler it looks.

**Merge styles are typed values, and must be permitted.** There are five, and each must be allowed
by the repository's pull-request settings. Check with the unit accessors, never by comparing
strings.

**Merging is not the permission check.** `Merge` merges. `IsUserAllowedToMerge` and
`CheckPullBranchProtections` sit next to it in the same file but are **separate calls the caller has
to make**. Calling `Merge` alone merges without checking anything.

**Review comments are anchored to a commit, not a line number.** `CreateCodeComment` records which
commit and which file path. When the branch moves, `InvalidateCodeComments` marks the ones that no
longer apply. Any review feature that stores only a line number will drift onto unrelated code the
first time the author pushes.

**There are two diff functions and they are not interchangeable.** `GetDiffForRender` carries
highlighting, escaping and the repository link needed to build URLs. `GetDiffForAPI` does not. Using
the API one to build HTML produces output that is unhighlighted and, worse, unescaped.

**Notifications happen after the transaction**, exactly as in `services-issue.md`.

## How to actually do it

**Add something to the diff viewer.** Extend the diff structures in `services/gitdiff/gitdiff.go`,
fill your new field in the parser or the render path, then display it in the repository's diff
templates and wire up any browser behaviour.

**Add a merge style.** Five places: the style constant and its settings flag in `models/repo`, an
implementation file, a branch in the merge function, the repository settings UI, and the locale
keys.

**Change when a pull request becomes mergeable.** The logic lives in `check.go` and
`merge_tree.go`. To make something *trigger* a recheck, call the enqueue function — do not compute
it where you are.

## Traps, and what they look like

**You go looking for pull requests in `models/pull` and it is nearly empty.** The rows are in
`models/issues/pull.go`. `models/pull` holds only automerge and review state. This package,
`services/pull`, is the logic.

**Your "pull request change" half works.** A PR's title, labels and comments are issue fields, so
they go through `services/issue`. Only PR-specific behaviour lives here. Many changes touch both.

**You assume `AddTestPullRequestTask` is test-only and skip it.** It reads like a test helper. It is
production code — the thing that queues PR rechecks after a push.

**A large diff renders incompletely and nothing errors.** The parser caps how many lines and files
it will process. Over the cap it truncates rather than failing, so check the flags on the result
before treating it as the whole diff.

**Something merged that should not have.** The caller called the merge function without first
calling the permission and branch-protection checks.

**Review comments end up on the wrong lines after a push.** Something stored a line number instead
of anchoring to a commit.

## Where to go next

- `docs/services-pull-and-gitdiff.md` (in this folder) — the reference page this was written from
- `services-issue.md` — the issue behaviour pull requests share
- `models-issues.md` — the pull request, review and comment rows
- `services-repository.md` — the push that triggers all those rechecks
