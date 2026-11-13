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
