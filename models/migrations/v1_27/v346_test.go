// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"testing"

	"gitea.dev/models/migrations/migrationtest"

	_ "gitea.dev/models/issues" // registers the comment table; fixtures for unregistered tables are skipped

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commentBeforeSuggestionApplied struct {
	ID          int64 `xorm:"pk autoincr"`
	Type        int
	PosterID    int64
	IssueID     int64
	TreePath    string
	Line        int64
	Content     string `xorm:"LONGTEXT"`
	Invalidated bool
	CreatedUnix int64
	UpdatedUnix int64
}

func (commentBeforeSuggestionApplied) TableName() string { return "comment" }

type commentSuggestionApplied struct {
	SuggestionAppliedByID   int64
	SuggestionAppliedCommit string
	SuggestionAppliedUnix   int64
}

func Test_AddSuggestionAppliedToComment(t *testing.T) {
	x, deferable := migrationtest.PrepareTestEnv(t, 0, new(commentBeforeSuggestionApplied))
	defer deferable()

	var before []commentBeforeSuggestionApplied
	require.NoError(t, x.Table("comment").Asc("id").Find(&before))
	require.Len(t, before, 3) // guards against the fixtures silently not loading

	require.NoError(t, AddSuggestionAppliedToComment(x))

	var after []commentBeforeSuggestionApplied
	require.NoError(t, x.Table("comment").Asc("id").Find(&after))
	assert.Equal(t, before, after) // #157 AC3 existing comments render as before

	var applied []commentSuggestionApplied
	require.NoError(t, x.Table("comment").Asc("id").Cols("suggestion_applied_by_id", "suggestion_applied_commit", "suggestion_applied_unix").Find(&applied))
	assert.Equal(t, []commentSuggestionApplied{{}, {}, {}}, applied) // #157 AC3 none is marked applied
}
