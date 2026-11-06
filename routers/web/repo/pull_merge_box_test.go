// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/translation"
	"gitea.dev/services/contexttest"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareMergeBoxDependencies(t *testing.T) {
	translation.InitLocales(t.Context()) // the expected texts are the English ones

	// viewerID opens pull request 2 (issue 3 in user2/repo1) after user2 gives it the dependencies depIDs.
	// It returns the merge box's blockers as text: each message, then each of its list entries as "text -> link".
	blockers := func(t *testing.T, viewerID int64, dependenciesEnabled bool, depIDs ...int64) []string {
		require.NoError(t, unittest.PrepareTestDatabase())
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		require.NoError(t, pr.LoadIssue(t.Context()))
		for _, depID := range depIDs {
			dep := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: depID})
			require.NoError(t, issues_model.CreateIssueDependency(t.Context(), owner, pr.Issue, dep))
		}
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		issuesUnit, err := repo.GetUnit(t.Context(), unit.TypeIssues)
		require.NoError(t, err)
		issuesUnit.IssuesConfig().EnableDependencies = dependenciesEnabled // fixture repos leave it unset, which reads as off
		require.NoError(t, repo_model.UpdateRepoUnitConfig(t.Context(), issuesUnit))

		ctx, _ := contexttest.MockContext(t, "user2/repo1/pulls/2")
		ctx.Locale = translation.NewLocale("en-US")
		contexttest.LoadUser(t, ctx, viewerID)
		contexttest.LoadRepo(t, ctx, 1)
		prInfo := &pullRequestViewInfo{issue: pr.Issue, MergeBoxData: &pullMergeBoxData{}}
		prInfo.prepareMergeBoxDependencies(ctx)
		require.False(t, ctx.Written())

		texts := []string{}
		for _, item := range prInfo.MergeBoxData.infoProtectionBlockers.items {
			texts = append(texts, string(item.InfoHTML))
			for _, listItem := range item.ListItems {
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(listItem)))
				require.NoError(t, err)
				text := strings.TrimSpace(doc.Text())
				if href, ok := doc.Find("a").Attr("href"); ok {
					text += " -> " + href
				}
				texts = append(texts, text)
			}
		}
		return texts
	}

	t.Run("AC1 links each blocking dependency", func(t *testing.T) {
		// issue 1 (#1) is in the same repository, issue 7 (#2) in user2/repo2
		texts := blockers(t, 2, true, 1, 7)
		require.NotEmpty(t, texts)
		assert.Equal(t, "This pull request is blocked by its dependencies:", texts[0])
		assert.ElementsMatch(t, []string{
			"#1 issue1 -> /user2/repo1/issues/1",
			"user2/repo2#2 - issue7 -> /user2/repo2/issues/2",
		}, texts[1:])
	})
	t.Run("AC3 no dependencies", func(t *testing.T) {
		assert.Empty(t, blockers(t, 2, true))
	})
	t.Run("AC4 counts a dependency the viewer cannot read", func(t *testing.T) {
		// user4 can read the public user2/repo1 but not the private user2/repo2
		texts := blockers(t, 4, true, 1, 7)
		require.NotEmpty(t, texts)
		assert.Equal(t, "This pull request is blocked by its dependencies:", texts[0])
		assert.ElementsMatch(t, []string{
			"#1 issue1 -> /user2/repo1/issues/1",
			"You do not have permission to read 1 dependency",
		}, texts[1:])
	})
	t.Run("AC6 dependencies turned off", func(t *testing.T) {
		assert.Empty(t, blockers(t, 2, false, 1, 7))
	})
}
