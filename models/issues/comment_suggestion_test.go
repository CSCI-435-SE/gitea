// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/optional"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentSuggestionOriginalLine(t *testing.T) {
	const header = "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,3 @@\n"
	cases := []struct {
		name     string
		typ      issues_model.CommentType
		line     int64
		patch    string
		expected optional.Option[string]
	}{
		{"added line keeps indentation", issues_model.CommentTypeCode, 3, header + " package main\n-\treturn nil\n+\treturn err", optional.Some("\treturn err")},
		{"context line", issues_model.CommentTypeCode, 2, header + " package main\n func main() {}", optional.Some("func main() {}")},
		{"trailing newline in patch", issues_model.CommentTypeCode, 2, header + " package main\n+x := 1\n", optional.Some("x := 1")},
		{"blank context line", issues_model.CommentTypeCode, 2, header + " package main\n ", optional.Some("")},
		{"CRLF kept", issues_model.CommentTypeCode, 1, header + "+foo\r", optional.Some("foo\r")},
		{"old side is not supported", issues_model.CommentTypeCode, -2, header + " package main\n-\treturn nil", optional.None[string]()},
		{"removed line on new side is inconsistent", issues_model.CommentTypeCode, 2, header + " package main\n-gone", optional.None[string]()},
		{"empty patch", issues_model.CommentTypeCode, 2, "", optional.None[string]()},
		{"header only", issues_model.CommentTypeCode, 2, header, optional.None[string]()},
		{"not a code comment", issues_model.CommentTypeComment, 2, header + "+x", optional.None[string]()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comment := &issues_model.Comment{Type: c.typ, Line: c.line, Patch: c.patch}
			assert.Equal(t, c.expected, comment.SuggestionOriginalLine())
		})
	}
}

func TestCommentSuggestionOriginalLineRange(t *testing.T) {
	const header = "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -16,6 +16,7 @@\n"
	// new-side lines 16..22; the removed line between 19 and 20 is not part of the range
	const patch = header + " l16\n l17\n l18\n+\tl19\n-old\n l20\n+l21\n l22"
	cases := []struct {
		name     string
		start    int64
		line     int64
		patch    string
		expected optional.Option[string]
	}{
		{"lines 18-22 in order, removed line skipped", 18, 22, patch, optional.Some("l18\n\tl19\nl20\nl21\nl22")},
		{"range reaching the hunk start", 16, 22, patch, optional.Some("l16\nl17\nl18\n\tl19\nl20\nl21\nl22")},
		{"range longer than the stored hunk", 15, 22, patch, optional.None[string]()},
		{"no newline at end of file marker", 21, 22, patch + "\n\\ No newline at end of file", optional.Some("l21\nl22")},
		{"empty patch", 18, 22, "", optional.None[string]()},
		{"start after end", 23, 22, patch, optional.None[string]()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comment := &issues_model.Comment{
				Type: issues_model.CommentTypeCode, Line: c.line, Patch: c.patch,
				CommentMetaData: &issues_model.CommentMetaData{StartLine: c.start},
			}
			assert.Equal(t, c.expected, comment.SuggestionOriginalLine())
		})
	}
}

func TestCommentSuggestionContextLines(t *testing.T) {
	const header = "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,4 @@\n"
	cases := []struct {
		name     string
		line     int64
		patch    string
		expected []string
	}{
		{"context and added lines above, in order", 4, header + " package main\n+import \"fmt\"\n \n+func main() {}", []string{"package main", "import \"fmt\"", ""}},
		{"removed lines are the old side", 3, header + " package main\n-\treturn nil\n+\treturn err\n+}", []string{"package main", "\treturn err"}},
		{"no newline marker is skipped", 2, header + " a\n\\ No newline at end of file\n+b", []string{"a"}},
		{"first line of the hunk has nothing above", 1, header + "+package main", nil},
		{"CRLF kept", 2, header + " a\r\n+b\r", []string{"a\r"}},
		{"old side is not supported", -2, header + " package main\n-\treturn nil", nil},
		{"no hunk header", 2, " a\n+b", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comment := &issues_model.Comment{Type: issues_model.CommentTypeCode, Line: c.line, Patch: c.patch}
			assert.Equal(t, c.expected, comment.SuggestionContextLines())
		})
	}
}

func TestCommentSuggestionBlocks(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected []string
	}{
		{"no block", "looks good", nil},
		{"one block", "try this:\n```suggestion\n\treturn nil\n```", []string{"\treturn nil\n"}},
		{"empty block deletes the line", "```suggestion\n```", []string{""}},
		{"one blank line", "```suggestion\n\n```", []string{"\n"}},
		{"several lines", "```suggestion\na\nb\n```", []string{"a\nb\n"}},
		{"two blocks", "```suggestion\na\n```\nor\n```suggestion\nb\n```", []string{"a\n", "b\n"}},
		{"tilde fence", "~~~suggestion\nx\n~~~", []string{"x\n"}},
		{"longer fence holds a shorter one", "````suggestion\n```\nx\n````", []string{"```\nx\n"}},
		{"only the first info word counts", "```suggestion title\nx\n```", []string{"x\n"}},
		{"capitalised Suggestion is an ordinary block", "```Suggestion\nx\n```", nil},
		{"other languages are left alone", "```go\nx\n```", nil},
		{"inline code is not a block", "use `suggestion` blocks", nil},
		{"inside a list item", "- item\n\n  ```suggestion\n  x\n  ```", []string{"x\n"}},
		{"inside a quote", "> ```suggestion\n> x\n> ```", []string{"x\n"}},
		{"CRLF is normalised", "```suggestion\r\nx\r\n```", []string{"x\n"}},
		{"an unclosed fence runs to the end", "```suggestion\nx", []string{"x\n"}}, // the renderer also ends the block with "\n"
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comment := &issues_model.Comment{Type: issues_model.CommentTypeCode, Content: c.content}
			assert.Equal(t, c.expected, comment.SuggestionBlocks(t.Context()))
		})
	}
}

func TestCommentSuggestionCommitMessage(t *testing.T) {
	ana := &user_model.User{Name: "ana"}
	// #156's user scenario
	comment := &issues_model.Comment{Poster: ana, TreePath: "main.go"}
	assert.Equal(t, "Apply suggestion from @ana on main.go", comment.SuggestionCommitMessage())
	// the file is named by its full path, as decided while implementing #156
	comment = &issues_model.Comment{Poster: ana, TreePath: "cmd/web/main.go"}
	assert.Equal(t, "Apply suggestion from @ana on cmd/web/main.go", comment.SuggestionCommitMessage())
}

func TestMarkSuggestionApplied(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	applier := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	const sha = "65f1bf27bc3bf70f64657658635e66094edbcb4d"

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4})
	assert.False(t, comment.SuggestionApplied()) // #157 AC3 an existing comment isn't applied

	marked, err := issues_model.MarkSuggestionApplied(t.Context(), comment, applier, sha)
	require.NoError(t, err)
	assert.True(t, marked)
	assert.True(t, comment.SuggestionApplied())

	stored := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4})
	assert.True(t, stored.SuggestionApplied())
	assert.Equal(t, applier.ID, stored.SuggestionAppliedByID)
	assert.Equal(t, sha, stored.SuggestionAppliedCommit)
	assert.NotZero(t, stored.SuggestionAppliedUnix)
	assert.Equal(t, "meh...", stored.Content) // only the applied columns change

	// #157 AC10 a comment records only the first commit that applied it
	marked, err = issues_model.MarkSuggestionApplied(t.Context(), stored, &user_model.User{ID: 1}, "1111111111111111111111111111111111111111")
	require.NoError(t, err)
	assert.False(t, marked)
	stored = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4})
	assert.Equal(t, applier.ID, stored.SuggestionAppliedByID)
	assert.Equal(t, sha, stored.SuggestionAppliedCommit)
}

func TestCommentListLoadSuggestionAppliedBys(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	const sha = "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	notApplied := &issues_model.Comment{}
	applied := &issues_model.Comment{SuggestionAppliedByID: 2, SuggestionAppliedCommit: sha}
	deleted := &issues_model.Comment{SuggestionAppliedByID: 1000, SuggestionAppliedCommit: sha} // no user 1000

	require.NoError(t, issues_model.CommentList{notApplied, applied, deleted}.LoadSuggestionAppliedBys(t.Context()))
	assert.Nil(t, notApplied.SuggestionAppliedBy) // not applied: nobody to load
	assert.Equal(t, "user2", applied.SuggestionAppliedBy.Name)
	assert.True(t, deleted.SuggestionAppliedBy.IsGhost()) // #157 AC8 a deleted applier shows as Ghost
}

func TestFetchCodeCommentsLoadsSuggestionAppliedBy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4}) // README.md line 4
	_, err := issues_model.MarkSuggestionApplied(t.Context(), comment, &user_model.User{ID: 2}, "65f1bf27bc3bf70f64657658635e66094edbcb4d")
	require.NoError(t, err)

	comments, err := issues_model.FetchCodeCommentsByLine(t.Context(), issue, user, "README.md", 4, true)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	// #157 AC1 the marker names who applied it on every page that shows code comments
	require.NotNil(t, comments[0].SuggestionAppliedBy)
	assert.Equal(t, "user2", comments[0].SuggestionAppliedBy.Name)
}
