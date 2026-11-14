# Mutation testing: Issue #155 -- Add a "Suggest change" button to the diff comment form

**PR(s):** #162

**Tool:** StrykerJS with the Vitest runner

**Config:** [controlled-opposition.stryker.config.mjs](controlled-opposition.stryker.config.mjs), ab-wm's personal config pattern with my own report paths, on top of the team's shared [stryker.config.mjs](../../../stryker.config.mjs) ([setup](../../../.claude-students/docs/mutation-testing.md))

**Command:** `pnpm exec stryker run docs/sprint3/mutation/controlled-opposition.stryker.config.mjs --mutate web_src/js/features/repo-diff-suggestion.ts,web_src/js/features/repo-issue.ts:289-289,web_src/js/features/repo-issue.ts:300-300`

**Reports:** [before](reports/155-controlled-opposition-before.json) · [after](reports/155-controlled-opposition-after.json)

**Mutation score:** before 81.10% → after 88.98% (covered code: 87.29% → 95.76%)

## Surviving mutants

PR #162 added `repo-diff-suggestion.ts` and changed two lines of `repo-issue.ts`, so the run mutates the whole new file and only those two lines (289 and 300) of the old one. That gives 127 mutants: 99 killed, 4 timed out, 15 survived and 9 not covered. All 15 survivors are below.

The 9 uncovered mutants are all on the two `repo-issue.ts` lines, inside the click handler that opens the code comment form. No Vitest test loads that handler; the Playwright test `tests/e2e/pr-suggest-change.test.ts` covers it, and Stryker does not run Playwright.

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `web_src/js/features/repo-diff-suggestion.ts:29` | `newline === -1` → `false` | Test gap | On a last line with no newline after it, the line is cut one character short, so an opening ```` ``` ```` there is no longer a fence. No test put the cursor on an unclosed fence on the last line (AC12, CommonMark: an unclosed block runs to the end). |
| `web_src/js/features/repo-diff-suggestion.ts:29` | `-1` → `+1` | Test gap | When the first line is one character long, that line is treated as running to the end of the text, so a block after it is missed. No test had a one-character first line before a block. |
| `web_src/js/features/repo-diff-suggestion.ts:30` | regex `^` anchor removed | Test gap | A run of backticks after other text (```` x ``` ````) would open a block. CommonMark allows at most 3 spaces before a fence; no test had text before one. |
| `web_src/js/features/repo-diff-suggestion.ts:30` | `~{3,}` → `~` | Test gap | One or two tildes would open a block. CommonMark needs at least 3; no test had a short tilde line. |
| `web_src/js/features/repo-diff-suggestion.ts:30` | regex `$` anchor removed | Not equivalent, but untestable in practice | Only differs on a line holding `\r`, U+2028 or U+2029, because `.` does not match them. Browsers turn the textarea's `\r` into `\n`. For U+2028 the mutant is the closer one to CommonMark, which does not count it as a line ending, so this survivor points to a small bug in the original code. |
| `web_src/js/features/repo-diff-suggestion.ts:33` | `` fence[1][0] === '`' `` → `true` | Test gap | Every fence with a backtick in its info string would be rejected, tilde fences included. CommonMark only forbids that for backtick fences; no test had a tilde fence with a backtick. |
| `web_src/js/features/repo-diff-suggestion.ts:35` | `pos <= lineEnd` → `false` | Equivalent | This early return is a shortcut. Without it the loop moves to the next line, where line 32 returns `pos` because `pos` is before that line's start; on the last line, line 42 returns `pos`. |
| `web_src/js/features/repo-diff-suggestion.ts:35` | the `return pos` block emptied | Equivalent | Same as above. |
| `web_src/js/features/repo-diff-suggestion.ts:35` | `<=` → `<` | Equivalent | Same as above, for `pos === lineEnd`. |
| `web_src/js/features/repo-diff-suggestion.ts:38` | `fence[1].length >= open.len` → `true` | Test gap | A shorter fence would close a longer one. The existing test for this used ```` ```js ````, which the info string check already rejects, so no test depended on the length check. |
| `web_src/js/features/repo-diff-suggestion.ts:38` | `fence[2].trim()` → `fence[2]` | Test gap | A closing fence followed by spaces would not close the block. CommonMark allows trailing spaces; no test had them. |
| `web_src/js/features/repo-diff-suggestion.ts:63` | `e.preventDefault()` removed | Test gap | The button has no `type` attribute and sits inside the comment `<form>`, so by the HTML spec a click submits the form. The unit test's button was not inside a form. |
| `web_src/js/features/repo-diff-suggestion.ts:68` | `/[\r\n]+$/` → `/[\r\n]+/` | Test gap, not worth a test | Only differs for a line holding a lone `\r` before its end, which Gitea's diff view rarely shows and no AC mentions. |
| `web_src/js/features/repo-diff-suggestion.ts:68` | `/[\r\n]+$/` → `/[\r\n]$/` | Test gap | On a CRLF file the block would keep the `\r`. Gitea's diff code does not strip it, and AC1 asks for the line's text alone. The test only used an LF line. |
| `web_src/js/features/repo-diff-suggestion.ts:69` | `textarea.value.slice(0, start)` → `textarea.value` | Test gap | Whether the block needs a newline in front would be decided by the end of the whole text, not the text before the cursor. AC2 says the block starts on a line of its own, but the click tests only inserted into an empty textarea or at its end. |

## New or strengthened test

Both tests are in one commit. Together they kill 10 of the 11 test-gap survivors; the one left is `/[\r\n]+/` on line 68, which the table explains. The after report has 5 survivors: the 3 equivalent ones on line 35 and the two untested ones on lines 30 and 68.

- Commit: [3d085eaf85](https://github.com/CSCI-435-SE/gitea/commit/3d085eaf85e06c41fd844cc9fc50639b6c0fbfc7) · Test: `web_src/js/features/repo-diff-suggestion.test.ts` › `suggestionInsertPos fence edge cases` · Kills: `repo-diff-suggestion.ts:29` (both), `:30` (`^` and `~`), `:33`, `:38` (both)
- Commit: [3d085eaf85](https://github.com/CSCI-435-SE/gitea/commit/3d085eaf85e06c41fd844cc9fc50639b6c0fbfc7) · Test: `web_src/js/features/repo-diff-suggestion.test.ts` › `initSuggestChangeButton in a form` · Kills: `repo-diff-suggestion.ts:63`, `:68` (`/[\r\n]$/`), `:69`

## Questions

1. What did the surviving mutants reveal about your tests?

The tests covered the cases the ACs list but not the CommonMark rules that the fence scan relies on. Most of the fence checks on lines 30 to 38 could be loosened without any test failing, because the tests only used backtick fences, and none had a short tilde run, text before a fence or spaces after a closing one. The click tests had the same blind spot: the button was not inside a form and the textarea was empty or the cursor was at its end, which hid the `preventDefault` call and the `slice` on line 69. Three equivalent mutants also showed that the early return on line 35 isn't needed, and the `$` survivor on line 30 turned up a small bug with U+2028.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

No. Every survivor was a case that no test asserted on. The new tests take their expected values from AC1, AC2 and AC12, from section 4.5 of the CommonMark spec, and from the HTML spec's rule that a button without a `type` submits its form.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

Claude Code (claude-opus-5-5) wrote the personal config from ab-wm's pattern, chose to mutate only the two lines PR #162 changed in `repo-issue.ts`, ran both Stryker runs, sorted the 15 survivors into gaps and equivalents, and wrote both tests. Before the re-run it checked that the new tests passed on the unchanged source. For two of the verdicts it read the code to confirm them: the `<button>` in `templates/shared/combomarkdowneditor.tmpl` has no `type` (so line 63 matters), and nothing in `services/gitdiff` strips `\r` (so line 68 matters). I approved the test commit after reading the tests against the ACs.
