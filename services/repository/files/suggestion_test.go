// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplySuggestionToContent(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		line       int
		original   string
		suggestion string
		expected   string
		err        error
	}{
		// AC1, AC12: only the commented line changes, as #154's diff shows it
		{"AC1 replaces only the commented line", "a\nb\nc\n", 2, "b", "B\n", "a\nB\nc\n", nil},
		{"AC1 first line", "a\nb\n", 1, "a", "A\n", "A\nb\n", nil},
		{"AC12 empty block deletes the line", "a\nb\nc\n", 2, "b", "", "a\nc\n", nil},
		{"AC12 one blank line replaces it with a blank line", "a\nb\nc\n", 2, "b", "\n", "a\n\nc\n", nil},
		{"AC12 several lines replace it in order", "a\nb\nc\n", 2, "b", "x\ny\n", "a\nx\ny\nc\n", nil},

		// AC11: line endings and the final newline are kept
		{"AC11 CRLF file keeps CRLF", "a\r\nb\r\nc\r\n", 2, "b\r", "B\n", "a\r\nB\r\nc\r\n", nil},
		{"AC11 several lines in a CRLF file all end in CRLF", "a\r\nb\r\nc\r\n", 2, "b\r", "x\ny\n", "a\r\nx\r\ny\r\nc\r\n", nil},
		{"AC11 last line without a final newline stays without one", "a\nb", 2, "b", "B\n", "a\nB", nil},
		{"AC11 several lines on a last line without a final newline", "a\nb", 2, "b", "x\ny\n", "a\nx\ny", nil},
		{"AC11 several lines on a CRLF last line without a final newline", "a\r\nb", 2, "b", "x\ny\n", "a\r\nx\r\ny", nil},
		{"AC11 deleting a last line without a final newline", "a\nb", 2, "b", "", "a", nil},
		{"AC11 deleting a last line that has a final newline", "a\nb\n", 2, "b", "", "a\n", nil},

		// AC3: the line no longer reads as the suggestion expects
		{"AC3 line changed since the review", "a\nX\nc\n", 2, "b", "B\n", "", ErrSuggestionOutdated},
		{"AC3 trailing whitespace counts as a change", "a\nb \nc\n", 2, "b", "B\n", "", ErrSuggestionOutdated},
		{"AC3 line number past the end of the file", "a\nb\n", 3, "c", "C\n", "", ErrSuggestionOutdated},
		{"AC3 empty file", "", 1, "a", "A\n", "", ErrSuggestionOutdated},

		// AC10: a suggestion identical to the line commits nothing
		{"AC10 same text makes no change", "a\nb\n", 2, "b", "b\n", "", ErrSuggestionNoChange},
		{"AC10 same text in a CRLF file makes no change", "a\r\nb\r\n", 2, "b\r", "b\n", "", ErrSuggestionNoChange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := applySuggestionToContent(c.content, c.line, c.original, c.suggestion)
			assert.ErrorIs(t, err, c.err)
			assert.Equal(t, c.expected, result)
		})
	}
}
