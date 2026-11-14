// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRangeChanged(t *testing.T) {
	original := []string{"l3", "l4", "l5"} // a comment on lines 3-5
	cases := []struct {
		name    string
		content string
		start   int64
		changed bool
	}{
		{"unchanged", "l1\nl2\nl3\nl4\nl5\nl6\n", 3, false},
		{"first line changed", "l1\nl2\nX\nl4\nl5\nl6\n", 3, true},
		{"middle line changed", "l1\nl2\nl3\nX\nl5\nl6\n", 3, true},
		{"last line changed", "l1\nl2\nl3\nl4\nX\nl6\n", 3, true},
		{"only a line after the range changed", "l1\nl2\nl3\nl4\nl5\nX\n", 3, false},
		{"range ends on the last line, no final newline", "l1\nl2\nl3\nl4\nl5", 3, false},
		{"last line deleted", "l1\nl2\nl3\nl4\n", 3, true},
		{"file deleted", "", 3, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.changed, rangeChanged(original, c.content, c.start))
		})
	}
}
