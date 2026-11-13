// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown_test

import (
	"testing"

	"gitea.dev/modules/markup/markdown"

	"github.com/stretchr/testify/assert"
)

// Expected values follow CommonMark's fenced and indented code block rules (spec sections 4.4 and 4.5).
// The suggestion-specific cases are in models/issues TestCommentSuggestionBlocks.
func TestFencedCodeBlocks(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		lang     string
		expected []string
	}{
		{"empty source", "", "go", nil},
		{"only the language asked for", "```go\nx := 1\n```\n```js\ny\n```", "go", []string{"x := 1\n"}},
		{"blocks in document order", "```go\na\n```\ntext\n```go\nb\n```", "go", []string{"a\n", "b\n"}},
		{"an indented code block has no language", "    x := 1\n", "go", nil},
		{"two backticks are not a fence", "``go\nx\n``", "go", nil},
		{"a fence inside a longer fence is content", "````md\n```go\nx\n```\n````", "go", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, markdown.FencedCodeBlocks(t.Context(), []byte(c.source), c.lang))
		})
	}
}
