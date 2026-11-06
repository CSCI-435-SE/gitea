// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/templates"
	"gitea.dev/modules/translation"
	"gitea.dev/services/contexttest"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueViewSidebarDependencyStates(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	translation.InitLocales(t.Context()) // the expected texts are the English ones
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issuesUnit, err := repo.GetUnit(t.Context(), unit.TypeIssues)
	require.NoError(t, err)
	issuesUnit.IssuesConfig().EnableDependencies = true // fixture repos leave it unset, which reads as off
	require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))

	// in user2/repo1: issue 1 (#1) open, issue 2 (#2) merged pull request 1, issue 3 (#3) open pull request 2,
	// issue 5 (#4) closed in the fixtures, issue 11 (#5) pull request 5 closed without merging; issue 7 is #2 in the private user2/repo2
	for _, id := range []int64{2, 11} { // pull request 1 is merged but the fixture leaves it open, so close it as a real merge would
		_, err := issues_model.CloseIssue(t.Context(), unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: id}), owner, issues_model.CloseReasonOptions{})
		require.NoError(t, err)
	}
	for _, dep := range [][2]int64{{3, 1}, {3, 2}, {3, 5}, {3, 11}, {3, 7}, {2, 1}, {5, 1}, {11, 1}} { // {issue, the issue it depends on}
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: dep[0]})
		require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, issue, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: dep[1]})))
	}

	// the entries of one sidebar list ("Depends on" or "Blocks") on issueID's page, as viewerID reads them
	sidebarList := func(t *testing.T, viewerID, issueID int64, heading string) []string {
		ctx, resp := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.PageRenderer()})
		ctx.Locale = translation.NewLocale("en-US")
		ctx.TemplateContext["Locale"] = ctx.Locale // templates read the locale the context was created with
		contexttest.LoadUser(t, ctx, viewerID)
		contexttest.LoadRepo(t, ctx, 1)
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issueID})
		require.NoError(t, issue.LoadRepo(ctx))
		ctx.Data["Issue"] = issue
		ctx.Data["Repository"] = ctx.Repo.Repository

		prepareIssueViewSidebarDependency(ctx, issue)
		require.False(t, ctx.Written())
		ctx.HTML(http.StatusOK, "repo/issue/sidebar/issue_dependencies")

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		require.NoError(t, err)
		entries := []string{}
		doc.Find("strong").FilterFunction(func(_ int, s *goquery.Selection) bool { return s.Text() == heading }).
			Parent().Next().Find(".item").Each(func(_ int, s *goquery.Selection) {
			// tests render each icon as its name in a span; AC5 asks for the state word, so leave the icons out
			s.Find("span").FilterFunction(func(_ int, e *goquery.Selection) bool {
				return e.Children().Length() == 0 && strings.HasPrefix(e.Text(), "octicon-")
			}).Remove()
			entries = append(entries, strings.Join(strings.Fields(s.Text()), " "))
		})
		return entries
	}

	t.Run("AC5 Depends on", func(t *testing.T) {
		assert.ElementsMatch(t, []string{
			"#1 issue1 user2/repo1 · Open",
			"#2 issue2 user2/repo1 · Merged",
			"#4 issue5 user2/repo1 · Closed",
			"#5 pull5 user2/repo1 · Closed",
			"#2 issue7 user2/repo2 · Open",
		}, sidebarList(t, 2, 3, "Depends on"))
	})
	t.Run("AC5 Blocks", func(t *testing.T) {
		assert.ElementsMatch(t, []string{
			"#3 issue3 user2/repo1 · Open",
			"#2 issue2 user2/repo1 · Merged",
			"#4 issue5 user2/repo1 · Closed",
			"#5 pull5 user2/repo1 · Closed",
		}, sidebarList(t, 2, 1, "Blocks"))
	})
	t.Run("AC5 an entry the viewer can remove but not read", func(t *testing.T) {
		// user40 may write to user2/repo1, so it manages dependencies, but cannot read user2/repo2
		assert.Contains(t, sidebarList(t, 40, 3, "Depends on"), "#2 issue7 user2/repo2 · Open")
	})
	t.Run("AC6 dependencies turned off", func(t *testing.T) {
		// the same dependencies as above, which the AC5 subtests show while dependencies are on
		issuesUnit.IssuesConfig().EnableDependencies = false
		require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))
		defer func() { // a defer, as t.Context() is already canceled when t.Cleanup runs
			issuesUnit.IssuesConfig().EnableDependencies = true
			require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))
		}()
		assert.Empty(t, sidebarList(t, 2, 3, "Depends on"))
		assert.Empty(t, sidebarList(t, 2, 1, "Blocks"))
	})
}
