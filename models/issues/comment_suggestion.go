// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"strings"

	"gitea.dev/modules/optional"
)

// SuggestionOriginalLine returns the commented line as the reviewer saw it: the last line of the
// stored patch, since CutDiffAroundLine stops right after the commented line.
// For a multi-line comment it returns every line of the range, joined by "\n".
// Old-side comments return None until #151 decides whether they may carry suggestions.
func (c *Comment) SuggestionOriginalLine() optional.Option[string] {
	if c.Type != CommentTypeCode || c.Line <= 0 {
		return optional.None[string]()
	}
	if c.RangeStartLine() > 0 {
		lines, ok := c.RangeOriginalLines()
		if !ok {
			return optional.None[string]()
		}
		return optional.Some(strings.Join(lines, "\n"))
	}
	patch := strings.TrimRight(c.Patch, "\n")
	last := patch[strings.LastIndexByte(patch, '\n')+1:]
	if last == "" || (last[0] != '+' && last[0] != ' ') {
		return optional.None[string]() // "-" is the other side; "@", "d", "\\" mean no code line
	}
	return optional.Some(last[1:])
}

// RangeOriginalLines returns lines RangeStartLine..Line of a multi-line code comment as the
// reviewer saw them, false if the stored patch doesn't hold all of them.
func (c *Comment) RangeOriginalLines() ([]string, bool) {
	start := c.RangeStartLine()
	if start <= 0 || c.Line < start {
		return nil, false
	}
	return PatchLastNewLines(c.Patch, int(c.Line-start+1))
}

// PatchLastNewLines returns the last n new-side lines of a one-hunk patch, which end at the
// commented line, skipping removed lines. False if the hunk has fewer than n.
func PatchLastNewLines(patch string, n int) ([]string, bool) {
	rows := strings.Split(strings.TrimRight(patch, "\n"), "\n")
	lines := make([]string, n)
	for i := len(rows) - 1; i >= 0 && n > 0; i-- {
		switch {
		case rows[i] == "" || rows[i][0] == '@':
			return nil, false // reached the hunk header
		case rows[i][0] == '+' || rows[i][0] == ' ':
			n--
			lines[n] = rows[i][1:]
		}
	}
	return lines, n == 0
}
