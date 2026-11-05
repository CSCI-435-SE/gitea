---
title: "Commit messages follow AGENTS.md"
description: "Each commit subject is a Conventional Commit, ends with the issue number when the PR closes an issue, and credits AI with an Assisted-by trailer rather than Co-Authored-By"
type: check
when: "Every pull request"
actions: "Post a comment listing each commit that breaks a rule below, and which rule"
---

# Commit messages

From `AGENTS.md`. Check every commit in the pull request except merge commits (subjects starting
with `Merge`).

1. The subject is a Conventional Commit, `type(scope): subject`, with `!` before the colon for a
   breaking change. A commit that only changes tests uses the `test` type.
2. When the pull request closes an issue (`Closes #123`), the subject ends with that issue's number,
   for example `(#123)`. Chore pull requests that close no issue are exempt.
3. AI help is credited with a trailer in the form `Assisted-by: AGENT_NAME:MODEL_VERSION`, for
   example `Assisted-by: Claude Code:claude-opus-5-5`, never with a `Co-Authored-By:` trailer that
   names an AI tool or model.

A `Signed-off-by:` trailer is not judged: signing off is the human author's own step.
