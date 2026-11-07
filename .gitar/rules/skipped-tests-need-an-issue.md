---
title: "Skipped tests and disabled checks link an issue"
description: "A test that is skipped or expected to fail, or a CI check that is turned off, needs a linked issue and a code comment giving the reason"
type: check
when: "PRs that add or change a skipped test, a test expected to fail, or a disabled CI job, step or check"
actions: "Post a comment naming each skip or disabled check that has no linked issue or no reason in a code comment"
---

# Skipped tests and disabled checks

The course and `AGENTS.md` say: never skip a test or disable a CI check without a linked issue and
a code comment giving the reason.

Count only what this pull request adds or changes:

- Go: `t.Skip`, `t.Skipf`, `t.SkipNow`.
- Vitest and Playwright: `test.skip`, `it.skip`, `describe.skip`, `test.fixme`, `test.fail`, and
  the same through `.skipIf` or `.runIf`.
- Workflows under `.github/workflows/`: a job or step that runs tests or a check and is turned off
  (`if: false`), removed, or allowed to fail (`continue-on-error: true`).

Each one is met when a comment next to it gives the reason and names an issue (`#123` or a link to
it). Skips that already existed and that the pull request does not touch are not counted.
