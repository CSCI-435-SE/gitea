# Report: Sprint 1

**Team Name:** GiCoffee

**Team member names:** Vishav Chopra (vvchopra), Carson Rackley (CarsonRackley), Arjun Bhat (arjunsb26), Aman Bhoot (ab-wm), Jack Donohue (Jack-Donohue), Ethan Turner (controlled-opposition)

**Project Name:** Gitea

**Project Link:** [https://github.com/CSCI-435-SE/gitea](https://github.com/CSCI-435-SE/gitea)

## Sprint Overview

Brief narrative: what the team set out to do and what was actually delivered. Be honest about scope changes

### What we planned to do

At the start of the sprint, we were hit with the problem that most issues were ambitious and large in scope, making it difficult to cleanly assign work between different members of the team.

A number of large issues, namely #7, #12, #23 and #25, were cut down into smaller medium and small issues that were easier to manage. The plan was to complete these large issues, with members who split the issues into smaller issues working on the backbone of the problem, and then distributing the issues among other members to complete them.

### Scope Changes

While we certainly got close to completing all of the aforementioned large issues, and managed to complete all 13 issues in the Sprint 1 milestone (#12 through its sub-issues #53, #54 and #55, plus #50, #51, #61, #62, #63, #64, #14, #34 and #35), larger than expected issues and requests from product owners made it difficult to achieve our goal. The request from the product owner to complete #14, or “Add a burndown chart to milestones with an ideal line and projected finish,” was one of the main reasons we had to change the scope of this sprint.

Issue #14 was a large issue, and with one of our team members completely devoted to this purpose, we had to change the scope of the sprint to ensure we are able to complete the main issues, and leave cleanup work for the next sprint. Because of this, #64 was only picked up on the last day of the sprint (PR #81), the keyboard shortcut issues (#56, #57) were deferred and moved out of the milestone, and #52 was moved to Sprint 2.

Fortunately, however, we were able to complete #12 and #14 in full, along with the core of #7 and #25 (#50, #51 and #61–#64), so we still consider this sprint to be a success. The remaining sub-issues (#52, #65 and #56–#60) carry over into Sprint 2.

## Sprint Backlog

Link to the Sprint 1 GitHub Milestone; table of issues (title, owner, estimated points, scope, status)

### Sprint 1 GitHub Milestone Link

[https://github.com/CSCI-435-SE/gitea/milestone/1](https://github.com/CSCI-435-SE/gitea/milestone/1)

### Table of Issues

Estimated points are taken from each issue's scope label on GitHub: small = 1, medium = 2, large = 4 (a large issue counts as two medium issues).

| Issue Title | Issue Owner | Estimated Points | Scope | Status |
| --- | --- | --- | --- | --- |
| #12 Let users choose a reason when closing an issue or pull request | ab-wm | 4 | Large | Done (split into #53, #54, #55) |
| #14 Add a burndown chart to milestones with an ideal line and projected finish | arjunsb26 | 4 | Large | Done (PR #77) |
| #34 Sort issues and pull requests by due date | Jack-Donohue | 2 | Medium | Done (PR #73) |
| #35 Require two-factor authentication for an organization's members | Jack-Donohue | 2 | Medium | Done (PR #76) |
| #50 Filter notifications by repository and type | CarsonRackley | 2 | Medium | Done (PR #69) |
| #51 Bulk actions on notifications (mark read/unread, pin, delete) | CarsonRackley | 2 | Medium | Done (PR #70) |
| #53 A way to store and retrieve "closed" reasons for issues and PRs | ab-wm | 2 | Medium | Done (PR #67) |
| #54 Select close reasons for Issues and PRs and show it under closed | ab-wm | 2 | Medium | Done (PR #68) |
| #55 Set close reason when issue or PR is closed by an automated process | vvchopra | 2 | Medium | Done (PR #72) |
| #61 Accessible menu web component, no Fomantic | controlled-opposition | 2 | Medium | Done (PR #71) |
| #62 Migrate the navbar, footer and comment menus to the accessible menu | controlled-opposition | 2 | Medium | Done (PR #74) |
| #63 Multi-select dropdown ARIA state and announcements | vvchopra | 2 | Medium | Done (PR #75) |
| #64 Accessible names for icon-only dropdown triggers | Jack-Donohue | 1 | Small | Done (PR #81) |

## Requirements & Design

Were specs complete before coding started? Any surprises? Key design decisions and brief rationale

### State of issues at the start of the sprint

Most of the issues carried over from Sprint 0 (#7, #12, #14, #23, #25, #34, #35) were written as summaries or proposals, not in the Sprint 1 spec format. Each owner rewrote their issue into a User Story, User Scenario, numbered Acceptance Criteria, Out of Scope and Open Questions, and a teammate posted a "Spec Reviewed" comment on the issue before implementation started.

Surprises:

- #35 was sent back ("Spec Reviewed: Not yet") for numbered acceptance criteria and was approved after it was revised.
- #14's spec was refined late, after implementation had already started.
- The review of #64 found that one of its acceptance criteria was already met by #62.
- Close reasons (#12) needed a database change and reached into automatic closes (commits, cross-references and the API), which became its own sub-issue (#55).

### Key decisions that led to the splitting of Issues

- **#7 Overhauled notifications system** → #50 (filter), #51 (bulk actions), #52 (sort). #52 was moved to Sprint 2 because it needs a database migration and a queue change.
- **#12 Close reasons** → #53 (storage and backend), #54 (UI), #55 (automatic and API closes). The issue was split by layer so that the backend could be merged first and the UI could build on it.
- **#25 Make dropdowns Accessible** → #61 (menu web component), #62 (menu migration), #63 (multi-select ARIA), #64 (icon-only triggers), #65 (Playwright regression suite). #61 had to be merged before #62 could use it.
- **#23 Keyboard navigation/shortcuts** → #56–#60, all deferred to keep the sprint scope achievable.
- **#14, #34 and #35** were kept as single issues. #14 is large and counts as two medium issues.

### Key design decisions

- **#50:** "Mark all as read" respects the active filter.
- **#51:** "Select all" only selects the notifications in the current view, with an explicit "Select all N notifications in this view" link. Bulk actions never reach outside the active view.
- **#53:** `CloseIssue` becomes a wrapper over a new `CloseIssueWithReason`, so existing callers keep working unchanged.
- **#54:** Picking "Duplicate" or "Other" opens a small popup under the close button, and each close reason gets its own icon.
- **#55:** Automatic closes record a default reason (completed for issues, not planned for pull requests).
- **#61:** The menu is built as a custom element under `web_src/js/webcomponents/`, without Fomantic.
- **#62:** Only static, server-rendered, single-select menus are migrated.
- **#63:** The existing Fomantic dropdown patch is extended rather than replaced.
- **#34:** `?due=` is parsed once into deadline bounds that both the database search and the issue indexer use.
- **#35:** A member or outside collaborator without two-factor authentication is treated as a non-member of the organization until they enrol. This is a reversible lockout: nothing is deleted.
- **#14:** Each milestone's history is rebuilt when the chart is requested, by replaying its items' timeline comments (milestone changes, closes, merges and reopens), and served from a separate JSON endpoint that the page loads after it renders, instead of a daily snapshot table that would have no history for existing milestones.

## Completed Issues

Table: issue title, issue owner, all associated PRs (PR link, author, reviewer(s), status), brief description of changes

| Issue Title | Issue Owner | SubIssue Name | SubIssue Owner | PR Link(s) | PR Author(s) | PR Reviewer(s) | PR Status | Description |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| #7 Overhauled notifications system | CarsonRackley | #50 Filter notifications by repository and type | CarsonRackley | [#69](https://github.com/CSCI-435-SE/gitea/pull/69) | CarsonRackley | ab-wm | Merged | Repository and type filters on the notifications page |
| #7 Overhauled notifications system | CarsonRackley | #51 Bulk actions on notifications | CarsonRackley | [#70](https://github.com/CSCI-435-SE/gitea/pull/70) | CarsonRackley | ab-wm | Merged | Select notifications and mark them read/unread, pin or delete them |
| #12 Let users choose a reason when closing | ab-wm | #53 Store and retrieve close reasons | ab-wm | [#67](https://github.com/CSCI-435-SE/gitea/pull/67) | ab-wm | CarsonRackley | Merged | Close reason stored in the database, `CloseIssueWithReason`, validation |
| #12 Let users choose a reason when closing | ab-wm | #54 Select close reasons and show them | ab-wm | [#68](https://github.com/CSCI-435-SE/gitea/pull/68) | ab-wm | CarsonRackley | Merged | Close reason menu, duplicate/other popup, reason icons, bulk close |
| #12 Let users choose a reason when closing | ab-wm | #55 Close reason on automated closes | vvchopra | [#72](https://github.com/CSCI-435-SE/gitea/pull/72) | vvchopra | controlled-opposition | Merged | Commits, cross-references and API closes record a reason |
| #25 Make dropdowns Accessible | Jack-Donohue | #61 Accessible menu web component | controlled-opposition | [#71](https://github.com/CSCI-435-SE/gitea/pull/71) | controlled-opposition | vvchopra | Merged | Keyboard-accessible `<aria-menu>` element without Fomantic |
| #25 Make dropdowns Accessible | Jack-Donohue | #62 Migrate navbar, footer and comment menus | controlled-opposition | [#74](https://github.com/CSCI-435-SE/gitea/pull/74) | controlled-opposition | ab-wm | Merged | Profile, create-new, footer and comment menus use `<aria-menu>` |
| #25 Make dropdowns Accessible | Jack-Donohue | #63 Multi-select dropdown ARIA | vvchopra | [#75](https://github.com/CSCI-435-SE/gitea/pull/75) | vvchopra | arjunsb26 | Merged | Selection state, live-region announcements, labelled delete icons |
| #25 Make dropdowns Accessible | Jack-Donohue | #64 Accessible names for icon-only dropdown triggers | Jack-Donohue | [#81](https://github.com/CSCI-435-SE/gitea/pull/81) | Jack-Donohue | CarsonRackley | Merged | Localised names on icon-only dropdown triggers (theme selector, lock reason, update-branch and kebab menus), plus a dev-mode warning for triggers with no name |
| #14 Add a burndown chart to milestones | arjunsb26 | — | — | [#77](https://github.com/CSCI-435-SE/gitea/pull/77) | arjunsb26 | CarsonRackley, Jack-Donohue, vvchopra | Merged | Milestone burndown chart with an ideal line, projected finish and a daily change list |
| #34 Sort issues and pull requests by due date | Jack-Donohue | — | — | [#73](https://github.com/CSCI-435-SE/gitea/pull/73) | Jack-Donohue | ab-wm, CarsonRackley | Merged | Due date options for the issue and pull request lists, in both the database and the search index |
| #35 Require two-factor authentication for an organization | Jack-Donohue | — | — | [#76](https://github.com/CSCI-435-SE/gitea/pull/76) | Jack-Donohue | CarsonRackley, vvchopra | Merged | Organization setting that treats members without 2FA as non-members, with a compliance view |

Every issue in the Sprint 1 milestone was completed. #56 and #57 were deferred and removed from the milestone, and #52 was moved to Sprint 2.

## Test Strategy

What kinds of tests were written? Were there changes you couldn't test automatically? Why?

Tests were written against the acceptance criteria, often one test per criterion:

- **Go unit tests** for models and services, run against the test database: #67, #68, #69, #70, #72, #73, #76, #77
- **Integration tests** that go through the full HTTP request path: #67, #68, #69, #70, #72, #73, #76, #77
- **Vitest** frontend unit tests: #68, #70, #71, #72, #75, #77, #81
- **Playwright end-to-end tests**: #68, #70, #71, #74, #75, #77, #81 (#71 also added a WebKit/Safari run)

In #67 and #68, the tests themselves were checked by deliberately breaking key lines of code and confirming that a test failed.

Changes we couldn't test automatically: screen reader behavior can't be driven in CI, so the accessibility work (#62/#74, #63/#75, #64/#81) asserts the ARIA attributes and live-region text instead. #81 also adds a development-mode warning for any dropdown trigger without an accessible name; the reviewer ran it on a local build, page by page, to find triggers the tests don't cover. Visual layout was checked by hand and with screenshots (#68, #74), and the Firefox end-to-end run only happens in CI.

## AI Tool Usage

Per member: tools used, sessions logged, link to AI log folder. Notable patterns vs. Sprint 0

Vishav Chopra

| **Vishav Chopra (vvchopra)** | |
| --- | --- |
| Tools Used | Claude Code Desktop |
| Total Sessions | 5 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/vvchopra](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/vvchopra) |
| Notable Obsv. vs. Sprint 0 | Claude is able to see and diagnose so many different issues, ranging from tests, code bugs, overlooked assumptions, etc. Although I have prior experience with using Claude Code in these ways, it is notable just in how many ways it can help you. |

Carson Rackley

| **Carson Rackley (CarsonRackley)** | |
| --- | --- |
| Tools Used | Claude Code (Terminal) + Specstory |
| Total Sessions | 5 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/CarsonRackley](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/CarsonRackley) |
| Notable Obsv. vs. Sprint 0 | Have prior experience, didn't have anything notable or unexpected |

Arjun Bhat

| **Arjun Bhat (arjunsb26)** | |
| --- | --- |
| Tools Used | Claude Code Desktop |
| Total Sessions | 1 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/arjunsb26](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/arjunsb26) |
| Notable Obsv. vs. Sprint 0 | Claude is a go-getter and very proactive, unprompted it will (for instance when I asked if new changes would break anything, it said it would be easier to test itself than explain and did exactly so) |

Aman Bhoot

| **Aman Bhoot (ab-wm)** | |
| --- | --- |
| Tools Used | Claude Code (Primarily Opus 5.5) via Claude-CLI |
| Total Sessions | 6 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/ab-wm](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/ab-wm) |
| Notable Obsv. vs. Sprint 0 | N/A. Worked within expectation (prior experience). |

Jack Donohue

| **Jack Donohue (Jack-Donohue)** | |
| --- | --- |
| Tools Used | Claude Code CLI, Claude Code VSCode |
| Total Sessions | 4 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/Jack-Donohue](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/Jack-Donohue) |
| Notable Obsv. vs. Sprint 0 | Is more capable of using sub agents than older versions. |

Ethan Turner

| **Ethan Turner (controlled-opposition)** | |
| --- | --- |
| Tools Used | Claude Code CLI, herdr |
| Total Sessions | 3 |
| Link to logs | [https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/controlled-opposition](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/controlled-opposition) |
| Notable Obsv. vs. Sprint 0 | With the introduction of Opus 5.5, implementing features was much faster and used less tokens. I did not hit session limits as I did in Sprint 0. |

## Release

Tag name and link to the Sprint 1 release on GitHub (see D7).

### Tag Name

v1.27.3-csci435-s1

### Release Link

[https://github.com/CSCI-435-SE/gitea/releases/tag/v1.27.3-csci435-s1](https://github.com/CSCI-435-SE/gitea/releases/tag/v1.27.3-csci435-s1)

## Risks and Retrospective

What worked:

- Splitting the large issues into sub-issues gave each member at least two medium issues and let work happen in parallel.
- The "Spec Reviewed" step caught gaps before coding started (#35 was revised before it was approved).
- Writing tests per acceptance criterion, and checking the tests by deliberately breaking code (#67, #68), produced well-specified PRs.
- CodeRabbit, added mid-sprint, gave quick first-pass reviews on #71–#77, alongside the required human rubric reviews.

What slowed the team:

- Review bottleneck: most PRs were opened in the last three days of the sprint (#71–#81, Oct 6–8), and one required approval with only a few active reviewers slowed merging. Four PRs (#73, #76, #77, #81) were merged in the final hours, and #81 was opened on the last evening.
- #77's burndown test passes on SQLite, MySQL and MSSQL but fails on PostgreSQL in CI, because events in the same second come back in a different order. It was merged on the last day with that check still failing, and the fix carries over into Sprint 2.
- #14 was a large issue that took one member the whole sprint, and its spec was refined late.
- Open PRs fell 25 commits behind `main` as other PRs merged and had to be updated.
- AI logs committed inside feature PRs made the diffs 8,000–36,000 added lines long, which made them harder to review.
- CodeRabbit's autofix pushed a commit directly to a teammate's branch (#76).
- Codebase size: Gitea has about 40 service packages, 85 module packages, and more than 300 migrations, so concept location still takes much longer than expected.

Changes for Sprint 2:

- Open PRs by the middle of the sprint and request a reviewer when opening them, aiming for a review within 24 hours.
- Don't start a branch until the spec is "Approved".
- Commit AI logs in a separate chore PR, as #78 did.
- Use CodeRabbit as a first pass only, with no autofix on someone else's branch without the author's OK.
- Update the branch from `main` before requesting a review.

## Sprint 2 Plan

Brief ideas on the features the team intends to tackle; any setup still needed

We plan to fix #77's failing PostgreSQL burndown test first, then address #52 (sort notifications by reason or type, spec already posted), #65 (Playwright keyboard regression suite for dropdowns) and the keyboard shortcut issues under #23 (#56–#60) first, along with two small follow-ups from the #81 review (the merge-style button in the pull request merge box and the repository topics editor still have no accessible name), split between 2-3 members between each. This will not be a strict split, though it should help complete issues faster, akin to pair programming under the extreme programming paradigm.

If we complete these issues quicker than the end of the sprint, we will attempt to implement more medium term issues to the extent possible.
