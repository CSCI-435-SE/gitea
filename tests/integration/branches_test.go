// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"gitea.dev/models/db"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/translation"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewBranches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	req := NewRequest(t, "GET", "/user2/repo1/branches")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	AssertHTMLElement(t, htmlDoc, "[data-testid=branches-default-branch-list]", 1)
	AssertHTMLElement(t, htmlDoc, "[data-testid=branches-default-branch-not-exist]", 0)

	repo1.DefaultBranch = "non-existent-branch"
	_, _ = db.GetEngine(t.Context()).ID(repo1.ID).Cols("default_branch").Update(repo1)
	req = NewRequest(t, "GET", "/user2/repo1/branches")
	resp = MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	AssertHTMLElement(t, htmlDoc, "[data-testid=branches-default-branch-list]", 0)
	AssertHTMLElement(t, htmlDoc, "[data-testid=branches-default-branch-not-exist]", 1)
}

func TestViewBranchesStaleBadge(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// repo1's "branch2" has a commit from 2020 (well past the 90-day default threshold)
	// and is not deleted, so its row must show the badge.
	req := NewRequest(t, "GET", "/user2/repo1/branches")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	branch2Row := htmlDoc.Find(`a.branch-name[href$="/src/branch/branch2"]`).Closest(".flex-text-block")
	assert.Equal(t, 1, branch2Row.Find("[data-testid=branch-stale-label]").Length())

	// "master" is repo1's default branch. Even though its own latest commit is also old,
	// it is rendered in the separate default-branch block and must never get the badge.
	defaultBranchSection := htmlDoc.Find("[data-testid=branches-default-branch-list]")
	assert.Equal(t, 0, defaultBranchSection.Find("[data-testid=branch-stale-label]").Length())

	// "foo" and "bar" are deleted branches; deleted rows never get the badge either.
	for _, name := range []string{"foo", "bar"} {
		deletedRow := htmlDoc.Find(`span.branch-name`).FilterFunction(func(_ int, s *goquery.Selection) bool {
			return s.Text() == name
		}).Closest("td")
		assert.Equal(t, 0, deletedRow.Find("[data-testid=branch-stale-label]").Length())
	}
}

func TestUndoDeleteBranch(t *testing.T) {
	branchAction := func(t *testing.T, button, attr string) (*HTMLDoc, string) {
		session := loginUser(t, "user2")
		req := NewRequest(t, "GET", "/user2/repo1/branches")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		link, exists := htmlDoc.doc.Find(button).Attr(attr)
		require.True(t, exists, "The template has changed")
		linkURL, err := url.Parse(link)
		require.NoError(t, err)

		req = NewRequest(t, "POST", link)
		session.MakeRequest(t, req, http.StatusOK)
		req = NewRequest(t, "GET", "/user2/repo1/branches")
		resp = session.MakeRequest(t, req, http.StatusOK)

		return NewHTMLParser(t, resp.Body), linkURL.Query().Get("name")
	}

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		htmlDoc, name := branchAction(t, ".delete-branch-button", "data-modal-form.url")
		assert.Contains(t,
			htmlDoc.doc.Find(".ui.positive.message").Text(),
			translation.NewLocale("en-US").TrString("repo.branch.deletion_success", name),
		)
		htmlDoc, name = branchAction(t, ".restore-branch-button", "data-url")
		assert.Contains(t,
			htmlDoc.doc.Find(".ui.positive.message").Text(),
			translation.NewLocale("en-US").TrString("repo.branch.restore_success", name),
		)
	})
}
