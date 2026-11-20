// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/gitrepo"
	"gitea.dev/modules/queue"
	issue_service "gitea.dev/services/issue"
	repo_service "gitea.dev/services/repository"
	files_service "gitea.dev/services/repository/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #159: applying several suggestions in one commit. The subtests share one pull request and run in order.
func TestPullReviewApplySuggestions(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owns the repository, so can push
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // signed in, cannot push
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_apply_suggestions",
			Readme:           "Default",
			AutoInit:         true,
			ObjectFormatName: git.Sha1ObjectFormat.Name(),
			DefaultBranch:    "master",
		}, true)
		require.NoError(t, err)
		_, err = files_service.ChangeRepoFiles(t.Context(), repo, user2, &files_service.ChangeRepoFilesOptions{
			NewBranch: "batch-branch",
			Files: []*files_service.ChangeRepoFile{{
				Operation:     "update",
				TreePath:      "README.md",
				ContentReader: strings.NewReader("# Title\nl2\nl3\nl4\nl5\nl6\n"),
			}},
		})
		require.NoError(t, err)

		ownerSession := loginUser(t, "user2")
		readerSession := loginUser(t, "user4")
		testPullCreate(t, ownerSession, "user2", "test_apply_suggestions", false, "master", "batch-branch", "Batch PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "batch-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))
		pullURL := fmt.Sprintf("/user2/test_apply_suggestions/pulls/%d", pr.Issue.Index)

		gitRepo, err := gitrepo.OpenRepository(t.Context(), repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		headCommit := func(t *testing.T) *git.Commit {
			commit, err := gitRepo.GetBranchCommit("batch-branch")
			require.NoError(t, err)
			return commit
		}
		readme := func(t *testing.T) string {
			blob, err := headCommit(t).GetBlobByPath("README.md")
			require.NoError(t, err)
			content, err := blob.GetBlobContent(1024)
			require.NoError(t, err)
			return content
		}

		// user4 reviews a line of README.md
		suggest := func(t *testing.T, line int, content string) *issues_model.Comment {
			req := NewRequestWithValues(t, "POST", pullURL+"/files/reviews/comments", map[string]string{
				"origin":        "diff",
				"content":       content,
				"side":          "proposed",
				"line":          strconv.Itoa(line),
				"path":          "README.md",
				"single_review": "true",
			})
			readerSession.MakeRequest(t, req, http.StatusOK)
			return unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode, Content: content})
		}
		applyBatch := func(t *testing.T, session *TestSession, comments []*issues_model.Comment, head, message string, status int) *httptest.ResponseRecorder {
			values := url.Values{"head_commit_id": {head}, "message": {message}}
			for _, c := range comments {
				values.Add("comment_ids", strconv.FormatInt(c.ID, 10))
			}
			resp := session.MakeRequest(t, NewRequestWithURLValues(t, "POST", pullURL+"/files/reviews/comments/apply", values), status)
			if status == http.StatusOK {
				// applying pushes, and the pull request is updated in the background; a comment posted before that
				// finishes can fail (seen on MySQL CI), so let it finish before the next request
				require.NoError(t, queue.GetManager().FlushAll(t.Context(), 5*time.Second))
			}
			return resp
		}
		errorMessage := func(t *testing.T, resp *httptest.ResponseRecorder) string {
			return DecodeJSON(t, resp, &struct {
				ErrorMessage string `json:"errorMessage"`
			}{}).ErrorMessage
		}
		batchToggles := func(t *testing.T, session *TestSession) int {
			doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", pullURL+"/files"), http.StatusOK).Body)
			return doc.Find("[data-suggestion-batch-url]").Length()
		}

		threeLines := suggest(t, 2, "```suggestion\na\nb\nc\n```")
		deleteLine := suggest(t, 4, "```suggestion\n```")
		sameLine := suggest(t, 2, "```suggestion\ntwo\n```")
		startHead := headCommit(t).ID.String()
		refusedWith := func(t *testing.T, resp *httptest.ResponseRecorder, message string) {
			assert.Equal(t, message, errorMessage(t, resp))
			assert.Equal(t, startHead, headCommit(t).ID.String()) // nothing committed
		}

		t.Run("AC5 someone who can push gets the batch button; a viewer who cannot gets none and a 403", func(t *testing.T) {
			assert.Equal(t, 3, batchToggles(t, ownerSession))
			assert.Equal(t, 0, batchToggles(t, readerSession))
			applyBatch(t, readerSession, []*issues_model.Comment{threeLines, deleteLine}, startHead, "", http.StatusForbidden)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC5 a signed-out viewer is sent to sign in", func(t *testing.T) {
			resp := applyBatch(t, emptyTestSession(t), []*issues_model.Comment{threeLines, deleteLine}, startHead, "", http.StatusSeeOther)
			assert.Contains(t, resp.Header().Get("Location"), "/user/login")
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		postIDs := func(t *testing.T, ids []string, status int) *httptest.ResponseRecorder {
			values := url.Values{"head_commit_id": {startHead}, "comment_ids": ids}
			return ownerSession.MakeRequest(t, NewRequestWithURLValues(t, "POST", pullURL+"/files/reviews/comments/apply", values), status)
		}

		t.Run("an ID that is not a number, or names no comment, is not found", func(t *testing.T) {
			postIDs(t, []string{strconv.FormatInt(threeLines.ID, 10), "abc"}, http.StatusNotFound)
			postIDs(t, []string{strconv.FormatInt(threeLines.ID, 10), "99999999"}, http.StatusNotFound)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC14 more than 50 IDs are refused before any is looked up", func(t *testing.T) {
			ids := make([]string, 51)
			for i := range ids {
				ids[i] = "abc" // would be a 404 if looked up
			}
			refusedWith(t, postIDs(t, ids, http.StatusBadRequest), "A batch can hold at most 50 suggestions.")
		})

		t.Run("AC12 an empty batch", func(t *testing.T) {
			refusedWith(t, applyBatch(t, ownerSession, nil, startHead, "", http.StatusBadRequest), "Select at least one suggestion to apply.")
		})

		t.Run("AC13 the same suggestion twice", func(t *testing.T) {
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{threeLines, threeLines}, startHead, "", http.StatusBadRequest)
			refusedWith(t, resp, "A suggestion can only be in the batch once.")
		})

		t.Run("AC14 more than 50 suggestions", func(t *testing.T) {
			many := make([]*issues_model.Comment, 51)
			for i := range many {
				many[i] = threeLines
			}
			refusedWith(t, applyBatch(t, ownerSession, many, startHead, "", http.StatusBadRequest), "A batch can hold at most 50 suggestions.")
		})

		t.Run("AC2 two suggestions on the same line", func(t *testing.T) {
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{threeLines, sameLine}, startHead, "", http.StatusBadRequest)
			refusedWith(t, resp, "This suggestion overlaps another suggestion in this batch.")
		})

		t.Run("AC7 an old head commit", func(t *testing.T) {
			baseHead, err := gitRepo.GetBranchCommitID("master")
			require.NoError(t, err)
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{threeLines, deleteLine}, baseHead, "", http.StatusBadRequest)
			refusedWith(t, resp, "This branch has new commits since the page was loaded. Reload the page and try again.")
		})

		t.Run("AC8 a protected branch", func(t *testing.T) {
			require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, &git_model.ProtectedBranch{
				RepoID:   repo.ID,
				RuleName: "batch-branch",
				CanPush:  false,
			}, git_model.WhitelistOptions{}))
			rule, err := git_model.GetFirstMatchProtectedBranchRule(t.Context(), repo.ID, "batch-branch")
			require.NoError(t, err)
			defer func() { require.NoError(t, git_model.DeleteProtectedBranch(t.Context(), repo, rule.ID)) }()
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{threeLines, deleteLine}, startHead, "", http.StatusBadRequest)
			refusedWith(t, resp, `Cannot commit to protected branch "batch-branch".`)
		})

		coAuthor := fmt.Sprintf("Co-authored-by: %s <%s>", user4.GitName(), user4.GetEmail())

		t.Run("AC1 AC9 one commit applies both: one line becomes three, a line further down is deleted", func(t *testing.T) {
			applyBatch(t, ownerSession, []*issues_model.Comment{deleteLine, threeLines}, startHead, "", http.StatusOK)
			commit := headCommit(t)
			require.Equal(t, 1, commit.ParentCount())
			parent, err := commit.ParentID(0)
			require.NoError(t, err)
			assert.Equal(t, startHead, parent.String())
			assert.Equal(t, "# Title\na\nb\nc\nl3\nl5\nl6\n", readme(t))
			assert.Equal(t, "Apply 2 suggestions from code review\n\n"+coAuthor, strings.TrimSpace(commit.MessageRaw))
			assert.Equal(t, user2.GetEmail(), commit.Author.Email)
			// AC1: each suggestion is marked applied (S4) with that commit
			for _, c := range []*issues_model.Comment{deleteLine, threeLines} {
				stored := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: c.ID})
				assert.Equal(t, commit.ID.String(), stored.SuggestionAppliedCommit)
				assert.Equal(t, user2.ID, stored.SuggestionAppliedByID)
			}
		})

		t.Run("AC10 an applied suggestion in a batch is refused and named", func(t *testing.T) {
			before := headCommit(t).ID.String()
			l5 := suggest(t, 6, "```suggestion\nL5\n```")
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{l5, deleteLine}, before, "", http.StatusBadRequest)
			assert.Equal(t, "This suggestion has already been applied. (README.md:4)", errorMessage(t, resp))
			assert.Equal(t, before, headCommit(t).ID.String())
		})

		t.Run("AC11 a batch of one commits what Apply would, with its default message", func(t *testing.T) {
			before := headCommit(t).ID.String()
			five := suggest(t, 6, "```suggestion\nfive\n```") // "l5" is line 6 now
			applyBatch(t, ownerSession, []*issues_model.Comment{five}, before, "", http.StatusOK)
			assert.Equal(t, "# Title\na\nb\nc\nl3\nfive\nl6\n", readme(t))
			assert.Equal(t, "Apply suggestion from @user4 on README.md\n\n"+coAuthor, strings.TrimSpace(headCommit(t).MessageRaw))
		})

		t.Run("AC3 an outdated suggestion stops the batch and is named", func(t *testing.T) {
			before := headCommit(t).ID.String()
			six := suggest(t, 7, "```suggestion\nsix\n```")                                                                // "l6"
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{six, sameLine}, before, "", http.StatusBadRequest) // line 2 was "l2"
			assert.Equal(t, "This suggestion is outdated: the code changed since it was written. (README.md:2)", errorMessage(t, resp))
			assert.Equal(t, before, headCommit(t).ID.String())
		})

		t.Run("AC6 a closed pull request gets no button and an error", func(t *testing.T) {
			before := headCommit(t).ID.String()
			pending := suggest(t, 7, "```suggestion\nsix // closed\n```")
			require.NoError(t, issue_service.CloseIssue(t.Context(), pr.Issue, user2, ""))
			assert.Equal(t, 0, batchToggles(t, ownerSession))
			resp := applyBatch(t, ownerSession, []*issues_model.Comment{pending}, before, "", http.StatusBadRequest)
			assert.Equal(t, "Suggestions can only be applied to an open pull request whose branch still exists.", errorMessage(t, resp))
			assert.Equal(t, before, headCommit(t).ID.String())
		})
	})
}
