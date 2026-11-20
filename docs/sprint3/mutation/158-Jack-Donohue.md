# Mutation testing: Issue #158 -- S5. Suggest a change across several lines

**PR(s):** #177

**Tool:** StrykerJS with the Vitest runner

**Config:** [Jack-Donohue.stryker.config.mjs](Jack-Donohue.stryker.config.mjs). It copies vvchopra's and ab-wm's personal config pattern, with my own report paths, on top of the team's shared [stryker.config.mjs](../../../stryker.config.mjs) ([setup](../../../.claude-students/docs/mutation-testing.md)).

**Command:** `pnpm exec stryker run docs/sprint3/mutation/Jack-Donohue.stryker.config.mjs --mutate "web_src/js/features/repo-diff-range.ts:1-64,web_src/js/features/repo-diff-suggestion.ts:57-58,web_src/js/features/repo-diff-suggestion.ts:68-68,web_src/js/features/repo-diff.ts:55-58,web_src/js/features/repo-issue.ts:14-14,web_src/js/features/repo-issue.ts:268-268,web_src/js/features/repo-issue.ts:292-293,web_src/js/features/repo-issue.ts:297-298,web_src/js/features/repo-issue.ts:305-305,web_src/js/markup/suggestion.ts:116-118"`

**Reports:** [before](reports/158-Jack-Donohue-before.json) · [after](reports/158-Jack-Donohue-after.json)

**Mutation score:** before 72.58% → after 75.81% (covered code: 91.84% → 95.92%)

## What was mutated

The `--mutate` list holds exactly the TypeScript lines that PR #177 added or changed and that are still on `main`:
- `git blame` of each file at `main` (`9124a600db`);
- keeping the lines that come from #177's six commits (`git rev-list --no-merges bb082e84b3^1..bb082e84b3^2`).

That covers:
- all of the new `repo-diff-range.ts` (range selection);
- the range prefill in `repo-diff-suggestion.ts`;
- the comment form's error toast in `repo-diff.ts`;
- the wiring in `repo-issue.ts`;
- the range split in `markup/suggestion.ts`.

The PR's Go changes (range storage, validation and invalidation in `models/issues` and `services/pull`) are covered by Go unit and integration tests and aren't part of this run. D6 asks for one tool. go-mutesting mutates whole files and re-runs each package's database-backed tests for every mutant, which would take hours for `services/pull/review.go`.

## Results

The before run has 124 mutants: 90 killed, 8 survived, 26 not covered.
- **The 26 uncovered mutants** are in `repo-diff.ts:55-56` (the error toast when the server refuses a range) and `repo-issue.ts` (wiring `initDiffRangeSelect`, the `+` handler that sends `start_line`, and passing the range to Suggest change).
  - These lines have no Vitest test. They're covered by the E2E test `pr-suggest-change.test.ts › suggest a change across several lines` (#158 AC1, AC4, AC11, AC15), which Stryker's Vitest runner doesn't run.
  - They aren't survivors: no unit test ran them, so the run says nothing about their tests.
- **All 8 survivors** are analysed below.

## Surviving mutants

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `web_src/js/features/repo-diff-range.ts:8` | `tr.querySelector('td.lines-num-new')?.getAttribute` → `.getAttribute` | Test gap (killed) | A conversation under a line is its own `<tr class="add-comment">` with one cell and no line numbers (`templates/repo/diff/section_unified.tmpl`). Without `?.`, a range that spans an existing thread throws a `TypeError` instead of skipping the row. Every fake row in the tests had a line-number cell, so none had that shape. |
| `web_src/js/features/repo-diff-range.ts:32` | `selected.length < 2` → `<= 2` | Test gap (killed) | A two-line range would no longer count as a range, so the form would open as a single-line comment. The tests only checked one line (AC3) and three lines (AC1). The boundary of two was never checked. |
| `web_src/js/features/repo-diff-range.ts:38` | `'left'` → `""` | Equivalent | `side` is only ever compared with `'right'` (lines 41 and 45). Any other value behaves the same as `'left'`. |
| `web_src/js/features/repo-diff-range.ts:44` | `window.getSelection()?.removeAllRanges()` → `.removeAllRanges()` | Equivalent | `getSelection()` returns null only for a document with no browsing context (HTML spec). The diff page is always shown in a window, so the `?.` never matters there. |
| `web_src/js/features/repo-diff-range.ts:45` | `side === 'right'` → `true` | Test gap (killed) | The mutant only checks the side the range started on. A range started on the new side and shift-clicked on the old side would be selected instead of refused. AC11 says the refusal holds "or the other way round", but the test only went from old to new. |
| `web_src/js/features/repo-diff-range.ts:61` | `e.preventDefault()` removed | Equivalent | Enter on a focused line-number cell (a `td` with `tabindex="0"`) has no default action. It isn't a link, button or form control, and the diff table isn't inside a form. |
| `web_src/js/features/repo-diff-suggestion.ts:68` | `/[\r\n]+$/` → `/[\r\n]$/` | Test gap (killed) | Only one line-ending character would be stripped, so in a CRLF file the `\r` would stay on each prefilled line. The test only had `\n` endings. |
| `web_src/js/features/repo-diff-suggestion.ts:68` | `/[\r\n]+$/` → `/[\r\n]+/` | Test gap (edge case, not fixed) | The unanchored pattern strips the first CR/LF run, not the trailing one. Git splits lines on LF only, so this differs only for a line with a bare `\r` before its end. Such a line is possible but very rare, so I recorded it rather than adding a test for it. |

## New or strengthened test

Commit: [39956a42d6](https://github.com/CSCI-435-SE/gitea/commit/39956a42d6)

| Test | Kills |
|---|---|
| `web_src/js/features/repo-diff-range.test.ts` › `collectRangeRows skips a comment thread inside the range` | `repo-diff-range.ts:8` |
| `web_src/js/features/repo-diff-range.test.ts` › `a two-line range is a range` | `repo-diff-range.ts:32` |
| `web_src/js/features/repo-diff-range.test.ts` › `a range from the new side to the old side is refused` | `repo-diff-range.ts:45` |
| `web_src/js/features/repo-diff-suggestion.test.ts` › `initSuggestChangeButton drops a CRLF line ending` | `repo-diff-suggestion.ts:68` (`/[\r\n]$/`) |

All four passed on the unchanged source before the re-run. The after report has 94 killed and 4 survived, and lists the four mutants above as killed. The 4 left are the three equivalent ones and the edge case.

## Questions

1. What did the surviving mutants reveal about your tests?

The tests were built on simplified DOM and only the obvious cases:
- every fake row had a line-number cell;
- the ranges were one line or three lines;
- the cross-side refusal was tried in one direction only;
- every code line ended in `\n`.

The survivors sat exactly at those gaps: the two-line boundary, the other direction of AC11, CRLF files, and a real comment-thread row. The thread row is the important one. In the real page a range can span an existing conversation, and without the `?.` that would throw. The fake rows could never show that.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

No. None of the survivors came from a wrong expected value; each came from a case with no test at all. The new tests' expected values come from #158's ACs:
- AC1: a range of more than one line, whose thread sits under its last line;
- AC3: only one line is not a range;
- AC11: "or the other way round";
- AC4: the block holds the lines themselves.

The thread row's markup comes from `section_unified.tmpl`. None of them came from what the code returned.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

Claude Code (claude-opus-5-5) did the following:
- wrote the personal config from vvchopra's;
- found #177's lines on current `main` with `git blame` against the PR's six commits;
- ran both Stryker runs (it called `node_modules/.bin/stryker` directly, because `pnpm exec` in a separate worktree tried to reinstall dependencies);
- classified the survivors;
- wrote the four tests, and checked that they pass on the unchanged source before the re-run.

I checked:
- each classification against the code, in particular the comment-row markup in the diff template and the claim that Enter on a `td` has no default action;
- the before and after counts in the JSON reports against the scores above;
- that each new test's expected value follows from the AC it cites.
