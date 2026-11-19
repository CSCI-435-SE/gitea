// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/gitrepo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// PR 2 is repo1 branch2 against master: README.md new lines 1-6 form one hunk
func TestCreateCodeCommentRange(t *testing.T) {
	unittest.PrepareTestEnv(t)
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadIssue(t.Context()))
	require.NoError(t, pr.Issue.LoadRepo(t.Context()))
	require.NoError(t, pr.Issue.LoadPoster(t.Context()))
	gitRepo, err := gitrepo.OpenRepository(t.Context(), pr.Issue.Repo)
	require.NoError(t, err)
	defer gitRepo.Close()
	create := func(line, startLine, replyReviewID int64) (*issues_model.Comment, error) {
		return CreateCodeComment(t.Context(), pr.Issue.Poster, gitRepo, pr.Issue, line, startLine, "range", "README.md", false, replyReviewID, pr.HeadCommitID, nil)
	}
	countCodeComments := func() int {
		return unittest.GetCount(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode})
	}

	// #158 AC1 AC4: lines 3-5 store their start and the text the reviewer saw
	comment, err := create(5, 3, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 3, comment.RangeStartLine())
	lines, ok := comment.RangeOriginalLines()
	assert.True(t, ok)
	assert.Equal(t, []string{"Description for repo1", "", "And change for branch2"}, lines)

	t.Run("AC12 invalid ranges are refused", func(t *testing.T) {
		before, reviewsBefore := countCodeComments(), unittest.GetCount(t, &issues_model.Review{IssueID: pr.IssueID})
		for _, c := range []struct {
			name        string
			line, start int64
		}{
			{"start equals end", 4, 4},
			{"start after end", 3, 4},
			{"old side", -3, 1},
			{"negative start", 4, -1},
		} {
			_, err := create(c.line, c.start, 0)
			assert.ErrorIs(t, err, ErrCodeCommentRangeInvalid, c.name)
		}
		assert.Equal(t, reviewsBefore, unittest.GetCount(t, &issues_model.Review{IssueID: pr.IssueID}), "no empty pending review left behind")
		// file 3 is deleted on branch2, so its diff has no new-side lines
		_, err := CreateCodeComment(t.Context(), pr.Issue.Poster, gitRepo, pr.Issue, 2, 1, "range", "3", false, 0, pr.HeadCommitID, nil)
		assert.ErrorIs(t, err, ErrCodeCommentRangeInvalid, "no hunk on the new side")
		assert.Equal(t, before, countCodeComments())
	})

	t.Run("a reply joins its thread's range", func(t *testing.T) {
		_, err := create(5, 4, comment.ReviewID)
		assert.ErrorIs(t, err, ErrCodeCommentRangeInvalid) // another start on the same thread
		reply, err := create(5, 0, comment.ReviewID)
		require.NoError(t, err)
		assert.EqualValues(t, 3, reply.RangeStartLine())
	})

	invalidated := func(t *testing.T, branch string) bool {
		require.NoError(t, InvalidateCodeComments(t.Context(), issues_model.PullRequestList{pr}, pr.Issue.Poster, pr.Issue.Repo, gitRepo, branch))
		return unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID}).Invalidated
	}
	// #158 AC8: the branch head still has the range's text
	assert.False(t, invalidated(t, "branch2"))
	// #158 AC7: master's README.md ends at line 3, so lines 4-5 of the range are gone
	assert.True(t, invalidated(t, "master"))
}
