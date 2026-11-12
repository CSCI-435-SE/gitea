// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	repo_service "gitea.dev/services/repository"
	files_service "gitea.dev/services/repository/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullReviewSuggestionAttribute(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_suggestion",
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

		session := loginUser(t, "user2")
		testPullCreate(t, session, "user2", "test_suggestion", false, "master", "suggestion-branch", "Suggestion PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "suggestion-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))
		commentURL := fmt.Sprintf("/user2/test_suggestion/pulls/%d/files/reviews/comments", pr.Issue.Index)

		postComment := func(t *testing.T, origin, side, line string, extra map[string]string) *HTMLDoc {
			values := map[string]string{
				"origin":        origin,
				"content":       "```suggestion\n\treturn nil\n```",
				"side":          side,
				"line":          line,
				"path":          "README.md",
				"single_review": "true",
			}
			maps.Copy(values, extra)
			req := NewRequestWithValues(t, "POST", commentURL, values)
			return NewHTMLParser(t, session.MakeRequest(t, req, http.StatusOK).Body)
		}

		for _, origin := range []string{"diff", "timeline"} { // repo/diff/comments.tmpl and view_content/conversation.tmpl
			t.Run(origin, func(t *testing.T) {
				markup := postComment(t, origin, "proposed", "2", nil).Find(".render-content.markup[data-suggestion-original]").First()
				require.Equal(t, 1, markup.Length())
				assert.Equal(t, "\treturn err", markup.AttrOr("data-suggestion-original", ""))
				assert.Equal(t, "Suggested change", markup.AttrOr("data-suggestion-label", ""))
			})
		}

		t.Run("reply in the thread gets the attribute", func(t *testing.T) {
			first := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode, Line: 2, TreePath: "README.md"}, unittest.OrderBy("id ASC"))
			doc := postComment(t, "diff", "proposed", "2", map[string]string{"reply": strconv.FormatInt(first.ReviewID, 10)})
			markups := doc.Find(".render-content.markup[data-suggestion-original]")
			assert.Equal(t, doc.Find(".render-content.markup").Length(), markups.Length()) // every comment in the thread, reply included
			assert.Equal(t, "\treturn err", markups.Last().AttrOr("data-suggestion-original", ""))
		})

		t.Run("old side gets no attribute", func(t *testing.T) {
			doc := postComment(t, "diff", "previous", "1", nil)
			require.Equal(t, 1, doc.Find(".render-content.markup code.language-suggestion").Length()) // the comment did render
			assert.Equal(t, 0, doc.Find("[data-suggestion-original]").Length())
		})
	})
}
