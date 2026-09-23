# Course workflow — issues, PRs, reviews, sprint deliverables

Course rules, not upstream Gitea ones. Source: the course site <https://csci-435-se.github.io/>
(Sprints pages) and `docs/sprint0/standards.md`. Everyone on the team, and every AI session, uses
the formats below so all issues and PRs look the same. The GitHub templates in `.github/` carry the
same formats; if they ever disagree with this page, fix the template.

AI disclosure has its own page: `AI-ASSISTANCE.md`.

## Current sprint: Sprint 1

- **Dates:** Sep 22 – Oct 8, 2026. Everything pushed and merged by **Oct 8, 11:59 PM**.
- **Points:** 100 (10% of the course grade).
- **Zero rule:** no merged PR (D4), no sprint report (D6) or no release (D7) means **0 for the
  whole sprint**.
- **Carries forward:** every Sprint 0 standard and convention still applies.

| # | Deliverable | Points | Who | Where |
| --- | --- | --- | --- | --- |
| D1 | Sprint backlog: a `Sprint 1` milestone with every selected issue on it | 10 | Team | GitHub milestone |
| D2 | Requirements spec for each issue (formats below), peer-reviewed by a teammate | 20 | Individual | Issue body |
| D3 | At least one non-trivial design decision per issue | 7 | Individual | Issue body |
| D4 | Code changes and PRs | 36 | Individual | PRs |
| D5 | AI logs | 7 | Individual | `ai-logs/sprint1/<github-username>/` |
| D6 | Sprint report | 10 | Team | `docs/sprint1/report.md` |
| D7 | Release: tag `v1.27.3-csci435-s1` plus a GitHub release | 2 | Team | Tags / Releases |
| D8 | Reflection survey | 8 | Individual | Blackboard/Zulip link |

- **D1 load:** each CSCI 435 student owns at least 2 medium issues; each CSCI 535 student owns at
  least 2 medium and 1 small. Every issue has exactly **one** assignee. Specs should be final by
  Sep 27.
- **D5:** conventions are in `ai-logs/README.md`. The folder must exist even if no AI was used. In
  that case, post an issue comment saying so. Log links go in **issue comments only**, never in PRs
  (`AI-ASSISTANCE.md`).
- **D7:** keep Sprint 0's base version (`v1.27.3-csci435-s0` → `v1.27.3-csci435-s1`), push the tag
  to origin, and write release notes listing the features, fixes and improvements.
- **Reviews:** each student reviews at least one PR per sprint. Over the semester, each student
  reviews every teammate at least once.
- **Extra credit:** up to 3 points per PR accepted by upstream Gitea.

## Branches and commits

- Branches: `feat/issue-<N>-short-description`, `fix/issue-<N>-short-description` or
  `chore/short-description` (`STUDENTS.md`). One issue per branch.
- Commit messages follow Conventional Commits and end with the issue number:
  `feat(repo): show empty state on releases page (#17)`. Add an
  `Assisted-by: AGENT_NAME:MODEL_VERSION` trailer when AI helped; never `Co-Authored-By` or
  `Signed-off-by` (`AGENTS.md`). A human approves every commit before it is made.

### Before each implementation: sync with main

Every new branch starts from an up-to-date `main`. AI sessions run this before writing any code:

```bash
git switch main
git pull origin main
git switch -c feat/issue-<N>-short-description
```

To pick up new work on `main` for a branch that already exists, **merge, don't rebase**. Rebasing
or squashing after review has started is not allowed (`docs/sprint0/standards.md`):

```bash
git fetch origin
git merge origin/main
```

Do this before opening the PR and again if `main` moves while it is in review. Never merge another
feature branch into yours.

### After merge: clean up

A PR is merged with a merge commit once it is approved, every review comment is addressed and CI
passes. Then:

```bash
git switch main
git pull origin main
git branch -d feat/issue-<N>-short-description               # local branch
git push origin --delete feat/issue-<N>-short-description    # or GitHub's "Delete branch" button
git fetch --prune                                            # drop stale remote-tracking refs
```

Also check that the issue closed (via `Closes #N`), is on the sprint milestone, and has its AI
Assistance comment. Then run `specstory sync` and file the session's log (see `ai-logs/README.md`).

## Issue format: feature (user story)

Template: `.github/ISSUE_TEMPLATE/user-story.yaml`. The user story must meet INVEST: Independent,
Negotiable, Valuable, Estimable, Small, Testable.

```markdown
## User Story
As a <role>, I want <capability> so that <benefit>.

## User Scenario
<A short concrete walk-through: who, where in the UI, what they do, what they see.>

## Acceptance Criteria
- [ ] AC1: Given <state>, when <action>, then <observable result>.
- [ ] AC2: ...
- [ ] AC3: ...            (at least 3; each has a clear pass/fail result)

## Out of Scope
- <What this issue deliberately does not do.>

## Open Questions
- <Unresolved questions, or "None".>

## Design Decision
**Decision:** <what was chosen>
**Alternatives Considered:** <at least one other option>
**Rationale:** <why this one>
**Consequences:** <trade-offs, follow-up work, what it makes harder>

## Scope
small | medium | large, and matching `scope:` label
```

## Issue format: bug report

Template: `.github/ISSUE_TEMPLATE/bug-report.yaml`.

```markdown
## Observed Behavior
## Expected Behavior
## Steps to Reproduce
1. ...
## Additional Information
<Version or commit, browser/OS, logs, screenshots.>
## Design Decision
<Same four fields as above: Decision, Alternatives Considered, Rationale, Consequences.>
```

- **Design Decision (D3):** fill it in once you've picked an approach, before opening the PR. Put
  a second decision under another `### Decision 2` heading. If a PR changes the decision, update
  the issue body.
- **Peer review (D2):** a teammate reads the spec and comments `Spec reviewed — <notes>` on the
  issue before implementation starts.

## Pull request format

Template: `.github/pull_request_template.md`. Title uses Conventional Commits, e.g.
`fix(issues): keep label filter after search`. Description sections:

1. `Closes #<N>`, which links the issue and closes it on merge.
2. **What changed:** the change in your own words.
3. **Why:** the problem, and which acceptance criteria (AC1…) it satisfies.
4. **How it was tested / Test strategy:** tests added or updated and why these ones. Name any
   change that can't be tested automatically and how it was checked by hand. Every behaviour change
   needs a new or updated test.
5. **CI / manual verification:** CI status, or the commands run and their results.
6. **Design:** one line pointing at the Design Decision in the issue, or a note on how it changed.
7. **AI assistance:** format in `AI-ASSISTANCE.md` §2, with **no log link**.

Per-PR grading (20 points): correctness against the acceptance criteria 6, test strategy with
explanation 5, description quality 4, CI or manual verification 3, review rubric posted 2.

## Code review rubric

The reviewer posts this as a **PR comment**, as well as the GitHub review itself. A copy-paste copy
is in `.github/code-review-rubric.md`, so keep the two in sync. Pull the branch and test it when you
can, say what you checked, and mark each point as required or optional
(`docs/community-governance.md`). Score each row against the issue's acceptance criteria. Anything
below 7 needs a note saying what would raise it.

```markdown
### Code Review — @reviewer-username

**Checked:** <what you ran or tried>

| Criterion | Score (1–10) | Notes |
| --- | --- | --- |
| Correctness — implements issue requirements | | |
| Test coverage — tests present and meaningful | | |
| Code quality — style, naming, no duplication | | |
| Documentation — comments and PR description clear | | |
| AI transparency — evidence of human verification if AI used | | |
| Overall — merge as-is? | | |
```

## Sprint report (D6) sections

`docs/sprint<N>/report.md`, following `docs/sprint0/report.md`:

- Team: names, GitHub usernames, project name, repository link.
- Sprint overview.
- Backlog: milestone link, plus a table of title, owner, points, scope and status.
- Requirements and design: when specs were done, what surprised you, design rationale.
- Completed issues: title, owner, PR link, author, reviewer, what changed.
- Test strategy, including untestable changes and why.
- AI tool usage per member: tools, session count, AI log folder link, and 2–3 sentences on lessons
  learned.
- Release: tag and release link.
- Risks and retrospective: what worked, what slowed the team down, Sprint 2 changes.
- Sprint 2 plan: initial ideas and carryover issues.
