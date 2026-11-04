# Course workflow — issues, PRs, reviews, sprint deliverables

Course rules, not upstream Gitea ones. Source: the course site <https://csci-435-se.github.io/>
(Sprints pages and Glossary) and `docs/sprint0/standards.md`. Everyone on the team, and every AI
session, uses the formats below so all issues and PRs look the same. The GitHub templates in
`.github/` carry the same formats; if they ever disagree with this page, fix the template.

AI disclosure has its own page: `AI-ASSISTANCE.md`.

## Current sprint: Sprint 3

- **Dates:** Nov 3 – Nov 19, 2026. Everything pushed and merged by **Nov 19, 11:59 PM**, except
  the D7 check-in, due **Nov 12, 11:59 PM** together with at least one code PR of your own, open
  and ready for review or merged.
- **Focus:** testing, meaning how well the tests check what the code is supposed to do.
- **Points:** 100.
- **Zero rule:** no merged PR (D4), no sprint report (D8) or no release (D9) means **0 for the
  whole sprint**.
- **Carries forward:** anything the Sprint 3 page doesn't change comes from the Sprint 2 and
  Sprint 1 pages, and every Sprint 0 standard still applies.
- **Start early:** record the test baseline at the Sprint 2 tag (see D8), and get the mutation tool
  running on any existing file in your area (see D6) before your PRs are ready.

| # | Deliverable | Points | Who | Where |
| --- | --- | --- | --- | --- |
| D1 | Sprint backlog: a `Sprint 3` milestone with every selected issue on it, each with a justified scope | 10 | Team | GitHub milestone |
| D2 | Requirements spec for each issue (formats below), spec-reviewed on the issue before coding | 20 | Individual | Issue body |
| D3 | At least one non-trivial design decision per issue, with **at least two alternatives** | 5 | Individual | Issue body |
| D4 | Code changes and PRs, each with `## Test Evidence` and `## Tech Debt` sections | 20 | Individual | PRs |
| D5 | AI logs | 5 | Individual | `ai-logs/sprint3/<github-username>/` |
| D6 | Mutation testing on one issue's PR (CSCI 535: two) | 15 | Individual | `docs/sprint3/mutation/` |
| D7 | Mid-sprint check-in, linking a code PR | 5 | Individual | `docs/sprint3/checkin/<github-username>.md` |
| D8 | Sprint report, with a Suite health section | 10 | Team | `docs/sprint3/report.md` |
| D9 | Release: tag `v1.27.3-csci435-s3` plus a GitHub release | 2 | Team | Tags / Releases |
| D10 | Reflection survey | 8 | Individual | Blackboard/Zulip link |

### Graded strictly this sprint

The Sprint 3 page lists these as the rules that cost the most points. Details are in the sections
named.

1. No merge without a teammate's approval on the PR's **latest commit** (Reviews, below).
2. A spec review comment on the issue that says what was checked (Spec review, under the issue
   formats).
3. Every code PR has `Closes #<N>`, and both the issue and the PR are on the `Sprint 3` milestone.
4. Real AI logs, linked from the issue (D5).
5. A Test Evidence table in every PR, with expected values not copied from the code's output
   (Pull request format).
6. Mutation testing with a tool, with before and after reports committed (Mutation testing (D6)).
7. The team introduces no new test failures. A skipped test or a disabled CI check needs a linked
   issue and a reason in a code comment (D8).
8. A sprint report that renders on GitHub and uses the Sprint 3 sections (D8).

### Deliverable notes

- **D1 load:** each CSCI 435 student owns at least 2 medium issues; each CSCI 535 student owns at
  least 2 medium and 1 small. Either load can be met with equivalent story points instead. Every
  issue has exactly **one** assignee. Unfinished Sprint 2 issues go into the backlog before new
  work. To drop one, write the reason on that issue. If the backlog is too thin to meet the load,
  find areas that still need work (features, improvements, test gaps, bugs) and write new issues
  for them the usual way.
- **D1 scope:** every issue states its scope (small or medium) and one or two sentences on why:
  which components it touches and how much change is expected. The instructor or TA may reclassify
  an issue whose scope is clearly overstated.
- **D1 hardening issues (optional):** one of your issues may be a hardening issue. See "Hardening
  issues" below.
- **D1 team feature:** the instructor or TA posts a larger feature for Sprints 3–4 as an issue and
  tells the team. Split it into sub-issues, each with one owner, its own spec (D2) and scope, linked
  to the feature issue and added to the `Sprint 3` milestone. Sub-issues count toward the load.
- **D2:** specs are written and spec-reviewed before coding starts. Features use the user story
  format; hardening issues use the bug report format when they fix a defect, otherwise the
  hardening task format.
- **D3:** each decision names at least two concrete alternatives and why one won; this is graded
  strictly. The site also accepts the decision in an issue comment or the PR description; the team
  keeps it in the issue body. Decisions that span issues go in `docs/sprint3/design.md`.
- **D4:** every PR description has the Test Evidence and Tech Debt sections (see "Pull request
  format"). Per-PR scores are combined across your PRs and scaled to 20.
- **D5:** conventions are in `ai-logs/README.md`. Logs hold the full prompts and output. Graded
  strictly this sprint: a session summary, paraphrased prompts, a share link, or tool output without
  the prompts that produced it is not a log. The link must open the log file itself, not a folder
  or another issue. The folder must exist even if no AI was used; in that case, post a comment
  saying so. The site accepts the link from the issue or the PR; the team keeps it in **issue
  comments only** (`AI-ASSISTANCE.md`).
- **D6:** see "Mutation testing (D6)" below.
- **D7:** about 100 words on each of: what you have implemented so far and what remains; one thing
  you learned about testing in the project; one risk to finishing by Nov 19 and what you will do
  about it. Link at least one **code PR you authored**, open and ready for review or merged, by Nov
  12. On time, substantive and with a PR linked scores 5; late, thin or no PR 3; missing 0.
- **D8:** graded on completeness, specificity, quality of evidence and the Suite health section;
  sections are in "Sprint report (D8) sections" below.
- **D9:** keep the base version (`v1.27.3-csci435-s2` → `v1.27.3-csci435-s3`). Tag **after the
  last Sprint 3 merge**, so the release matches the final `main`, and push the tag to origin. The
  release description summarizes the sprint's features, fixes and hardening work, not just the tag
  name.
- **Reviews:** nobody approves or merges their own PR. Request the review on GitHub and tell the
  reviewer on Zulip; reviewers respond (approve or request changes) within 24 hours, with
  substantive comments and the rubric, not a bare approval. A PR is merged only after a teammate
  approves its **latest commit**: any push after the approval, including merging `main` in, needs
  a new approval. Once approved, the author merges. Each student reviews at least one PR per
  sprint, and every teammate at least once per semester.
- **Extra credit:** up to 3 points per PR submitted to upstream Gitea that is accepted or under
  active review, and up to 2 points per bug report filed upstream that comes from your own testing
  and is confirmed by the maintainers by the time Sprint 4 is graded. Both at the instructor's
  discretion.

## Branches and commits

- Branches: `feat/issue-<N>-short-description`, `fix/issue-<N>-short-description` or
  `chore/short-description` (`STUDENTS.md`). One issue per branch.
- Commit messages follow Conventional Commits and end with the issue number:
  `feat(repo): show empty state on releases page (#17)`. Add an
  `Assisted-by: AGENT_NAME:MODEL_VERSION` trailer when AI helped; never `Co-Authored-By` or
  `Signed-off-by` (`AGENTS.md`). A human approves every commit before it is made.

### Before each implementation: sync with main

Every new branch starts from an up-to-date `main`. AI sessions run this before writing any code:

```bash
git switch main
git pull origin main
git switch -c feat/issue-<N>-short-description
```

To pick up new work on `main` for a branch that already exists, **merge, don't rebase**. Rebasing
or squashing after review has started is not allowed (`docs/sprint0/standards.md`):

```bash
git fetch origin
git merge origin/main
```

Do this before opening the PR and again if `main` moves while it is in review. A merge pushed after
approval needs a new approval (see Reviews). Never merge another feature branch into yours.

### After merge: clean up

A PR is merged with a merge commit once its latest commit is approved, every review comment is
addressed and CI passes. Then:

```bash
git switch main
git pull origin main
git branch -d feat/issue-<N>-short-description               # local branch
git push origin --delete feat/issue-<N>-short-description    # or GitHub's "Delete branch" button
git fetch --prune                                            # drop stale remote-tracking refs
```

Also check that the issue closed (via `Closes #N`), that the issue and PR are on the sprint
milestone, and that the issue has its AI Assistance comment. Then run `specstory sync` and file the
session's log (see `ai-logs/README.md`).

## Issue format: feature (user story)

Template: `.github/ISSUE_TEMPLATE/user-story.yaml`. The user story must meet INVEST: Independent,
Negotiable, Valuable, Estimable, Small, Testable.

```markdown
## User Story
As a <role>, I want <capability> so that <benefit>.

## User Scenario
<A short concrete walk-through: who, where in the UI, what they do, what they see.>

## Acceptance Criteria
- [ ] AC1: Given <state>, when <action>, then <observable result>.
- [ ] AC2: ...
- [ ] AC3: ...            (at least 3; each has a clear pass/fail result)

## Out of Scope
- <What this issue deliberately does not do.>

## Open Questions
- <Unresolved questions, or "None".>

## Design Decision
**Decision:** <what was chosen>
**Alternatives Considered:** <at least two concrete options, and why each lost>
**Rationale:** <why this one>
**Consequences:** <trade-offs, follow-up work, what it makes harder>

## Scope
small | medium | large, and matching `scope:` label
<One or two sentences on why: the components it touches and how much change you expect.>
```

## Issue format: bug report

Template: `.github/ISSUE_TEMPLATE/bug-report.yaml`. Also used by hardening issues that fix a
defect.

```markdown
## Observed Behavior
## Expected Behavior
## Steps to Reproduce
1. ...
## Additional Information
<Version or commit, browser/OS, logs, screenshots.>
## Design Decision
<Same four fields as above: Decision, Alternatives Considered, Rationale, Consequences.>
## Scope
<Same as above: size, label and one or two sentences on why.>
```

## Issue format: hardening task

No GitHub template; write it in a blank issue. Used by hardening issues that don't fix a defect,
such as a flaky test or a broken CI check (Sprint 3, D2).

```markdown
## Problem
<What is unreliable now, and how you know: failing test names, run counts, CI run links.>
## Goal
<What "fixed" means.>
## Acceptance Criteria
- [ ] AC1: ...
## Evidence of Fix
<Which evidence from the table below the fix will show.>
## Design Decision
<Same four fields as above.>
## Scope
<Same as above: size, label and one or two sentences on why.>
```

### Hardening issues

Work that makes the existing system more reliable rather than adding a feature. Good sources: the
D8 baseline run (failing or flaky tests, broken CI checks on `main`); the "Introduced" and
"Encountered" Tech Debt fields of Sprint 1 and Sprint 2 PRs; bugs found in reviews or after
merging; and the upstream issue tracker for the parts of Gitea the team works on. It is scoped like
any other issue and must include evidence that the problem is fixed:

| Kind | Evidence |
| --- | --- |
| Bug fix | A test that fails before the fix and passes after |
| Flaky test | The root cause, and repeated runs before and after the fix (e.g. 20 for a unit test, 10 for E2E) |
| Broken or missing CI check | Links to CI runs before and after the fix |
| Failing tests in the baseline | Pass/fail counts before and after, and the cause of each group of failures |

The site doesn't say where the evidence goes: plan it in the issue (Evidence of Fix, or Additional
Information for a bug), then post the results on the issue and in the PR's Test Evidence once the
fix is in. Regenerating snapshots, reformatting code, or silencing a check without fixing its cause
counts as small at most.

### Design Decision and spec review

- **Design Decision (D3):** fill it in once you've picked an approach, before opening the PR. Put
  a second decision under another `### Decision 2` heading. If a PR changes the decision, update
  the issue body.
- **Spec review (D2):** before coding starts, a teammate posts a review comment **on the issue**.
  Start it with `Spec reviewed` so it is easy to find. The layout is the reviewer's choice, but the
  comment must say what was checked:
  1. each acceptance criterion: clear and testable, naming any that aren't;
  2. the cases considered (e.g. empty input, errors, permissions) and whether the ACs cover them;
  3. whether the scope fits the stated size;

  and end with an approval, or the problems for the author to fix in the spec. `Looks good!` with
  nothing about what was checked does not count. An optional layout:

  ```markdown
  Spec reviewed — @reviewer-username

  **Acceptance criteria:** <clear and testable? which aren't, and why>
  **Cases considered:** <which cases, and whether the ACs cover them>
  **Scope:** <does it fit the stated size?>
  **Verdict:** Approved | Changes needed: <list>
  ```

## Pull request format

Template: `.github/pull_request_template.md`. Title uses Conventional Commits, e.g.
`fix(issues): keep label filter after search`. Description sections:

1. `Closes #<N>`, which links the issue and closes it on merge. Every code PR links its issue, and
   the PR goes on the `Sprint 3` milestone too.
2. **What changed:** the change in your own words.
3. **Why:** the problem, and which acceptance criteria (AC1…) it satisfies.
4. **Test Evidence:** required from Sprint 3. It replaces the old "How it was tested" section.
   Every behaviour change still needs a new or updated test.

   ```markdown
   ## Test Evidence

   | Acceptance criterion | Test(s) | Level | Source of expected value |
   |---|---|---|---|
   | AC1: ... | ... | ... | ... |

   **Not tested automatically, and why:** [...]

   **AI-written tests:** [Which tests were written or modified by AI, and how you checked that their expected values are correct. If none, write "None."]
   ```

   - **One row per acceptance criterion** from the issue; for a bug report, the Expected Behavior.
     A PR that changes no product behaviour (e.g. a CI change) writes `N/A` and why.
   - **Test(s):** the test name, plus the file if not obvious. For a manual check, write `Manual:`
     followed by the steps and what you saw. An AC tested both ways lists both in the same row.
   - **Level:** `unit`, `integration`, `E2E` or `manual`.
   - **Source of expected value:** `AC`, `issue discussion`, `docs`, `upstream behavior`, or
     `other` (explain). **Never the current output of your own code:** a test whose expected value
     was copied from what the code returns cannot detect that the code is wrong.
   - **Not tested automatically, and why:** only for ACs checked manually only.
   - AI may draft this section; the author checks that it is accurate.

5. **CI / manual verification:** CI status. If CI doesn't run your tests, the commands you ran and
   a summary of the results (passed, failed, skipped).
6. **Design:** one line pointing at the Design Decision in the issue, or a note on how it changed.
7. **Tech Debt:** required from Sprint 2. Both fields are required, and `None.` is a valid answer:

   ```markdown
   ## Tech Debt

   **Introduced:** <shortcuts, workarounds or design compromises this PR makes, and why>

   **Encountered:** <tech debt that was already there and that you ran into, and how you handled it>
   ```

8. **AI assistance:** format in `AI-ASSISTANCE.md` §2, with **no log link**.

Per-PR grading (20 points): correctness against the acceptance criteria 6; tests written or updated
appropriately, with the Test Evidence section complete, every AC covered and expected values
justified 6; description explains what and why, Tech Debt section present, issue linked 3; CI passes
or manual verification documented 3; reviewed with the rubric (including the test evidence row),
reviewer comment posted, merged only after an approval on the latest commit 2.

## Code review rubric

The reviewer posts this as a **PR comment**, as well as the GitHub review itself. A copy-paste copy
is in `.github/code-review-rubric.md`, so keep the two in sync. Pull the branch and test it when you
can, say what you checked, and mark each point as required or optional
(`docs/community-governance.md`). Score each row against the issue's acceptance criteria. Anything
below 7 needs a note saying what would raise it.

```markdown
### Code Review — @reviewer-username

**Checked:** <what you ran or tried>

| Criterion | Score (1–10) | Notes |
| --- | --- | --- |
| Correctness — implements issue requirements | | |
| Test coverage — tests present and meaningful | | |
| Test evidence — does every AC have a test? Are the expected values justified? Which tests did you run locally? | | |
| Code quality — style, naming, no duplication | | |
| Documentation — comments and PR description clear | | |
| AI transparency — evidence of human verification if AI used | | |
| Overall — merge as-is? | | |
```

## Mutation testing (D6)

A mutation testing tool makes small changes to the code (*mutants*), such as flipping a condition
or a boundary value, and runs the tests against each one. If a test fails, the mutant is
**killed**. If every test passes, it **survived**: the tests didn't notice the change, which points
to a weak or missing test. An **equivalent** mutant behaves exactly like the original code, so no
test can kill it. Definitions are in the course Glossary.

**Which issue:** one of your issues whose PR changes source code and tests (CSCI 535: two, and the
small issue may be one of them). A tool is required; mutating code by hand doesn't count.

| Code changed | Tool |
| --- | --- |
| Frontend TypeScript (Vitest) | StrykerJS with the Vitest runner |
| Go backend | Gremlins or go-mutesting |

1. Run the tool on the **source files changed by your PR(s)** for that issue, not the whole
   project.
2. Analyse every surviving mutant: equivalent, or a gap in your tests? If more than 10 survive,
   analyse at least 10, starting with lines your PR added or changed.
3. Add or strengthen **at least one test that kills a survivor** that revealed a gap, and re-run
   the tool to confirm it is killed. If no survivor reveals a gap (all killed or equivalent), say so
   and justify it in the surviving-mutants table; that completes this step.
4. Answer the three questions in the write-up.

Files:

- Write-up: `docs/sprint3/mutation/<issue#>-<github-username>.md`, using the template below.
- Reports: machine-readable output of the first run and the re-run, as
  `docs/sprint3/mutation/reports/<issue#>-<github-username>-before.json` and `-after.json` (`.txt`
  for go-mutesting). StrykerJS: add `"json"` to `reporters`. Gremlins: `--output=<file>.json`.
  go-mutesting: save the terminal output to a text file. The scores in the write-up must match the
  reports, and the mutant your new test kills must show as killed in the "after" report.
- The tool configuration you used, committed and linked from the write-up. Reusing a teammate's
  setup is encouraged, as long as you run the tool yourself and credit them. The team's shared
  setup (go-mutesting for Go), its commands and its pitfalls: `.claude-students/docs/mutation-testing.md`.

Common problems:

- **The tool stops before mutating.** It first runs the selected tests on unchanged code and stops
  if any fail. Fix or exclude them first; the D8 baseline shows which ones fail already.
- **Tests run against built code.** Mutants in the source are never seen by tests that run on
  compiled output. Configure a rebuild, or pick tests that run on source.
- **Tests need services.** Start the database or service first. How each test type runs is in
  `.claude-students/docs/testing.md`.
- **False survivors.** When limiting which tests run, include the tests that cover the mutated
  files. Otherwise mutants they would catch show as survived or not covered. Check that the
  relevant tests ran before writing a new test for a survivor.
- **Blocked?** Ask teammates first. Then tell the TA **as early as possible**, with the errors and
  the AI log of what you tried; the TA may approve a different issue or area. Problems reported at
  the end of the sprint don't excuse a missing D6.

Template, copied from the Sprint 3 page:

```markdown
# Mutation testing: Issue #<N> -- <issue title>

**PR(s):** #...  **Tool:** ...  **Config:** [link; credit the teammate if you reused their setup]  **Command:** `...`
**Reports:** [before](reports/<issue#>-<username>-before.json) · [after](reports/<issue#>-<username>-after.json)
**Mutation score:** before ...% → after ...%

## Surviving mutants

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| | | | |

## New or strengthened test

Commit: [link] · Test: [file › test name] · Kills: [mutant location]

## Questions

1. What did the surviving mutants reveal about your tests?
2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.
3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."
```

Grading (15 points): tool run on the files changed by the PR, with configuration, command, results
and reports committed 4; surviving mutants analysed correctly (equivalent or test gap, with an
explanation) 4; at least one new or strengthened test kills a survivor, confirmed by the re-run 4;
answers show understanding, not just description 3. CSCI 535: the average of the two write-ups.

## Sprint report (D8) sections

`docs/sprint<N>/report.md` (or `.pdf` or `.docx`), following `docs/sprint0/report.md`:

- Team: names, GitHub usernames, project name, repository link.
- Sprint overview.
- Backlog: milestone link, why the team chose these issues, plus a table of title, owner, points,
  scope and status.
- Requirements and design: when specs were done, what surprised you, design rationale.
- Completed issues: title, owner, PR link, author, reviewer, what changed.
- Test strategy, including untestable changes and why.
- AI tool usage per member: tools, session count, AI log folder link, and 2–3 sentences on lessons
  learned.
- Release: tag and release link.
- Risks and retrospective: what worked, what slowed the team down, next-sprint changes.
- Next sprint plan: initial ideas and carryover issues.
- New in Sprint 3:
  - **Suite health** (keep it short). A baseline table of the test suites relevant to the team (the
    packages or areas it works on, not the whole repository): for each suite, the command or CI
    workflow and the runner's summary line (passed, failed, skipped) at the Sprint 2 tag
    `v1.27.3-csci435-s2` and at the end of the sprint. Use CI results where available; locally, run
    only the packages you change. Failures are grouped by cause (e.g. environment-specific, already
    broken upstream, flaky, caused by our changes), each cause linking one issue that lists the
    affected tests. Any test the team's changes broke is fixed by the end of the sprint;
    pre-existing failures only need recording and grouping. Every skipped test and disabled CI
    check has a linked issue and a reason in a code comment.
  - **Mutation testing:** a link to each member's D6 write-up.
  - **Hardening issues:** which issues they are and where they came from.
  - **Carryovers:** the Sprint 2 issues carried over, and what changed.
- The report covers this sprint's work only. Write the Sprint 3 sections fresh rather than copying
  a previous report, and check that tables and links render on GitHub (no escaped `\|` or `\[`).
