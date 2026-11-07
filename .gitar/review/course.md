# Reviewing this repository

This repository is a CSCI 435 course fork of Gitea. Pull requests are graded against the course
rules as well as Gitea's own conventions, so review against both. `AGENTS.md` and `CLAUDE.md` are
read already; these are the course and project rules they point to:

@.claude-students/COURSE-WORKFLOW.md
@.claude-students/AI-ASSISTANCE.md
@CONTRIBUTING.md
@docs/sprint0/standards.md

## Focus on

- Whether the change does what the linked issue's acceptance criteria ask, and nothing outside its
  scope.
- Whether each test checks real behaviour and can fail: no assertion that passes for an unrelated
  reason, no "nothing happened" check without waiting, no test that depends on fixture data by
  chance.
- Bugs and security problems, with the input or state that triggers them.

## Leave alone

- The pull request description's sections and format: CodeRabbit checks those.
- The pull request title's Conventional Commit format: the `pr-title` CI job checks it.
- Formatting and anything else the repository's linters already enforce (`make lint`).
- Style preferences that no rule above asks for. These are students learning a large codebase;
  one clear finding helps more than several small ones.
