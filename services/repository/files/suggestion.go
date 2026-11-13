// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"errors"
	"strings"
)

var (
	// ErrSuggestionOutdated means the commented line no longer reads as it did when the suggestion was written (#156 AC3)
	ErrSuggestionOutdated = errors.New("the suggestion is outdated")
	// ErrSuggestionNoChange means applying the suggestion would leave the file as it is (#156 AC10)
	ErrSuggestionNoChange = errors.New("the suggestion makes no changes")
)

// applySuggestionToContent replaces line lineNum (1-based) of content with the suggestion's lines, the way
// #154 draws the diff: "" deletes the line, "\n" leaves one blank line, otherwise one line per suggestion line.
// original is the line the suggestion was written against; content and original may use CRLF.
func applySuggestionToContent(content string, lineNum int, original, suggestion string) (string, error) {
	lines := strings.SplitAfter(content, "\n") // every element but the last keeps its "\n"
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1] // content ended with a newline
	}
	idx := lineNum - 1
	if idx < 0 || idx >= len(lines) {
		return "", ErrSuggestionOutdated
	}

	target := lines[idx]
	ending := ""
	switch {
	case strings.HasSuffix(target, "\r\n"):
		ending = "\r\n"
	case strings.HasSuffix(target, "\n"):
		ending = "\n"
	}
	if strings.TrimSuffix(target, ending) != strings.TrimSuffix(original, "\r") {
		return "", ErrSuggestionOutdated
	}

	var replacement string
	if suggestion != "" {
		innerEnding := ending
		if innerEnding == "" { // the last line has no newline, so new lines in between take the file's own
			innerEnding = "\n"
			if strings.Contains(content, "\r\n") {
				innerEnding = "\r\n"
			}
		}
		newLines := strings.Split(strings.TrimSuffix(suggestion, "\n"), "\n")
		replacement = strings.Join(newLines, innerEnding) + ending
	}

	before := strings.Join(lines[:idx], "")
	if replacement == "" && ending == "" && idx > 0 {
		// deleting a last line that has no newline: the line before it becomes last and loses its newline,
		// so the file still ends without one (#156 AC11)
		prev := lines[idx-1]
		before = strings.Join(lines[:idx-1], "") + strings.TrimSuffix(strings.TrimSuffix(prev, "\n"), "\r")
	}
	result := before + replacement + strings.Join(lines[idx+1:], "")
	if result == content {
		return "", ErrSuggestionNoChange
	}
	return result, nil
}
