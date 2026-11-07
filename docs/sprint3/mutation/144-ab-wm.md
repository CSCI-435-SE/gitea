# Mutation testing: Issue #144 -- Show which dependencies block a pull request from merging, and whether each is open, merged or closed

**PR(s):** #150

**Tool:** go-mutesting

**Config:** no config file; the options are the command's flags ([setup](../../../.claude-students/docs/mutation-testing.md))

**Command:** `GOTOOLCHAIN=go1.26.4 go-mutesting --exec-timeout=30 --match='^(MergeBlockedByDependencies|GetMergeBlockingDependencies|mergeBlockingDependencyCond|prepareMergeBoxDependencies|prepareMergeBoxInfoItems|prepareMergeBoxFormProps|prepareMergeBox|prepareIssueViewSidebarDependency|loadDependencyPullRequests)$' models/issues/dependency.go routers/web/repo/pull_merge_box.go routers/web/repo/pull_merge_form.go routers/web/repo/issue_view.go`

**Reports:** [before](reports/144-ab-wm-before.txt) · [after](reports/144-ab-wm-after.txt)

**Mutation score:** before 23.6% → after 29.5%

## Surviving mutants

The run mutates the four Go files the PR changed, limited to the nine functions it changed. Of the 258 mutants, 38 are on lines the PR added or changed: 26 killed, 12 survived. All 12 are below. The other 184 survivors are in the unchanged code of the same functions.

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `routers/web/repo/issue_view.go:483` | `branch/if`: removes `ctx.ServerError("LoadPullRequests", err)` and `return` | Test gap | Differs only when loading the dependencies' pull requests fails in the database. No test makes it fail, and no AC covers that case. |
| `routers/web/repo/issue_view.go:483` | `statement/remove`: removes `ctx.ServerError("LoadPullRequests", err)` | Test gap | The same error path: the page would carry on instead of showing the error page. |
| `routers/web/repo/issue_view.go:489` | `branch/if`: removes the `return` after the sidebar's permission check | Test gap | Differs only when a dependency's permission lookup fails, which no test causes. |
| `routers/web/repo/issue_view.go:956` | `statement/remove`: removes the call `prInfo.prepareMergeBoxDependencies(ctx)` (line 957) | Test gap | The merge box would never mention dependencies (AC1, AC2). The unit tests call that step directly, and the browser test that notices is not run by the tool, which runs only the package's Go tests. |
| `routers/web/repo/issue_view.go:959` | `branch/if`: removes the `return` after that call | Test gap | Error path: the box would keep being built after an error page was written. |
| `routers/web/repo/pull_merge_box.go:97` | `branch/if`: removes `ctx.ServerError("GetMergeBlockingDependencies", err)` and `return` | Test gap | Differs only when the dependency query fails in the database. |
| `routers/web/repo/pull_merge_box.go:97` | `statement/remove`: removes `ctx.ServerError("GetMergeBlockingDependencies", err)` | Test gap | The same error path. |
| `routers/web/repo/pull_merge_box.go:106` | `branch/if`: removes the `return` after the permission check | Test gap | Differs only when a permission lookup fails. |
| `routers/web/repo/pull_merge_box.go:109` | `arithmetic/base`: the list's capacity `len(canRead)+1` becomes `len(canRead)-1` | Test gap | When every blocking dependency is unreadable, the capacity is -1 and the page crashes. That is AC4's own case, but the AC4 test always had a readable dependency too. With a readable one, the capacity makes no visible difference. |
| `routers/web/repo/pull_merge_box.go:202` | `expression/remove`: drops `data.canMergeNow &&` | Test gap | Without dependencies, someone who cannot merge would see the "can be merged" lines. AC3 says the box looks as it does today, but no test uses such a viewer without dependencies. |
| `routers/web/repo/pull_merge_form.go:111` | `expression/remove`: drops `pull.IsStatusMergeable() &&` | Test gap | A pull request with conflicts would be offered merge buttons. AC3 again, but no test uses a pull request that cannot be merged. |
| `routers/web/repo/pull_merge_form.go:166` | `expression/remove`: drops `canUseManualMerge &&`, so the repository setting, work in progress and a running check stop mattering | Test gap | While blocked, a viewer who can merge now would be offered "Manually merged" even where the repository does not allow it. AC2 says "where the repository allows it", and no test turns it off for that viewer. |

## New or strengthened test

I had AI write and address five tests and kill the five survivors since they related to an AC. All of the fixes are in a single commit.

- Commit: [a42a1d48ef](https://github.com/CSCI-435-SE/gitea/commit/a42a1d48efe58c91d8e13ff76c5c45501d0222ad) · Test: `routers/web/repo/pull_merge_box_test.go` › `TestPrepareMergeBoxDependencies` › "AC4 a dependency the viewer cannot read, and no other" · Kills: `routers/web/repo/pull_merge_box.go:109`
- Commit: [a42a1d48ef](https://github.com/CSCI-435-SE/gitea/commit/a42a1d48efe58c91d8e13ff76c5c45501d0222ad) · Test: `routers/web/repo/pull_merge_box_test.go` › `TestPrepareMergeBoxWithDependencies` · Kills: `routers/web/repo/issue_view.go:956`
- Commit: [a42a1d48ef](https://github.com/CSCI-435-SE/gitea/commit/a42a1d48efe58c91d8e13ff76c5c45501d0222ad) · Test: `routers/web/repo/pull_merge_box_test.go` › `TestPrepareMergeBoxFormWithDependencies` › "AC2 no merge form while blocked where the repository does not allow Manually merged" · Kills: `routers/web/repo/pull_merge_form.go:166`
- Commit: [a42a1d48ef](https://github.com/CSCI-435-SE/gitea/commit/a42a1d48efe58c91d8e13ff76c5c45501d0222ad) · Test: `routers/web/repo/pull_merge_box_test.go` › `TestPrepareMergeBoxFormWithDependencies` › "AC3 no merge prompt as before without dependencies, while waiting for checks" · Kills: `routers/web/repo/pull_merge_box.go:202`
- Commit: [a42a1d48ef](https://github.com/CSCI-435-SE/gitea/commit/a42a1d48efe58c91d8e13ff76c5c45501d0222ad) · Test: `routers/web/repo/pull_merge_box_test.go` › `TestPrepareMergeBoxFormWithDependencies` › "AC3 a conflicted pull request offers only Manually merged, as before" · Kills: `routers/web/repo/pull_merge_form.go:111`

## Questions

1. What did the surviving mutants reveal about your tests?

A general pattern was that AI created tests tended to only address changes introduced during implementation; in other words, old logic was left alone while the new logic was well tested. While AI has clearly been quite thorough in the changes it has implemented, it might be taking "preserving old behavior and make minimal changes" a bit too far when it comes to tests.

Another problem was that the examples used in the tests were more detailed than the ACs. What then happened is that the tests covered a combined case while missing the main case as the AC states it.

However, it should be emphasized that some of the mutants found were behaviors that rely on external factors (database or permission lookup failures) that cannot be killed without making the database or the lookup fail on purpose.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

No, none of the surviving mutants pointed to an assertion whose expected value was not based on the requirements. Or more specifically, the surviving mutants represented situations that no test had an assertion for, either because no test set them up yet (such as a viewer who cannot merge, or only a dependency the viewer cannot read) or because they need a failure that no test simulates.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

Most mutants uncovered by the tool were not from the changes themselves; the tool has no knowledge of what changes have occurred. Thus, it was a challenge to narrow down the mutants to actually take care of, which was then narrowed down to the aforementioned 12 surviving mutants that came from the code that was actually changed.

Throughout the implementation of this issue, I had AI produce a table per commit, describing the tests implemented, the requirements for which the test was created, and how the tests were verified, with me providing feedback for those that did not have a clear expected value. Thus, tests were implemented methodically, adding an element of "doubt" that forces the AI to pay attention to, and verify, the changes that occurred and for what reason.

Getting back to the narrowing of mutants, the tool was run with the help of options, which was easy to understand (regex match function names within selected files to narrow the scope of the tool), along with carefully reading not only the AI-generated summaries, but watching the steps it took to understand what it was trying to do.
