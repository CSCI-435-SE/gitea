// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/gitrepo"
	api "gitea.dev/modules/structs"
	issue_service "gitea.dev/services/issue"
	repo_service "gitea.dev/services/repository"
	files_service "gitea.dev/services/repository/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A bot posting a review through the existing API gets suggestions with no API change (#160 AC1).
func TestAPIPullReviewSuggestionRenders(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		const (
			commentedLine = "\treturn err" // line 2 of the head README, written below
			replacement   = "\treturn nil"
			suggestion    = "```suggestion\n" + replacement + "\n```"
		)

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_api_suggestion",
			Readme:           "Default",
			AutoInit:         true,
			ObjectFormatName: git.Sha1ObjectFormat.Name(),
			DefaultBranch:    "master",
		}, true)
		require.NoError(t, err)

		_, err = files_service.ChangeRepoFiles(t.Context(), repo, user2, &files_service.ChangeRepoFilesOptions{
			NewBranch: "suggestion-branch",
			Files: []*files_service.ChangeRepoFile{{
				Operation:     "update",
				TreePath:      "README.md",
				ContentReader: strings.NewReader("# Title\n" + commentedLine + "\n"),
			}},
		})
		require.NoError(t, err)

		session := loginUser(t, "user2")
		testPullCreate(t, session, "user2", "test_api_suggestion", false, "master", "suggestion-branch", "API suggestion PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "suggestion-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))

		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/user2/test_api_suggestion/pulls/%d/reviews", pr.Issue.Index), &api.CreatePullReviewOptions{
			Event: api.ReviewStateComment,
			Body:  "lint bot",
			Comments: []api.CreatePullReviewComment{
				{Path: "README.md", Body: suggestion, NewLineNum: 2},
				{Path: "README.md", Body: suggestion, OldLineNum: 1}, // old side: stays a plain code block
			},
		}).AddTokenAuth(token)
		review := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &api.PullReview{})
		assert.Equal(t, api.ReviewStateComment, review.State)
		assert.Equal(t, 2, review.CodeCommentsCount)

		req = NewRequestf(t, http.MethodGet, "/api/v1/repos/user2/test_api_suggestion/pulls/%d/reviews/%d/comments", pr.Issue.Index, review.ID).AddTokenAuth(token)
		comments := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), []*api.PullReviewComment{})
		require.Len(t, comments, 2)
		for _, c := range comments {
			assert.Equal(t, suggestion, c.Body) // stored verbatim, no API-side rewriting
		}

		pages := map[string]string{ // repo/diff/comments.tmpl and view_content/conversation.tmpl
			"files tab":        fmt.Sprintf("/user2/test_api_suggestion/pulls/%d/files", pr.Issue.Index),
			"conversation tab": fmt.Sprintf("/user2/test_api_suggestion/pulls/%d", pr.Issue.Index),
		}
		for name, link := range pages {
			t.Run(name, func(t *testing.T) {
				doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, http.MethodGet, link), http.StatusOK).Body)
				assert.Equal(t, 2, doc.Find(".render-content.markup code.language-suggestion").Length())

				markup := doc.Find(".render-content.markup[data-suggestion-original]")
				require.Equal(t, 1, markup.Length()) // only the new-side comment
				assert.Equal(t, commentedLine, markup.AttrOr("data-suggestion-original", ""))
				assert.Equal(t, "Suggested change", markup.AttrOr("data-suggestion-label", ""))
				assert.Equal(t, replacement+"\n", markup.Find("code.language-suggestion").Text())
			})
		}
	})
}

// A token that can push applies a suggestion through the API; everything else commits nothing (#160 AC2, AC3).
// The subtests share one pull request and run in order.
func TestAPIPullReviewApplySuggestion(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owns the repository, so can push
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // the lint bot: can review, cannot push
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_api_apply_suggestion",
			Readme:           "Default",
			AutoInit:         true,
			ObjectFormatName: git.Sha1ObjectFormat.Name(),
			DefaultBranch:    "master",
		}, true)
		require.NoError(t, err)
		_, err = files_service.ChangeRepoFiles(t.Context(), repo, user2, &files_service.ChangeRepoFilesOptions{
			NewBranch: "suggestion-branch",
			Files: []*files_service.ChangeRepoFile{{
				Operation:     "update",
				TreePath:      "README.md",
				ContentReader: strings.NewReader("# Title\n\treturn err\n"),
			}},
		})
		require.NoError(t, err)

		ownerSession := loginUser(t, "user2")
		testPullCreate(t, ownerSession, "user2", "test_api_apply_suggestion", false, "master", "suggestion-branch", "API apply PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "suggestion-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))
		repoAPI := "/api/v1/repos/user2/test_api_apply_suggestion"

		ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeWriteRepository)
		ownerReadToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeReadRepository)
		botToken := getTokenForLoggedInUser(t, loginUser(t, "user4"), auth_model.AccessTokenScopeWriteRepository)

		gitRepo, err := gitrepo.OpenRepository(t.Context(), repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		headCommit := func(t *testing.T) *git.Commit {
			commit, err := gitRepo.GetBranchCommit("suggestion-branch")
			require.NoError(t, err)
			return commit
		}

		// the bot reviews line 2 through the existing reviews API, as AC1 does
		suggest := func(t *testing.T, body string) int64 {
			req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("%s/pulls/%d/reviews", repoAPI, pr.Issue.Index), &api.CreatePullReviewOptions{
				Event:    api.ReviewStateComment,
				Comments: []api.CreatePullReviewComment{{Path: "README.md", Body: body, NewLineNum: 2}},
			}).AddTokenAuth(botToken)
			review := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), &api.PullReview{})
			req = NewRequestf(t, http.MethodGet, "%s/pulls/%d/reviews/%d/comments", repoAPI, pr.Issue.Index, review.ID).AddTokenAuth(botToken)
			comments := DecodeJSON(t, MakeRequest(t, req, http.StatusOK), []*api.PullReviewComment{})
			require.Len(t, comments, 1)
			return comments[0].ID
		}
		apply := func(t *testing.T, token string, commentID int64, opts *api.ApplyPullReviewSuggestionOptions, status int) *httptest.ResponseRecorder {
			req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("%s/pulls/comments/%d/suggestion/apply", repoAPI, commentID), opts)
			if token != "" {
				req.AddTokenAuth(token)
			}
			return MakeRequest(t, req, status)
		}

		useNil := suggest(t, "```suggestion\n\treturn nil\n```")
		startHead := headCommit(t).ID.String()
		atStart := &api.ApplyPullReviewSuggestionOptions{HeadSHA: startHead}

		t.Run("no token is refused", func(t *testing.T) {
			apply(t, "", useNil, atStart, http.StatusUnauthorized)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC2 a read:repository token is refused", func(t *testing.T) {
			apply(t, ownerReadToken, useNil, atStart, http.StatusForbidden)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC3 a token without push access to the head branch gets 403", func(t *testing.T) {
			apply(t, botToken, useNil, atStart, http.StatusForbidden)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC3 a stale head_sha gets 409", func(t *testing.T) {
			baseHead, err := gitRepo.GetBranchCommitID("master")
			require.NoError(t, err)
			apply(t, ownerToken, useNil, &api.ApplyPullReviewSuggestionOptions{HeadSHA: baseHead}, http.StatusConflict)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("a missing head_sha gets 422", func(t *testing.T) {
			apply(t, ownerToken, useNil, &api.ApplyPullReviewSuggestionOptions{}, http.StatusUnprocessableEntity)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("a comment that doesn't exist gets 404", func(t *testing.T) {
			apply(t, ownerToken, 999999, atStart, http.StatusNotFound)
		})

		t.Run("a comment with no suggestion block gets 422", func(t *testing.T) {
			apply(t, ownerToken, suggest(t, "looks good"), atStart, http.StatusUnprocessableEntity)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC2 a token that can push commits the suggestion and gets the commit SHA", func(t *testing.T) {
			resp := apply(t, ownerToken, useNil, atStart, http.StatusCreated)
			applied := DecodeJSON(t, resp, &api.AppliedPullReviewSuggestion{})
			commit := headCommit(t)
			assert.Equal(t, commit.ID.String(), applied.CommitSHA)
			require.Equal(t, 1, commit.ParentCount())
			parent, err := commit.ParentID(0)
			require.NoError(t, err)
			assert.Equal(t, startHead, parent.String())

			blob, err := commit.GetBlobByPath("README.md")
			require.NoError(t, err)
			content, err := blob.GetBlobContent(1024)
			require.NoError(t, err)
			assert.Equal(t, "# Title\n\treturn nil\n", content)
			// no message given: S3's default, crediting the reviewer (#156 AC1)
			coAuthor := fmt.Sprintf("Co-authored-by: %s <%s>", user4.GitName(), user4.GetEmail())
			assert.Equal(t, "Apply suggestion from @user4 on README.md\n\n"+coAuthor, strings.TrimSpace(commit.MessageRaw))
		})

		t.Run("AC3 an outdated suggestion gets 409", func(t *testing.T) {
			before := headCommit(t).ID.String()
			outdated := suggest(t, "```suggestion\n\treturn errors.New(\"x\")\n```") // written against "\treturn nil"
			_, err := files_service.ChangeRepoFiles(t.Context(), repo, user2, &files_service.ChangeRepoFilesOptions{
				OldBranch: "suggestion-branch",
				Files: []*files_service.ChangeRepoFile{{
					Operation:     "update",
					TreePath:      "README.md",
					ContentReader: strings.NewReader("# Title\n\treturn nil // changed\n"),
				}},
			})
			require.NoError(t, err)
			changed := headCommit(t).ID.String()
			require.NotEqual(t, before, changed)
			apply(t, ownerToken, outdated, &api.ApplyPullReviewSuggestionOptions{HeadSHA: changed}, http.StatusConflict)
			assert.Equal(t, changed, headCommit(t).ID.String())
		})

		t.Run("a closed pull request gets 409", func(t *testing.T) {
			before := headCommit(t).ID.String()
			pending := suggest(t, "```suggestion\n\treturn nil // closed\n```")
			require.NoError(t, issue_service.CloseIssue(t.Context(), pr.Issue, user2, ""))
			apply(t, ownerToken, pending, &api.ApplyPullReviewSuggestionOptions{HeadSHA: before}, http.StatusConflict)
			assert.Equal(t, before, headCommit(t).ID.String())
		})
	})
}
