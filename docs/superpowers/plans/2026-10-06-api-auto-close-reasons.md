# Close Reasons for Automatic and API Closes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every automatic or API close of an issue or pull request records a close reason that scripts can set and read back through the API, and the hover popup shows it.

**Architecture:** `issue_service.CloseIssue` records `DefaultCloseReason(isPull)` instead of no reason, which covers every automatic path at once. A small helper in `routers/api/v1/repo` reads `close_reason` / `close_reason_text` / `close_duplicate_of`, checks them before the endpoint writes anything, and closes through `CloseIssueWithReason`. The converters expose the same three fields, and the popup's JS icon logic mirrors `close_reason_icon.tmpl`.

**Tech Stack:** Go (xorm models, Gitea services, chi API routers, go-swagger), TypeScript + Vue (Vitest), Gitea's unittest fixtures and integration test harness.

**Spec:** `docs/superpowers/specs/2026-10-06-api-auto-close-reasons-design.md`
**Issue:** CSCI-435-SE/gitea#55 · **Branch:** `feat/issue-55-api-auto-close-reasons`

## Ground rules for whoever executes this

- `make` is not installed on this machine. Use the commands written in each step.
- **Do not run `git commit` yourself.** The human commits (AGENTS.md: commit only after a human approves that commit). Each "Commit" step gives the exact command to hand over; leave the work uncommitted and move on.
- New `.go` files get the header `// Copyright 2026 The Gitea Authors. All rights reserved.` + `// SPDX-License-Identifier: MIT`.
- No trailing whitespace. Keep comments short, explain why, same-line where possible. Keep existing comments that are still true.
- After any Go edit run `gofmt -l <files>`; it must print nothing (fix with `gofmt -w`).

## File map

| File | Change |
| --- | --- |
| `models/issues/issue_close_reason.go` | add `CloseReasonOptions.DuplicateIssueID` (the duplicate lookup, shared by the model and the API) |
| `models/issues/issue_update.go` | `SetIssueAsClosed` uses `DuplicateIssueID` |
| `models/issues/issue_close_reason_test.go` | test `DuplicateIssueID` |
| `services/issue/status.go` | `CloseIssue` records the default reason |
| `services/issue/status_test.go` (new) | default reason for an issue and a pull request |
| `services/issue/commit_test.go` | AC1: `fixes #1` closes as completed |
| `services/pull/close_reason_test.go` (new) | AC2: merged PR reference closes as completed; AC3: deleted head branch closes as not planned |
| `modules/structs/issue.go`, `modules/structs/pull.go` | new input and output fields with swagger comments |
| `services/convert/issue.go`, `services/convert/pull.go` | fill the output fields |
| `services/convert/issue_test.go` | test the output fields |
| `routers/api/v1/repo/issue_close_reason.go` (new) | `apiCloseReason`: read, check (422), close |
| `routers/api/v1/repo/issue_close_reason_test.go` (new) | unit test of the checks |
| `routers/api/v1/repo/issue.go`, `routers/api/v1/repo/pull.go` | wire the helper into `CreateIssue`, `EditIssue`, `EditPullRequest`, `closeOrReopenIssue` |
| `tests/integration/api_issue_close_reason_test.go` (new) | AC4–AC7 end to end |
| `templates/swagger/v1_json.tmpl`, `templates/swagger/v1_openapi3_json.tmpl` | regenerated (AC9) |
| `web_src/js/types.ts`, `web_src/js/features/issue.ts`, `web_src/js/svg.ts` | popup icons by reason (AC8) |
| `web_src/js/features/issue.test.ts` (new) | icon/colour per reason |
| `services/issue/suggestion.go` | `#` suggestions carry `close_reason` so their icons match too |
| `templates/shared/issueicon.tmpl`, `templates/shared/close_reason_icon.tmpl` | update the "JS doesn't show reasons yet" comments |
| `.claude-students/docs/services-issue.md`, `.claude-students/docs/models-issues.md` + their `guide/` twins | describe the new behaviour |

---

### Task 1: Share the duplicate lookup as `CloseReasonOptions.DuplicateIssueID`

The API must check a duplicate target before writing anything, and `SetIssueAsClosed` already has that check inline. Move it to one method both use.

**Files:**
- Modify: `models/issues/issue_close_reason.go`
- Modify: `models/issues/issue_update.go:65-77` (the `var duplicateID int64` block in `SetIssueAsClosed`)
- Test: `models/issues/issue_close_reason_test.go`

- [ ] **Step 1: Write the failing test**

Append to `models/issues/issue_close_reason_test.go`:

```go
func TestCloseReasonOptionsDuplicateIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	target := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: 4})

	id, err := issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonCompleted}.DuplicateIssueID(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Zero(t, id) // only the duplicate reason names a target

	dup := issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 4}
	id, err = dup.DuplicateIssueID(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, target.ID, id) // the global ID, not the number

	id, err = dup.DuplicateIssueID(t.Context(), 1, 0) // an item not created yet has no number of its own
	require.NoError(t, err)
	assert.Equal(t, target.ID, id)

	_, err = dup.DuplicateIssueID(t.Context(), 1, 4)
	assert.True(t, issues_model.IsErrInvalidCloseDuplicate(err), "an item cannot duplicate itself")

	_, err = dup.DuplicateIssueID(t.Context(), 2, 1) // repo 2 has no #4
	assert.True(t, issues_model.IsErrInvalidCloseDuplicate(err), "the target must be in the same repository")
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test -run '^TestCloseReasonOptionsDuplicateIssueID$' ./models/issues/`
Expected: FAIL to compile, `dup.DuplicateIssueID undefined`.

- [ ] **Step 3: Add the method**

In `models/issues/issue_close_reason.go`, add `"context"` to the imports (first in the block), and add after `Validate`:

```go
// DuplicateIssueID returns the ID of the issue the options mark as duplicated, or 0 for any other reason.
// It checks the target is in the repository and isn't the item itself; selfIndex is 0 for an item not created yet.
func (opts CloseReasonOptions) DuplicateIssueID(ctx context.Context, repoID, selfIndex int64) (int64, error) {
	if opts.Reason != CloseReasonDuplicate {
		return 0, nil
	}
	if opts.DuplicateIndex == selfIndex {
		return 0, ErrInvalidCloseDuplicate{Index: opts.DuplicateIndex, Detail: "an issue cannot duplicate itself"}
	}
	target, err := GetIssueByIndex(ctx, repoID, opts.DuplicateIndex) // the repo ID keeps the target in the same repository
	if IsErrIssueNotExist(err) {
		return 0, ErrInvalidCloseDuplicate{Index: opts.DuplicateIndex, Detail: "no issue with this number in the repository"}
	} else if err != nil {
		return 0, err
	}
	return target.ID, nil
}
```

Also update `Validate`'s doc comment, which names where the existence check lives:

```go
// Validate checks the options for an issue or pull request, without looking anything up;
// DuplicateIssueID checks that the duplicate target exists.
```

- [ ] **Step 4: Use it in `SetIssueAsClosed`**

In `models/issues/issue_update.go`, replace this block:

```go
	var duplicateID int64
	if reason.Reason == CloseReasonDuplicate {
		if reason.DuplicateIndex == issue.Index {
			return nil, ErrInvalidCloseDuplicate{Index: reason.DuplicateIndex, Detail: "an issue cannot duplicate itself"}
		}
		target, err := GetIssueByIndex(ctx, issue.RepoID, reason.DuplicateIndex) // the repo ID keeps the target in the same repository
		if IsErrIssueNotExist(err) {
			return nil, ErrInvalidCloseDuplicate{Index: reason.DuplicateIndex, Detail: "no issue with this number in the repository"}
		} else if err != nil {
			return nil, err
		}
		duplicateID = target.ID
	}
```

with:

```go
	duplicateID, err := reason.DuplicateIssueID(ctx, issue.RepoID, issue.Index)
	if err != nil {
		return nil, err
	}
```

- [ ] **Step 5: Run the model tests**

Run: `go test -run 'CloseReason|CloseIssue|ReopenIssue' ./models/issues/`
Expected: PASS (the new test and the existing `TestCloseIssueAsDuplicate`, `TestCloseIssueInvalidReasonLeavesItOpen`).

Run: `gofmt -l models/issues/` — expect no output.

---

### Task 2: Automatic closes record the default reason

**Files:**
- Modify: `services/issue/status.go:17-23` (`CloseIssue` and its doc comment)
- Modify: `models/issues/issue_close_reason.go` (`DefaultCloseReason` doc comment)
- Create: `services/issue/status_test.go`
- Modify: `services/issue/commit_test.go`
- Create: `services/pull/close_reason_test.go`

- [ ] **Step 1: Write the failing service test**

Create `services/issue/status_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseIssueRecordsDefaultReason(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	cases := []struct {
		name    string
		issueID int64
		want    issues_model.CloseReason
	}{
		{"issue", 1, issues_model.CloseReasonCompleted},
		{"pull request", 3, issues_model.CloseReasonNotPlanned}, // pull request 2, open
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			require.NoError(t, CloseIssue(t.Context(), issue, doer, ""))

			issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.issueID})
			assert.True(t, issue.IsClosed)
			assert.Equal(t, c.want, issue.CloseReason)
			comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: c.issueID, Type: issues_model.CommentTypeClose})
			assert.Equal(t, c.want, comment.MetaCloseReason().Reason) // what the timeline shows
		})
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test -run '^TestCloseIssueRecordsDefaultReason$' ./services/issue/`
Expected: FAIL, `expected: 1 actual: 0` (and `2` / `0` for the pull request).

- [ ] **Step 3: Record the default in `CloseIssue`**

In `services/issue/status.go`, replace `CloseIssue` and its comment with:

```go
// CloseIssue closes an issue with the default reason for its kind: completed for an issue, not planned for a
// pull request. It is for closes that give no reason: commit keywords, merged pull requests closing the issues
// they reference, deleted branches, and API requests without one. Callers with a reason use CloseIssueWithReason.
func CloseIssue(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, commitID string) error {
	return CloseIssueWithReason(ctx, issue, doer, commitID, issues_model.CloseReasonOptions{Reason: issues_model.DefaultCloseReason(issue.IsPull)})
}
```

In `models/issues/issue_close_reason.go`, change the `DefaultCloseReason` doc comment to:

```go
// DefaultCloseReason is the reason the close button starts on, and the one recorded by closes that give none
// (issue_service.CloseIssue); consumers may still send any allowed reason.
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test -run '^TestCloseIssueRecordsDefaultReason$' ./services/issue/`
Expected: PASS.

- [ ] **Step 5: Add the AC1 test (commit keyword)**

Append to `services/issue/commit_test.go`, and add `"github.com/stretchr/testify/require"` to its imports:

```go
func TestUpdateIssuesCommitClosesAsCompleted(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo.Owner = user
	pushCommits := []*repository.PushCommit{{
		Sha1:           "abcdef9",
		CommitterEmail: "user2@example.com",
		CommitterName:  "User Two",
		AuthorEmail:    "user2@example.com",
		AuthorName:     "User Two",
		Message:        "fixes #1",
	}}

	require.NoError(t, UpdateIssuesCommit(t.Context(), user, repo, pushCommits, repo.DefaultBranch))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	assert.True(t, issue.IsClosed)
	assert.Equal(t, issues_model.CloseReasonCompleted, issue.CloseReason)
}
```

- [ ] **Step 6: Add the AC2 and AC3 tests (merge reference, deleted branch)**

Create `services/pull/close_reason_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/references"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCloseCrossReferencesClosesAsCompleted(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // open, in repo 1
	require.NoError(t, pr.LoadIssue(t.Context()))
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// what "Fixes #1" in the pull request's description records on issue #1
	require.NoError(t, db.Insert(t.Context(), &issues_model.Comment{
		Type:       issues_model.CommentTypePullRef,
		PosterID:   doer.ID,
		IssueID:    1,
		RefRepoID:  pr.Issue.RepoID,
		RefIssueID: pr.Issue.ID,
		RefAction:  references.XRefActionCloses,
		RefIsPull:  true,
	}))

	require.NoError(t, handleCloseCrossReferences(t.Context(), pr, doer))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.True(t, issue.IsClosed)
	assert.Equal(t, issues_model.CloseReasonCompleted, issue.CloseReason)
}

func TestAdjustPullsCausedByBranchDeletedClosesAsNotPlanned(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}) // open, head branch "branch2" in repo 1
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pr.HeadRepoID})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	require.NoError(t, AdjustPullsCausedByBranchDeleted(t.Context(), doer, repo, pr.HeadBranch))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: pr.IssueID})
	assert.True(t, issue.IsClosed)
	assert.Equal(t, issues_model.CloseReasonNotPlanned, issue.CloseReason)
}
```

If `TestAdjustPullsCausedByBranchDeletedClosesAsNotPlanned` fails inside `retargetBranchPulls` (it needs a git repo when a pull request targets `branch2`), wrap the call with `defer test.MockVariableValue(&setting.Repository.PullRequest.RetargetChildrenOnMerge, false)()` (imports `gitea.dev/modules/setting`, `gitea.dev/modules/test`) and re-run. Do not change production code for it.

- [ ] **Step 7: Run every touched package**

Run: `go test ./services/issue/ ./services/pull/ ./models/issues/`
Expected: PASS. If an existing test asserted `CloseReasonNone` after `issue_service.CloseIssue`, it now sees the default: update that assertion to `DefaultCloseReason`'s value and note it in the hand-over (merges must still assert `CloseReasonNone`; `TestSetMergedRecordsNoCloseReason` must stay green unchanged).

Run: `grep -rn "CloseReasonNone" tests/integration/ services/ routers/ --include=*_test.go` and check each hit still matches the new behaviour.

Run: `gofmt -l services/issue/ services/pull/ models/issues/` — expect no output.

- [ ] **Step 8: Commit (hand over to the human)**

```bash
git add models/issues/issue_close_reason.go models/issues/issue_update.go models/issues/issue_close_reason_test.go services/issue/status.go services/issue/status_test.go services/issue/commit_test.go services/pull/close_reason_test.go
git commit -m "feat(issues): record a default close reason for automatic closes (#55)" -m "Commit keywords and merged pull requests now close issues as completed, and deleted branches close pull requests as not planned. The duplicate lookup moves to CloseReasonOptions.DuplicateIssueID so the API can run it before writing." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 3: API structs

**Files:**
- Modify: `modules/structs/issue.go` (`Issue`, `CreateIssueOption`, `EditIssueOption`)
- Modify: `modules/structs/pull.go` (`PullRequest`, `EditPullRequestOption`)

No test of its own: the converter and API tests in Tasks 4–5 exercise these fields.

- [ ] **Step 1: Output fields on `api.Issue`**

In `modules/structs/issue.go`, in `type Issue struct`, directly after the `State     StateType `json:"state"`` line group (after `Comments  int       `json:"comments"``), add:

```go
	// Why it was closed: completed, not_planned, duplicate or other; empty when open or closed without a reason
	CloseReason string `json:"close_reason"`
	// The close reason's description, only for other
	CloseReasonText string `json:"close_reason_text"`
	// The number of the issue it duplicates, in the same repository, only for duplicate; 0 otherwise
	CloseDuplicateOf int64 `json:"close_duplicate_of"`
```

No `omitempty`: AC7 wants items without a reason to return an empty reason.

- [ ] **Step 2: Input fields on `CreateIssueOption` and `EditIssueOption`**

In `CreateIssueOption`, after `Closed   bool    `json:"closed"``, add:

```go
	// Why the issue is closed, only with closed: completed, not_planned, duplicate or other; defaults to completed
	CloseReason string `json:"close_reason"`
	// Required with the other close reason, up to 255 characters
	CloseReasonText string `json:"close_reason_text"`
	// Required with the duplicate close reason: the number of an issue in the same repository
	CloseDuplicateOf int64 `json:"close_duplicate_of"`
```

In `EditIssueOption`, after `State    *string  `json:"state"``, add:

```go
	// Why the item is closed, only when state closes it: completed (issues only), not_planned, duplicate or other;
	// defaults to completed for an issue and not_planned for a pull request
	CloseReason string `json:"close_reason"`
	// Required with the other close reason, up to 255 characters
	CloseReasonText string `json:"close_reason_text"`
	// Required with the duplicate close reason: the number of an issue in the same repository
	CloseDuplicateOf int64 `json:"close_duplicate_of"`
```

- [ ] **Step 3: Fields on `api.PullRequest` and `EditPullRequestOption`**

In `modules/structs/pull.go`, in `type PullRequest struct`, after the `State StateType `json:"state"`` line, add:

```go
	// Why it was closed: not_planned, duplicate or other; empty when open, merged or closed without a reason
	CloseReason string `json:"close_reason"`
	// The close reason's description, only for other
	CloseReasonText string `json:"close_reason_text"`
	// The number of the issue it duplicates, in the same repository, only for duplicate; 0 otherwise
	CloseDuplicateOf int64 `json:"close_duplicate_of"`
```

In `EditPullRequestOption`, after `State *string `json:"state"``, add:

```go
	// Why the pull request is closed, only when state closes it: not_planned, duplicate or other; defaults to not_planned
	CloseReason string `json:"close_reason"`
	// Required with the other close reason, up to 255 characters
	CloseReasonText string `json:"close_reason_text"`
	// Required with the duplicate close reason: the number of an issue in the same repository
	CloseDuplicateOf int64 `json:"close_duplicate_of"`
```

- [ ] **Step 4: Build**

Run: `go build ./modules/structs/ && gofmt -l modules/structs/`
Expected: builds, no gofmt output.

---

### Task 4: Converters fill the output fields

**Files:**
- Modify: `services/convert/issue.go` (in `toIssue`, after the `issue.ClosedUnix` block)
- Modify: `services/convert/pull.go` (both `&api.PullRequest{...}` literals, around lines 74 and 350)
- Test: `services/convert/issue_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/convert/issue_test.go` (add `"github.com/stretchr/testify/require"` and `user_model "gitea.dev/models/user"` to the imports if missing):

```go
func TestToAPIIssueCloseReason(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1}) // open
	apiIssue := ToAPIIssue(t.Context(), doer, issue)
	assert.Empty(t, apiIssue.CloseReason)
	assert.Empty(t, apiIssue.CloseReasonText)
	assert.Zero(t, apiIssue.CloseDuplicateOf)

	_, err := issues_model.CloseIssue(t.Context(), issue, doer, issues_model.CloseReasonOptions{Reason: issues_model.CloseReasonDuplicate, DuplicateIndex: 4})
	require.NoError(t, err)
	issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	apiIssue = ToAPIIssue(t.Context(), doer, issue)
	assert.Equal(t, "duplicate", apiIssue.CloseReason)
	assert.Equal(t, int64(4), apiIssue.CloseDuplicateOf) // the number scripts send, not the stored ID

	issue.CloseDuplicateIssueID = 999999 // the target was deleted
	assert.Zero(t, ToAPIIssue(t.Context(), doer, issue).CloseDuplicateOf)
	assert.Equal(t, "duplicate", ToAPIIssue(t.Context(), doer, issue).CloseReason)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test -run '^TestToAPIIssueCloseReason$' ./services/convert/`
Expected: FAIL, `expected: "duplicate" actual: ""`.

- [ ] **Step 3: Fill the fields in `toIssue`**

In `services/convert/issue.go`, right after:

```go
	if issue.ClosedUnix != 0 {
		apiIssue.Closed = issue.ClosedUnix.AsTimePtr()
	}
```

add:

```go
	apiIssue.CloseReason = issue.CloseReason.String()
	apiIssue.CloseReasonText = issue.CloseReasonText
	if issue.CloseDuplicateIssueID != 0 {
		target, err := issues_model.GetIssueByID(ctx, issue.CloseDuplicateIssueID)
		if err == nil {
			apiIssue.CloseDuplicateOf = target.Index
		} else if !issues_model.IsErrIssueNotExist(err) { // a deleted target leaves 0
			return &api.Issue{}
		}
	}
```

- [ ] **Step 4: Copy them into both pull request literals**

In `services/convert/pull.go`, in **both** `&api.PullRequest{` literals (the one in `ToAPIPullRequest` and the one in `ToAPIPullRequests`), add after the `State:          apiIssue.State,` line:

```go
		CloseReason:      apiIssue.CloseReason,
		CloseReasonText:  apiIssue.CloseReasonText,
		CloseDuplicateOf: apiIssue.CloseDuplicateOf,
```

(Match the literal's indentation; `gofmt -w` realigns the colons.)

- [ ] **Step 5: Run the tests**

Run: `go test ./services/convert/ && gofmt -l services/convert/`
Expected: PASS, no gofmt output.

---

### Task 5: The API reads, checks and records the reason

**Files:**
- Create: `routers/api/v1/repo/issue_close_reason.go`
- Create: `routers/api/v1/repo/issue_close_reason_test.go`
- Modify: `routers/api/v1/repo/issue.go` (`CreateIssue`, `EditIssue`, `closeOrReopenIssue`)
- Modify: `routers/api/v1/repo/pull.go` (`EditPullRequest`)
- Create: `tests/integration/api_issue_close_reason_test.go`

- [ ] **Step 1: Write the failing unit test**

Create `routers/api/v1/repo/issue_close_reason_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"
	"testing"

	"gitea.dev/models/unittest"
	"gitea.dev/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestAPICloseReasonCheck(t *testing.T) {
	unittest.PrepareTestEnv(t)

	cases := []struct {
		name   string
		reason apiCloseReason
		closes bool
		isPull bool
		index  int64
		ok     bool
	}{
		{"no fields", newAPICloseReason("", "", 0), false, false, 1, true}, // old scripts
		{"valid", newAPICloseReason("not_planned", "", 0), true, false, 1, true},
		{"duplicate", newAPICloseReason("duplicate", "", 4), true, false, 1, true},
		{"new item duplicate", newAPICloseReason("duplicate", "", 4), true, false, 0, true},
		{"not closing", newAPICloseReason("completed", "", 0), false, false, 1, false},
		{"text alone, not closing", newAPICloseReason("", "why", 0), false, false, 1, false},
		{"unknown name", newAPICloseReason("wontfix", "", 0), true, false, 1, false},
		{"completed on a pull request", newAPICloseReason("completed", "", 0), true, true, 2, false},
		{"other without text", newAPICloseReason("other", "  ", 0), true, false, 1, false},
		{"other too long", newAPICloseReason("other", strings.Repeat("a", 256), 0), true, false, 1, false},
		{"text with completed", newAPICloseReason("completed", "why", 0), true, false, 1, false},
		{"duplicate of itself", newAPICloseReason("duplicate", "", 1), true, false, 1, false},
		{"duplicate missing", newAPICloseReason("duplicate", "", 999), true, false, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, resp := contexttest.MockAPIContext(t, "user2/repo1")
			contexttest.LoadRepo(t, ctx, 1)
			assert.Equal(t, c.ok, c.reason.check(ctx, c.closes, c.isPull, c.index))
			if !c.ok {
				assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
			}
		})
	}
}

func TestAPICloseReasonNotAllowedMessage(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, resp := contexttest.MockAPIContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	newAPICloseReason("completed", "", 0).check(ctx, true, true, 2)
	// scripts see names, never the stored numbers
	assert.Contains(t, resp.Body.String(), `close_reason \"completed\" is not allowed for a pull request, use one of: not_planned, duplicate, other`)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test -run '^TestAPICloseReason' ./routers/api/v1/repo/`
Expected: FAIL to compile, `undefined: newAPICloseReason`.

- [ ] **Step 3: Write the helper**

Create `routers/api/v1/repo/issue_close_reason.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	issues_model "gitea.dev/models/issues"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/util"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
)

// apiCloseReason is the close reason an API request gives in close_reason, close_reason_text and close_duplicate_of.
type apiCloseReason struct {
	name  string
	opts  issues_model.CloseReasonOptions
	given bool // without any of the fields, a close records the default reason
}

func newAPICloseReason(name, text string, duplicateOf int64) apiCloseReason {
	return apiCloseReason{
		name:  name,
		opts:  issues_model.CloseReasonOptions{Reason: issues_model.AsCloseReason(name), Text: text, DuplicateIndex: duplicateOf},
		given: name != "" || text != "" || duplicateOf != 0,
	}
}

// editCloses reports whether an edit's state field closes an open item.
func editCloses(state *string, issue *issues_model.Issue) bool {
	return state != nil && api.StateType(*state) == api.StateClosed && !issue.IsClosed
}

// check answers 422 and returns false when the reason can't be used. It runs before the request writes anything,
// so a rejected request leaves the item as it was. index is 0 for an item not created yet.
func (r apiCloseReason) check(ctx *context.APIContext, closes, isPull bool, index int64) bool {
	if !r.given {
		return true
	}
	if !closes { // changing the reason of a closed item is not supported
		ctx.APIError(http.StatusUnprocessableEntity, "close_reason, close_reason_text and close_duplicate_of are only allowed when the request closes an open issue or pull request")
		return false
	}
	if err := r.opts.Validate(isPull); err != nil {
		msg := err.Error()
		if issues_model.IsErrCloseReasonNotAllowed(err) { // its own message shows the stored number
			names := make([]string, 0, 4)
			for _, allowed := range issues_model.AllowedCloseReasons(isPull) {
				names = append(names, allowed.String())
			}
			msg = fmt.Sprintf("close_reason %q is not allowed for a%s, use one of: %s", r.name, util.Iif(isPull, " pull request", "n issue"), strings.Join(names, ", "))
		}
		ctx.APIError(http.StatusUnprocessableEntity, msg)
		return false
	}
	if _, err := r.opts.DuplicateIssueID(ctx, ctx.Repo.Repository.ID, index); err != nil {
		if issues_model.IsErrInvalidCloseDuplicate(err) {
			ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		} else {
			ctx.APIErrorInternal(err)
		}
		return false
	}
	return true
}

// closeIssue closes the item with the request's reason, or the default one when it gave none,
// answering the error itself; it returns false when it did.
func (r apiCloseReason) closeIssue(ctx *context.APIContext, issue *issues_model.Issue) bool {
	var err error
	if r.given {
		err = issue_service.CloseIssueWithReason(ctx, issue, ctx.Doer, "", r.opts)
	} else {
		err = issue_service.CloseIssue(ctx, issue, ctx.Doer, "")
	}
	switch {
	case err == nil:
		return true
	case issues_model.IsErrDependenciesLeft(err):
		ctx.APIError(http.StatusPreconditionFailed, "cannot close this issue or pull request because it still has open dependencies")
	case errors.Is(err, util.ErrInvalidArgument): // check already ran; only a concurrent change gets here
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
	default:
		ctx.APIErrorInternal(err)
	}
	return false
}
```

- [ ] **Step 4: Run the unit test**

Run: `go test -run '^TestAPICloseReason' ./routers/api/v1/repo/`
Expected: PASS. If the message assertion fails only on JSON escaping, compare against `resp.Body.String()` printed by the failure and adjust the expected escaping, not the message.

- [ ] **Step 5: Wire it into `closeOrReopenIssue`**

In `routers/api/v1/repo/issue.go`, replace `closeOrReopenIssue` with:

```go
func closeOrReopenIssue(ctx *context.APIContext, issue *issues_model.Issue, state api.StateType, closeReason apiCloseReason) {
	if state != api.StateOpen && state != api.StateClosed {
		ctx.APIError(http.StatusPreconditionFailed, fmt.Sprintf("unknown state: %s", state))
		return
	}

	if state == api.StateClosed && !issue.IsClosed {
		closeReason.closeIssue(ctx, issue)
	} else if state == api.StateOpen && issue.IsClosed {
		if err := issue_service.ReopenIssue(ctx, issue, ctx.Doer, ""); err != nil {
			ctx.APIErrorInternal(err)
			return
		}
	}
}
```

- [ ] **Step 6: Wire it into `EditIssue`**

In `EditIssue`, directly after the "Fail fast" `form.ContentVersion` check (the block ending `return` / `}` that answers `http.StatusConflict`), add:

```go
	closeReason := newAPICloseReason(form.CloseReason, form.CloseReasonText, form.CloseDuplicateOf)
	if !closeReason.check(ctx, editCloses(form.State, issue), issue.IsPull, issue.Index) {
		return
	}
```

Change the call further down from `closeOrReopenIssue(ctx, issue, state)` to `closeOrReopenIssue(ctx, issue, state, closeReason)`.

In `EditIssue`'s swagger comment, add after the `"412"` response:

```go
	//   "422":
	//     "$ref": "#/responses/validationError"
```

- [ ] **Step 7: Wire it into `EditPullRequest`**

In `routers/api/v1/repo/pull.go`, `EditPullRequest`, directly after its "Fail fast" `form.ContentVersion` check, add:

```go
	closeReason := newAPICloseReason(form.CloseReason, form.CloseReasonText, form.CloseDuplicateOf)
	if !closeReason.check(ctx, editCloses(form.State, issue), true, issue.Index) {
		return
	}
```

Change `closeOrReopenIssue(ctx, issue, state)` to `closeOrReopenIssue(ctx, issue, state, closeReason)`. (`EditPullRequest` already documents 422.)

- [ ] **Step 8: Wire it into `CreateIssue`**

In `CreateIssue`, directly after `form := web.GetForm(ctx).(*api.CreateIssueOption)`, add:

```go
	closeReason := newAPICloseReason(form.CloseReason, form.CloseReasonText, form.CloseDuplicateOf)
	if !closeReason.check(ctx, form.Closed, false, 0) { // before NewIssue, so a rejected reason creates nothing
		return
	}
```

Replace the existing close block:

```go
	if form.Closed {
		if err := issue_service.CloseIssue(ctx, issue, ctx.Doer, ""); err != nil {
			if issues_model.IsErrDependenciesLeft(err) {
				ctx.APIError(http.StatusPreconditionFailed, "cannot close this issue because it still has open dependencies")
				return
			}
			ctx.APIErrorInternal(err)
			return
		}
	}
```

with:

```go
	if form.Closed && !closeReason.closeIssue(ctx, issue) {
		return
	}
```

- [ ] **Step 9: Build and run the package tests**

Run: `go build ./routers/... && go test ./routers/api/v1/repo/ && gofmt -l routers/api/v1/repo/`
Expected: builds, PASS, no gofmt output. If `errors` becomes unused in `issue.go`, the compiler says so; leave imports as the compiler requires.

- [ ] **Step 10: Write the integration test (AC4–AC7)**

Create `tests/integration/api_issue_close_reason_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	api "gitea.dev/modules/structs"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPICloseReason(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue, auth_model.AccessTokenScopeWriteRepository)
	issuesURL := fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name)
	pullsURL := fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner.Name, repo.Name)
	closed := "closed"
	open := "open"

	newIssue := func(t *testing.T, title string) int64 {
		req := NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: title}).AddTokenAuth(token)
		return DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).Index
	}
	load := func(t *testing.T, index int64) *issues_model.Issue {
		return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: index})
	}

	t.Run("IssueWithReason", func(t *testing.T) { // AC4, AC7
		index := newIssue(t, "close as other")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{
			State: &closed, CloseReason: "other", CloseReasonText: "moved to the forum",
		}).AddTokenAuth(token)
		apiIssue := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		assert.Equal(t, "other", apiIssue.CloseReason)
		assert.Equal(t, "moved to the forum", apiIssue.CloseReasonText)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d", issuesURL, index)).AddTokenAuth(token)
		apiIssue = DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &api.Issue{})
		assert.Equal(t, "other", apiIssue.CloseReason) // read back
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: apiIssue.ID, Type: issues_model.CommentTypeClose})
		assert.Equal(t, issues_model.CloseReasonOther, comment.MetaCloseReason().Reason) // the timeline shows it
	})

	t.Run("IssueDuplicate", func(t *testing.T) {
		index := newIssue(t, "close as duplicate")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{
			State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 1,
		}).AddTokenAuth(token)
		apiIssue := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{})
		assert.Equal(t, "duplicate", apiIssue.CloseReason)
		assert.Equal(t, int64(1), apiIssue.CloseDuplicateOf)
	})

	t.Run("IssueInvalid", func(t *testing.T) { // AC5
		index := newIssue(t, "stays open")
		cases := map[string]api.EditIssueOption{
			"other without text":   {State: &closed, CloseReason: "other"},
			"other too long":       {State: &closed, CloseReason: "other", CloseReasonText: strings.Repeat("a", 256)},
			"duplicate missing":    {State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 999},
			"duplicate of itself":  {State: &closed, CloseReason: "duplicate", CloseDuplicateOf: index},
			"unknown name":         {State: &closed, CloseReason: "wontfix"},
			"text with completed":  {State: &closed, CloseReason: "completed", CloseReasonText: "why"},
			"reason without state": {CloseReason: "completed"},
			"reason while opening": {State: &open, CloseReason: "completed"},
		}
		for name, opts := range cases {
			t.Run(name, func(t *testing.T) {
				opts.Title = "changed by a rejected request"
				req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &opts).AddTokenAuth(token)
				MakeRequest(t, req, http.StatusUnprocessableEntity)
				issue := load(t, index)
				assert.False(t, issue.IsClosed)
				assert.Equal(t, "stays open", issue.Title) // nothing in the request was applied
			})
		}
	})

	t.Run("AlreadyClosed", func(t *testing.T) {
		req := NewRequestWithJSON(t, "PATCH", issuesURL+"/4", &api.EditIssueOption{State: &closed, CloseReason: "not_planned"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		assert.Equal(t, issues_model.CloseReasonNone, load(t, 4).CloseReason) // the fixture's closed issue keeps no reason
	})

	t.Run("Pull", func(t *testing.T) { // AC4, AC5 on a pull request
		req := NewRequestWithJSON(t, "PATCH", pullsURL+"/3", &api.EditPullRequestOption{State: &closed, CloseReason: "completed"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		assert.False(t, load(t, 3).IsClosed)

		req = NewRequestWithJSON(t, "PATCH", pullsURL+"/3", &api.EditPullRequestOption{
			State: &closed, CloseReason: "duplicate", CloseDuplicateOf: 1,
		}).AddTokenAuth(token)
		apiPull := DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.PullRequest{})
		assert.Equal(t, "duplicate", apiPull.CloseReason)
		assert.Equal(t, int64(1), apiPull.CloseDuplicateOf)
	})

	t.Run("Defaults", func(t *testing.T) { // AC6
		index := newIssue(t, "closed by an old script")
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", issuesURL, index), &api.EditIssueOption{State: &closed}).AddTokenAuth(token)
		assert.Equal(t, "completed", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "PATCH", pullsURL+"/5", &api.EditPullRequestOption{State: &closed}).AddTokenAuth(token)
		assert.Equal(t, "not_planned", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.PullRequest{}).CloseReason)
	})

	t.Run("CreateClosed", func(t *testing.T) { // AC5, AC6 on create
		req := NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "starts closed", Closed: true}).AddTokenAuth(token)
		assert.Equal(t, "completed", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "starts not planned", Closed: true, CloseReason: "not_planned"}).AddTokenAuth(token)
		assert.Equal(t, "not_planned", DecodeJSON(t, MakeRequest(t, req, http.StatusCreated), &api.Issue{}).CloseReason)

		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "rejected create", Closed: true, CloseReason: "other"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		req = NewRequestWithJSON(t, "POST", issuesURL, &api.CreateIssueOption{Title: "rejected create", CloseReason: "completed"}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
		unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Title: "rejected create"}) // nothing was created
	})

	t.Run("NoReasonReadsEmpty", func(t *testing.T) { // AC7
		for _, index := range []int64{1, 4} { // open, and closed before close reasons existed
			req := NewRequest(t, "GET", fmt.Sprintf("%s/%d", issuesURL, index)).AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Body.String(), `"close_reason":""`)
		}
	})
}
```

- [ ] **Step 11: Run the integration test**

Run: `go test -run '^TestAPICloseReason$' ./tests/integration/`
Expected: PASS, in about 2s once compiled. Also re-run the neighbours that close through the API:
`go test -run '^(TestAPIIssue|TestAPIEditPull)$' ./tests/integration/` — expect PASS.

---

### Task 6: Regenerate the API docs (AC9)

**Files:**
- Modify (generated): `templates/swagger/v1_json.tmpl`, `templates/swagger/v1_openapi3_json.tmpl`

- [ ] **Step 1: Regenerate the Swagger 2.0 spec**

Run (the Makefile's `generate-swagger` recipe, spelled out):

```bash
go run github.com/go-swagger/go-swagger/cmd/swagger@v0.35.0 generate spec --enable-allof-compounding --skip-enum-desc --exclude "gitea.dev/sdk" --input "templates/swagger/v1_input.json" --output './templates/swagger/v1_json.tmpl'
```

Expected: no output besides `go:` download lines (any other line is a warning the Makefile treats as failure — fix the comment it names).

- [ ] **Step 2: Regenerate the OpenAPI 3 spec**

Run: `go run build/generate-openapi.go`

- [ ] **Step 3: Check the diff**

Run: `git diff --stat templates/swagger/` and `git diff templates/swagger/v1_json.tmpl | grep '^+' | grep -c close_`
Expected: both files changed; `close_reason`, `close_reason_text` and `close_duplicate_of` appear under `CreateIssueOption`, `EditIssueOption`, `EditPullRequestOption`, `Issue` and `PullRequest`, and `issueEditIssue` gains a `422` response. Nothing unrelated changed.

- [ ] **Step 4: Commit (hand over to the human)**

```bash
git add modules/structs/issue.go modules/structs/pull.go services/convert/issue.go services/convert/pull.go services/convert/issue_test.go routers/api/v1/repo/issue_close_reason.go routers/api/v1/repo/issue_close_reason_test.go routers/api/v1/repo/issue.go routers/api/v1/repo/pull.go tests/integration/api_issue_close_reason_test.go templates/swagger/v1_json.tmpl templates/swagger/v1_openapi3_json.tmpl
git commit -m "feat(api): accept and return close reasons on issue and pull request endpoints (#55)" -m "Creating or editing an issue or pull request takes close_reason, close_reason_text and close_duplicate_of, checked before anything is written so a rejected request (422) changes nothing. Requests without them close as completed (issues) or not planned (pull requests). Responses carry the same three fields." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 7: Popup and suggestion icons show the reason (AC8)

**Files:**
- Modify: `web_src/js/types.ts` (`Issue`)
- Modify: `web_src/js/features/issue.ts`
- Modify: `web_src/js/svg.ts` (register two icons)
- Create: `web_src/js/features/issue.test.ts`
- Modify: `services/issue/suggestion.go`
- Modify: `templates/shared/issueicon.tmpl:1`, `templates/shared/close_reason_icon.tmpl` (the last comment paragraph)

- [ ] **Step 1: Write the failing test**

Create `web_src/js/features/issue.test.ts`:

```ts
import {getIssueColorClass, getIssueIcon} from './issue.ts';
import type {Issue} from '../types.ts';

function makeIssue(fields: Partial<Issue>): Issue {
  return {
    id: 1, number: 1, title: '', body: '', state: 'closed', close_reason: '', created_at: '', html_url: '',
    repository: {full_name: 'user2/repo1', html_url: ''}, labels: [], ...fields,
  };
}

test('closed issues by close reason, as close_reason_icon.tmpl draws them', () => {
  const looks = [
    ['completed', 'octicon-issue-closed', 'tw-text-purple'],
    ['not_planned', 'octicon-skip', 'tw-text-text-light'],
    ['duplicate', 'octicon-duplicate', 'tw-text-text-light'],
    ['other', 'octicon-note', 'tw-text-text-light'],
    ['', 'octicon-issue-closed', 'tw-text-red'], // closed before close reasons, keeps its old look
  ] as const;
  for (const [close_reason, icon, color] of looks) {
    const issue = makeIssue({close_reason});
    expect(getIssueIcon(issue)).toEqual(icon);
    expect(getIssueColorClass(issue)).toEqual(color);
  }
});

test('closed pull requests by close reason', () => {
  const pull = {draft: false, merged: false};
  expect(getIssueIcon(makeIssue({pull_request: pull, close_reason: 'not_planned'}))).toEqual('octicon-skip');
  expect(getIssueColorClass(makeIssue({pull_request: pull, close_reason: 'not_planned'}))).toEqual('tw-text-text-light');
  expect(getIssueIcon(makeIssue({pull_request: pull}))).toEqual('octicon-git-pull-request-closed');
  expect(getIssueColorClass(makeIssue({pull_request: pull}))).toEqual('tw-text-red');
});

test('merged and open items ignore close reasons', () => {
  const merged = makeIssue({pull_request: {draft: false, merged: true}});
  expect(getIssueIcon(merged)).toEqual('octicon-git-merge');
  expect(getIssueColorClass(merged)).toEqual('tw-text-purple');
  const open = makeIssue({state: 'open'});
  expect(getIssueIcon(open)).toEqual('octicon-issue-opened');
  expect(getIssueColorClass(open)).toEqual('tw-text-green');
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `pnpm exec vitest run web_src/js/features/issue.test.ts`
Expected: FAIL, e.g. `expected 'octicon-issue-closed' to equal 'octicon-skip'`.

- [ ] **Step 3: Add `close_reason` to the `Issue` type**

In `web_src/js/types.ts`, in `export type Issue`, after `state: 'open' | 'closed',` add:

```ts
  close_reason: '' | 'completed' | 'not_planned' | 'duplicate' | 'other', // '' when open or closed without a reason
```

- [ ] **Step 4: Register the two missing icons**

In `web_src/js/svg.ts`, add imports in alphabetical position among the others:

```ts
import octiconDuplicate from '../../public/assets/img/svg/octicon-duplicate.svg';
import octiconNote from '../../public/assets/img/svg/octicon-note.svg';
```

and entries in the `svgs` object, also alphabetical:

```ts
  'octicon-duplicate': octiconDuplicate,
  'octicon-note': octiconNote,
```

- [ ] **Step 5: Draw the reason in `issue.ts`**

Replace the whole of `web_src/js/features/issue.ts` with:

```ts
import type {Issue} from '../types.ts';
import type {SvgName} from '../svg.ts';

// the getIssueIcon/getIssueColorClass logic should be kept the same as "templates/shared/issueicon.tmpl",
// and closeReasonLooks the same as the "list" look in "templates/shared/close_reason_icon.tmpl"
const closeReasonLooks: Record<Exclude<Issue['close_reason'], ''>, {icon: SvgName, color: string}> = {
  completed: {icon: 'octicon-issue-closed', color: 'tw-text-purple'},
  not_planned: {icon: 'octicon-skip', color: 'tw-text-text-light'},
  duplicate: {icon: 'octicon-duplicate', color: 'tw-text-text-light'},
  other: {icon: 'octicon-note', color: 'tw-text-text-light'},
};

export function getIssueIcon(issue: Issue): SvgName {
  if (issue.pull_request) {
    if (issue.state === 'open') {
      if (issue.pull_request.draft) {
        return 'octicon-git-pull-request-draft'; // WIP PR
      }
      return 'octicon-git-pull-request'; // Open PR
    } else if (issue.pull_request.merged) {
      return 'octicon-git-merge'; // Merged PR
    }
    return issue.close_reason ? closeReasonLooks[issue.close_reason].icon : 'octicon-git-pull-request-closed'; // Closed PR
  }

  if (issue.state === 'open') {
    return 'octicon-issue-opened'; // Open Issue
  }
  return issue.close_reason ? closeReasonLooks[issue.close_reason].icon : 'octicon-issue-closed'; // Closed Issue
}

export function getIssueColorClass(issue: Issue) {
  if (issue.pull_request) {
    if (issue.state === 'open') {
      if (issue.pull_request.draft) {
        return 'tw-text-text-light'; // WIP PR
      }
      return 'tw-text-green'; // Open PR
    } else if (issue.pull_request.merged) {
      return 'tw-text-purple'; // Merged PR
    }
    return issue.close_reason ? closeReasonLooks[issue.close_reason].color : 'tw-text-red'; // Closed PR
  }

  if (issue.state === 'open') {
    return 'tw-text-green'; // Open Issue
  }
  return issue.close_reason ? closeReasonLooks[issue.close_reason].color : 'tw-text-red'; // Closed Issue
}
```

(The comments `// WIP PR` etc. are the originals; keep them.)

- [ ] **Step 6: Run the test and the type check**

Run: `pnpm exec vitest run web_src/js/features/issue.test.ts`
Expected: PASS.

Run: `pnpm exec vue-tsc`
Expected: no errors. If another test file builds an `Issue` object literal, it now needs `close_reason: ''`; add it there.

- [ ] **Step 7: Send `close_reason` with `#` suggestions**

`web_src/js/features/comp/TextExpander.ts` draws the `#` suggestion list with the same `getIssueIcon`, from `services/issue/suggestion.go`, which builds `structs.Issue` by hand. In `GetSuggestion`, in the `suggestion := &structs.Issue{...}` literal, add after `State: issue.State(),`:

```go
			CloseReason: issue.CloseReason.String(),
```

Run: `go build ./services/issue/ && gofmt -l services/issue/` — builds, no output.

- [ ] **Step 8: Update the template comments**

In `templates/shared/issueicon.tmpl`, line 1, replace:

```
{{/* the logic should be kept the same as getIssueIcon/getIssueColorClass in JS code, which doesn't show close reasons yet (see shared/close_reason_icon) */}}
```

with:

```
{{/* the logic should be kept the same as getIssueIcon/getIssueColorClass in JS code, whose closeReasonLooks mirrors shared/close_reason_icon */}}
```

In `templates/shared/close_reason_icon.tmpl`, replace the comment line:

```
The hover popup on issue links (getIssueIcon in web_src/js/features/issue.ts) doesn't show reasons yet: it reads the API.
```

with:

```
The hover popup and the # suggestions draw the "list" look in JS: keep closeReasonLooks in web_src/js/features/issue.ts in step.
```

- [ ] **Step 9: Lint**

Run: `pnpm exec eslint --color --max-warnings=0 web_src/js tools *.ts tests/e2e` then `pnpm exec vue-tsc` then `pnpm exec vitest run`
Expected: all pass.

Run: `node tools/lint-templates-svg.ts` — expect pass.

- [ ] **Step 10: Commit (hand over to the human)**

```bash
git add web_src/js/types.ts web_src/js/features/issue.ts web_src/js/features/issue.test.ts web_src/js/svg.ts services/issue/suggestion.go templates/shared/issueicon.tmpl templates/shared/close_reason_icon.tmpl
git commit -m "feat(issues): show close reasons in the issue hover popup (#55)" -m "getIssueIcon and getIssueColorClass now mirror close_reason_icon.tmpl, so the popup and the # suggestions show the same icon as issue lists." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 8: Keep the course context docs true

**Files:**
- Modify: `.claude-students/docs/services-issue.md` (the "A close that records why" bullet and the `status.go` table row)
- Modify: `.claude-students/docs/models-issues.md` (the `issue_close_reason.go` table row)
- Regenerate: `.claude-students/guide/services-issue.md`, `.claude-students/guide/models-issues.md`

- [ ] **Step 1: Update `docs/services-issue.md`**

Replace the table row:

```
| `services/issue/status.go` | `CloseIssueWithReason`, `CloseIssue` (the no-reason wrapper), `ReopenIssue` |
```

with:

```
| `services/issue/status.go` | `CloseIssueWithReason`, `CloseIssue` (the default-reason wrapper), `ReopenIssue` |
```

Replace the bullet:

```
- A close that records why goes through `CloseIssueWithReason`. `CloseIssue` passes an empty
  `issues_model.CloseReasonOptions`, for callers with no reason to give (commit keywords, the API);
  bulk close on the list passes the picked reason, from `issues_model.BulkCloseReasons`. The reason is
  validated in `models/issues/issue_update.go` → `SetIssueAsClosed`, inside the close transaction.
```

with:

```
- A close that records why goes through `CloseIssueWithReason`. `CloseIssue` records
  `issues_model.DefaultCloseReason` (completed for an issue, not planned for a pull request), for
  callers with no reason to give: commit keywords, merged-PR references, deleted branches, and API
  requests without `close_reason`. Bulk close on the list passes the picked reason. The reason is
  validated in `models/issues/issue_update.go` → `SetIssueAsClosed`, inside the close transaction;
  `routers/api/v1/repo/issue_close_reason.go` runs the same checks before an API request writes anything.
```

Set the frontmatter `verified-at:` to the output of `git rev-parse --short HEAD`.

- [ ] **Step 2: Update `docs/models-issues.md`**

In the `models/issues/issue_close_reason.go` row, change `` `CloseReasonOptions.Validate` and its three errors`` to `` `CloseReasonOptions.Validate` and its three errors, `CloseReasonOptions.DuplicateIssueID` (the duplicate lookup, shared with the API)``. Set `verified-at:` the same way.

- [ ] **Step 3: Regenerate the two guide twins**

For each of `services-issue` and `models-issues`: reread `docs/<name>.md`, rewrite the matching passages of `guide/<name>.md` in plain English (in `guide/services-issue.md` that is the `status.go` table row and the "Closing with a reason uses `CloseIssueWithReason`" paragraph; in `guide/models-issues.md`, wherever it lists what `issue_close_reason.go` holds). Add no fact the doc doesn't have, cite no path the doc doesn't cite. Then update its frontmatter:
- `source-hash:` ← `sha256sum .claude-students/docs/<name>.md | cut -c1-16`
- `verified-at:` ← `git rev-parse --short HEAD`

- [ ] **Step 4: Run the docs check**

Run: `bash ./.claude-students/check.sh`
Expected: exits 0 with no `drift` or rule errors.

- [ ] **Step 5: Commit (hand over to the human)**

```bash
git add .claude-students/docs/services-issue.md .claude-students/docs/models-issues.md .claude-students/guide/services-issue.md .claude-students/guide/models-issues.md
git commit -m "docs(issues): describe default close reasons and the API's close reason checks (#55)" -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 9: Final verification

- [ ] **Step 1: Go tests for every touched package**

Run: `go test ./models/issues/ ./services/issue/ ./services/pull/ ./services/convert/ ./routers/api/v1/repo/`
Expected: PASS.

- [ ] **Step 2: Integration tests**

Run: `go test -run '^(TestAPICloseReason|TestAPIIssue|TestAPIEditPull|TestAPIMergePull)$' ./tests/integration/`
Expected: PASS.

- [ ] **Step 3: Go lint and vet on the changed packages**

Run: `go vet ./models/issues/ ./services/issue/ ./services/pull/ ./services/convert/ ./routers/api/v1/repo/ ./modules/structs/` and `gofmt -l $(git diff --name-only main -- '*.go')`
Expected: no output from either.

- [ ] **Step 4: Frontend**

Run: `pnpm exec eslint --color --max-warnings=0 web_src/js tools *.ts tests/e2e`, `pnpm exec vue-tsc`, `pnpm exec vitest run`
Expected: all pass.

- [ ] **Step 5: Whitespace and docs**

Run: `git diff main --check` (expect no output) and `bash ./.claude-students/check.sh` (expect exit 0).

- [ ] **Step 6: Manual check of AC8 in the real app**

Run: `pnpm exec vite build`, `go build -o gitea.exe`, then `./gitea.exe web` and, in a browser at `http://localhost:3000`, close an issue as "Not planned" from the API (or the UI), then hover a `#N` reference to it in another issue's comment. Expected: the popup shows the grey skip icon. Close another as "Completed": purple closed icon.

- [ ] **Step 7: Hand-over reminders**

Remind the human to: run `specstory sync`, file the session log under `ai-logs/sprint<N>/vvchopra/` (copy and rename only), draft the PR in the course format (`Closes #55`, What changed, test strategy, Design pointing at the issue's Design Decision, AI assistance section with no log link), and post the AI Assistance comment on #55.
