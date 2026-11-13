// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/optional"

	"github.com/stretchr/testify/assert"
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
