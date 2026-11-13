// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gitea.dev/models/db"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/test"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/contexttest"
	"gitea.dev/services/forms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Apply POST's responses (#156), against fixture pull request 2: user2/repo1 #3, head branch2, whose
// README.md has "And change for branch2" on line 5. The mock locale returns message keys, not their text.
func TestApplySuggestion(t *testing.T) {
	unittest.PrepareTestEnv(t)
	const branch2Head = "985f0301dba5e7b34be866819cd15ad3d8f508ee" // tests/gitea-repositories-meta/user2/repo1.git
	const masterHead = "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	comment := func(t *testing.T, original, content string) int64 {
		c := &issues_model.Comment{
			Type: issues_model.CommentTypeCode, IssueID: 3, PosterID: 4, TreePath: "README.md", Line: 5,
			Patch: "@@ -4,1 +4,2 @@\n \n+" + original, Content: content,
		}
		require.NoError(t, db.Insert(t.Context(), c))
		return c.ID
	}
	fix := comment(t, "And change for branch2", "```suggestion\nAnd a change for branch2\n```")

	apply := func(t *testing.T, doerID, index, commentID int64, head string) (*context.Context, *httptest.ResponseRecorder) {
		ctx, resp := contexttest.MockContext(t, "user2/repo1/pulls/3/files/reviews/comments/apply")
		contexttest.LoadUser(t, ctx, doerID)
		contexttest.LoadRepo(t, ctx, repo.ID)
		ctx.SetPathParam("index", strconv.FormatInt(index, 10))
		ctx.SetPathParam("id", strconv.FormatInt(commentID, 10))
		web.SetForm(ctx, &forms.ApplySuggestionForm{HeadCommitID: head})
		ApplySuggestion(ctx)
		return ctx, resp
	}
	refused := func(t *testing.T, resp *httptest.ResponseRecorder, ctx *context.Context, message string) {
		assert.Equal(t, http.StatusBadRequest, ctx.Resp.WrittenStatus())
		assert.Equal(t, message, test.ParseJSONError(resp.Body.Bytes()).ErrorMessage)
	}

	t.Run("AC1 the page gets the head commit and the apply link for someone who can push", func(t *testing.T) {
		ctx, _ := contexttest.MockContext(t, "user2/repo1/pulls/3/files")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, repo.ID)
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
		prepareSuggestionApply(ctx, issue)
		assert.Equal(t, branch2Head, ctx.Data["SuggestionApplyHeadCommitID"])
		assert.Equal(t, "/user2/repo1/pulls/3/files/reviews/comments", ctx.Data["SuggestionApplyLink"])
	})

	t.Run("AC2 the page gets nothing for someone who cannot push", func(t *testing.T) {
		ctx, _ := contexttest.MockContext(t, "user2/repo1/pulls/3/files")
		contexttest.LoadUser(t, ctx, 4)
		contexttest.LoadRepo(t, ctx, repo.ID)
		prepareSuggestionApply(ctx, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3}))
		assert.NotContains(t, ctx.Data, "SuggestionApplyHeadCommitID")
	})

	t.Run("AC2 someone who cannot push gets a 403", func(t *testing.T) {
		ctx, _ := apply(t, 4, 3, fix, branch2Head)
		assert.Equal(t, http.StatusForbidden, ctx.Resp.WrittenStatus())
	})

	t.Run("an issue that is not a pull request is not found", func(t *testing.T) {
		ctx, _ := apply(t, 2, 1, fix, branch2Head) // repo1 #1 is an issue
		assert.Equal(t, http.StatusNotFound, ctx.Resp.WrittenStatus())
	})

	t.Run("a comment that does not exist is not found", func(t *testing.T) {
		ctx, _ := apply(t, 2, 3, 999999, branch2Head)
		assert.Equal(t, http.StatusNotFound, ctx.Resp.WrittenStatus())
	})

	t.Run("AC7 a comment with two suggestion blocks is not found", func(t *testing.T) {
		twoBlocks := comment(t, "And change for branch2", "```suggestion\na\n```\nor\n```suggestion\nb\n```")
		ctx, _ := apply(t, 2, 3, twoBlocks, branch2Head)
		assert.Equal(t, http.StatusNotFound, ctx.Resp.WrittenStatus())
	})

	t.Run("AC4 the head moved", func(t *testing.T) {
		ctx, resp := apply(t, 2, 3, fix, masterHead)
		refused(t, resp, ctx, "repo.diff.suggestion.head_moved")
	})

	t.Run("AC3 the line changed", func(t *testing.T) {
		outdated := comment(t, "And an older change for branch2", "```suggestion\nAnd a change for branch2\n```")
		ctx, resp := apply(t, 2, 3, outdated, branch2Head)
		refused(t, resp, ctx, "repo.diff.suggestion.outdated")
	})

	t.Run("AC10 the suggestion changes nothing", func(t *testing.T) {
		same := comment(t, "And change for branch2", "```suggestion\nAnd change for branch2\n```")
		ctx, resp := apply(t, 2, 3, same, branch2Head)
		refused(t, resp, ctx, "repo.diff.suggestion.no_change")
	})

	t.Run("AC6 a protected branch gets the web editor's message", func(t *testing.T) {
		rule := &git_model.ProtectedBranch{RepoID: repo.ID, RuleName: "branch2", CanPush: false}
		require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, rule, git_model.WhitelistOptions{}))
		defer func() { require.NoError(t, git_model.DeleteProtectedBranch(t.Context(), repo, rule.ID)) }()
		ctx, resp := apply(t, 2, 3, fix, branch2Head)
		refused(t, resp, ctx, "repo.editor.cannot_commit_to_protected_branch:branch2")
	})

	t.Run("AC5 a closed pull request", func(t *testing.T) {
		_, err := db.GetEngine(t.Context()).ID(3).Cols("is_closed").Update(&issues_model.Issue{IsClosed: true})
		require.NoError(t, err)
		ctx, resp := apply(t, 2, 3, fix, branch2Head)
		refused(t, resp, ctx, "repo.diff.suggestion.pull_closed")
	})
}
