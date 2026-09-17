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

func TestGetRepoIssueLifetimes(t *testing.T) {
	unittest.PrepareTestEnv(t)

	lifetimes, err := issues_model.GetRepoIssueLifetimes(t.Context(), 1)
	require.NoError(t, err)

	// repo 1 also holds pull requests, which must not reach an issues chart
	require.Len(t, lifetimes, 2)
	assert.EqualValues(t, 946684800, lifetimes[0].CreatedUnix)
	assert.False(t, lifetimes[0].IsClosed)
	assert.EqualValues(t, 946684840, lifetimes[1].CreatedUnix)
	assert.True(t, lifetimes[1].IsClosed)
}

func TestGetRepoIssueLifetimesNoIssues(t *testing.T) {
	unittest.PrepareTestEnv(t)

	lifetimes, err := issues_model.GetRepoIssueLifetimes(t.Context(), unittest.NonexistentID)
	require.NoError(t, err)
	assert.Empty(t, lifetimes)
}
