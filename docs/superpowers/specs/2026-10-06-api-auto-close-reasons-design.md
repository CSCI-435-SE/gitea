# Close reasons for automatic and API closes

**Status:** approved, ready for implementation plan
**Branch:** `feat/issue-55-api-auto-close-reasons`
**Upstream issue:** CSCI-435-SE/gitea#55 (was blocked by #53 and #54, both merged)

Follows the user story format in `.claude-students/COURSE-WORKFLOW.md`. The technical design, testing and
acceptance criteria map follow Scope as appendices.

## User Story

As a repository maintainer whose team relies on scripts and automation, I want every automatic or API close
of an issue or PR to record a reason that my tools can set and read back through the API, so that all closing
activity can be audited.

## User Scenario

As there are alternative ways to close an issue or PR that is not via the UI, that is, via Gitea's automated
closing paths or via an API, these actions either need to be able to set a reason, or fall back onto a default
value so closed issues/PRs have an established reason.

### Automated closing

Automated closing (Gitea itself) has four distinct pathways that have been found:

| How it closes | Reason |
| --- | --- |
| Commit message keyword ("fixes #5") | Completed |
| Merged PR that references an issue | Completed (on the referenced issue) |
| PR auto-closed because its head branch was deleted | Not planned |
| PRs auto-closed because the repo's branches were removed | Not planned |

### API

Any closing via the API come from scripts and tools that interact with the REST API. As such, there are more
technical requirements for them:

- Bad inputs are rejected via a 422 with a message.
- Old scripts keep working with a default fallback.
- The issue and PR endpoints accept a close reason when closing.
- Responses from the server will include the new reasons so they can be read by tools or scripts.

### Extra in scope: popups

Popups that detail a brief summary of the issue/PR read that information from a JSON. Since modifying this
JSON comes under the API's jurisdiction, this change will be implemented as well.

### Where it stands today

#53 and #54 added close reasons (`completed`, `not_planned`, `duplicate`, `other`) and let people pick
one in the web UI. Every other close path still records no reason:

- `issue_service.CloseIssue` (`services/issue/status.go`) passes empty `CloseReasonOptions`, and it is
  what commit keywords (`services/issue/commit.go`), merged pull requests closing the issues they
  reference (`services/pull/merge.go` `handleCloseCrossReferences`), head and base branch deletion
  (`services/pull/pull.go` `AdjustPullsCausedByBranchDeleted`, `CloseRepoBranchesPulls`) and the API
  (`routers/api/v1/repo/issue.go` `CreateIssue`, `closeOrReopenIssue`) all call.
- The API has no way to send a reason and its responses do not show one.
- The hover popup (`getIssueIcon` / `getIssueColorClass` in `web_src/js/features/issue.ts`) ignores reasons,
  unlike the list icons from `templates/shared/close_reason_icon.tmpl`.

## Acceptance Criteria

- [ ] AC1: Given an open issue, when a pushed commit's message closes it with a keyword (e.g. "fixes #5"),
  then it closes as Completed.
- [ ] AC2: Given an open issue referenced by a pull request, when that pull request is merged and closes the
  issue, then the issue closes as Completed.
- [ ] AC3: Given an open pull request, when its head branch is deleted or the repository's branches are
  removed, then the pull request closes as Not planned.
- [ ] AC4: Given a script closing an issue or pull request through the API, when it sends a valid close
  reason, then the item closes with that reason and the timeline shows it.
- [ ] AC5: Given a script closing through the API, when it sends an invalid reason ("other" with no text or
  more than 255 characters, a duplicate that does not exist in the same repository, or "completed" on a pull
  request), then the request is rejected with a 422 and a message, and the item stays open.
- [ ] AC6: Given an existing script that closes issues or pull requests through the API without a reason,
  including creating an issue that starts closed, when it runs unchanged, then it succeeds and the item
  closes as Completed (issues) or Not planned (pull requests).
- [ ] AC7: Given an issue or pull request closed with a reason, when a tool reads it through the API, then
  the response includes the reason and any custom text; items without a reason return an empty reason.
- [ ] AC8: Given an item closed with a reason, when a user hovers over a reference to it (e.g. #5), then the
  popup shows that reason's icon.
- [ ] AC9: Given the API documentation page, when a script writer looks up the issue and pull request
  endpoints, then the new close reason inputs and response fields are listed.

## Out of Scope

- Showing reasons in UI: addressed by issue #54. This is the blocker for the timeline (AC4) and the popup
  icons/labels that will be needed (AC8).
- Back "dating"/"reasoning" old issues and PRs.
- Changing the reason of an already closed item.
- Any other UI change (#54).
- A `closed` flag on pull request creation (the API has none today).
- Merged pull requests, which keep no reason since "merged" is already their own state.

In scope: default reasons for automatic closes, close reason input and output on the issue and pull request
API endpoints, swagger docs, and reason icons in the hover popup.

## Open Questions

- Should old issues and PRs have a way to be updated? From the perspective of archival, likely not, but it
  is still something to consider. This is, however, out of scope of the current issue but a consequence that
  should be considered.

## Design Decision

### Decision 1: the default reason lives in `issue_service.CloseIssue`

**Decision:** `CloseIssue` records `DefaultCloseReason(issue.IsPull)` (`completed` for issues, `not_planned`
for pull requests) instead of no reason. The automatic close paths and old API scripts get their reason from
it, with no change at their call sites.
**Alternatives Considered:**
- Pass an explicit reason at each of the six call sites (commit keywords, merged-PR references, head branch
  deletion, base branch deletion, `CloseRepoBranchesPulls`, the API).
- Default inside `issues_model.SetIssueAsClosed`, so every close in Gitea gets one.

**Rationale:** Every caller left on `CloseIssue` wants exactly the default, so one change produces the issue's
whole table. Explicit call sites mean six edits, and a future caller that forgets would record no reason,
which works against "all closing activity can be audited". The API also needs the default for old scripts,
and it closes both issues and pull requests through the same code, so it needs the `IsPull` choice anyway.
Defaulting in the model would also change the web close with no reason that #54 settled, and the merge path
would need a special case.
**Consequences:** The reason a path records isn't visible at its call site; the function's doc comment and
one test per path document it. Any new caller of `CloseIssue` gets a reason without asking for one.

### Decision 2: pull requests default to Not planned; a merged pull request keeps no reason

**Decision:** A pull request closed without a given reason (any branch deletion, or an old API script) closes
as `not_planned`. A merged pull request itself keeps no reason.
**Alternatives Considered:** Record no reason for automatic pull request closes; or give a merged pull
request `completed`.
**Rationale:** `completed` isn't allowed for pull requests (they're completed by merging), and `not_planned`
is the only other reason that needs no extra input. No reason would leave a gap in the audit. "Merged" is
already a state of its own, and `merge.go` closes through `SetIssueAsClosed` directly, which #54 tests to
record no reason.
**Consequences:** Base branch deletion, which the issue didn't list, also gives `not_planned`.

### Decision 3: the API rejects a reason that doesn't come with a close

**Decision:** 422 when `close_reason`, `close_reason_text` or `close_duplicate_of` is sent but the request
doesn't close the item: `closed` is false on create, or `state` is missing, `open`, or the item is already
closed on edit.
**Alternatives Considered:** Ignore the stray fields, the way `state: closed` on a closed item is a no-op
today; or reject only when `state` isn't `closed`, but ignore the reason on an already closed item.
**Rationale:** A script that sends a reason expects it to be recorded. Silently dropping it lets the script
believe the audit trail has data it doesn't. Changing the reason of a closed item is out of scope (the
issue's open question), so it is refused rather than half supported.
**Consequences:** A script that resends a full payload to an item that is already closed gets a 422 if the
payload carries a reason. Supporting reason changes later means relaxing this rule. Because the check runs
first, a request that carries a reason gets this 422 before the older errors: on a merged pull request instead
of 412 "already merged", and with an unknown `state` instead of 412 "unknown state". Requests without a
reason see the old errors unchanged.

### Decision 4: the API checks the reason before writing anything

**Decision:** The API helper runs `Validate` and the duplicate lookup before `CreateIssue` creates the issue
and before `EditIssue` / `EditPullRequest` apply any field.
**Alternatives Considered:** Rely on the checks inside `SetIssueAsClosed` and map its errors to 422 where the
close happens today.
**Rationale:** Today `CreateIssue` creates the issue and then closes it, and the edit endpoints apply title,
assignee and milestone changes before the state change. Failing at the close would leave a new open issue or
a half-applied edit behind, breaking AC5's "the item stays open" in spirit.
**Consequences:** The checks run twice (up front and again in `SetIssueAsClosed`'s transaction); the second
run only matters in a race, where its error still maps to 422.

### Decision 5: responses include the duplicate's number

**Decision:** `api.Issue` and `api.PullRequest` gain `close_duplicate_of` (the duplicate's issue number, `0`
when none) next to `close_reason` and `close_reason_text`. Input and output use the same field names.
**Alternatives Considered:** Return only `close_reason` and `close_reason_text`, the minimum AC7 asks for.
**Rationale:** An audit that sees `duplicate` can't tell which issue it duplicates without scraping the
timeline. Using the number scripts send in, rather than the internal ID, lets them round-trip it.
**Consequences:** One extra lookup per duplicate-closed item when converting, so a list with many duplicates
makes a query per duplicate. It can be batched later if it shows up.

### Decision 6: the popup only swaps its icon

**Decision:** The hover popup shows the reason through `getIssueIcon` / `getIssueColorClass`, mirroring
`close_reason_icon.tmpl`. Its layout and the `/info` endpoint are unchanged.
**Alternatives Considered:** Add a state label with the reason's text to the popup.
**Rationale:** AC8 asks for the reason's icon. #22 deferred richer popup content to its own design pass, and
it made the popup appear on lists, boards and dependency links, so the icon change reaches all of them for
free. Matching the list icons keeps one visual language.
**Consequences:** The popup's per-page cache can show an outdated icon if the item closes while the page is
open, as it already does for open versus closed. The JS and template icon logic must be kept in sync by
hand; the comments in both point at each other.

## Scope

medium (`scope: medium` label)

---

## Appendix A: Technical design

### 1. Automatic closes get a default reason

`issue_service.CloseIssue` passes `{Reason: issues_model.DefaultCloseReason(issue.IsPull)}` instead of no
reason: `completed` for an issue, `not_planned` for a pull request. Its doc comment is updated to match.

Every remaining caller wants exactly this, so no call site changes, and the issue's table follows:

| Path | Result |
| --- | --- |
| Commit keyword (`fixes #5`) | `completed` |
| Merged PR closing an issue it references | `completed` |
| PR closed because its head branch was deleted | `not_planned` |
| PR closed because its base branch was deleted (no retargeting) | `not_planned` |
| PR closed by `CloseRepoBranchesPulls` | `not_planned` |

Commit keywords and merged-PR references only ever close issues: `UpdateIssuesCommit` skips pull requests
(`services/issue/commit.go`, "Only issues can be closed/reopened this way"), and `verifyReferencedIssue`
(`models/issues/issue_xref.go`) only lets close actions point from a pull request to an issue.

`services/pull/merge.go`'s own close of the merged PR (`SetIssueAsClosed(..., true, {})`) is unchanged.
See Decisions 1 and 2.

### 2. API input

New optional fields on `api.CreateIssueOption` (next to `closed`), `api.EditIssueOption` and
`api.EditPullRequestOption` (next to `state`):

| JSON | Go | Meaning |
| --- | --- | --- |
| `close_reason` | `CloseReason string` | `completed`, `not_planned`, `duplicate` or `other` |
| `close_reason_text` | `CloseReasonText string` | only with `other`, up to 255 characters |
| `close_duplicate_of` | `CloseDuplicateOf int64` | only with `duplicate`: an issue number in the same repository |

Closing a PR with a reason, for example:

```http
PATCH /api/v1/repos/{owner}/{repo}/pulls/{index}
{"state": "closed", "close_reason": "duplicate", "close_duplicate_of": 42}
```

`PATCH .../issues/{index}` on a PR behaves identically.

A helper in `routers/api/v1/repo` builds `issues_model.CloseReasonOptions` from the fields (name via
`issues_model.AsCloseReason`) and checks them **before the endpoint writes anything**, answering 422 with
the error's message when:

- any reason field is set but the request does not close the item: `closed` is false on create, or `state`
  is missing, `open`, or the item is already closed on edit;
- `Validate(isPull)` fails: unknown name, `completed` on a pull request, `other` with empty or over-long text,
  text or number sent with the wrong reason;
- the duplicate number is the item itself or no issue with that number exists in the repository (looked up
  with `GetIssueByIndex`; on create the item has no number yet, so only existence is checked).

Checking up front keeps AC5's "the item stays open" true for the whole request: `CreateIssue` does not
leave a new open issue behind, and `EditIssue` / `EditPullRequest` do not apply the title, assignee or
milestone part of an edit before failing.

On success, the close goes through `issue_service.CloseIssueWithReason` with the options. A request with
no reason fields keeps calling `issue_service.CloseIssue`, which now supplies the default (AC6). Error
handling around the close (412 for open dependencies) is unchanged; `SetIssueAsClosed` re-runs the
checks inside its transaction, and any `util.ErrInvalidArgument` it still returns (a race) maps to 422.

### 3. API output

New fields on `api.Issue` and `api.PullRequest`, set by `services/convert` (`toIssue`, and both PR
converters in `convert/pull.go`):

| JSON | Value |
| --- | --- |
| `close_reason` | `issue.CloseReason.String()`; `""` for no reason |
| `close_reason_text` | `issue.CloseReasonText` |
| `close_duplicate_of` | the number of `issue.CloseDuplicateIssueID`'s issue; `0` when none or the target is gone |

The duplicate number costs one lookup per duplicate-closed item, only when one exists. Webhook payloads use
the same converters and pick the fields up too.

Swagger comments document every new field, then `templates/swagger/v1_json.tmpl` and
`v1_openapi3_json.tmpl` are regenerated (AC9).

### 4. Hover popup

`GET .../{issues|pulls}/{index}/info` already returns `convert.ToIssue`, so `close_reason` reaches
`ContextPopup.vue` with no endpoint change. The `Issue` type in `web_src/js/types.ts` gains
`close_reason`, and `getIssueIcon` / `getIssueColorClass` mirror `close_reason_icon.tmpl` for closed,
unmerged items:

| Reason | Icon | Colour |
| --- | --- | --- |
| `completed` | `octicon-issue-closed` | `tw-text-purple` |
| `not_planned` | `octicon-skip` | `tw-text-text-light` |
| `duplicate` | `octicon-duplicate` | `tw-text-text-light` |
| `other` | `octicon-note` | `tw-text-text-light` |
| none | unchanged closed icon | `tw-text-red` |

The `#` suggestion list in the editor (`TextExpander.ts`) and the similar-issues panel on the new-issue page
(`repo-issue-similar.ts`) draw with the same functions from `services/issue/suggestion.go` and
`services/issue/similar.go`, which build their `structs.Issue` by hand, so both gain `CloseReason` too and
their icons match.

`octicon-duplicate` and `octicon-note` are added to `web_src/js/svg.ts` (the SVGs exist in
`public/assets/img/svg/`). The "doesn't show reasons yet" notes in `issueicon.tmpl`,
`close_reason_icon.tmpl` and `issue.ts` are updated. This builds on #22, which made the popup appear on
lists, boards and dependency links, and does not touch its layout.

## Appendix B: Testing

- **Unit (Go):** the API reason helper's rules (stray reason, each `Validate` failure, duplicate checks).
- **Unit (TS):** `getIssueIcon` / `getIssueColorClass` for each reason, issue and PR, and merged PR.
- **Integration:**
  - AC1: push a commit with `fixes #N`, issue is `completed`.
  - AC2: merge a PR referencing an issue, issue is `completed`.
  - AC3: delete a PR's head branch, PR is `not_planned`.
  - AC4/AC7: close an issue and a PR through the API with each reason, the response and a re-read show it.
  - AC5: each invalid input gives 422, the item stays open and other edited fields are unchanged; create with
    an invalid reason creates nothing.
  - AC6: close without a reason (edit, and create with `closed: true`) gives `completed` / `not_planned`.
  - AC7: an open item, and one closed with no reason, return `""`.
- **Swagger:** `make swagger-check` (or its underlying commands) passes.

## Appendix C: Acceptance criteria map

| AC | Covered by |
| --- | --- |
| AC1, AC2, AC3 | Appendix A, section 1 |
| AC4, AC5, AC6 | Appendix A, section 2 (AC4's timeline entry comes from the close comment's reason, shown since #54) |
| AC7 | Appendix A, section 3 |
| AC8 | Appendix A, section 4 |
| AC9 | Appendix A, section 3 (swagger) |
