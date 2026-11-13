// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"
	"strings"

	"gitea.dev/modules/markup/markdown"
	"gitea.dev/modules/optional"
)

// SuggestionOriginalLine returns the commented line as the reviewer saw it: the last line of the
// stored patch, since CutDiffAroundLine stops right after the commented line.
// Old-side comments return None until #151 decides whether they may carry suggestions.
func (c *Comment) SuggestionOriginalLine() optional.Option[string] {
	if c.Type != CommentTypeCode || c.Line <= 0 {
		return optional.None[string]()
	}
	patch := strings.TrimRight(c.Patch, "\n")
	last := patch[strings.LastIndexByte(patch, '\n')+1:]
	if last == "" || (last[0] != '+' && last[0] != ' ') {
		return optional.None[string]() // "-" is the other side; "@", "d", "\\" mean no code line
	}
	return optional.Some(last[1:])
}

// SuggestionBlocks returns the text of each ```suggestion block in the comment, found the way the
// rendered page finds them (#154 AC9). An empty block is "", one blank line is "\n".
func (c *Comment) SuggestionBlocks(ctx context.Context) []string {
	return markdown.FencedCodeBlocks(ctx, []byte(c.Content), "suggestion")
}

// SuggestionCommitMessage is the default message for committing the comment's suggestion (#156 AC1).
// The poster must be loaded.
func (c *Comment) SuggestionCommitMessage() string {
	return fmt.Sprintf("Apply suggestion from @%s on %s", c.Poster.Name, c.TreePath)
}
