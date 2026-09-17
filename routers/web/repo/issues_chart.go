// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"gitea.dev/modules/templates"
	"gitea.dev/services/context"
	repo_service "gitea.dev/services/repository"
)

const (
	tplIssuesChart templates.TplName = "repo/activity"
)

// IssuesChart renders the activity sub-tab charting a repository's issues over time
func IssuesChart(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.activity.navbar.issues")
	ctx.Data["PageIsActivity"] = true
	ctx.Data["PageIsIssuesChart"] = true
	ctx.PageData["repoLink"] = ctx.Repo.RepoLink

	ctx.HTML(http.StatusOK, tplIssuesChart)
}

// IssuesChartData returns JSON of weekly issue counts, or null for a repository with no issues
func IssuesChartData(ctx *context.Context) {
	weeks, err := repo_service.GetRepoIssueWeeks(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		ctx.ServerError("GetRepoIssueWeeks", err)
		return
	}
	ctx.JSON(http.StatusOK, weeks)
}
