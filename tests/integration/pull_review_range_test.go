// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/gitrepo"
	pull_service "gitea.dev/services/pull"
	repo_service "gitea.dev/services/repository"
	files_service "gitea.dev/services/repository/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileLines returns "l1\n".."lN\n" with the given lines replaced
func fileLines(n int, replace map[int]string) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		if s, ok := replace[i]; ok {
			sb.WriteString(s + "\n")
		} else {
			fmt.Fprintf(&sb, "l%d\n", i)
		}
	}
	return sb.String()
}

func TestPullReviewRangeComment(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_range",
			AutoInit:         true,
			Readme:           "Default",
			ObjectFormatName: git.Sha1ObjectFormat.Name(),
			DefaultBranch:    "master",
		}, true)
		require.NoError(t, err)
		writeFile := func(branch, newBranch, content string) {
			_, err := files_service.ChangeRepoFiles(t.Context(), repo, user2, &files_service.ChangeRepoFilesOptions{
				OldBranch: branch,
				NewBranch: newBranch,
				Files:     []*files_service.ChangeRepoFile{{Operation: "update", TreePath: "README.md", ContentReader: strings.NewReader(content)}},
			})
			require.NoError(t, err)
		}
		// lines 2 and 13 of 14 change, so the diff has two hunks: new lines 1-5 and 10-14
		writeFile("master", "", fileLines(14, nil))
		writeFile("master", "range-branch", fileLines(14, map[int]string{2: "X2", 13: "X13"}))

		session := loginUser(t, "user2")
		testPullCreate(t, session, "user2", "test_range", false, "master", "range-branch", "Range PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "range-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))
		commentURL := fmt.Sprintf("/user2/test_range/pulls/%d/files/reviews/comments", pr.Issue.Index)

		post := func(t *testing.T, origin, side, startLine, line string, status int) *HTMLDoc {
			req := NewRequestWithValues(t, "POST", commentURL, map[string]string{
				"origin":        origin,
				"content":       "```suggestion\nnew\n```",
				"side":          side,
				"start_line":    startLine,
				"line":          line,
				"path":          "README.md",
				"single_review": "true",
			})
			resp := session.MakeRequest(t, req, status)
			if status != http.StatusOK {
				return nil
			}
			return NewHTMLParser(t, resp.Body)
		}

		for _, origin := range []string{"diff", "timeline"} {
			t.Run("AC1 AC5 range on lines 1-3 from "+origin, func(t *testing.T) {
				doc := post(t, origin, "proposed", "1", "3", http.StatusOK)
				assert.Equal(t, "Lines 1–3", strings.TrimSpace(doc.Find(".code-comment-range").First().Text()))
				markup := doc.Find(".render-content.markup[data-suggestion-original]").Last()
				assert.Equal(t, "l1\nX2\nl3", markup.AttrOr("data-suggestion-original", ""))
			})
		}

		t.Run("AC3 one-line range is a single-line comment", func(t *testing.T) {
			doc := post(t, "diff", "proposed", "11", "11", http.StatusOK)
			assert.Equal(t, 0, doc.Find(".code-comment-range").Length())
			comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode, Line: 11})
			assert.EqualValues(t, 0, comment.RangeStartLine())
		})

		t.Run("AC12 invalid ranges are refused", func(t *testing.T) {
			before := unittest.GetCount(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode})
			post(t, "diff", "proposed", "4", "3", http.StatusBadRequest)  // start after end
			post(t, "diff", "previous", "1", "3", http.StatusBadRequest)  // old side
			post(t, "diff", "proposed", "4", "11", http.StatusBadRequest) // two hunks
			post(t, "diff", "proposed", "7", "8", http.StatusBadRequest)  // between hunks, no hunk at all
			assert.Equal(t, before, unittest.GetCount(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode}))
		})

		gitRepo, err := gitrepo.OpenRepository(t.Context(), repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		invalidate := func(t *testing.T) {
			require.NoError(t, pull_service.InvalidateCodeComments(t.Context(), issues_model.PullRequestList{pr}, user2, repo, gitRepo, "range-branch"))
		}
		post(t, "diff", "proposed", "", "2", http.StatusOK) // AC13: a single-line comment on line 2
		ranged := func() []*issues_model.Comment {
			all, err := issues_model.FindComments(t.Context(), &issues_model.FindCommentsOptions{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode})
			require.NoError(t, err)
			var cs []*issues_model.Comment
			for _, c := range all {
				if c.RangeStartLine() > 0 || c.Line == 2 {
					cs = append(cs, c)
				}
			}
			return cs
		}

		t.Run("AC8 a change after the range keeps it current", func(t *testing.T) {
			writeFile("range-branch", "", fileLines(14, map[int]string{2: "X2", 13: "Y13"}))
			invalidate(t)
			for _, c := range ranged() {
				assert.False(t, c.Invalidated, "comment on line %d", c.Line)
			}
		})

		t.Run("AC7 AC13 a change inside the range makes it outdated", func(t *testing.T) {
			writeFile("range-branch", "", fileLines(14, map[int]string{2: "Y2", 13: "Y13"}))
			invalidate(t)
			cs := ranged()
			require.Len(t, cs, 3) // two range comments and the single-line one
			for _, c := range cs {
				assert.True(t, c.Invalidated, "comment on line %d", c.Line)
			}
		})
	})
}
