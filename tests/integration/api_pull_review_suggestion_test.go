// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	auth_model "gitea.dev/models/auth"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	api "gitea.dev/modules/structs"
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
