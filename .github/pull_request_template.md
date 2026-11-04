<!--
CSCI 435 PR format, see .claude-students/COURSE-WORKFLOW.md.
- Title: Conventional Commits, e.g. `fix(issues): keep label filter after search`.
- Put the PR and its issue on the sprint milestone.
- Keep as a draft until ready, then request a teammate's review on GitHub and tell them on Zulip.
- The reviewer posts the Code Review rubric (.github/code-review-rubric.md) as a PR comment.
- Merge only after a teammate approves the latest commit; a push after the approval needs a new one.
- Do NOT link AI logs here; they go in the issue's AI Assistance comment.
-->

Closes #

## What changed

## Why

<!-- The problem, and which acceptance criteria (AC1, AC2, ...) from the issue this satisfies. -->

## Test Evidence

<!--
One row per acceptance criterion (for a bug, the Expected Behavior); "N/A" and why if the PR changes no product behaviour.
Test(s): test name (and file); for a manual check, "Manual:" then the steps and what you saw.
Level: unit, integration, E2E or manual.
Source of expected value: AC, issue discussion, docs, upstream behavior, or other (explain). Never your code's current output.
-->

| Acceptance criterion | Test(s) | Level | Source of expected value |
| --- | --- | --- | --- |
| AC1: | | | |

**Not tested automatically, and why:** <!-- Only for ACs checked manually only. -->

**AI-written tests:** <!-- Which tests AI wrote or changed, and how you checked their expected values. "None." if none. -->

## CI / manual verification

<!-- CI status. If CI doesn't run your tests, the commands you ran and a summary (passed, failed, skipped). -->

## Design

<!-- Point to the Design Decision in the issue, or say how it changed during implementation. -->

## Tech Debt

<!-- Both fields are required; "None." is a valid answer. -->

**Introduced:** <!-- Shortcuts, workarounds or design compromises this PR makes, and why. -->

**Encountered:** <!-- Tech debt that was already there and that you ran into, and how you handled it. -->

## AI assistance

<!-- Tool and model, what AI did, what you did and how you verified it. No log link. "No AI used" if none. -->
