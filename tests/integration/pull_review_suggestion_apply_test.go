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

	"gitea.dev/models/db"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/gitrepo"
	issue_service "gitea.dev/services/issue"
	repo_service "gitea.dev/services/repository"
	files_service "gitea.dev/services/repository/files"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The subtests share one pull request and run in order: each later one starts from the branch the earlier ones left.
func TestPullReviewApplySuggestion(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owns the repository, so can push
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // signed in, cannot push
		repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_apply_suggestion",
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
		readerSession := loginUser(t, "user4")
		testPullCreate(t, ownerSession, "user2", "test_apply_suggestion", false, "master", "suggestion-branch", "Suggestion PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "suggestion-branch"})
		require.NoError(t, pr.LoadIssue(t.Context()))
		pullURL := fmt.Sprintf("/user2/test_apply_suggestion/pulls/%d", pr.Issue.Index)

		gitRepo, err := gitrepo.OpenRepository(t.Context(), repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		headCommit := func(t *testing.T) *git.Commit {
			commit, err := gitRepo.GetBranchCommit("suggestion-branch")
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

		// user4 reviews line 2, as #154's tests do
		suggest := func(t *testing.T, content string) *issues_model.Comment {
			req := NewRequestWithValues(t, "POST", pullURL+"/files/reviews/comments", map[string]string{
				"origin":        "diff",
				"content":       content,
				"side":          "proposed",
				"line":          "2",
				"path":          "README.md",
				"single_review": "true",
			})
			readerSession.MakeRequest(t, req, http.StatusOK)
			return unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pr.IssueID, Type: issues_model.CommentTypeCode, Content: content})
		}
		apply := func(t *testing.T, session *TestSession, comment *issues_model.Comment, head, message string, status int) *httptest.ResponseRecorder {
			req := NewRequestWithValues(t, "POST", fmt.Sprintf("%s/files/reviews/comments/%d/apply", pullURL, comment.ID), map[string]string{
				"head_commit_id": head,
				"message":        message,
			})
			return session.MakeRequest(t, req, status)
		}
		errorMessage := func(t *testing.T, resp *httptest.ResponseRecorder) string {
			return DecodeJSON(t, resp, &struct {
				ErrorMessage string `json:"errorMessage"`
			}{}).ErrorMessage
		}
		applyAttrs := func(t *testing.T, session *TestSession, page string) *HTMLDoc {
			return NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", page), http.StatusOK).Body)
		}

		useNil := suggest(t, "```suggestion\n\treturn nil\n```")
		startHead := headCommit(t).ID.String()

		t.Run("AC1 someone who can push gets Apply with the PR's head and the default message", func(t *testing.T) {
			for _, page := range []string{pullURL + "/files", pullURL} { // Files and Conversation tabs
				markup := applyAttrs(t, ownerSession, page).Find(".render-content.markup[data-suggestion-apply-url]")
				require.Equal(t, 1, markup.Length(), page)
				assert.Equal(t, fmt.Sprintf("%s/files/reviews/comments/%d/apply", pullURL, useNil.ID), markup.AttrOr("data-suggestion-apply-url", ""))
				assert.Equal(t, startHead, markup.AttrOr("data-suggestion-apply-head", ""))
				assert.Equal(t, "Apply suggestion from @user4 on README.md", markup.AttrOr("data-suggestion-apply-message", ""))
			}
		})

		t.Run("AC2 a viewer who cannot push gets no button and a 403", func(t *testing.T) {
			assert.Equal(t, 0, applyAttrs(t, readerSession, pullURL+"/files").Find("[data-suggestion-apply-url]").Length())
			apply(t, readerSession, useNil, startHead, "", http.StatusForbidden)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC2 a signed-out viewer gets no button and is sent to sign in", func(t *testing.T) {
			anonymous := emptyTestSession(t)
			assert.Equal(t, 0, applyAttrs(t, anonymous, pullURL+"/files").Find("[data-suggestion-apply-url]").Length())
			resp := apply(t, anonymous, useNil, startHead, "", http.StatusSeeOther)
			assert.Contains(t, resp.Header().Get("Location"), "/user/login")
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC7 a comment with two suggestion blocks cannot be applied", func(t *testing.T) {
			twoBlocks := suggest(t, "```suggestion\na\n```\nor\n```suggestion\nb\n```")
			apply(t, ownerSession, twoBlocks, startHead, "", http.StatusNotFound)
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC10 a suggestion identical to the line commits nothing", func(t *testing.T) {
			same := suggest(t, "```suggestion\n\treturn err\n```")
			resp := apply(t, ownerSession, same, startHead, "", http.StatusBadRequest)
			assert.Equal(t, "This suggestion makes no changes to the code.", errorMessage(t, resp))
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC4 an old head commit is refused", func(t *testing.T) {
			baseHead, err := gitRepo.GetBranchCommitID("master")
			require.NoError(t, err)
			resp := apply(t, ownerSession, useNil, baseHead, "", http.StatusBadRequest)
			assert.Equal(t, "This branch has new commits since the page was loaded. Reload the page and try again.", errorMessage(t, resp))
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		t.Run("AC6 a protected branch is refused with the web editor's message", func(t *testing.T) {
			require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, &git_model.ProtectedBranch{
				RepoID:   repo.ID,
				RuleName: "suggestion-branch",
				CanPush:  false,
			}, git_model.WhitelistOptions{}))
			rule, err := git_model.GetFirstMatchProtectedBranchRule(t.Context(), repo.ID, "suggestion-branch")
			require.NoError(t, err)
			defer func() { require.NoError(t, git_model.DeleteProtectedBranch(t.Context(), repo, rule.ID)) }()

			resp := apply(t, ownerSession, useNil, startHead, "", http.StatusBadRequest)
			assert.Equal(t, `Cannot commit to protected branch "suggestion-branch".`, errorMessage(t, resp))
			assert.Equal(t, startHead, headCommit(t).ID.String())
		})

		coAuthor := fmt.Sprintf("Co-authored-by: %s <%s>", user4.GitName(), user4.GetEmail())
		var appliedSHA string                                   // the commit that applied useNil
		useZero := suggest(t, "```suggestion\n\treturn 0\n```") // also on "\treturn err", never applied: AC3 below needs it

		t.Run("AC1 AC8 applying adds one commit that changes only that line, with the given message", func(t *testing.T) {
			apply(t, ownerSession, useNil, startHead, "Return nil instead", http.StatusOK)
			commit := headCommit(t)
			require.Equal(t, 1, commit.ParentCount())
			parent, err := commit.ParentID(0)
			require.NoError(t, err)
			assert.Equal(t, startHead, parent.String())
			assert.Equal(t, "# Title\n\treturn nil\n", readme(t))
			assert.Equal(t, "Return nil instead\n\n"+coAuthor, strings.TrimSpace(commit.MessageRaw))
			// customer question 1's default: the person who applied it authors and commits
			assert.Equal(t, user2.GetEmail(), commit.Author.Email)
			assert.Equal(t, user2.GetEmail(), commit.Committer.Email)
			appliedSHA = commit.ID.String()
		})

		t.Run("#157 AC10 AC6 the comment records the commit that applied it, and its conversation stays open", func(t *testing.T) {
			applied := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: useNil.ID})
			assert.Equal(t, user2.ID, applied.SuggestionAppliedByID)
			assert.Equal(t, appliedSHA, applied.SuggestionAppliedCommit)
			assert.NotZero(t, applied.SuggestionAppliedUnix)
			assert.Zero(t, applied.ResolveDoerID) // customer question 3's default: applying doesn't resolve
		})

		// the marker suggestion.ts moves into the header of a comment's suggestion, found beside the comment's raw content
		marker := func(t *testing.T, session *TestSession, page string, comment *issues_model.Comment) *goquery.Selection {
			return applyAttrs(t, session, page).Find(fmt.Sprintf("#issuecomment-%d-raw", comment.ID)).PrevAll().Filter("template.markup-suggestion-applied")
		}
		applyURL := func(comment *issues_model.Comment) string {
			return fmt.Sprintf(`[data-suggestion-apply-url="%s/files/reviews/comments/%d/apply"]`, pullURL, comment.ID)
		}

		t.Run("#157 AC1 every viewer sees who applied it and the commit, and nobody gets Apply", func(t *testing.T) {
			for _, session := range []*TestSession{ownerSession, readerSession, emptyTestSession(t)} {
				for _, page := range []string{pullURL + "/files", pullURL} { // Files and Conversation tabs
					found := marker(t, session, page, useNil)
					require.Equal(t, 1, found.Length(), page)
					assert.Equal(t, "Applied by @user2 in "+appliedSHA[:10], strings.TrimSpace(found.Text()), page)
					assert.Equal(t, 1, found.Find(fmt.Sprintf(`a[href="/user2/test_apply_suggestion/commit/%s"]`, appliedSHA)).Length(), page)
					assert.Equal(t, 0, applyAttrs(t, session, page).Find(applyURL(useNil)).Length(), page)
				}
			}
		})

		t.Run("#157 AC2 applying it again is refused, even for someone who cannot push", func(t *testing.T) {
			before := headCommit(t).ID.String()
			for _, session := range []*TestSession{ownerSession, readerSession} {
				resp := apply(t, session, useNil, before, "", http.StatusBadRequest)
				assert.Equal(t, "This suggestion has already been applied.", errorMessage(t, resp))
			}
			assert.Equal(t, before, headCommit(t).ID.String())
		})

		t.Run("AC1 an empty message gets the default", func(t *testing.T) {
			before := headCommit(t).ID.String()
			addComment := suggest(t, "```suggestion\n\treturn nil // done\n```")
			apply(t, ownerSession, addComment, before, "", http.StatusOK)
			assert.Equal(t, "# Title\n\treturn nil // done\n", readme(t))
			assert.Equal(t, "Apply suggestion from @user4 on README.md\n\n"+coAuthor, strings.TrimSpace(headCommit(t).MessageRaw))
		})

		t.Run("AC3 a suggestion whose line has changed is outdated", func(t *testing.T) {
			before := headCommit(t).ID.String()
			resp := apply(t, ownerSession, useZero, before, "", http.StatusBadRequest) // written against "\treturn err"
			assert.Equal(t, "This suggestion is outdated: the code changed since it was written.", errorMessage(t, resp))
			assert.Equal(t, before, headCommit(t).ID.String())
		})

		t.Run("#157 AC9 editing an applied comment keeps the marker", func(t *testing.T) {
			req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/test_apply_suggestion/comments/%d", useNil.ID), map[string]string{
				"content":         "```suggestion\n\treturn nil // edited\n```",
				"content_version": "0",
			})
			readerSession.MakeRequest(t, req, http.StatusOK)
			assert.Equal(t, 1, marker(t, ownerSession, pullURL+"/files", useNil).Length())
			assert.Equal(t, 0, applyAttrs(t, ownerSession, pullURL+"/files").Find(applyURL(useNil)).Length())
		})

		t.Run("#157 AC4 AC8 a commit that is gone is shown unlinked, and a deleted applier as Ghost", func(t *testing.T) {
			const gone = "0000000000000000000000000000000000000001" // force-pushed away and garbage-collected
			_, err := db.GetEngine(t.Context()).ID(useNil.ID).Cols("suggestion_applied_by_id", "suggestion_applied_commit").
				Update(&issues_model.Comment{SuggestionAppliedByID: 1000, SuggestionAppliedCommit: gone}) // there is no user 1000
			require.NoError(t, err)
			found := marker(t, ownerSession, pullURL, useNil)
			require.Equal(t, 1, found.Length())
			assert.Equal(t, "Applied by Ghost in "+gone[:10], strings.TrimSpace(found.Text()))
			assert.Equal(t, 0, found.Find("a").Length())
		})

		t.Run("#157 AC5 deleting an applied comment keeps the commit and the pages render", func(t *testing.T) {
			before := headCommit(t).ID.String()
			readerSession.MakeRequest(t, NewRequest(t, "POST", fmt.Sprintf("/user2/test_apply_suggestion/comments/%d/delete", useNil.ID)), http.StatusOK)
			unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: useNil.ID})
			for _, page := range []string{pullURL + "/files", pullURL} {
				ownerSession.MakeRequest(t, NewRequest(t, "GET", page), http.StatusOK)
			}
			assert.Equal(t, before, headCommit(t).ID.String())
		})

		t.Run("AC5 a closed pull request gets no button and an error", func(t *testing.T) {
			before := headCommit(t).ID.String()
			pending := suggest(t, "```suggestion\n\treturn nil // closed\n```")
			require.NoError(t, issue_service.CloseIssue(t.Context(), pr.Issue, user2, ""))
			assert.Equal(t, 0, applyAttrs(t, ownerSession, pullURL+"/files").Find("[data-suggestion-apply-url]").Length())
			resp := apply(t, ownerSession, pending, before, "", http.StatusBadRequest)
			assert.Equal(t, "Suggestions can only be applied to an open pull request whose branch still exists.", errorMessage(t, resp))
			assert.Equal(t, before, headCommit(t).ID.String())
		})
	})
}
