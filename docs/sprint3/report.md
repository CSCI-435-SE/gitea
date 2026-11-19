# Report: Sprint 3

<!-- DRAFT. Every "⬜ PENDING" marks something a teammate hasn't finished yet; fill it in when they do, then delete this comment. -->

**Team Name:** GiCoffee

**Team member names:** Vishav Chopra (vvchopra), Carson Rackley (CarsonRackley), Arjun Bhat (arjunsb26), Aman Bhoot (ab-wm), Jack Donohue (Jack-Donohue), Ethan Turner (controlled-opposition)

**Project Name:** Gitea

**Project Link:** [https://github.com/CSCI-435-SE/gitea](https://github.com/CSCI-435-SE/gitea)

## Sprint Overview

Sprint 3 (Nov 3 – Nov 19) was about testing: how well our tests check what the code is supposed to do. The work fell into three parts:

- **The team feature, code suggestions in pull request reviews (#151).** The instructor's feature issue was split into eight sub-issues, S1–S8 (#154–#160, #187), each with one owner and its own spec. S1 (render a `suggestion` block as a diff), S2 ("Suggest change" button), S3 (apply a suggestion as a commit), S4 (record who applied a suggestion) and S5 (multi-line ranges) merged. S8 (#187) moves to Sprint 4: its owner finished three other issues this sprint (#155, #181, #186). ⬜ PENDING: status of S6a (#189) and S6 (#159), both still in progress by Jack-Donohue, and S7 (#160, PR #193) at the deadline.
- **Finishing older work.** #118 (tab badge and desktop notifications), deferred from Sprint 2, merged in #161. ab-wm finished #20, blocking a pull request's merge until the pull requests it depends on are merged, as #143, #144 and #145.
- **Hardening the test suites and CI.** #181 made `main` run every suite after each merge and cut backend PR time, #186 keeps a Playwright trace when an e2e test fails in CI, and #171 records a flaky Vitest pairing.

Every member also ran mutation testing on one of their PRs (D6) and recorded the suite's health at the Sprint 2 tag and at the end of the sprint (Suite health, below).

## Sprint 2 Carryovers

| Sprint 2 item | What happened in Sprint 3 |
| --- | --- |
| #118 Tab icon badge and opt-in desktop notifications (deferred in Sprint 2) | Taken. Merged in PR #161, with the mutation-testing follow-up test in PR #175. This completes the notifications overhaul (#7). |
| #101 and #102, private and personal calendar feeds (listed in the Sprint 2 plan) | Not taken: both are still open and off the milestone. ⬜ PENDING (vvchopra): the reason, also to be written on each issue. |

## Sprint Backlog

### Sprint 3 GitHub Milestone Link

[https://github.com/CSCI-435-SE/gitea/milestone/4](https://github.com/CSCI-435-SE/gitea/milestone/4)

### Why we chose these issues

- **The carryover first.** #118 went into the backlog before new work, as the course rules require.
- **The team feature.** #151 came from the instructor. It was split along the order the code has to be built in. S1 (#154) renders a suggestion, and everything else reads that suggestion. S2 (#155) and S5 (#158) change how suggestions are written. S3 (#156) applies one. S4 (#157), S6 (#159) and S7 (#160) build on S3. The split gave every member at least one sub-issue.
- **An unfinished epic.** #20 (dependency blocking) had been waiting since Sprint 0. ab-wm split it into the blocking rule (#143), the merge box and sidebar that name each blocker (#144), and a bug found along the way (#145).
- **The sprint's focus on testing.** #181, #186 and #171 came from failures the team ran into this sprint (see Hardening issues). #171 was moved to Sprint 4 on the last day, with the reason on the issue.

### Table of Issues

Estimated points come from each issue's scope label, as in Sprints 1 and 2: small = 1, medium = 2.

| Issue Title | Issue Owner | Estimated Points | Scope | Status |
| --- | --- | --- | --- | --- |
| #118 Alert me to new notifications with a tab icon badge and opt-in desktop notifications | CarsonRackley | 2 | Medium | Done (PR #161) |
| #156 S3. Apply a single suggestion as a commit on the PR branch | CarsonRackley | 2 | Medium | Done (PR #179) |
| #143 A pull request stays blocked until the pull requests it depends on are merged | ab-wm | 2 | Medium | Done (PR #146) |
| #144 Show which dependencies block a pull request from merging | ab-wm | 2 | Medium | Done (PR #150) |
| #145 Dependency loops of three or more, and self-dependencies through the API, are accepted | ab-wm | 1 | Small | Done (PR #149) |
| #20 Block pull request merges until prerequisite pull requests are merged | ab-wm | — | Large (parent of #143–#145) | Done |
| #154 S1. Render `suggestion` blocks in review comments as a diff | vvchopra | 2 | Medium | Done (PR #168) |
| #157 S4. Record and show that a suggestion was applied | vvchopra | 2 | Medium | Done (PR #192) |
| #155 S2. Add a "Suggest change" button to the diff comment form | controlled-opposition | 1 | Small | Done (PR #162) |
| #181 Main's CI is red on every push and skips most test suites | controlled-opposition | 2 | Medium | Done (PR #182) |
| #186 Repo topics keyboard test fails in CI Chromium and leaves nothing to debug | controlled-opposition | 1 | Small | Done (PR #191) |
| #187 S8. Add the "Suggest change" button to reply forms in the Files tab | controlled-opposition | 1 | Small | Moved to Sprint 4 |
| #158 S5. Suggest a change across several lines | Jack-Donohue | 2 | Medium | Done (PR #177) |
| #189 S6a. Apply a suggestion across several lines | Jack-Donohue | 1 | Small | ⬜ PENDING (in progress) |
| #159 S6. Apply several suggestions in one commit | Jack-Donohue | 2 | Medium | ⬜ PENDING (in progress; needs #189) |
| #160 S7. Create and apply suggestions through the API | arjunsb26 | 2 | Medium | ⬜ PENDING (PR #193 open; the API test from PR #164 merged) |
| #171 Shortcut tests fail when two test files share a vitest worker | CarsonRackley | 1 | Small | Moved to Sprint 4 (reason on the issue) |

## Requirements & Design

### Were specs complete before coding started?

Yes, for every issue that was implemented. Each spec uses the course format (user story, bug report or hardening task) and got a "Spec reviewed" comment from a teammate before its PR was opened:

- **#143–#145** were written and reviewed by controlled-opposition on Nov 5, the day #146 was opened.
- **The #151 sub-issues** were written on Nov 11. S1 and S2 were reviewed by CarsonRackley on Nov 11, and S3–S7 by controlled-opposition on Nov 11–12. S3 and S4 each got a second review after their revisions (#156 on Nov 12, #157 on Nov 17).
- **#118** kept its Sprint 2 spec and vvchopra's spec review from Oct 26.
- **#189 (S6a)**, split out of S6 on Nov 17 so that S6 could build on applying one multi-line suggestion, was reviewed by controlled-opposition the same day.
- **#181** was reviewed by ab-wm on Nov 15 (its AC7 target was measured and changed during review), **#186** by Jack-Donohue on Nov 17 and **#187** by vvchopra on Nov 17.

**Surprises:**

- **#151's open questions decided most of the design.** The feature issue asked who may apply a suggestion, who authors the commit, what happens when the code changed, and what happens on forks. Each answer became acceptance criteria spread across several sub-issues. S3's spec had to settle the commit author (the person who applies it, crediting the reviewer as co-author) before S4 could store who applied it.
- **The chain of dependencies.** S4 needed S3 merged (Nov 17), and S6 needs S3, S4, S5 and S6a (#189). That left the last links of the chain with little time.
- **Fixing CI exposed hidden failures.** Once #182 made `main` run every suite, flaky tests that PR runs had hidden started to show up on `main` (Suite health).

### Key design decisions

Every implemented issue records at least one decision with two or more alternatives in its issue body (D3). #143–#145 record theirs per commit, as #53 and #54 did in Sprint 1.

- **#118:** The badge is a dot drawn on the existing favicon, reusing the code that already badges the icon with an Actions run's status, rather than a number in the icon or the page title. "Enable desktop notifications" is remembered in the browser rather than as a server setting.
- **#154 (S1):** The diff is drawn in the browser, the way Gitea already renders `mermaid` and `math` blocks. The server only adds the commented line from the comment's stored diff context.
- **#155 (S2):** The button reads the line text from the diff row already on the page. The code fence is one backtick longer than the longest run of backticks in the line, so any line can be quoted.
- **#156 (S3):** A new `ApplySuggestion` service next to `ChangeRepoFiles` refuses the commit if the head branch has moved since the page loaded, rather than merging the change in. The server only adds the Apply attributes for a viewer who may push, so the page never shows a button that would be refused.
- **#157 (S4):** Who applied a suggestion, and the commit, are stored in three new `comment` columns with defaults, so the migration is one `ALTER TABLE`, and a conditional `UPDATE` makes a second apply lose.
- **#158 (S5):** A range comment's start line is stored in the comment's existing metadata JSON, not in a new column. A range is marked outdated by comparing its lines' text, not by blame.
- **#143–#145:** ⬜ PENDING (ab-wm): one line on the main decision, e.g. where the "is this dependency resolved" rule lives.
- **#181:** The upstream release workflows are guarded with upstream's own `if: github.repository == 'go-gitea/gitea'`, rather than deleted or disabled in settings. Backend PRs got faster by refreshing the build cache and running the unit passes in parallel, not by dropping databases or `-race`.
- **#186:** Get a Playwright trace out of CI first, and fix the test only once the trace shows the cause, rather than adding waits blindly.

## Completed Issues

| Issue Title | Issue Owner | Parent | PR Link | PR Author | PR Reviewer(s) | PR Status | Description |
| --- | --- | --- | --- | --- | --- | --- | --- |
| #143 Blocked until dependencies merge | ab-wm | #20 | [#146](https://github.com/CSCI-435-SE/gitea/pull/146) | ab-wm | CarsonRackley | Merged | The merge button, the API and scheduled merges all refuse a pull request whose dependencies aren't merged |
| #145 Dependency loops of any length | ab-wm | #20 | [#149](https://github.com/CSCI-435-SE/gitea/pull/149) | ab-wm | controlled-opposition | Merged | Refuses dependency loops of any length and self-dependencies; the API answers 422 or 409 |
| #144 Name the blocking dependencies | ab-wm | #20 | [#150](https://github.com/CSCI-435-SE/gitea/pull/150) | ab-wm | controlled-opposition | Merged | The merge box names each blocking dependency, and the sidebar shows whether each is open, merged or closed |
| #118 Tab badge and desktop alerts | CarsonRackley | #7 | [#161](https://github.com/CSCI-435-SE/gitea/pull/161) | CarsonRackley | ab-wm | Merged | A dot on the tab icon for unread notifications, and opt-in desktop notifications; [#175](https://github.com/CSCI-435-SE/gitea/pull/175) added the test that kills a mutation-testing survivor |
| #155 S2 "Suggest change" button | controlled-opposition | #151 | [#162](https://github.com/CSCI-435-SE/gitea/pull/162) | controlled-opposition | ab-wm | Merged | A button in the diff comment form inserts a `suggestion` block holding the commented line; [#180](https://github.com/CSCI-435-SE/gitea/pull/180) added tests that kill mutation-testing survivors |
| #154 S1 Render suggestions as a diff | vvchopra | #151 | [#168](https://github.com/CSCI-435-SE/gitea/pull/168) | vvchopra | ab-wm | Merged | `suggestion` blocks in review comments render as a removed and added diff. An earlier PR for it (#163) was merged by mistake and reverted in #166 |
| #160 S7 (API test only) | arjunsb26 | #151 | [#164](https://github.com/CSCI-435-SE/gitea/pull/164) | arjunsb26 | CarsonRackley | Merged | Proves that suggestions posted through the reviews API render as suggestions; the apply endpoint is PR #193 |
| #156 S3 Apply a suggestion | CarsonRackley | #151 | [#179](https://github.com/CSCI-435-SE/gitea/pull/179) | CarsonRackley | vvchopra | Merged | An Apply button commits a single suggestion to the PR's head branch, crediting the reviewer, and refuses with a reason when it can't |
| #158 S5 Multi-line suggestions | Jack-Donohue | #151 | [#177](https://github.com/CSCI-435-SE/gitea/pull/177) | Jack-Donohue | controlled-opposition | Merged | Code comments can cover a range of lines, which is stored, validated and marked outdated when those lines change |
| #181 Main's CI | controlled-opposition | — | [#182](https://github.com/CSCI-435-SE/gitea/pull/182) | controlled-opposition | ab-wm | Merged | Every suite runs on `main`, on sprint tags and on demand; upstream's release workflows are skipped; backend PRs run faster |
| #186 Playwright trace in CI | controlled-opposition | — | [#191](https://github.com/CSCI-435-SE/gitea/pull/191) | controlled-opposition | Jack-Donohue | Merged | A failed e2e test in CI now keeps a Playwright trace and uploads it |
| #157 S4 Applied marker | vvchopra | #151 | [#192](https://github.com/CSCI-435-SE/gitea/pull/192) | vvchopra | CarsonRackley | Merged | Records who applied a suggestion, and in which commit, and shows it on the comment |
| #160 S7 Apply through the API | arjunsb26 | #151 | [#193](https://github.com/CSCI-435-SE/gitea/pull/193) | arjunsb26 | ⬜ PENDING | ⬜ PENDING | An API endpoint applies a suggestion with S3's service |
| ⬜ PENDING: #189 and #159 (Jack-Donohue), once their PRs are open | | | | | | | |

Every code PR above has `## Test Evidence` and `## Tech Debt` sections (D4). CodeRabbit, Gitar and SonarQube Cloud also commented on every code PR as automated first passes, alongside the human reviews.

Other PRs merged this sprint, outside the issue table:

- Course and tooling setup: #140 (Sprint 3 course documents), #141 (mutation testing tools), #142 (Nix), #147 (SonarQube Cloud in CI), #148 (Gitar review checks), #169 (Playwright 1.65)
- Mutation testing: #175, #180
- Check-ins: #165, #172, #173, #174, #176, #178
- AI logs: #170, #188, #190, ⬜ PENDING (later log PRs, including CarsonRackley's)
- Showcase 2: #153

## Test Strategy

Tests were written against each acceptance criterion, with expected values taken from the criteria, the issue discussion or upstream behaviour, never from the code's output. Each PR's Test Evidence table maps every AC to its test and says which tests AI wrote.

- **Go unit tests** for models, services and handlers: #146, #149, #150, #168, #177, #179, #192, #193
- **Integration tests** through the full HTTP request path, on SQLite, MySQL, MSSQL and PostgreSQL in CI: #146, #149, #164, #168, #177, #179, #192, #193
- **Migration test:** #192 (S4's new `comment` columns)
- **Vitest** frontend unit tests: #150, #161, #162, #168, #175, #177, #179, #180, #192
- **Playwright end-to-end tests:** #150, #162, #177
- **CI only:** #182 and #191 change workflows, so their evidence is CI runs: throwaway draft PRs (#183, #184) before merge and the first runs on `main` after it.

Mutation testing (D6) led to new tests: #175 for #118 and #180 for #155 each add a test that kills a surviving mutant.

Changes we couldn't test automatically:

- **Desktop notifications (#118):** the browser's permission prompt and the operating system's notification can't be driven in CI. The unit tests replace them, and #161's Test Evidence records a manual check in Chrome for each of them (AC3 to AC5).
- **The tab icon (#118):** the tests check which icon is chosen and when. What the dot looks like in a real tab was checked by hand in Chrome, as #161's Test Evidence records for AC1 and AC2.
- **What GitHub-style suggestions look like (#154, #156, #158):** layout was checked by hand with screenshots. The tests check the DOM.
- **CI timing (#181):** runner speed varies, so AC7's 22-minute median is measured on real runs rather than tested.
- ⬜ PENDING: anything from #159, #160 or #189.

## Suite health

Suites relevant to our work, at the Sprint 2 tag `v1.27.3-csci435-s2` (`40e0aa79`) and at the end of the sprint. Before #182, nothing ran on tags, so the baseline comes from a manual run of the same workflows against the tag on Nov 19 (#181, AC3).

⬜ PENDING: after the last Sprint 3 merge, replace the "End of sprint" column with the run on the final `main` commit. It currently shows `bb082e84` (the #177 merge).

| Suite | Command or CI job | At `v1.27.3-csci435-s2` | End of sprint (`main` at `bb082e84`) |
| --- | --- | --- | --- |
| Go unit tests | `db-tests` → `test-unit` (`make test-backend`) | 224 packages ok, 0 failed ([run](https://github.com/CSCI-435-SE/gitea/actions/runs/44458153829)) | 224 packages ok, 0 failed ([run](https://github.com/CSCI-435-SE/gitea/actions/runs/44585673568)) |
| Go unit tests, `gogit` build | `db-tests` → `test-unit-gogit` | 224 packages ok, 0 failed | 224 packages ok, 0 failed |
| Migration and integration tests, SQLite | `db-tests` → `test-sqlite` | Passed, 1,532 tests run, 0 failed | Passed, 1,542 tests run, 0 failed |
| Migration and integration tests, MySQL | `db-tests` → `test-mysql` | Passed, 1,531 run, 0 failed | Passed, 1,541 run, 0 failed |
| Migration and integration tests, MSSQL | `db-tests` → `test-mssql` | Passed, 1,531 run, 0 failed | **Failed**, 1,541 run, 1 failed: `TestPullReviewApplySuggestion/AC1_an_empty_message_gets_the_default` |
| Migration and integration tests, PostgreSQL | `db-tests` → `test-pgsql-shard-1` and `-2` | Passed, 834 + 704 run, 0 failed | Passed, 715 + 833 run, 0 failed |
| Vitest | `db-tests` → `test-frontend-coverage` (`pnpm exec vitest run --coverage`) | 63 files, 388 passed, 0 failed (run locally; in CI the job can't start at this tag, see below) | 70 files, 444 passed, **1 failed**: `repo-issue-edit.test.ts` › "r quotes the selection after the existing text, or only moves to the reply box" |
| Playwright e2e (Chromium) | `e2e-tests` → `test-e2e` | 297 passed, 3 skipped, 0 failed ([run](https://github.com/CSCI-435-SE/gitea/actions/runs/44458157446)) | 312 passed, 3 skipped, 0 failed ([run](https://github.com/CSCI-435-SE/gitea/actions/runs/44585673715)) |

"Tests run" counts the top-level Go tests the runner started. Go prints no total, so these are counted from the job log.

### Failures, by cause

- **Flaky, caused by our changes, Vitest:** the shortcut tests and the `r` quote-reply test (Sprint 2's #116) fail when Vitest puts `shortcut.test.ts` and `repo-issue-edit.test.ts` in the same worker, because `isolate: false` lets them share one `document`. It failed on `main` above and on #192's first run, and passes when the files land in different workers. Tracked in **#171**.
- **Flaky, caused by our changes, MSSQL:** S3's integration test failed once on MSSQL (`main`, run above). The comment it posts right after the previous Apply got a 500 after 1.9 s, while that Apply's push was still being processed. It passed on MSSQL in 14 other runs from Nov 13 to Nov 19, on #179, #177 and `main`. Tracked in **#195** (Sprint 4).
- **Flaky, caused by our changes, PostgreSQL:** `TestRepoCalendar` (Sprint 2's #123) fails under the race detector when the mocked clock is reset while an indexer worker reads it. Not in the two runs above. Tracked in **#185**.
- **Flaky, e2e:** "repo topics can be added and removed by keyboard" failed in CI Chromium on Nov 15 (#183, #184). It passed in both runs above. **#186** now keeps a trace of any failed e2e test so the cause can be found.
- **Environment-specific:** at the s2 tag, the CI Vitest job stops with `Cannot find dependency '@vitest/coverage-v8'`, because that package was added after s2 (#141). The baseline was run locally instead (`pnpm exec vitest run` at `40e0aa79`, Debian WSL). Recorded in **#181**, Decision 4.

No test the team added this sprint is skipped. The 3 skipped Playwright tests were skipped at s2 too, each with a reason in a code comment: two WebKit Tab-order skips (`aria-menu.test.ts:44`, `menus.test.ts:54`) and an upstream Firefox skip (`file-view-render.test.ts:6`). ⬜ PENDING: an issue link for the two WebKit skips, which come from our Sprint 1 `<aria-menu>` work. The only CI change that turns something off is #182's skip of upstream's release workflows, which links #181 in a same-line comment on every job.

## Mutation Testing

Each member's D6 write-up, with the before and after reports in [`docs/sprint3/mutation/reports/`](https://github.com/CSCI-435-SE/gitea/tree/main/docs/sprint3/mutation/reports):

| Member | Issue (PR) | Write-up |
| --- | --- | --- |
| ab-wm | #144 (#150) | [144-ab-wm.md](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint3/mutation/144-ab-wm.md) |
| CarsonRackley | #118 (#161) | [118-CarsonRackley.md](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint3/mutation/118-CarsonRackley.md) |
| controlled-opposition | #155 (#162) | [155-controlled-opposition.md](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint3/mutation/155-controlled-opposition.md) |
| vvchopra | #157 (#192) | [157-vvchopra.md](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint3/mutation/157-vvchopra.md) |
| Jack-Donohue | ⬜ PENDING | ⬜ PENDING |
| arjunsb26 | ⬜ PENDING | ⬜ PENDING |

## Hardening Issues

| Issue | Owner | Where it came from |
| --- | --- | --- |
| #181 Main's CI is red on every push and skips most test suites | controlled-opposition | Collecting the Suite health baseline: the s2 tag had no test runs, `main` ran only unit tests, and every `main` commit was already red from upstream release workflows, which hid a real `db-tests` failure |
| #186 Repo topics keyboard test fails in CI Chromium | controlled-opposition | The e2e failures on #183 and #184, #181's evidence runs, which left nothing to debug |
| #145 Dependency loops of three or more are accepted | ab-wm | Found by ab-wm while working on #20's dependency rules: only loops of two were refused |
| #171 Shortcut tests fail when two test files share a vitest worker (moved to Sprint 4) | CarsonRackley | #161's `sonarqube` job failed in a test #161 didn't touch. CarsonRackley traced it to `isolate: false` in `vitest.config.ts` |

## AI Tool Usage

Per member: tools used, sessions logged, a link to the AI log folder, and 2–3 sentences on lessons learned this sprint.

Vishav Chopra

| **Vishav Chopra (vvchopra)** | |
| --- | --- |
| Tools Used | Claude Code Desktop |
| Total Sessions | 4 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/vvchopra](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/vvchopra) |
| Lessons learned | Claude is able to see and diagnose so many different issues, ranging from tests, code bugs, overlooked assumptions, etc. Although I have prior experience with using Claude Code in these ways, it is notable just in how many ways it can help you. |

Carson Rackley

| **Carson Rackley (CarsonRackley)** | |
| --- | --- |
| Tools Used | Claude Code (Terminal) + Specstory |
| Total Sessions | 3 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/CarsonRackley](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/CarsonRackley) |
| Lessons learned | Have prior experience, didn't have anything notable or unexpected. This sprint, mutation testing on #118 showed a line my tests ran but never checked, which taught me that coverage alone doesn't mean the behaviour is tested. |

Arjun Bhat

| **Arjun Bhat (arjunsb26)** | |
| --- | --- |
| Tools Used | Claude Code (VS Code extension) + SpecStory |
| Total Sessions | 2 |
| Link to logs | [arjunsb26 folder](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/arjunsb26):<br>[2026-11-05_claude-code_sprint3-quiz-prep.md](https://github.com/CSCI-435-SE/gitea/blob/main/ai-logs/sprint3/arjunsb26/2026-11-05_claude-code_sprint3-quiz-prep.md)<br>[2026-11-12_claude-code_issue-160-api-suggestion-test.md](https://github.com/CSCI-435-SE/gitea/blob/main/ai-logs/sprint3/arjunsb26/2026-11-12_claude-code_issue-160-api-suggestion-test.md) |
| Lessons learned | Claude is a go-getter and very proactive, unprompted it will (for instance when I asked if new changes would break anything, it said it would be easier to test itself than explain and did exactly so) |

Aman Bhoot

| **Aman Bhoot (ab-wm)** | |
| --- | --- |
| Tools Used | Claude Code (Primarily Opus 5.5) via Claude-CLI |
| Total Sessions | 12 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/ab-wm](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/ab-wm) |
| Lessons learned | N/A. Worked within expectation (prior experience). |

Jack Donohue

| **Jack Donohue (Jack-Donohue)** | |
| --- | --- |
| Tools Used | Claude Code CLI, Claude Code VSCode |
| Total Sessions | 4 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/Jack-Donohue](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/Jack-Donohue) |
| Lessons learned | Is more capable of using sub agents than older versions. |

Ethan Turner

| **Ethan Turner (controlled-opposition)** | |
| --- | --- |
| Tools Used | Claude Code CLI, herdr |
| Total Sessions | 13 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/controlled-opposition](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint3/controlled-opposition) |
| Lessons learned | With the introduction of Opus 5.5, implementing features was much faster and used less tokens. I did not hit session limits as I did in Sprint 0. |

## Release

### Tag Name

v1.27.3-csci435-s3

### Release Link

⬜ PENDING: tag after the last Sprint 3 merge, then link the release here: `https://github.com/CSCI-435-SE/gitea/releases/tag/v1.27.3-csci435-s3`

## Deliverables Summary

| # | Deliverable | Status |
| --- | --- | --- |
| D1 | Sprint backlog | [Sprint 3 milestone](https://github.com/CSCI-435-SE/gitea/milestone/4), each issue with a scope label and a Scope section |
| D2 | Requirements specs, peer-reviewed | Every implemented issue in the course format, each with a "Spec reviewed" comment before its PR |
| D3 | Design decisions with at least two alternatives | In each issue's Design Decision section; #143–#145 per commit |
| D4 | PRs with Test Evidence and Tech Debt | 14 code PRs merged as of Nov 19, all with both sections (table above). ⬜ PENDING: final count |
| D5 | AI logs | `ai-logs/sprint3/<username>/` for each member, linked from each issue's AI Assistance comment |
| D6 | Mutation testing | See Mutation Testing. ⬜ PENDING: two write-ups |
| D7 | Mid-sprint check-in (due Nov 12) | All six in [`docs/sprint3/checkin/`](https://github.com/CSCI-435-SE/gitea/tree/main/docs/sprint3/checkin) |
| D8 | Sprint report | This document |
| D9 | Release | ⬜ PENDING |
| D10 | Reflection survey | Submitted individually |

## Risks and Retrospective

### What worked

- **Splitting #151 along its dependencies.** S1 merged on Nov 12, the day after the split, so S2, S3 and S5 could be built on it in parallel.
- **Spec reviews that settled the hard questions first.** S3's two reviews settled who authors the commit, what "outdated" means and how forks behave before any code was written. Its PR review then found edge cases (an archived fork, a line above the comment that changed) rather than design problems.
- **Mutation testing found real gaps.** It showed new lines that ran under tests but whose results no test checked. #175 and #180 added tests that kill those survivors.
- **Fixing CI.** After #182, `main` runs every suite on every merge, and the s2 tag could be tested for the baseline. That is why this report has a Suite health table, and why the flaky tests above were found.

### What slowed the team down

- **The feature was split late.** #151 was posted on Nov 7, and the sub-issues were written on Nov 11, eight days into a seventeen-day sprint. Our Sprint 2 goal was to write and review every spec in the first three days.
- **The end of the chain ran out of time.** S4 couldn't start its PR until S3 merged on Nov 17, so it merged on the last day, and S6 depends on four other issues. #187 (S8) carries over. ⬜ PENDING: which of #159, #160 and #189 do too.
- **Review load.** ab-wm reviewed 6 of the 14 code PRs, controlled-opposition 3, CarsonRackley 3 and vvchopra and Jack-Donohue 1 each. Our Sprint 2 goal was at least two reviews per member.
- **A wrong merge.** #163 (S1) was merged by mistake while another PR was meant, then reverted (#166) and merged again as #168, all on Nov 12.
- **Flaky tests.** Until #182, `main` didn't run the integration and e2e suites, and its status was red anyway, so flaky failures were noticed late.

### Changes for Sprint 4

- Split the team feature into sub-issues, and write and review their specs, within the first three days.
- When one issue blocks another, merge the blocking PR by mid-sprint. Otherwise move the blocked issue out at planning time instead of on the last day.
- Name a reviewer when each PR is opened, rotating so that every member reviews at least two PRs.
- Treat a red `main` as a stop: whoever's merge turned it red fixes it or files an issue that day.

## Sprint 4 Plan

Carryover issues:

- #187 (S8, controlled-opposition): the "Suggest change" button in reply forms. Its spec is reviewed and it builds only on S2, so it can start on day one.
- ⬜ PENDING: whichever of #159 (S6), #160 (S7) and #189 (S6a) are not merged by the deadline.
- #171 (flaky Vitest pairing) and #185 (`TestRepoCalendar` data race), plus #195 (the MSSQL flake in S3's integration test).
- #101 and #102, the private and personal calendar feeds, if the team takes them.

Initial ideas, from this sprint's Tech Debt sections:

- Stop computing the Apply check on every page that renders code comments, and add it where "Expand" reveals more of a file (#179, and the #192 review).
- Refuse an outdated suggestion less often: S3 compares the lines above the comment, which can refuse a suggestion whose own line is unchanged (#179).
- Ask the TA to make the combined CI check required and to protect tags, which needs admin rights (#181).
