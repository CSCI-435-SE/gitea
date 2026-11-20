// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"gitea.dev/modules/git"
	"gitea.dev/services/contexttest"
	pull_service "gitea.dev/services/pull"
	files_service "gitea.dev/services/repository/files"

	"github.com/stretchr/testify/assert"
)

// Each refusal's status comes from #160's revised spec: 403 stop, 409 refetch and retry, 422 nothing to apply.
func TestApplyPullReviewSuggestionError(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"AC3a cannot push", files_service.ErrSuggestionPermission, http.StatusForbidden},
		{"decision 2 protected branch", files_service.ErrSuggestionProtectedBranch{Branch: "main"}, http.StatusForbidden},
		{"decision 2 signed commits required", files_service.ErrUserCannotCommit{UserName: "user2"}, http.StatusForbidden},
		{"decision 2 protected file", pull_service.ErrFilePathProtected{Path: "README.md"}, http.StatusForbidden},
		{"decision 2 pre-receive hook", &git.ErrPushRejected{Message: "rejected by policy"}, http.StatusForbidden},
		{"AC3b outdated", files_service.ErrSuggestionOutdated, http.StatusConflict},
		{"AC3c stale head_sha", files_service.ErrSuggestionHeadMoved, http.StatusConflict},
		{"AC3c pushed to between check and commit", &git.ErrPushOutOfDate{}, http.StatusConflict},
		{"AC3d closed pull request", files_service.ErrSuggestionPullClosed, http.StatusConflict},
		{"already applied (#157)", files_service.ErrSuggestionAlreadyApplied, http.StatusConflict},
		{"AC3e no single-line suggestion", files_service.ErrSuggestionNotApplicable, http.StatusUnprocessableEntity},
		{"decision 2 no change", files_service.ErrSuggestionNoChange, http.StatusUnprocessableEntity},
		{"a wrapped refusal keeps its status", fmt.Errorf("apply: %w", files_service.ErrSuggestionOutdated), http.StatusConflict},
		{"anything else is a server error", errors.New("disk full"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, resp := contexttest.MockAPIContext(t, "POST /api/v1/repos/user2/repo1/pulls/comments/1/suggestion/apply")
			applyPullReviewSuggestionError(ctx, c.err)
			assert.Equal(t, c.status, resp.Code)
		})
	}

	t.Run("decision 2 a pre-receive hook's message reaches the client", func(t *testing.T) {
		ctx, resp := contexttest.MockAPIContext(t, "POST /api/v1/repos/user2/repo1/pulls/comments/1/suggestion/apply")
		applyPullReviewSuggestionError(ctx, &git.ErrPushRejected{Message: "rejected by policy"})
		assert.Contains(t, resp.Body.String(), "rejected by policy")
	})
}
