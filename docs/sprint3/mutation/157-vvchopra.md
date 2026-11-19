# Mutation testing: Issue #157 -- S4. Record and show that a suggestion was applied

**PR(s):** #192

**Tool:** StrykerJS with the Vitest runner

**Config:** [vvchopra.stryker.config.mjs](vvchopra.stryker.config.mjs). It follows ab-wm's personal config pattern with my own report paths, on top of the team's shared [stryker.config.mjs](../../../stryker.config.mjs) ([setup](../../../.claude-students/docs/mutation-testing.md)).

**Command:** `pnpm exec stryker run docs/sprint3/mutation/vvchopra.stryker.config.mjs --mutate "web_src/js/markup/suggestion.ts:91-98,web_src/js/markup/suggestion.ts:123-123"`

**Reports:** [before](reports/157-vvchopra-before.json) · [after](reports/157-vvchopra-after.json)

**Mutation score:** before 81.25% → after 100.00% (covered code: 81.25% → 100.00%)

## Surviving mutants

PR #192's only TypeScript change is in `web_src/js/markup/suggestion.ts`:
- It adds `addAppliedMarker` (lines 89–99; the run mutates its body, 91–98).
- It adds the line that calls it in `initMarkupCodeSuggestion` (line 123).

The rest of the file belongs to #154 and #156, so it isn't mutated.

The run gives 16 mutants: 13 killed, 3 survived, none uncovered. All 3 survivors are below.

The PR's Go changes are covered by Go unit and integration tests and aren't part of this run. D6 asks for one tool on the issue's changed files, and the PR's frontend logic is all in this file.

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `web_src/js/markup/suggestion.ts:95` | `'tw-flex'` → `""` | Test gap | The header would lose its flex layout, so "Suggested change" and the marker would no longer sit at opposite ends. In a real browser it's worse: the DOM spec says `classList.add('')` throws a `SyntaxError`, so the marker would never be appended (AC1). It survived because happy-dom accepts an empty token without throwing, and no test checked the header's classes or layout. |
| `web_src/js/markup/suggestion.ts:95` | `'tw-items-center'` → `""` | Test gap | Same as above: the marker would lose its vertical centring, and a real browser would throw before appending it. |
| `web_src/js/markup/suggestion.ts:95` | `'tw-justify-between'` → `""` | Test gap | Same as above: the marker would no longer be pushed to the right of the header, where #156 puts Apply, and a real browser would throw first. |

## New or strengthened test

Commit: <link to the commit that adds this test, once pushed> · Test: `web_src/js/markup/suggestion.test.ts` › `an applied suggestion` › `#157 AC1 the header keeps the label on the left and the marker on the right, as for Apply` · Kills: `web_src/js/markup/suggestion.ts:95` (all three)

The test checks three things:
- the header carries the flex layout #156 gives it for Apply;
- the label is its first child;
- the marker is its last element.

The after report has 16 of 16 mutants killed, the three line 95 ones included.

## Questions

1. What did the surviving mutants reveal about your tests?

The marker tests checked *what* the header shows ("Applied by @ben in …", the commit link, no Apply), but not *how* it's laid out. Line 95 sets up the same header layout that #156 uses for the Apply button: label left, marker right. It could be emptied class by class without any test failing. The survivors also showed a difference between the test DOM and a browser. happy-dom ignores `classList.add('')`, but a browser throws, so in production these mutants would have stopped the marker from rendering at all. A test that only checks the marker's text would never notice the layout.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

No. None of the survivors came from a wrong assertion: the layout simply had no assertion. The new test's expected values come from:
- AC1: the marker is shown in the suggestion's header;
- #157's Design Decision: the marker takes the header slot #156 uses for Apply;
- the header classes #156's `addApplyButton` already sets for that slot.

They don't come from what the code happened to output.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

Claude Code (claude-opus-5-5) did the following:
- wrote the personal config from ab-wm's pattern;
- worked out from the git diff which lines PR #192 added to `suggestion.ts`, and limited `--mutate` to them;
- ran both Stryker runs;
- classified the three survivors;
- wrote the new test and checked that it passes on the unchanged source before the re-run.

For the verdict, it looked at why an empty class name survived and checked both environments:
- **Chromium:** `classList.add('tw-flex', '')` throws `SyntaxError: ... The token provided must not be empty`, as the DOM spec's `DOMTokenList.add` says.
- **happy-dom:** the same call silently leaves `class="a "`.

So the mutants are not equivalent in a browser, even though happy-dom lets them through. I checked the before and after reports' counts against the scores above. I also read the test against AC1 and the Design Decision before commiting. 
