// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"gitea.dev/modules/log"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/structs"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
)

// similarIssuesLimit is how many suggestions the new-issue form shows.
const similarIssuesLimit = 5

// similarQueryMaxRunes mirrors the maxlength="255" the title input already enforces in
// templates/repo/issue/new_form.tmpl, so a request can't make this anonymous endpoint tokenize
// and rank an unbounded string.
const similarQueryMaxRunes = 255

// SimilarIssues returns issues whose titles resemble the title being typed on the new
// issue/pull request form, so the author can spot a duplicate before filing one.
func SimilarIssues(ctx *context.Context) {
	wantPull := ctx.FormBool("is_pull")

	// Only search what this viewer may read: someone who cannot see pull requests must not
	// learn about them through suggestions. An empty list rather than an error keeps the panel
	// invisible instead of surfacing a failure on a form that is working fine.
	if !ctx.Repo.Permission.CanReadIssuesOrPulls(wantPull) {
		ctx.JSON(http.StatusOK, []*structs.Issue{})
		return
	}

	query := ctx.FormString("q")
	// Truncate by runes, not bytes: slicing mid-rune would corrupt a multi-byte character and
	// change the tokens the rest of this feature derives from it.
	if runes := []rune(query); len(runes) > similarQueryMaxRunes {
		query = string(runes[:similarQueryMaxRunes])
	}

	similar, err := issue_service.FindSimilarIssues(ctx, ctx.Repo.Repository,
		optional.Some(wantPull), query, similarIssuesLimit)
	if err != nil {
		log.Error("FindSimilarIssues: %v", err)
		// This endpoint is only ever called by XHR that reads the status and discards the body,
		// so rendering the HTML error template would be wasted work with a misleading content type.
		ctx.JSON(http.StatusInternalServerError, nil)
		return
	}

	ctx.JSON(http.StatusOK, similar)
}
