---
title: "Test expected values come from a source"
description: "Every expected value in a new or changed test can be traced to the source its Test Evidence row names, not to the code's current output"
type: check
when: "PRs that add or change tests (*_test.go, *.test.ts, or files under tests/)"
actions: "Post a comment naming each expected value whose source cannot be found, and what the description cites for it"
---

# Expected values in tests

The course's Test Evidence rule (`.claude-students/COURSE-WORKFLOW.md`): an expected value comes
from the acceptance criteria, the issue discussion, docs or upstream behaviour, never from the
current output of the code under test. A test that copies the code's output cannot catch the code
being wrong.

For each assertion the pull request adds or changes (`assert`/`require` in Go, `expect` in Vitest
and Playwright), find the matching row of the description's "Test Evidence" table and its "Source of
expected value". The value is met when it appears in, or follows directly from, that source: the
linked issue's acceptance criteria or discussion, quoted docs, or the behaviour the row says it
keeps.

Flag only clear cases:

- a value that the cited source does not contain and the row does not explain under "other";
- a snapshot or golden file that was regenerated with no source given;
- a value that only repeats a constant or string from the code under test.

Setup values such as fixture IDs, user names and timeouts are not expected values.
