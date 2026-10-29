# Report: Sprint 2

**Team Name:** GiCoffee

**Team member names:** Vishav Chopra (vvchopra), Carson Rackley (CarsonRackley), Arjun Bhat (arjunsb26), Aman Bhoot (ab-wm), Jack Donohue (Jack-Donohue), Ethan Turner (controlled-opposition)

**Project Name:** Gitea

**Project Link:** [https://github.com/CSCI-435-SE/gitea](https://github.com/CSCI-435-SE/gitea)

## Sprint Overview

### What we planned to do

Sprint 2 started from the Sprint 1 carryovers. The keyboard-shortcut sub-issues of #23 (#56–#60, plus #95), the dropdown regression suite from #25 (#65), notification sorting from #7 (#52), and the PostgreSQL failure in #77's burndown test all went into the backlog first, as the Sprint 1 plan said they would.

The shortcut work was planned in dependency order. #56 replaces Gitea's single page-wide key handler with a central shortcut registry, and #57, #58, #59, #60 and #95 all build on it, so #56 had to merge first. The rest of the team took new medium issues so that every CSCI 435 member owned at least two:

- Calendar deadlines (#36), split into #98 and #100
- More `<aria-menu>` migrations and an ARIA bug under #25 (#104, #108, #109, #110)
- A second notifications feature under #7 (#117)
- A milestone follow-up to Sprint 1's burndown chart (#131)

### Scope changes

- **#52 depended on #117.** #52's sort had to work inside #117's repository groups, so #117 was written and merged first (PR #120), and #52 followed in PR #122.
- **#118 was deferred to Sprint 3.** The tab badge and desktop notifications issue under #7 was taken off the milestone. Its owner explained on the issue that #52's migration and notifier changes left no time to finish it this sprint.
- **#101 and #102 were left off the milestone.** #36 was split into four issues: the calendar view (#98) and the public iCalendar feed (#100) were kept for this sprint, and the private-repository feed with revocable tokens (#101) and the personal cross-repository feed (#102) were left for later.

In total, all 16 issues on the Sprint 2 milestone are closed, each by a merged PR. The PostgreSQL burndown fix was merged as PR #125.

## Sprint 1 Carryovers

| Sprint 1 item | What happened in Sprint 2 |
| --- | --- |
| #52 Sort notifications by reason or type (moved from Sprint 1, needed a migration and a queue change) | Taken. Merged in PR #122 with a database migration that stores each notification's reason. |
| #56–#60 keyboard shortcuts (deferred from Sprint 1) | Taken. All five merged: #96, #115, #97, #114 and #121. |
| #65 Playwright keyboard regression suite for dropdowns | Taken. Merged in PR #105. |
| #77's burndown test failing on PostgreSQL | Taken. Fixed in PR #125: changes made within the same second are now ordered by issue, oldest first, so the result no longer depends on the database's row order. |

## Sprint Backlog

### Sprint 2 GitHub Milestone Link

[https://github.com/CSCI-435-SE/gitea/milestone/3](https://github.com/CSCI-435-SE/gitea/milestone/3)

### Why we chose these issues

- **Carryovers first.** The course rules require unfinished Sprint 1 issues to go into the backlog before new work, so #52, #56–#60 and #65 were added first.
- **Finishing #23's shortcut work.** Building #56's registry first made the other five shortcut issues small and independent, so they could be split between ab-wm and controlled-opposition. #95 (`r` to quote-reply) was added because it uses the same registry.
- **Finishing #25's accessibility work.** Sprint 1 built the `<aria-menu>` component (#61, #62). #108, #109 and #110 move the remaining action menus onto it, and #104 fixes an ARIA bug that #65's new tests are meant to catch.
- **New features with a clear user.** #117 (group notifications by repository) and #52 finish the notifications overhaul (#7). #98 and #100 start the calendar epic (#36). #131 builds on the burndown chart from #14.

### Table of Issues

Estimated points come from each issue's scope label, as in Sprint 1: small = 1, medium = 2. #98 and #100 have no scope label on GitHub, and both are counted as medium from the Scope sections in their specs.

| Issue Title | Issue Owner | Estimated Points | Scope | Status |
| --- | --- | --- | --- | --- |
| #52 Sort notifications by reason or type | CarsonRackley | 2 | Medium | Done (PR #122) |
| #56 Central keyboard-shortcut registry with chords | ab-wm | 2 | Medium | Done (PR #96) |
| #57 `?` opens a shortcut help dialog built from the registry | controlled-opposition | 2 | Medium | Done (PR #115) |
| #58 `g` navigation shortcuts for repository tabs and notifications | ab-wm | 2 | Medium | Done (PR #97) |
| #59 GitHub's `c` shortcuts for the issue and PR lists and the PR commit picker | ab-wm | 1 | Small | Done (PR #114) |
| #60 Character-key shortcuts can be turned off (WCAG 2.1.4) | controlled-opposition | 1 | Small | Done (PR #121) |
| #65 Playwright keyboard regression suite for dropdowns | controlled-opposition | 1 | Small | Done (PR #105) |
| #95 `r` quotes selected text into the reply box | controlled-opposition | 1 | Small | Done (PR #116) |
| #98 Show milestone and issue deadlines in a repository calendar view | vvchopra | 2 | Medium | Done (PR #123) |
| #100 Subscribe to a public repository's milestones and issue deadlines as an iCalendar feed | vvchopra | 2 | Medium | Done (PR #129) |
| #104 Label picker keeps `aria-expanded="true"` after Escape closes it | Jack-Donohue | 1 | Small | Done (PR #124) |
| #108 Move the Code tab, branch list and commit page action menus to `<aria-menu>` | Jack-Donohue | 2 | Medium | Done (PR #127) |
| #109 Move the "…" row menus on the Actions, Projects and Packages pages to `<aria-menu>` | arjunsb26 | 2 | Medium | Done (PR #135) |
| #110 Move the "Add webhook", "Link account" and admin notice "Operations" menus to `<aria-menu>` | Jack-Donohue | 2 | Medium | Done (PR #128) |
| #117 Group notifications by repository | CarsonRackley | 2 | Medium | Done (PR #120) |
| #131 Show each open milestone's projected finish on the milestones list | arjunsb26 | 2 | Medium | Done (PR #134) |

## Requirements & Design

### Were specs complete before coding started?

Yes. Every issue was written in the course format: User Story, User Scenario, numbered Acceptance Criteria, Out of Scope, Open Questions and Design Decisions. Every issue also got a "Spec reviewed" comment from a teammate before its PR was opened:

- The shortcut and test issues (#56–#60, #65, #95) were reviewed on Oct 18, the day #56's PR was opened.
- The later issues (#52, #98, #100, #104, #108, #110, #117) were reviewed between Oct 26 and Oct 27, before or on the day their PRs were opened.

This sprint, reviewers checked each spec against the code, not just its wording. That caught real problems before implementation:

- **#104:** the review found that the planned fix, wrapping the dropdown's `onHide`, would never run for the label picker, because the picker later replaces `onHide` with its own setting. Decision 1 was changed before any code was written.
- **#108:** the commit page's cherry-pick, branch and tag dialogs sit inside the menu being migrated. `<aria-menu>` treats typed keys as type-ahead, so moving them as-is would have broken typing a branch or tag name. The modals were moved out of the menu, and an acceptance criterion was added for it.
- **#52:** the review found that repository transfer notifications never pass through the code that sets a reason. It also suggested storing the reason as a number that matches its rank, so "Sort: Reason" is a plain `ORDER BY` that keeps paging and #117's grouping working. Both points became design decisions.
- **#58 and #59** were approved with empty Design Decision sections. Their decisions were recorded in comments on the issues during implementation instead of in the issue bodies.

**Surprises:**

- **The cost of #52's migration.** Storing a reason on every notification needed a migration plus changes to how the notifier queues each kind of notification. That was a large part of why #118 was deferred.
- **#117 and #52 overlapped.** #52's sort had to work inside #117's groups, so #52 was redesigned (its Decision 5) after #117 merged.
- **#100 added a dependency.** It brought in the first new Go library of the sprint, `golang-ical`. Its spec recorded the alternatives (a hand-written serializer and `emersion/go-ical`) and the fallback if the dependency was rejected.

### Key design decisions

Every implemented issue records at least one decision with two or more alternatives, as D3 requires.

- **#56:** All shortcuts are kept in one registry in the page code, and a shortcut can be scoped to part of the page, instead of being found through `<kbd>` hints in the HTML. This is what made two-key sequences such as `g i` possible.
- **#57:** The help dialog uses the native `<dialog>` element, and its markup and text are rendered on the server so they can be translated.
- **#60:** The on/off setting is stored in the browser, and `?` keeps working when shortcuts are off, so the user can always turn them back on.
- **#65:** ARIA state is asserted in Playwright, by extending the existing label picker and navbar tests rather than adding a new suite.
- **#95:** `r` shares one quoting helper with the existing "Quote reply" menu item, and it is only registered on pages that have a reply box.
- **#98:** The month grid is rendered on the server with a Go template, reusing the deadline bounds from Sprint 1's #34. Month navigation and filters are plain links.
- **#100:** The iCalendar feeds are generated with `golang-ical` and served from new handlers in `routers/web/feed/`, behind the same `feedEnabled` guard and permission check as the RSS and Atom feeds. Private repositories return 404 until #101 adds tokens.
- **#104:** The ARIA state is refreshed when the dropdown module's own `hide` runs, rather than through `onHide`, which the label picker overrides.
- **#108 and #110:** Only action menus move to `<aria-menu>`. Menus that show a current selection, like Sort and Filter, stay on Fomantic. #110 records why the webhook type switcher still counts as an action menu.
- **#117:** Grouping is done in the SQL query, not on the page, so each repository's rows stay together across pages. Sections are ordered A–Z, the choice lives in the URL like Sprint 1's #50 filters, and collapsed sections are kept in page memory so a live refresh doesn't reopen them.
- **#52:** The reason is stored in a new column as a number that matches its priority, and the highest-priority reason wins. Existing notifications get "Other" and sort last, and the sort works inside #117's groups.

## Completed Issues

| Issue Title | Issue Owner | Parent | PR Link | PR Author | PR Reviewer(s) | PR Status | Description |
| --- | --- | --- | --- | --- | --- | --- | --- |
| #56 Central keyboard-shortcut registry | ab-wm | #23 | [#96](https://github.com/CSCI-435-SE/gitea/pull/96) | ab-wm | controlled-opposition | Merged | Replaces the page-wide key handler with a registry that supports two-key sequences and shortcuts without a text box |
| #58 `g` navigation shortcuts | ab-wm | #23 | [#97](https://github.com/CSCI-435-SE/gitea/pull/97) | ab-wm | CarsonRackley, vvchopra | Merged | `g c`, `g i`, `g p`, `g a`, `g w` go to a repository's tabs, and `g n` goes to notifications |
| #59 `c` shortcuts | ab-wm | #23 | [#114](https://github.com/CSCI-435-SE/gitea/pull/114) | ab-wm | CarsonRackley | Merged | `c` starts a new issue or PR from the lists and opens the commit picker on a PR's changed files |
| #57 `?` shortcut help dialog | controlled-opposition | #23 | [#115](https://github.com/CSCI-435-SE/gitea/pull/115) | controlled-opposition | ab-wm | Merged | `?` opens a "Keyboard shortcuts" dialog listing the shortcuts that work on the current page, grouped by section |
| #95 `r` quotes selected text | controlled-opposition | #23 | [#116](https://github.com/CSCI-435-SE/gitea/pull/116) | controlled-opposition | CarsonRackley | Merged | `r` quotes the selected text into the reply box, or jumps to the reply box when nothing is selected |
| #60 Turn character-key shortcuts off | controlled-opposition | #23 | [#121](https://github.com/CSCI-435-SE/gitea/pull/121) | controlled-opposition | ab-wm | Merged | A checkbox in the `?` dialog turns off every shortcut except `?` (WCAG 2.1.4) |
| #65 Playwright keyboard regression suite | controlled-opposition | #25 | [#105](https://github.com/CSCI-435-SE/gitea/pull/105) | controlled-opposition | ab-wm | Merged | End-to-end checks of the label picker's and navbar menus' ARIA state; test files only |
| #104 Label picker `aria-expanded` bug | Jack-Donohue | #25 | [#124](https://github.com/CSCI-435-SE/gitea/pull/124) | Jack-Donohue | ab-wm | Merged | The Fomantic ARIA patch now refreshes ARIA state after the dropdown's own `hide`, so Escape resets `aria-expanded` |
| #108 Code tab, branch and commit menus to `<aria-menu>` | Jack-Donohue | #25 | [#127](https://github.com/CSCI-435-SE/gitea/pull/127) | Jack-Donohue | ab-wm | Merged | Four action menus move from Fomantic to `<aria-menu>`, with the commit page modals moved out of the menu |
| #110 Webhook, account link and notice menus to `<aria-menu>` | Jack-Donohue | #25 | [#128](https://github.com/CSCI-435-SE/gitea/pull/128) | Jack-Donohue | ab-wm | Merged | Four settings-page action menus move to `<aria-menu>` |
| #117 Group notifications by repository | CarsonRackley | #7 | [#120](https://github.com/CSCI-435-SE/gitea/pull/120) | CarsonRackley | ab-wm | Merged | A "Group by repository" button splits the list into collapsible A–Z sections with unread counts |
| #52 Sort notifications by reason or type | CarsonRackley | #7 | [#122](https://github.com/CSCI-435-SE/gitea/pull/122) | CarsonRackley | vvchopra | Merged | Each notification stores why it was sent, and a Sort dropdown orders by Newest, Reason or Type; includes a migration |
| #98 Repository calendar view | vvchopra | #36 | [#123](https://github.com/CSCI-435-SE/gitea/pull/123) | vvchopra | ab-wm | Merged | A month grid at `/{owner}/{repo}/calendar` with milestone due dates and issue and PR deadlines |
| #100 iCalendar feeds | vvchopra | #36 | [#129](https://github.com/CSCI-435-SE/gitea/pull/129) | vvchopra | ab-wm | Merged | `milestones.ics` and `issues.ics` feeds for public repositories, with "Subscribe (iCal)" buttons on the lists |
| Sprint 1 carryover: #77's PostgreSQL test failure | ab-wm | #14 | [#125](https://github.com/CSCI-435-SE/gitea/pull/125) | ab-wm | controlled-opposition | Merged | The burndown chart now orders same-second changes by issue, fixing the flaky test |
| #109 Actions, Projects and Packages row menus | arjunsb26 | #25 | [#135](https://github.com/CSCI-435-SE/gitea/pull/135) | arjunsb26 | CarsonRackley | Merged | The workflow, run, project column and package cleanup rule "…" menus move to `<aria-menu>`, which now supports disabled items |
| #131 Milestone projected finish on the list | arjunsb26 | #14 | [#134](https://github.com/CSCI-435-SE/gitea/pull/134) | arjunsb26 | vvchopra | Merged | Each open milestone on the Milestones tab shows its projected finish, worded like the burndown chart's summary |

Every PR above has a `## Tech Debt` section in its description (D4). CodeRabbit also commented on every code PR as an automated first pass, alongside the human reviews.

Documentation and log PRs merged this sprint, outside the issue table:

- Check-ins: #99, #103, #107, #111, #112, #113
- AI logs: #106, #130, #132
- Course rules update: #94
- UML diagrams: #126

## Test Strategy

As in Sprint 1, tests were written against each acceptance criterion:

- **Go unit tests** for models, services and handlers: #120, #122, #123, #125, #129, #134
- **Integration tests** through the full HTTP request path: #120, #122, #123, #128, #129, #134
- **Vitest** frontend unit tests: #96, #97, #114, #115, #116, #120, #121, #124, #135
- **Playwright end-to-end tests**: #96, #97, #105, #114, #115, #116, #120, #121, #124, #127, #128, #135

The shortcut PRs are mostly frontend, so they lean on Vitest for the registry logic and on Playwright for real key presses on real pages. #52's migration has its own migration test with fixtures (`v345_test.go`). #125 fixed a test that failed only on PostgreSQL by making the code's ordering deterministic, rather than loosening the test.

Changes we couldn't test automatically:

- **Screen reader output** can't be driven in CI. The `<aria-menu>` and ARIA work (#104, #108, #110, #65) asserts ARIA attributes and roles instead.
- **Real calendar apps** can't subscribe to a CI server. #129 tests the feed builder (`services/feed/ical_test.go`) and the endpoints (`tests/integration/repo_ical_test.go`), but not how a calendar app displays the feed.
- **Visual layout** of new UI was checked by hand and with light and dark screenshots in `docs/sprint2/screenshots/` (#52, #57, #60, #95).
- **Firefox and WebKit** end-to-end runs happen only in CI.

## AI Tool Usage

Per member: tools used, sessions logged, a link to the AI log folder, and notable patterns compared with Sprint 1.

Vishav Chopra

| **Vishav Chopra (vvchopra)** | |
| --- | --- |
| Tools Used | Claude Code Desktop |
| Total Sessions | 5 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/vvchopra](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/vvchopra) |
| Notable Obsv. vs. Sprint 1 | Claude is able to see and diagnose so many different issues, ranging from tests, code bugs, overlooked assumptions, etc. Although I have prior experience with using Claude Code in these ways, it is notable just in how many ways it can help you. |

Carson Rackley

| **Carson Rackley (CarsonRackley)** | |
| --- | --- |
| Tools Used | Claude Code (Terminal) + Specstory |
| Total Sessions | 2 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/CarsonRackley](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/CarsonRackley) |
| Notable Obsv. vs. Sprint 1 | Have prior experience, didn't have anything notable or unexpected |

Arjun Bhat

| **Arjun Bhat (arjunsb26)** | |
| --- | --- |
| Tools Used | Claude Code Desktop |
| Total Sessions | **[TODO (Arjun)]** |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/arjunsb26](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/arjunsb26) |
| Notable Obsv. vs. Sprint 1 | Claude is a go-getter and very proactive, unprompted it will (for instance when I asked if new changes would break anything, it said it would be easier to test itself than explain and did exactly so) |

Aman Bhoot

| **Aman Bhoot (ab-wm)** | |
| --- | --- |
| Tools Used | Claude Code (Primarily Opus 5.5) via Claude-CLI |
| Total Sessions | 10 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/ab-wm](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/ab-wm) |
| Notable Obsv. vs. Sprint 1 | N/A. Worked within expectation (prior experience). |

Jack Donohue

| **Jack Donohue (Jack-Donohue)** | |
| --- | --- |
| Tools Used | Claude Code CLI, Claude Code VSCode |
| Total Sessions | 5 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/Jack-Donohue](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/Jack-Donohue) |
| Notable Obsv. vs. Sprint 1 | Is more capable of using sub agents than older versions. |

Ethan Turner

| **Ethan Turner (controlled-opposition)** | |
| --- | --- |
| Tools Used | Claude Code CLI, herdr |
| Total Sessions | 9 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/controlled-opposition](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint2/controlled-opposition) |
| Notable Obsv. vs. Sprint 1 | With the introduction of Opus 5.5, implementing features was much faster and used less tokens. I did not hit session limits as I did in Sprint 0. |

## Release

### Tag Name

v1.27.3-csci435-s2

### Release Link

[https://github.com/CSCI-435-SE/gitea/releases/tag/v1.27.3-csci435-s2](https://github.com/CSCI-435-SE/gitea/releases/tag/v1.27.3-csci435-s2)

## Deliverables Summary

| # | Deliverable | Status |
| --- | --- | --- |
| D1 | Sprint backlog | [Sprint 2 milestone](https://github.com/CSCI-435-SE/gitea/milestone/3) with 16 issues, one assignee each |
| D2 | Requirements specs, peer-reviewed | All 16 issues in the course format, each with a "Spec reviewed" comment from a teammate |
| D3 | Design decisions with at least two alternatives | Recorded in each issue's Design Decisions section; #58 and #59 recorded theirs in issue comments |
| D4 | PRs with a `## Tech Debt` section | 17 code PRs merged (table above), all with Tech Debt sections |
| D5 | AI logs | `ai-logs/sprint2/<username>/` for each member, linked from each issue's AI Assistance comment |
| D6 | UML class and sequence diagrams plus Q&A | ab-wm ([#56](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/56-ab-wm.md)), controlled-opposition ([#57](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/57-controlled-opposition.md)), vvchopra ([#98](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/98-vvchopra.md)), Jack-Donohue ([#108](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/108-Jack-Donohue.md)), CarsonRackley ([#117](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/117-CarsonRackley.md)), arjunsb26 ([#131](https://github.com/CSCI-435-SE/gitea/blob/main/docs/sprint2/diagrams/131-arjunsb26.md)) |
| D7 | Mid-sprint check-in (due Oct 22) | All six in [`docs/sprint2/checkin/`](https://github.com/CSCI-435-SE/gitea/tree/main/docs/sprint2/checkin) |
| D8 | Sprint report | This document |
| D9 | Release | `v1.27.3-csci435-s2` (see Release) |
| D10 | Reflection survey | Submitted individually |

## Risks and Retrospective

### What worked

- **Building the registry first.** Merging #56 first turned five more shortcut features (#57–#60, #95) into small PRs that each added a few registry entries and could be reviewed on their own.
- **Spec reviews that read the code.** These caught design problems in #104, #108 and #52 before any code was written, which is cheaper than finding them in PR review.
- **Sprint 1 changes that stuck.** AI logs went in separate PRs, specs were approved before branches were started, and the PostgreSQL test failure carried over from Sprint 1 was fixed (#125).
- **Screenshots for UI changes.** These were committed in light and dark mode under `docs/sprint2/screenshots/`, which made UI PRs quicker to review.

### What slowed the team down

- **PRs still bunched up at the end.** 14 of the 17 code PRs were opened on or after Oct 24, and eight of them (#122, #123, #124, #127, #128, #129, #134, #135) merged in the last two days. Our Sprint 1 goal of opening PRs by mid-sprint was met only by the first shortcut PRs (#96 and #97).
- **Specs were written late.** Most of the new issues (#98, #100, #104, #108, #109, #110) were written on Oct 22, and #117 on Oct 26, so the write, review and implement cycle for them fit into the last week.
- **Review load fell on one person.** ab-wm reviewed 9 of the 17 code PRs. Spreading reviews more evenly would have shortened the wait at the end.
- **CI trouble around #114.** After #114 merged, a revert (#119) was opened to get CI passing and then closed without merging.
- **#52's migration.** Adding a column to every notification row and changing the notifier queue was more work than its medium estimate suggested, which pushed #118 into Sprint 3.

### Changes for Sprint 3

- Write and review every Sprint 3 spec in the first three days of the sprint.
- Open each member's first PR by the middle of the sprint.
- Spread reviews so that each member reviews at least two PRs, and name the reviewer when the PR is opened.
- Size issues that need a migration or a queue change as large, or split them.

## Sprint 3 Plan

Carryover and follow-up issues:

- **#118** (CarsonRackley): tab icon badge and opt-in desktop notifications, deferred from Sprint 2, which finishes #7.
- **#101 and #102** (#36): private repository feeds with a revocable calendar token, and one personal deadline feed across all of a user's repositories, building on #100.
