// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"net/http"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/git"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	pull_service "gitea.dev/services/pull"
	files_service "gitea.dev/services/repository/files"
)

// ApplyPullReviewSuggestion commits a review comment's suggestion to the pull request's head branch (#160)
func ApplyPullReviewSuggestion(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/pulls/comments/{id}/suggestion/apply repository repoApplyPullReviewSuggestion
	// ---
	// summary: Apply a review comment's suggestion as a commit on the pull request's head branch
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the review comment
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ApplyPullReviewSuggestionOptions"
	// responses:
	//   "201":
	//     "$ref": "#/responses/AppliedPullReviewSuggestion"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "423":
	//     "$ref": "#/responses/repoArchivedError"
	form := web.GetForm(ctx).(*api.ApplyPullReviewSuggestionOptions)
	comment := getPullReviewCommentToResolve(ctx) // 404 outside this repo, 400 unless a code comment on a pull request
	if comment == nil {
		return
	}
	// someone else's pending review is invisible through the API, so don't confirm it exists
	if err := comment.LoadReview(ctx); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	if comment.Review != nil && comment.Review.Type == issues_model.ReviewTypePending && comment.Review.ReviewerID != ctx.Doer.ID {
		ctx.APIErrorNotFound()
		return
	}
	if err := comment.Issue.LoadPullRequest(ctx); err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	commitSHA, err := files_service.ApplySuggestion(ctx, ctx.Doer, comment.Issue.PullRequest, comment, form.HeadSHA, form.Message)
	if err != nil {
		applyPullReviewSuggestionError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, &api.AppliedPullReviewSuggestion{CommitSHA: commitSHA})
}

// applyPullReviewSuggestionError answers a refused apply: 403 when the doer may not push, 409 when the
// pull request or the code moved on, 422 when the comment has nothing to apply. Nothing was committed.
func applyPullReviewSuggestionError(ctx *context.APIContext, err error) {
	var errProtected files_service.ErrSuggestionProtectedBranch
	var errPushRejected *git.ErrPushRejected
	switch {
	case errors.Is(err, files_service.ErrSuggestionPermission):
		ctx.APIError(http.StatusForbidden, "you cannot push to the pull request's head branch")
	case errors.As(err, &errProtected):
		ctx.APIError(http.StatusForbidden, err.Error())
	case files_service.IsErrUserCannotCommit(err), pull_service.IsErrFilePathProtected(err):
		ctx.APIError(http.StatusForbidden, err.Error()) // signed commits or protected files, as the file API answers them
	case errors.As(err, &errPushRejected):
		ctx.APIError(http.StatusForbidden, errPushRejected.Message) // a pre-receive hook, as the file API answers it
	case errors.Is(err, files_service.ErrSuggestionHeadMoved), git.IsErrPushOutOfDate(err):
		ctx.APIError(http.StatusConflict, "head_sha is not the head of the pull request's branch")
	case errors.Is(err, files_service.ErrSuggestionOutdated):
		ctx.APIError(http.StatusConflict, "the suggestion is outdated: the code changed since it was written")
	case errors.Is(err, files_service.ErrSuggestionPullClosed):
		ctx.APIError(http.StatusConflict, "the pull request is closed or merged, or its head branch is gone")
	case errors.Is(err, files_service.ErrSuggestionAlreadyApplied):
		ctx.APIError(http.StatusConflict, "the suggestion was already applied") // #157 records it; 409 as #160's first AC3 asked
	case errors.Is(err, files_service.ErrSuggestionNotApplicable):
		ctx.APIError(http.StatusUnprocessableEntity, "the comment has no single-line suggestion to apply")
	case errors.Is(err, files_service.ErrSuggestionNoChange):
		ctx.APIError(http.StatusUnprocessableEntity, "the suggestion makes no changes to the code")
	default:
		ctx.APIErrorInternal(err)
	}
}
