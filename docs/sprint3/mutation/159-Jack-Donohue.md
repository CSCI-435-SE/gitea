# Mutation testing: Issue #159 -- S6. Apply several suggestions in one commit

**PR(s):** #196

**Tool:** StrykerJS with the Vitest runner

**Config:** [Jack-Donohue.stryker.config.mjs](Jack-Donohue.stryker.config.mjs). It copies vvchopra's and ab-wm's personal config pattern, with my own report paths, on top of the team's shared [stryker.config.mjs](../../../stryker.config.mjs) ([setup](../../../.claude-students/docs/mutation-testing.md)). It's the same file as in my #158 write-up.

**Command:** `pnpm exec stryker run docs/sprint3/mutation/Jack-Donohue.stryker.config.mjs --mutate "web_src/js/markup/suggestion.ts:3-3,web_src/js/markup/suggestion.ts:25-43,web_src/js/markup/suggestion.ts:47-48,web_src/js/markup/suggestion.ts:50-51,web_src/js/markup/suggestion.ts:85-86,web_src/js/markup/suggestion.ts:99-185,web_src/js/markup/suggestion.ts:225-228"`

**Reports:** [before](reports/159-Jack-Donohue-before.json) · [after](reports/159-Jack-Donohue-after.json)

**Mutation score:** before 89.55% → after 96.27% (no uncovered mutants, so covered code is the same)

## What was mutated

The `--mutate` list holds exactly the lines of `web_src/js/markup/suggestion.ts` that #159 added or changed. They come from `git diff -U0 origin/main HEAD` after `main` (with S4) was merged into the branch:
- `postApply`, the POST-and-reload code moved out of #156's Apply so the batch bar can share it;
- the button group Apply now sits in;
- the batch state, `batchMessage`, `updateBatchBar` and `addBatchToggle`;
- the call in `initMarkupCodeSuggestion`.

The PR's Go changes (`ApplySuggestions`, the handler, the error mapping) are covered by Go unit and integration tests and aren't part of this run. D6 asks for one tool. go-mutesting mutates whole files and re-runs `services/repository/files`' database-backed tests for every mutant, which would take hours.

## Results

The before run has 134 mutants: 120 killed and 14 survived. None are uncovered. All 14 survivors are below.

## Surviving mutants

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `web_src/js/markup/suggestion.ts:32` | `return ''` → `return "Stryker was here!"` | Test gap (killed) | A successful Apply would be treated as refused: the page would show the text as an error and re-enable Commit while it reloads. No test checked the page after a success, only that it reloaded. |
| `web_src/js/markup/suggestion.ts:35` | `if (json.errorMessage)` → `if (true)` | Test gap (killed) | A refusal whose JSON has no `errorMessage` would return `undefined`. The caller would then treat it as a success and show nothing, leaving Commit disabled. The tests covered a body that isn't JSON, not JSON without a message. |
| `web_src/js/markup/suggestion.ts:48` | `'tw-flex tw-gap-1'` → `""` | Test gap (styling, not fixed) | Apply and Add to batch would stack instead of sitting side by side in the header. Only the layout changes; the buttons still work. The unit tests don't check Tailwind classes, and the E2E test sees both buttons. |
| `web_src/js/markup/suggestion.ts:86` | `if (!errorMessage) return;` → `if (false)` | Test gap (killed) | Same as line 32, from the other side: after a successful single Apply, an empty error box would be shown and Commit re-enabled. |
| `web_src/js/markup/suggestion.ts:116` | `batchBar?.el.remove()` → `batchBar.el.remove()` | Equivalent | The bar is removed when the batch becomes empty, which only happens by removing a suggestion that was in it. The bar exists by then, so `batchBar` is never null here. |
| `web_src/js/markup/suggestion.ts:126` | `'ui small input tw-flex-1'` → `""` | Test gap (styling, not fixed) | The message field would lose Fomantic's input style and wouldn't stretch across the bar. Layout only. |
| `web_src/js/markup/suggestion.ts:128` | `message.type = 'text'` → `""` | Equivalent | An input with an empty or missing `type` is a text input (HTML spec, the "missing value default"). |
| `web_src/js/markup/suggestion.ts:129` | `'aria-label'` → `""` | Test gap (killed) | The bar's message field would have no accessible name, so screen readers would announce an unlabelled text box. #156's form labels its field "Commit message", and no test checked the bar's. |
| `web_src/js/markup/suggestion.ts:131` | `'ui small primary button'` → `""` | Test gap (styling, not fixed) | Apply suggestions would render as an unstyled button. Layout only; its label and click handling are tested. |
| `web_src/js/markup/suggestion.ts:140` | `applyButton.disabled = true` → `false` | Test gap (killed) | A second click while the first batch is posting would post again. The repeat would be refused as "new commits since the page was loaded", and that message would show right after the batch had succeeded. #156 has this test for Apply; the bar had none. |
| `web_src/js/markup/suggestion.ts:141` | `hideElem(error)` removed | Test gap (killed) | After a refusal, retrying would keep the old message on screen while the new request posts. |
| `web_src/js/markup/suggestion.ts:147` | `if (!errorMessage) return;` → `if (false)` | Test gap (killed) | The batch version of line 86: after a successful batch, an empty error would show and Apply would be re-enabled while the page reloads. |
| `web_src/js/markup/suggestion.ts:149` | `showElem(error)` removed | Test gap (killed) | A refusal's message would be written into the bar but stay hidden. The AC3 test checked the error's text, not that it was visible. |
| `web_src/js/markup/suggestion.ts:177` | `other.start <= item.end` → `true` | Test gap (killed) | Any suggestion *above* one already in the batch, in the same file, would be refused as overlapping. That would block AC9's own example if the lower suggestion is added first. Every AC2 test added suggestions top to bottom. |

## New or strengthened test

Commit: [5c529c2458](https://github.com/CSCI-435-SE/gitea/commit/5c529c2458)

| Test (`web_src/js/markup/suggestion.test.ts`) | Kills |
|---|---|
| `applying a suggestion` › `AC1 a commit that succeeds shows no error and keeps the button disabled while the page reloads` | `:32`, `:86` |
| `applying a suggestion` › `a refusal whose JSON has no message shows the generic error` | `:35` |
| `applying several suggestions in one commit` › `AC2 AC9 a suggestion above one already in the batch, in the same file, is accepted` | `:177` |
| `applying several suggestions in one commit` › `the bar's message field is labelled like Apply's` | `:129` |
| `applying several suggestions in one commit` › `Apply posts once while posting, shows a refusal, hides it on retry, and shows nothing once it succeeds` | `:140`, `:141`, `:147`, `:149` |

All five passed on the unchanged source before the re-run. The after report has 129 killed and 5 survived, with the nine mutants above listed as killed. The 5 left are the two equivalent mutants and the three styling class strings.

## Questions

1. What did the surviving mutants reveal about your tests?

The tests checked what happens when things go as expected: a refusal shows its text, a success reloads. They missed the edges:
- what the page shows *after* a success (nothing), and what it does on a second click while posting (nothing);
- whether a refusal's message is actually visible, not just written into the bar;
- JSON without a message;
- adding a suggestion that sits above one already in the batch, rather than below.

The last one would have broken AC9's own example (a change on line 2 and one on line 5) whenever the line-5 suggestion was picked first. Moving #156's POST code into the shared `postApply` also left its success path with no direct test, which the line 32 and 86 survivors showed.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain.

No. None of the survivors came from a wrong expected value; each came from a case or a property with no assertion. The new tests' expected values come from:
- #159's ACs: AC2 refuses only suggestions that share a line; AC9 is lines 2 and 5 in one file; AC3 says the refusal is shown;
- #156's existing behaviour, which #159 must keep: the generic error when a refusal has no message, the "Commit message" label, and one commit per click.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."

Claude Code (claude-opus-5-5) did the following:
- found #159's lines with `git diff` against `main`;
- ran both Stryker runs;
- classified the 14 survivors;
- wrote the five tests, and checked that they pass on the unchanged source before the re-run.

I checked:
- the two equivalent verdicts: when the batch can become empty with no bar, and the HTML default for an input with no `type`;
- that the three styling survivors only change layout;
- the before and after counts in the JSON reports against the scores above;
- each new test's expected value against the AC or #156 behaviour it cites.
