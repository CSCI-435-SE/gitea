// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	pull_model "gitea.dev/models/pull"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
	"gitea.dev/modules/translation"
	"gitea.dev/services/context"
	"gitea.dev/services/contexttest"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPullWithDependencies opens pull request 2 (issue 3 in user2/repo1) as viewerID, after user2 gives it the dependencies depIDs.
// It loads the real English locale, as the expected texts are the English ones.
func mockPullWithDependencies(t *testing.T, viewerID int64, dependenciesEnabled bool, depIDs ...int64) (*context.Context, *pullRequestViewInfo) {
	require.NoError(t, unittest.PrepareTestDatabase())
	translation.InitLocales(t.Context())
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
	return ctx, &pullRequestViewInfo{issue: pr.Issue, MergeBoxData: &pullMergeBoxData{}}
}

func TestPrepareMergeBoxDependencies(t *testing.T) {
	// the merge box's blockers as text: each message, then each of its list entries as "text -> link"
	blockers := func(t *testing.T, viewerID int64, dependenciesEnabled bool, depIDs ...int64) []string {
		ctx, prInfo := mockPullWithDependencies(t, viewerID, dependenciesEnabled, depIDs...)
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
	t.Run("AC4 a dependency the viewer cannot read, and no other", func(t *testing.T) {
		assert.Equal(t, []string{
			"This pull request is blocked by its dependencies:",
			"You do not have permission to read 1 dependency",
		}, blockers(t, 4, true, 7))
	})
}

func TestPrepareMergeBoxWithDependencies(t *testing.T) {
	// the merge box built as the page and its reload build it, so the dependency step must be wired in
	ctx, prInfo := mockPullWithDependencies(t, 2, true, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()
	prInfo.prepareViewInfo(ctx, prInfo.issue)
	require.False(t, ctx.Written())
	prInfo.prepareMergeBox(ctx, prInfo.issue)
	require.False(t, ctx.Written())

	var messages []string
	for _, section := range prInfo.MergeBoxData.InfoSections {
		for _, item := range section.InfoItems {
			messages = append(messages, string(item.InfoHTML))
		}
	}
	assert.Contains(t, messages, "This pull request is blocked by its dependencies:")
}

func TestPrepareMergeBoxFormWithDependencies(t *testing.T) {
	type offer struct {
		styles     []string // the merge styles offered, "+auto" where "(When checks succeed)" is offered with it; nil when there is no merge form
		messages   []string
		scheduled  bool   // the form shows a scheduled merge's notice and cancel button
		stylesJSON string // the style list as the page hands it to the form
	}
	// user2 owns user2/repo1 and may merge pull request 2, whose branch merges cleanly unless conflicted is set
	type situation struct {
		manualMerge   bool // user2/repo1 allows "Manually merged"
		checksPending bool // status checks user2 cannot bypass hold the merge back, which is when "(When checks succeed)" is offered
		conflicted    bool // the branch has conflicts
		scheduled     bool // user2 has scheduled the merge for when checks succeed
	}
	mergeBox := func(t *testing.T, s situation, depIDs ...int64) offer {
		ctx, prInfo := mockPullWithDependencies(t, 2, true, depIDs...)
		contexttest.LoadGitRepo(t, ctx)
		defer ctx.Repo.GitRepo.Close()
		ctx.Repo.Repository.MustGetUnit(ctx, unit.TypePullRequests).PullRequestsConfig().AllowManualMerge = s.manualMerge
		if s.conflicted {
			prInfo.issue.PullRequest.Status = issues_model.PullRequestStatusConflict
		}
		if s.scheduled {
			require.NoError(t, pull_model.ScheduleAutoMerge(ctx, ctx.Doer, prInfo.issue.PullRequest.ID, repo_model.MergeStyleMerge, "", false))
		}
		data := prInfo.MergeBoxData
		data.hasPermToMerge = true
		data.hasOverridableBlockers = s.checksPending
		data.canMergeNow = !s.checksPending

		// in the order prepareMergeBox runs them
		prInfo.prepareMergeBoxDependencies(ctx)
		prInfo.prepareMergeBoxFormProps(ctx)
		prInfo.prepareMergeBoxInfoItems(ctx)
		require.False(t, ctx.Written())

		var o offer
		if data.MergeFormProps != nil {
			o.styles = []string{} // a form whose styles are all disallowed is still a form, not "no merge form"
			o.scheduled, _ = data.MergeFormProps["hasPendingPullRequestMerge"].(bool)
			stylesJSON, err := json.Marshal(data.MergeFormProps["mergeStyles"])
			require.NoError(t, err)
			o.stylesJSON = string(stylesJSON)
			for _, s := range data.MergeFormProps["mergeStyles"].([]any) {
				style := s.(map[string]any)
				if !style["allowed"].(bool) {
					continue
				}
				name := style["name"].(string)
				if !style["hideAutoMerge"].(bool) {
					name += "+auto"
				}
				o.styles = append(o.styles, name)
			}
		}
		for _, section := range data.InfoSections {
			for _, item := range section.InfoItems {
				o.messages = append(o.messages, string(item.InfoHTML))
			}
		}
		return o
	}

	t.Run("AC2 only Manually merged while blocked", func(t *testing.T) {
		o := mergeBox(t, situation{manualMerge: true}, 1)
		assert.Equal(t, []string{"manually-merged"}, o.styles)
		assert.NotContains(t, o.messages, "This pull request can be merged automatically.")
	})
	t.Run("AC2 no merge form while blocked where the repository does not allow Manually merged", func(t *testing.T) {
		assert.Nil(t, mergeBox(t, situation{}, 1).styles)
	})
	t.Run("AC2 no merge form while blocked and waiting for checks", func(t *testing.T) {
		o := mergeBox(t, situation{checksPending: true}, 1)
		assert.Nil(t, o.styles)
		assert.NotContains(t, o.messages, "This pull request cannot be merged because all repository merge options are disabled.")
	})
	t.Run("AC2 no merge form while blocked for a viewer held back by checks", func(t *testing.T) {
		// "Manually merged" is allowed, but alone it would open as "Manually merged (When checks succeed)"
		o := mergeBox(t, situation{manualMerge: true, checksPending: true}, 1)
		assert.Nil(t, o.styles)
	})
	t.Run("AC3 Manually merged as before without dependencies, while waiting for checks", func(t *testing.T) {
		assert.Contains(t, mergeBox(t, situation{manualMerge: true, checksPending: true}).styles, "manually-merged")
	})
	t.Run("AC3 no merge prompt as before without dependencies, while waiting for checks", func(t *testing.T) {
		assert.Empty(t, mergeBox(t, situation{checksPending: true}).messages) // the can-merge lines are only for a viewer who can merge now
	})
	t.Run("AC3 a conflicted pull request offers only Manually merged, as before", func(t *testing.T) {
		// Gitea's own note in prepareMergeBoxFormProps: "Manually Merged" marks a non-mergeable (e.g. conflicted) pull request as merged
		assert.Equal(t, []string{"manually-merged"}, mergeBox(t, situation{manualMerge: true, conflicted: true}).styles)
	})
	t.Run("Decision 3 a scheduled merge keeps its notice and cancel button while blocked", func(t *testing.T) {
		// no merge style is offered, so without the schedule there would be no form at all (the AC2 subtest above)
		o := mergeBox(t, situation{scheduled: true}, 1)
		assert.Equal(t, []string{}, o.styles) // a form, with no merge style in it
		assert.Equal(t, "[]", o.stylesJSON)   // a list the form can read, not null
		assert.True(t, o.scheduled)
	})
	t.Run("Decision 3 a conflicted pull request with a scheduled merge keeps its cancel button too", func(t *testing.T) {
		o := mergeBox(t, situation{conflicted: true, scheduled: true})
		assert.Equal(t, []string{}, o.styles)
		assert.True(t, o.scheduled)
	})
	t.Run("control: waiting for checks offers (When checks succeed)", func(t *testing.T) {
		assert.Contains(t, mergeBox(t, situation{checksPending: true}).styles, "merge+auto")
	})
	t.Run("AC3 resolved dependencies look as with none", func(t *testing.T) {
		none := mergeBox(t, situation{})
		// the styles user2/repo1 allows: four in its fixture, plus fast-forward-only, which is on by default
		// (models/repo/repo_unit.go → DefaultPullRequestsConfig)
		assert.Equal(t, []string{"merge", "rebase", "rebase-merge", "squash", "fast-forward-only"}, none.styles)
		assert.Contains(t, none.messages, "This pull request can be merged automatically.")
		resolved := mergeBox(t, situation{}, 5)                // issue 5 is closed
		assert.JSONEq(t, none.stylesJSON, resolved.stylesJSON) // with GOEXPERIMENT=jsonv2, map keys come out in any order
		none.stylesJSON, resolved.stylesJSON = "", ""
		assert.Equal(t, none, resolved)
	})
}
