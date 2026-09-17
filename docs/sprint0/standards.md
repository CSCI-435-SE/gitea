# Standards & Guidelines Document

CSCI 435 - Gitea

## Baseline

Existing documents define and govern every single aspect of the project. The CONTRIBUTING.md document
contains all the information that may be needed by a contributor, and thus, will govern our activities
throughout the semester on this project.

- `CONTRIBUTING.md` - issues, PR format, code review, AI contribution policy
- `AGENTS.md` / `CLAUDE.md` - instructions for AI agents, test commands, commit conventions and trailers
- `docs/guidelines-backend.md` and `docs/guidelines-frontend.md` - code structure and style
- `docs/testing.md` - how to write and run tests
- `docs/community-governance.md` - review process
- `STUDENTS.md` - course setup and branch naming
- `.golangci.yml`, `eslint.config.ts`, `stylelint.config.ts`, `.markdownlint.yaml`, `.editorconfig` -
  linter and formatter configs

## Coding conventions

Formatting is handled, for the most part, via Golangci-lint, which can be invoked by `make fmt` (or
manually running the commands described in the Makefile). Ensure to run the formatter before committing
any code.

Similarly, the Makefile describes various lint and test commands that can be run depending on what
portion of the system you are working on. So just have a peek at the file to understand what kind of
lint command that should be run. Do not just simply run `make lint`, as that will scan the entire system
and uncover errors that are not related to your work.

There are Make commands for almost every possible file, such as markdown, so even documentation will
fail to pass CI unless it is checked beforehand. Furthermore, there are corresponding "fix" commands
that will fix styling problems to the extent possible.

## Branching and commit conventions

The `STUDENTS.md` file defines the branch naming conventions. As upstream Gitea squashes commits on
merge, there is no settled convention on commit styles. Judging by the original repository's commit
history, it would be best to use the types defined for PRs in the `CONTRIBUTING.md` document.

Other general conventions are:

- Avoid merging another branch (that is not main) with your branch. Prefer only including data that is
  already on main.
- Delete branches after every merge. The GitHub UI provides the option once the branch has been merged.
- Only address one issue or problem on a branch. In other words, keep PRs small.
- We may not be actually publishing PRs on upstream Gitea, but follow their commit and PR styles to the
  extent possible.
- Do not rebase or squash commits on a branch once review has started. To update the branch, merge main
  into it. PRs are merged with a merge commit (upstream Gitea squash-merges instead).

## Pull request process

The `STUDENTS.md` file defines the naming conventions and the requirements that need to be followed when
creating and reviewing pull requests, and merging branches. Since these are course requirements, they
will be followed to the maximum extent, along with the following guidelines:

- A PR is opened from its issue branch into main, and main is protected, so every change goes through a PR.
- The PR title follows Conventional Commits, and the description is written in your own words, explaining
  what changed and why.
- Follow `CONTRIBUTING.md` for the description: features include how to use them and how they were tested,
  UI changes include screenshots, and each closed issue is listed as `Closes #<N>` on its own line.
- Add a short note on how AI was used (see AI tool use).
- Keep the PR as a draft while working on it, then mark it ready for review and request a reviewer.
- At least one teammate must review and approve the PR, and the review must include a written comment, not
  just an approval. Reviewers should pull the branch and test it when possible, say what they checked, and
  mark each point as required or optional (`docs/community-governance.md`). Answer reviewer questions
  yourself, without AI (`CONTRIBUTING.md`).
- A PR can be merged once it is approved, all review comments are addressed, and CI passes.

## Testing expectations

- Every code change must add or update tests. Prefer unit tests when the logic can be tested on its own
  (`AGENTS.md`).
- Go unit tests go next to the code in `*_test.go` files, named `TestXxx`. Run a single test with
  `go test -run '^TestName$' ./modulepath/`.
- Tests for routes and pages go in `tests/integration/`.
- Frontend tests go next to the file as `*.test.ts`. Run them with `pnpm exec vitest <path>`.
- End-to-end tests use Playwright: `GITEA_TEST_E2E_FLAGS='<filepath>' make test-e2e`.
- Test the normal case, invalid input, and permission checks where they apply, and try UI changes in a
  local instance.
- Run tests in WSL or Linux. Native Windows produces errors that are not related to your change.
- Some tests may already fail on main. Run the same test on main before assuming your change broke it.
- See `docs/testing.md` for more details.
- Run tests on localhost to validate the fix does what you want and you get the expected results in the
  dev environment.

## AI tool use

Given the nature of the course, the use of AI is permitted, if not encouraged. It goes without saying
that using AI comes with the following caveats:

- Do not rely on AI to catch every single problem and grasp every single implementation detail.
- Analyse and check every single change AI does on your behalf, to the extent possible in a foreign
  codebase.
- For issues you implement, find the relevant code yourself first, without AI, then use AI to confirm
  your understanding (course requirement).
- Make the AI agent use the provided Make commands to lint and format your code, as AI agents like to
  find their own solutions even if a solution already exists.
- It is NOT permitted for AI to commit on your behalf, a human must approve it first. Additionally,
  Gitea disallows the "Co-Authored" tag on commits, so keep that in mind.
- Drafting issues and pull requests with AI is perfectly acceptable, but attempt to keep the description
  as concise and precise as possible. AI has the tendency to be overly descriptive and use technical
  words and statements when a much simpler alternative exists.
- Per the course requirements, all AI sessions must be logged in the ai folder per sprint. Following the
  naming convention as defined on the course website.
- Reviewers are responsible for checking AI-generated code the same way as any other code.

The `.claude-students` folder contains AI generated and maintained documentation for both us as
developers and AI. Given the size and complexity of the project, it is unlikely that either us or an AI
agent will understand every part of the program even by the end of the semester. This is especially true
about AI that will make assumptions of potentially significant portions of the applications. Currently,
there are 30+ pieces of documentation for both us and AI, and it comes with its own maintenance criteria:

- The documentation is strictly AI maintained. Do not attempt to edit documentation as there are checks
  in place to prevent changes to them.
- Have your AI agent update documentation when it has found something that should be included within it.
  This folder is a living document, so to speak.
- The `check.sh` file checks all reference documentation to ensure that they have not been edited without
  updating the corresponding AI documentation.
- As usual, do not blindly trust the documentation. Its purpose is to guide us through the codebase, but
  with the complexity of the project, it is unlikely that the documentation has covered every single
  aspect of the codebase.

It should be completely fine for other AI solutions to use and maintain the documentation, although I am
not sure if it will match Claude's style of agent documentation. Please ensure that the documentation and
edits done by your agent of choice matches as closely as possible with Claude's style, before we have AI
fighting with each other on the semantics of style.

## Definition of done

An issue is done when:

- The change does what the issue asks, and the PR only contains changes for that issue.
- Tests were added or updated and pass locally.
- The code is formatted, and the lint commands for the changed files pass.
- CI passes.
- Locale strings and documentation (including `.claude-students`) are updated if needed.
- The PR description follows the pull request process above.
- A teammate reviewed and approved the PR with a written comment.
- The author can explain every line of the change.
- AI logs are pushed and AI Assistance comments are posted.
- The PR is merged, the issue is closed, and the branch is deleted.

## AI Assistance

This document was written by the team. Claude Code (Claude Opus 5) was used to convert it to Markdown
and to check its statements against the project's own files (`CONTRIBUTING.md`, `AGENTS.md`,
`docs/guidelines-*.md`, `docs/testing.md`, `docs/community-governance.md`). The team reviewed and edited
the result and remains responsible for its contents. See [ai-logs/sprint0/](../../ai-logs/sprint0/).
