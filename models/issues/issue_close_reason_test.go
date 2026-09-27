// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueCloseReasonPersists(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	other := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issues_model.CloseReasonNone, other.CloseReason) // fixtures set no reason, so this is the column default
	assert.Empty(t, other.CloseReasonText)
	assert.Zero(t, other.CloseDuplicateIssueID)

	other.CloseReason = issues_model.CloseReasonOther
	other.CloseReasonText = "superseded by a new design"
	require.NoError(t, issues_model.UpdateIssueCols(t.Context(), other, "close_reason", "close_reason_text"))

	duplicate := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	duplicate.CloseReason = issues_model.CloseReasonDuplicate
	duplicate.CloseDuplicateIssueID = 1
	require.NoError(t, issues_model.UpdateIssueCols(t.Context(), duplicate, "close_reason", "close_duplicate_issue_id"))

	// reload and compare: bean conditions skip unmapped and zero-valued fields, so they can't catch a missing column
	other = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issues_model.CloseReasonOther, other.CloseReason)
	assert.Equal(t, "superseded by a new design", other.CloseReasonText)

	duplicate = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	assert.Equal(t, issues_model.CloseReasonDuplicate, duplicate.CloseReason)
	assert.EqualValues(t, 1, duplicate.CloseDuplicateIssueID)
}
