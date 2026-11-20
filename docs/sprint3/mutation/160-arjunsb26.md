# Mutation testing: Issue #160 -- S7. Create and apply suggestions through the API

**PR(s):** #193

**Tool:** go-mutesting

**Config:** no config file; the options are the command's flags. Setup from the team's guide ([mutation-testing.md](../../../.claude-students/docs/mutation-testing.md)), and the `--match` approach from ab-wm's [#144 write-up](144-ab-wm.md).

**Command:** `GOTOOLCHAIN=go1.26.4 go-mutesting --exec-timeout=60 --match='^(ApplyPullReviewSuggestion|applyPullReviewSuggestionError|ApplySuggestion)$' routers/api/v1/repo/pull_review_suggestion.go services/repository/files/suggestion.go`

**Reports:** [before](reports/160-arjunsb26-before.txt) · [after](reports/160-arjunsb26-after.txt)

**Mutation score:** before 41.3% (26 of 63 killed) → after 57.1% (36 of 63 killed)

Both runs are on PR #193 as of `4968efea46`, before `main` (with #196's batch apply) was merged into it; the after run adds only the new unit test. The merge moved `ApplySuggestion`'s body into `ApplySuggestions`, so the service line numbers below refer to `4968efea46`.

## Surviving mutants

The run mutates the two Go files the PR changed, limited to the three functions it added or changed: the new API handler `ApplyPullReviewSuggestion`, its error mapping `applyPullReviewSuggestionError`, and S3's service `ApplySuggestion`, whose returns the PR changed to carry the commit SHA. go-mutesting runs only each package's own Go tests, not `tests/integration`, so the PR's integration test `TestAPIPullReviewApplySuggestion` was never run against these mutants.

Before: 37 survived; after: 27. The ten mutants in the error mapping are the ones the new test kills, and the after report shows them killed (PASS).

Before: 37 survived. All 21 handler mutants survived, because `routers/api/v1/repo` had no unit test for the handler. The 16 service survivors are below; 10 are on lines the PR changed, and 6 are in S3's unchanged code.

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `routers/api/v1/repo/pull_review_suggestion.go:99` | removes the 403 for `ErrSuggestionPermission` | Test gap | AC3a. Covered only by the integration test, which this tool doesn't run. **Killed after** by the new unit test. |
| `routers/api/v1/repo/pull_review_suggestion.go:101` | removes the 403 for a protected branch | Test gap | Design decision 2; no test at all, unit or integration. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:103` | removes the 403 for required signed commits or a protected file | Test gap | Design decision 2; no test at all. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:105` | removes the 403 carrying a pre-receive hook's message | Test gap | Design decision 2; no test at all. **Killed after**, and the new test also checks that the hook's message reaches the client. |
| `routers/api/v1/repo/pull_review_suggestion.go:107` | removes the 409 for a stale `head_sha` | Test gap | AC3c. Integration only. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:109` | removes the 409 for an outdated suggestion | Test gap | AC3b. Integration only. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:111` | removes the 409 for a closed pull request | Test gap | AC3d. Integration only. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:113` | removes the 422 for a comment with nothing to apply | Test gap | AC3e. Integration only. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:115` | removes the 422 for a suggestion that changes nothing | Test gap | Design decision 2; no test at all. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:117` | removes the 500 for any other error | Test gap | No test made the service fail unexpectedly. **Killed after.** |
| `routers/api/v1/repo/pull_review_suggestion.go:86-87` | removes the call to the error mapping (2 mutants) | Test gap | The integration test's refusal cases catch it, but this tool doesn't run them. A unit test would need a real pull request and comment in the fixtures. |
| `routers/api/v1/repo/pull_review_suggestion.go:75-77` | the pending-review check: `ReviewerID != Doer.ID` becomes `true`, `Review != nil` becomes `true`, and the 404 is removed (4 mutants) | Test gap | No test, unit or integration, applies a suggestion from a pending review. With the 404 removed, the service still refuses (its own pending check), so only the status (404 vs 422) would change. The reviewer's own pending review applying is untested too. |
| `routers/api/v1/repo/pull_review_suggestion.go:68, 72-73, 80-81` | removes `return`s and 500s after a failed lookup (6 mutants) | Test gap | Differ only when the comment, its review or its pull request fails to load from the database, which no test causes. |
| `services/repository/files/suggestion.go:115` | removes `return "", ErrSuggestionHeadMoved` after the head check | Equivalent | The commit is made with `LastCommitID: expectedHeadSHA`, so `ChangeRepoFiles` refuses the stale head anyway, and line 178 maps that to the same `ErrSuggestionHeadMoved`. The service test "AC4 the page showed an older head" still passes because the outcome is identical. |
| `services/repository/files/suggestion.go:178` | removes `return "", ErrSuggestionHeadMoved` when the branch moves between the check and the commit | Test gap | Only reachable in a race (a push between line 113 and the commit). The mutant would then dereference a nil response. Hard to provoke in a test. |
| `services/repository/files/suggestion.go:180` | removes `return "", err` after a refused commit | Test gap | A hook rejection or protected file that the pre-check misses. No service test makes `ChangeRepoFiles` fail, and without the return the new SHA line dereferences a nil response. |
| `services/repository/files/suggestion.go:104, 121, 140, 145, 151, 155, 163` | removes a `return "", err` after a database or git error (7 mutants) | Test gap | The PR only added `""` to these returns. They differ only when loading the review, the protection rule, the repository, the commit, the blob or the poster fails, which no test causes. |
| `services/repository/files/suggestion.go:106` | S3's pending-review check: `Type == Pending` becomes `true`, and `doer == nil` becomes `false` (2 mutants) | Test gap (unchanged S3 code) | The service test's comments have no review, so a submitted review by someone else is never applied there. The web and API integration tests do this, but aren't run by the tool. |
| `services/repository/files/suggestion.go:126` | S3's unprotected-file check: `!canPush` and `UnprotectedFilePatterns != ""` each become `true` (2 mutants) | Test gap (unchanged S3 code) | No service test has a protection rule with file patterns that the doer *can* push to. |
| `services/repository/files/suggestion.go:94` | `RangeStartLine() > 0` becomes `> 1` | Test gap (unchanged #158 code) | No test has a range starting at line 1. |
| `services/repository/files/suggestion.go:124` | removes `protectedBranch.Repo = pr.HeadRepo` | Not analysed further (unchanged S3 code) | No test checks a rule that needs the repository loaded (e.g. team whitelists). |

## New or strengthened test

- Commit: [2e1b3eaf26](https://github.com/CSCI-435-SE/gitea/commit/2e1b3eaf26afd78fe3e68cc29e6de5cc7800cd48) · Test: `routers/api/v1/repo/pull_review_suggestion_test.go` › `TestApplyPullReviewSuggestionError` (14 subtests) · Kills: `routers/api/v1/repo/pull_review_suggestion.go:99, 101, 103, 105, 107, 109, 111, 113, 115, 117`

It calls the error mapping directly with each refusal the service can return and checks the status. The expected statuses come from #160's revised ACs (AC3a–AC3e) and design decision 2 (403 stop, 409 refetch and retry, 422 nothing to apply), not from the handler's output. It also checks that a wrapped error keeps its status and that a hook's message reaches the client.

## Questions

1. What did the surviving mutants reveal about your tests?

   Every API behaviour was tested only end to end. `TestAPIPullReviewApplySuggestion` checks each status code, but it lives in `tests/integration`, so the handler's own package had no test at all and all 21 handler mutants survived. Three refusals the design decision promises (protected branch, signed commits or protected files, and a hook's rejection) had no test anywhere, not even in the integration test. The service survivors were mostly error paths that only a failing database or git call reaches, plus one equivalent mutant: the stale-head check at line 115 is backed up by the commit's own `LastCommitID` check, so removing it changes nothing a test can see.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

   No. The survivors came from missing assertions, not wrong ones: cases that no test set up, or error paths that no test makes fail. The closest is the equivalent mutant at `suggestion.go:115`. The service test "AC4 the page showed an older head" expects `ErrSuggestionHeadMoved`, which comes from #156 AC4, but it can't tell which of the two checks produced it. The expected value is still right.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

   Claude Code (claude-opus-5-5) installed go-mutesting, chose the `--match` list of the functions PR #193 changed, ran both passes, mapped each surviving mutant to its line and drafted the table above, and wrote `TestApplyPullReviewSuggestionError` with its expected statuses taken from #160's ACs and design decision 2.

TODO(arjunsb26): before committing, replace this line with what you checked yourself, e.g. that the after report shows the ten mapping mutants as PASS (killed), that each expected status in the test matches the AC or decision it names, and whether you agree with each "equivalent" or "test gap" call.
