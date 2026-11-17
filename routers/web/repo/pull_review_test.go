// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/templates"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/contexttest"
	"gitea.dev/services/forms"
	"gitea.dev/services/pull"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderConversation(t *testing.T) {
	unittest.PrepareTestEnv(t)

	pr, _ := issues_model.GetPullRequestByID(t.Context(), 2)
	_ = pr.LoadIssue(t.Context())
	_ = pr.Issue.LoadPoster(t.Context())
	_ = pr.Issue.LoadRepo(t.Context())

	run := func(name string, cb func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder)) {
		t.Run(name, func(t *testing.T) {
			ctx, resp := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.PageRenderer()})
			contexttest.LoadUser(t, ctx, pr.Issue.PosterID)
			contexttest.LoadRepo(t, ctx, pr.BaseRepoID)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()
			cb(t, ctx, resp)
		})
	}

	var preparedComment *issues_model.Comment
	run("prepare", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		comment, err := pull.CreateCodeComment(ctx, pr.Issue.Poster, ctx.Repo.GitRepo, pr.Issue, 1, 0, "content", "", false, 0, pr.HeadCommitID, nil)
		require.NoError(t, err)

		comment.Invalidated = true
		err = issues_model.UpdateCommentInvalidate(ctx, comment)
		require.NoError(t, err)

		preparedComment = comment
	})
	require.NotNil(t, preparedComment)

	run("diff with outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "diff")
		assert.Contains(t, resp.Body.String(), `<div class="content comment-container"`)
	})
	run("diff without outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = false
		renderConversation(ctx, preparedComment, "diff")
		assert.Contains(t, resp.Body.String(), `conversation-not-existing`)
	})
	run("timeline with outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "timeline")
		assert.Contains(t, resp.Body.String(), `<div id="code-comments-`)
	})
	run("timeline is not affected by ShowOutdatedComments=false", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = false
		renderConversation(ctx, preparedComment, "timeline")
		assert.Contains(t, resp.Body.String(), `<div id="code-comments-`)
	})
	run("diff non-existing review", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		err := db.TruncateBeans(t.Context(), &issues_model.Review{})
		assert.NoError(t, err)
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "diff")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotContains(t, resp.Body.String(), `status-page-500`)
	})
	run("timeline non-existing review", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		err := db.TruncateBeans(t.Context(), &issues_model.Review{})
		assert.NoError(t, err)
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "timeline")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotContains(t, resp.Body.String(), `status-page-500`)
	})
}

// PR 2 is repo1 branch2 against master: README.md new lines 1-6 form one hunk
func TestCodeCommentRange(t *testing.T) {
	unittest.PrepareTestEnv(t)
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadIssue(t.Context()))

	mockCtx := func(t *testing.T, url string) (*context.Context, *httptest.ResponseRecorder) {
		ctx, resp := contexttest.MockContext(t, url, contexttest.MockContextOption{Render: templates.PageRenderer()})
		contexttest.LoadUser(t, ctx, pr.Issue.PosterID)
		contexttest.LoadRepo(t, ctx, pr.BaseRepoID)
		contexttest.LoadGitRepo(t, ctx)
		t.Cleanup(func() { ctx.Repo.GitRepo.Close() })
		ctx.SetPathParam("index", "3")
		return ctx, resp
	}

	t.Run("new comment form shows the range", func(t *testing.T) {
		ctx, _ := mockCtx(t, "/?start_line=2&line=4")
		RenderNewCodeCommentForm(ctx)
		assert.EqualValues(t, 2, ctx.Data["RangeStartLine"])
		assert.EqualValues(t, 4, ctx.Data["RangeEndLine"])

		ctx, _ = mockCtx(t, "/?start_line=4&line=4") // #158 AC3: one line is no range
		RenderNewCodeCommentForm(ctx)
		assert.Nil(t, ctx.Data["RangeStartLine"])
	})

	post := func(t *testing.T, startLine, line int64) *httptest.ResponseRecorder {
		ctx, resp := mockCtx(t, "/")
		ctx.Data["ShowOutdatedComments"] = true // set by a middleware in the real router
		web.SetForm(ctx, &forms.CodeCommentForm{
			Origin: "diff", Content: "range", Side: "proposed", Line: line, StartLine: startLine,
			TreePath: "README.md", SingleReview: true,
		})
		CreateCodeComment(ctx)
		return resp
	}

	t.Run("AC12 an invalid range is refused with a message", func(t *testing.T) {
		resp := post(t, 4, 3)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "repo.diff.comment.range_invalid")
	})

	t.Run("AC3 a one-line range is a single-line comment", func(t *testing.T) {
		resp := post(t, 2, 2)
		assert.Equal(t, http.StatusOK, resp.Code)
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode, Line: 2, Content: "range"})
		assert.EqualValues(t, 0, comment.RangeStartLine())
	})
}
