# Mutation testing: Issue #118 -- Alert me to new notifications with a tab icon badge and opt-in desktop notifications

**PR(s):** #161

**Tool:** StrykerJS with the Vitest runner

**Config:** [CarsonRackley.stryker.config.mjs](CarsonRackley.stryker.config.mjs), copied from @ab-wm's [ab-wm.stryker.config.mjs](ab-wm.stryker.config.mjs) with my own report paths; both build on the team's shared [stryker.config.mjs](../../../stryker.config.mjs) ([setup](../../../.claude-students/docs/mutation-testing.md))

**Command:** `pnpm exec stryker run docs/sprint3/mutation/CarsonRackley.stryker.config.mjs --mutate web_src/js/modules/favicon-status.ts,web_src/js/modules/notification-alert.ts,web_src/js/features/notification-desktop.ts,web_src/js/features/notification.ts,web_src/js/eventsource.sharedworker.ts,web_src/js/modules/worker.ts`, run from the repository root in WSL

**Reports:** [before](reports/118-CarsonRackley-before.json) · [after](reports/118-CarsonRackley-after.json)

**Mutation score:** before 33.50% → after 33.67% (on lines #161 added: 63.01% → 63.47%)

The run mutates six of the seven source files #161 changed. `web_src/js/index.ts` is left out: #161 only added an import and one entry to its list of init functions there, no logic, and no test imports `index.ts`, so all its mutants would be "no coverage". The before report comes from re-running the config commit [132d9b0823](https://github.com/CSCI-435-SE/gitea/commit/132d9b08230f3fe2c383cfd1300cbe9e68150dd8), which doesn't have the new test. The first run of that commit had 14 timeouts on `favicon-status.ts` lines 6–20 (older code that sets the badge size), which Stryker counts as killed and which gave 35.59%. None of the later runs (one before, two after) had any timeouts. Between the before and after reports, the only mutant whose status changes is the one the new test kills.

Each run made 591 mutants: 197 killed before and 198 after, 96 → 95 survived, 295 with no coverage, and 3 that crashed the test runner. Most no-coverage mutants are in `eventsource.sharedworker.ts` and `worker.ts`, which have no automated tests (#161's Test Evidence explains why). Of the 219 mutants on lines #161 added, 138 were killed, 38 survived and 43 had no coverage.

## Surviving mutants

38 mutants survived on lines #161 added (37 after the new test). This table covers 10 of them, from all four tested files. The other 58 survivors are on lines #161 didn't change.

| Location (file:line) | Mutation | Equivalent or test gap? | Why |
|---|---|---|---|
| `web_src/js/modules/notification-alert.ts:45` | `if (count < this.baseline)` → `if (false)` | Test gap | If the count drops while Gitea is hidden (5 → 2) and then one notification arrives (3), the mutant computes 3 − 5 = −2 new, and `showDesktopAlert` ignores counts below 1, so a real new notification gets no alert (AC5). No test sent a rise after a drop. **Killed by the new test.** |
| `web_src/js/features/notification-desktop.ts:62` | `permission === 'granted'` → `true` | Test gap | Denying the browser prompt would still save the setting as on, which breaks AC4's "the setting stays off". The AC4 test checks `desktopAlertsEnabled()`, and that function also checks the permission, so it hides the stored value. |
| `web_src/js/features/notification-desktop.ts:57` | `render(false)` → `render(true)` | Test gap | After clicking Disable, the page would say notifications are blocked by the browser when they aren't. The AC9 test checks that alerts stop and Enable shows, not the blocked message. |
| `web_src/js/features/notification-desktop.ts:20` | `newCount <= 0` → `false` | Test gap | `showDesktopAlert(0)` would show "You have 0 new notifications". No test calls it with 0. Today's callers never pass 0, but the function is exported and nothing checks its guard. |
| `web_src/js/features/notification.ts:26` | `getCurrentCount() > 0` → `true` | Test gap | The tab icon would show the dot on page load even with no unread notifications. AC1 says the dot reflects the count "when any Gitea page loads", but no test checks the icon on page load. |
| `web_src/js/features/notification.ts:130` | `data.new !== 0` → `false` | Test gap | On the polling fallback the dot would disappear at the first poll and the poller would never bring it back (AC1 when the event stream is unavailable). The poller tests check the count text and alerts, never the icon. |
| `web_src/js/features/notification.ts:79` | `newCount > lastCount` → `newCount >= lastCount` | Equivalent | This line is inside `if (lastCount !== newCount)` (line 76), so the two counts are never equal here and `>` and `>=` always give the same result. |
| `web_src/js/modules/notification-alert.ts:45` | `count < this.baseline` → `count <= this.baseline` | Equivalent | The only new case is when the two are equal, and then it assigns `baseline` the value it already has. |
| `web_src/js/modules/notification-alert.ts:46` | `lastCount === null \|\| …` → `false \|\| …` | Equivalent | `lastCount` is only `null` before any tab has reported, and with no tabs `onCount` has already returned at line 40. |
| `web_src/js/modules/favicon-status.ts:125` | `if (show === notificationDot) return` → `if (false) return` | Equivalent | Without the early return the icon is redrawn with the same cached image. It is extra work, but the result is identical. |


## New or strengthened test

Commit: [8f9e3faecd](https://github.com/CSCI-435-SE/gitea/commit/8f9e3faecda1f0395b25b676c99c8e6df2d1297c) · Test: `web_src/js/modules/notification-alert.test.ts` › "AC5 after notifications are read elsewhere while hidden, the next alert counts only the new one" · Kills: `web_src/js/modules/notification-alert.ts:45` (`if (count < this.baseline) this.baseline = count;` → `if (false) this.baseline = count;`), Survived in the before report and Killed in the after report

The tab is visible with 5 unread, goes hidden, the count drops to 2 (read on another device) and then rises to 3. The test expects exactly one alert, for 1 new notification. That value comes from the scenario (one notification arrived) and AC5's alert saying how many new notifications there are, not from running the code. With the mutant, the tracker reports −2 and the test fails. In the after report this mutant is Killed by this test.

## Questions

1. What did the surviving mutants reveal about your tests? My tests checked the normal case for each AC, but missed some cases around it. For example no test had the count go down and then back up while Gitea was hidden. Because of that, a real missed alert went unnoticed. My new test covers this. As well as no test checked the tab icon dot from `notification.ts`, on page load or after polling. The favicon tests only test the favicon file itself, and the poller tests only check the number shown. Some survivors didn't need a test at all. For example, line 76 already makes sure the two counts are different, so `>` and `>=` on line 79 always give the same answer.

2. Did any surviving mutant point to an assertion whose expected value was not based on the requirements? Explain. Yes. AC4 says that if the user denies permission, "the setting stays off". My AC4 test checks that `desktopAlertsEnabled()` is false. But that function is false whenever permission isn't granted, no matter what was saved. So the mutant at `notification-desktop.ts:62`, which saves the setting as on after a denial, still passes the test. The test should check the saved setting itself, since that is what the AC talks about.

3. If you used AI to configure the tool, analyze survivors, or write tests, what did it produce and what did you verify or correct? If AI was not used, write "N/A."
I used Claude Code (claude-opus-5-5). It copied @ab-wm's config with my report paths, ran Stryker, listed every survivor on #161's lines with what would break, suggested a verdict for each (I picked the 10 for the table), and wrote the new test. It also drafted the header, scope and new-test sections of this write-up, and fixed two errors in my table at my request.

What I checked:

- For each of the 10 mutants, I opened the line in the code and made sure the verdict made sense. For the equivalent ones, I checked the line it depends on (for example, line 76 for line 79).
- I read the new test and worked through it by hand: 5 → 2 → 3 should give 1 new, and the mutant gives 3 − 5 = −2.
- I checked that the test's expected value comes from AC5 and the scenario, not from the code's output.
- I checked that the mutant shows as Survived in the before report and Killed in the after report, and that the scores here match both reports.

What was corrected: the AI first called line 79 (`>` → `>=`) a test gap, then saw that line 76 makes it equivalent. My first before run scored higher only because of timeouts in older code, so I re-ran it to get a fair before report.
