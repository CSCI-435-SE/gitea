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
