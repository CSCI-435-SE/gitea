// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"
	"strings"

	"gitea.dev/models/db"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/container"
	"gitea.dev/modules/markup/markdown"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/timeutil"
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

// SuggestionContextLines returns the new-side lines the stored patch shows above the commented line, in file
// order. Applying a suggestion checks them too, since blame-based invalidation can miss the commented line
// moving onto an identical one, like another "}" (#156 AC3).
func (c *Comment) SuggestionContextLines() []string {
	if !c.SuggestionOriginalLine().Has() {
		return nil
	}
	lines := strings.Split(strings.TrimRight(c.Patch, "\n"), "\n")
	hunkStart := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "@@") {
			hunkStart = i
		}
	}
	if hunkStart < 0 {
		return nil
	}
	var above []string
	for _, line := range lines[hunkStart+1 : len(lines)-1] { // the last line is the commented one
		if line != "" && (line[0] == ' ' || line[0] == '+') {
			above = append(above, line[1:])
		} // "-" is the old side, "\" is "No newline at end of file"
	}
	return above
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

// SuggestionApplied reports whether the comment's suggestion was committed to the PR branch (#157)
func (c *Comment) SuggestionApplied() bool {
	return c.SuggestionAppliedCommit != ""
}

// MarkSuggestionApplied records that doer committed the comment's suggestion as commitSHA (#157). It returns
// false, changing nothing, when the comment was already marked: only the first commit is recorded (AC10).
func MarkSuggestionApplied(ctx context.Context, c *Comment, doer *user_model.User, commitSHA string) (bool, error) {
	now := timeutil.TimeStampNow()
	n, err := db.GetEngine(ctx).ID(c.ID).Where("suggestion_applied_commit = ''").
		Cols("suggestion_applied_by_id", "suggestion_applied_commit", "suggestion_applied_unix").NoAutoTime().
		Update(&Comment{SuggestionAppliedByID: doer.ID, SuggestionAppliedCommit: commitSHA, SuggestionAppliedUnix: now})
	if err != nil || n == 0 {
		return false, err
	}
	c.SuggestionAppliedByID, c.SuggestionAppliedBy, c.SuggestionAppliedCommit, c.SuggestionAppliedUnix = doer.ID, doer, commitSHA, now
	return true, nil
}

// LoadSuggestionAppliedBys loads who applied each comment's suggestion: the Ghost user once their account is
// deleted (#157 AC8)
func (comments CommentList) LoadSuggestionAppliedBys(ctx context.Context) error {
	ids := container.FilterSlice(comments, func(c *Comment) (int64, bool) {
		return c.SuggestionAppliedByID, c.SuggestionApplied() && c.SuggestionAppliedBy == nil
	})
	if len(ids) == 0 {
		return nil
	}
	users, err := user_model.GetUsersMapByIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, c := range comments {
		if c.SuggestionApplied() && c.SuggestionAppliedBy == nil {
			c.SuggestionAppliedBy = user_model.GetPossibleUserFromMap(c.SuggestionAppliedByID, users)
		}
	}
	return nil
}
