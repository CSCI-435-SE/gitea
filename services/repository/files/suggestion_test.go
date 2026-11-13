// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"testing"

	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// The checks before the commit (#156), against fixture pull request 2: user2/repo1, head branch2, whose README.md
// has "And change for branch2" on line 5. The commit itself needs git hooks, so it is in the integration test.
func TestApplySuggestion(t *testing.T) {
	unittest.PrepareTestEnv(t)
	const branch2Head = "985f0301dba5e7b34be866819cd15ad3d8f508ee" // tests/gitea-repositories-meta/user2/repo1.git
	const masterHead = "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})  // owns repo1
	reader := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // cannot push to repo1
	loadPR := func(t *testing.T, id int64) *issues_model.PullRequest {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: id})
		require.NoError(t, pr.LoadIssue(t.Context()))
		return pr
	}
	suggest := func(pr *issues_model.PullRequest, treePath, original, content string) *issues_model.Comment {
		return &issues_model.Comment{
			Type: issues_model.CommentTypeCode, IssueID: pr.IssueID, PosterID: 4, TreePath: treePath, Line: 5,
			Patch: "@@ -4,1 +4,2 @@\n \n+" + original, Content: content,
		}
	}
	const fix = "```suggestion\nAnd a change for branch2\n```"

	t.Run("AC1 someone who can push gets the head branch's commit", func(t *testing.T) {
		head, err := CanApplySuggestions(t.Context(), owner, loadPR(t, 2))
		require.NoError(t, err)
		assert.Equal(t, branch2Head, head)
	})

	cannotApply := []struct {
		name   string
		doer   *user_model.User
		change func(pr *issues_model.PullRequest)
		err    error
	}{
		{"AC2 signed out", nil, nil, ErrSuggestionPermission},
		{"AC2 cannot push", reader, nil, ErrSuggestionPermission},
		{"AC5 closed", owner, func(pr *issues_model.PullRequest) { pr.Issue.IsClosed = true }, ErrSuggestionPullClosed},
		{"AC5 merged", owner, func(pr *issues_model.PullRequest) { pr.HasMerged = true }, ErrSuggestionPullClosed},
		{"AC5 AGit flow", owner, func(pr *issues_model.PullRequest) { pr.Flow = issues_model.PullRequestFlowAGit }, ErrSuggestionPullClosed},
		{"AC5 head branch deleted", owner, func(pr *issues_model.PullRequest) { pr.HeadBranch = "deleted-branch" }, ErrSuggestionPullClosed},
	}
	for _, c := range cannotApply {
		t.Run(c.name, func(t *testing.T) {
			pr := loadPR(t, 2)
			if c.change != nil {
				c.change(pr)
			}
			_, err := CanApplySuggestions(t.Context(), c.doer, pr)
			assert.ErrorIs(t, err, c.err)
			// ApplySuggestion makes the same check before anything else that touches the branch
			assert.ErrorIs(t, ApplySuggestion(t.Context(), c.doer, pr, suggest(pr, "README.md", "And change for branch2", fix), branch2Head, ""), c.err)
		})
	}

	refused := []struct {
		name    string
		comment func(pr *issues_model.PullRequest) *issues_model.Comment
		head    string
		err     error
	}{
		{"AC7 a comment from another pull request", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.IssueID = 2
			return c
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 an old-side comment", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.Line = -5
			return c
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 no suggestion block", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", "looks good")
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 two suggestion blocks", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", fix+"\nor\n"+fix)
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC4 the page showed an older head", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", fix)
		}, masterHead, ErrSuggestionHeadMoved},
		{"AC3 the file was deleted or renamed", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "gone.md", "And change for branch2", fix)
		}, branch2Head, ErrSuggestionOutdated},
		{"AC3 the line changed", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And an older change for branch2", fix)
		}, branch2Head, ErrSuggestionOutdated},
		{"AC10 the suggestion is the line as it is", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", "```suggestion\nAnd change for branch2\n```")
		}, branch2Head, ErrSuggestionNoChange},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			pr := loadPR(t, 2)
			assert.ErrorIs(t, ApplySuggestion(t.Context(), owner, pr, c.comment(pr), c.head, ""), c.err)
		})
	}

	t.Run("AC6 a protected branch", func(t *testing.T) {
		rule := &git_model.ProtectedBranch{RepoID: repo.ID, RuleName: "branch2", CanPush: false}
		require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, rule, git_model.WhitelistOptions{}))
		defer func() { require.NoError(t, git_model.DeleteProtectedBranch(t.Context(), repo, rule.ID)) }()

		pr := loadPR(t, 2)
		err := ApplySuggestion(t.Context(), owner, pr, suggest(pr, "README.md", "And change for branch2", fix), branch2Head, "")
		var errProtected ErrSuggestionProtectedBranch
		require.ErrorAs(t, err, &errProtected)
		assert.Equal(t, "branch2", errProtected.Branch)

		// like the web editor, a file matching the rule's unprotected patterns may still be changed,
		// so this one gets past protection and stops at its outdated line instead
		rule.UnprotectedFilePatterns = "README.md"
		require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, rule, git_model.WhitelistOptions{}))
		err = ApplySuggestion(t.Context(), owner, pr, suggest(pr, "README.md", "And an older change for branch2", fix), branch2Head, "")
		assert.ErrorIs(t, err, ErrSuggestionOutdated)
	})
}
