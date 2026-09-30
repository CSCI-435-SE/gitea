// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/indexer/issues"
	"gitea.dev/modules/references"
	"gitea.dev/modules/setting"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/test"
	issue_service "gitea.dev/services/issue"
	repo_service "gitea.dev/services/repository"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getIssuesSelection(t testing.TB, htmlDoc *HTMLDoc) *goquery.Selection {
	issueList := htmlDoc.doc.Find("#issue-list")
	assert.Equal(t, 1, issueList.Length())
	return issueList.Find(".item").Find(".list-item-large-title")
}

func getIssue(t *testing.T, repoID int64, issueSelection *goquery.Selection) *issues_model.Issue {
	href, exists := issueSelection.Attr("href")
	assert.True(t, exists)
	indexStr := href[strings.LastIndexByte(href, '/')+1:]
	index, err := strconv.Atoi(indexStr)
	assert.NoError(t, err, "Invalid issue href: %s", href)
	return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repoID, Index: int64(index)})
}

func assertMatch(t testing.TB, issue *issues_model.Issue, keyword string) {
	matches := strings.Contains(strings.ToLower(issue.Title), keyword) ||
		strings.Contains(strings.ToLower(issue.Content), keyword)
	for _, comment := range issue.Comments {
		matches = matches || strings.Contains(
			strings.ToLower(comment.Content),
			keyword,
		)
	}
	assert.True(t, matches)
}

func TestNoLoginViewIssues(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/issues")
	MakeRequest(t, req, http.StatusOK)
}

func TestViewIssuesSortByType(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	session := loginUser(t, user.Name)
	req := NewRequest(t, "GET", repo.Link()+"/issues?type=created_by")
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	issuesSelection := getIssuesSelection(t, htmlDoc)
	expectedNumIssues := min(unittest.GetCount(t,
		&issues_model.Issue{RepoID: repo.ID, PosterID: user.ID},
		unittest.Cond("is_closed=?", false),
		unittest.Cond("is_pull=?", false),
	), setting.UI.IssuePagingNum)
	assert.Equal(t, expectedNumIssues, issuesSelection.Length())

	issuesSelection.Each(func(_ int, selection *goquery.Selection) {
		issue := getIssue(t, repo.ID, selection)
		assert.Equal(t, user.ID, issue.PosterID)
	})
}

func TestViewIssuesKeyword(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{
		RepoID: repo.ID,
		Index:  1,
	})
	issues.UpdateIssueIndexer(t.Context(), issue.ID)
	time.Sleep(time.Second * 1)
	const keyword = "first"
	req := NewRequestf(t, "GET", "%s/issues?q=%s", repo.Link(), keyword)
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	issuesSelection := getIssuesSelection(t, htmlDoc)
	assert.Equal(t, 1, issuesSelection.Length())
	issuesSelection.Each(func(_ int, selection *goquery.Selection) {
		issue := getIssue(t, repo.ID, selection)
		assert.False(t, issue.IsClosed)
		assert.False(t, issue.IsPull)
		assertMatch(t, issue, keyword)
	})
}

func TestNoLoginViewIssue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/issues/1")
	MakeRequest(t, req, http.StatusOK)
}

func testNewIssue(t *testing.T, session *TestSession, user, repo, title, content string) string {
	req := NewRequest(t, "GET", "/"+path.Join(user, repo, "issues", "new"))
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"title":   title,
		"content": content,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)

	issueURL := test.RedirectURL(resp)
	req = NewRequest(t, "GET", issueURL)
	resp = session.MakeRequest(t, req, http.StatusOK)

	htmlDoc = NewHTMLParser(t, resp.Body)
	val := htmlDoc.doc.Find("#issue-title-display").Text()
	assert.Contains(t, val, title)
	val = htmlDoc.doc.Find(".comment .render-content p").First().Text()
	assert.Equal(t, content, val)

	return issueURL
}

func testIssueDelete(t *testing.T, session *TestSession, issueURL string) {
	req := NewRequest(t, "POST", path.Join(issueURL, "delete"))
	session.MakeRequest(t, req, http.StatusSeeOther)
}

func testIssueAssign(t *testing.T, session *TestSession, repoLink string, issueID, assigneeID int64) {
	req := NewRequestWithValues(t, "POST", fmt.Sprintf(repoLink+"/issues/assignee?issue_ids=%d", issueID), map[string]string{
		"id":     strconv.FormatInt(assigneeID, 10),
		"action": "", // empty action means assign
	})
	session.MakeRequest(t, req, http.StatusOK)
}

func testIssueAddComment(t *testing.T, session *TestSession, issueURL, content, status string) int64 {
	req := NewRequest(t, "GET", issueURL)
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("#comment-form").Attr("action")
	assert.True(t, exists, "The template has changed")

	commentCount := htmlDoc.doc.Find(".comment-list .comment .render-content").Length()

	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"content": content,
		"status":  status,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", test.RedirectURL(resp))
	resp = session.MakeRequest(t, req, http.StatusOK)

	htmlDoc = NewHTMLParser(t, resp.Body)

	val := strings.TrimSpace(htmlDoc.doc.Find(".comment-list .comment .render-content").Eq(commentCount).Text())
	assert.Equal(t, content, val)

	idAttr, has := htmlDoc.doc.Find(".comment-list .comment").Eq(commentCount).Attr("id")
	idStr := idAttr[strings.LastIndexByte(idAttr, '-')+1:]
	assert.True(t, has)
	id, err := strconv.Atoi(idStr)
	assert.NoError(t, err)
	return int64(id)
}

func testIssueChangeMilestone(t *testing.T, session *TestSession, repoLink string, issueID, milestoneID int64) {
	req := NewRequestWithValues(t, "POST", fmt.Sprintf(repoLink+"/issues/milestone?issue_ids=%d", issueID), map[string]string{
		"id": strconv.FormatInt(milestoneID, 10),
	})
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, `{"ok":true}`, strings.TrimSpace(resp.Body.String()))
}

func TestNewIssue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	testNewIssue(t, session, "user2", "repo1", "Title", "Description")
}

func TestEditIssue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")

	req := NewRequestWithValues(t, "POST", issueURL+"/content", map[string]string{
		"content": "modified content",
		"context": fmt.Sprintf("/%s/%s", "user2", "repo1"),
	})
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequestWithValues(t, "POST", issueURL+"/content", map[string]string{
		"content": "modified content",
		"context": fmt.Sprintf("/%s/%s", "user2", "repo1"),
	})
	session.MakeRequest(t, req, http.StatusBadRequest)

	req = NewRequestWithValues(t, "POST", issueURL+"/content", map[string]string{
		"content":         "modified content",
		"content_version": "1",
		"context":         fmt.Sprintf("/%s/%s", "user2", "repo1"),
	})
	session.MakeRequest(t, req, http.StatusOK)
}

func TestIssueCommentClose(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")
	testIssueAddComment(t, session, issueURL, "Test comment 1", "")
	testIssueAddComment(t, session, issueURL, "Test comment 2", "")
	testIssueAddComment(t, session, issueURL, "Test comment 3", "close")

	// Validate that issue content has not been updated
	req := NewRequest(t, "GET", issueURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	val := htmlDoc.doc.Find(".comment-list .comment .render-content p").First().Text()
	assert.Equal(t, "Description", val)
}

func TestIssueCommentCloseWithReason(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

	// closeWithReason opens a new issue, posts the close button with the given form fields and expects status
	closeWithReason := func(t *testing.T, fields map[string]string, status int) (*issues_model.Issue, string) {
		issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")
		resp := session.MakeRequest(t, NewRequest(t, "GET", issueURL), http.StatusOK)
		link, exists := NewHTMLParser(t, resp.Body).doc.Find("#comment-form").Attr("action")
		require.True(t, exists, "The template has changed")

		fields["status"] = "close"
		resp = session.MakeRequest(t, NewRequestWithValues(t, "POST", link, fields), status)

		index, err := strconv.ParseInt(path.Base(issueURL), 10, 64)
		require.NoError(t, err)
		return unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: index}), resp.Body.String()
	}

	t.Run("not planned", func(t *testing.T) {
		issue, _ := closeWithReason(t, map[string]string{"close_reason": "not_planned"}, http.StatusOK)
		assert.True(t, issue.IsClosed)
		assert.Equal(t, issues_model.CloseReasonNotPlanned, issue.CloseReason)
	})

	t.Run("other with text", func(t *testing.T) {
		issue, _ := closeWithReason(t, map[string]string{"close_reason": "other", "close_reason_text": "superseded"}, http.StatusOK)
		assert.True(t, issue.IsClosed)
		assert.Equal(t, "superseded", issue.CloseReasonText)
	})

	t.Run("duplicate", func(t *testing.T) {
		target := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
		issue, _ := closeWithReason(t, map[string]string{"close_reason": "duplicate", "close_duplicate_index": "1"}, http.StatusOK)
		assert.True(t, issue.IsClosed)
		assert.Equal(t, target.ID, issue.CloseDuplicateIssueID)
		session.MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/user2/repo1/issues/%d", issue.Index)), http.StatusOK) // the page loads the target for the close comment
	})

	t.Run("refused reason leaves the issue open", func(t *testing.T) {
		issue, _ := closeWithReason(t, map[string]string{"close_reason": "other"}, http.StatusOK)
		assert.False(t, issue.IsClosed)
		assert.Equal(t, `The description is not valid. It is required for "Other", can be up to 255 characters, and is only used with "Other".`, session.GetCookieFlashMessage().ErrorMsg)

		issue, _ = closeWithReason(t, map[string]string{"close_reason": "duplicate", "close_duplicate_index": "9999"}, http.StatusOK)
		assert.False(t, issue.IsClosed)
		assert.Equal(t, `The duplicate must be another issue or pull request in this repository, and a number is only used with "Duplicate".`, session.GetCookieFlashMessage().ErrorMsg)

		issue, _ = closeWithReason(t, map[string]string{"close_reason": "bogus"}, http.StatusOK)
		assert.False(t, issue.IsClosed)
		assert.Equal(t, "This close reason is not available for this issue or pull request.", session.GetCookieFlashMessage().ErrorMsg)
	})

	t.Run("too long a description is refused by the form", func(t *testing.T) {
		issue, body := closeWithReason(t, map[string]string{"close_reason": "other", "close_reason_text": strings.Repeat("a", 256)}, http.StatusBadRequest)
		assert.False(t, issue.IsClosed)
		assert.Contains(t, body, "Close reason description must contain at most 255 characters.")
	})
}

func TestIssueCloseReasonMenu(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	// the button texts each menu item hands to the button when it is picked
	itemTexts := map[string][2]string{
		"completed":   {"Close as completed", "Close as completed with comment"},
		"not_planned": {"Close as not planned", "Close as not planned with comment"},
		"duplicate":   {"Close as duplicate", "Close as duplicate with comment"},
		"other":       {"Close with other reason", "Close with other reason and comment"},
	}

	// closeFromPage checks the close button and its menu, then closes with the reason the page itself sends
	closeFromPage := func(t *testing.T, link, wantText, wantTextWithComment string, wantReasons, wantLabels []string) {
		htmlDoc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", link), http.StatusOK).Body)
		button := htmlDoc.doc.Find("#comment-form .ui.buttons #status-button")
		assert.Equal(t, wantText, strings.TrimSpace(button.Find(".status-button-text").Text()))
		assert.Equal(t, wantTextWithComment, button.AttrOr("data-status-and-comment", ""))

		var reasons, labels []string
		htmlDoc.doc.Find("#comment-form .ui.buttons .menu .item").Each(func(_ int, item *goquery.Selection) {
			reason := item.AttrOr("data-value", "")
			reasons = append(reasons, reason)
			labels = append(labels, strings.TrimSpace(item.Text())) // a missing locale key would show as the key
			rendered := [2]string{item.AttrOr("data-status", ""), item.AttrOr("data-status-and-comment", "")}
			assert.Equal(t, itemTexts[reason], rendered, "reason %q", reason)
			assert.True(t, item.HasClass("js-aria-clickable"), "reason %q: without it, Enter does not pick the item", reason)
		})
		assert.Equal(t, wantReasons, reasons)
		assert.Equal(t, wantLabels, labels)

		// what duplicate and other need is typed in their popups into fields that are switched off, so not sent, until picked
		assert.Equal(t, 1, htmlDoc.doc.Find(`#comment-form input[type="hidden"][name="close_duplicate_index"][disabled]`).Length())
		assert.Equal(t, 1, htmlDoc.doc.Find(`#comment-form input[type="hidden"][name="close_reason_text"][disabled]`).Length())
		duplicatePopup := htmlDoc.doc.Find(`#comment-form [data-close-reason-popup="duplicate"]`)
		assert.Equal(t, "Close as duplicate of #%s", duplicatePopup.AttrOr("data-locale-status", ""))
		assert.Equal(t, "Close as duplicate of #%s with comment", duplicatePopup.AttrOr("data-locale-status-and-comment", ""))
		assert.Equal(t, 1, duplicatePopup.Find(`.field label[for="close-duplicate-index"] + input#close-duplicate-index[type="number"][min="1"]`).Length(), "in a field, so an unusable number can be marked as an error")
		preview := duplicatePopup.Find(`[data-close-duplicate-preview]`)
		assert.Equal(t, "No #%s found in this repository", preview.AttrOr("data-locale-not-found", ""))
		assert.Equal(t, "Can't be a duplicate of itself", preview.AttrOr("data-locale-self", ""))
		assert.Equal(t, 1, htmlDoc.doc.Find(`#comment-form [data-close-reason-popup="other"] .field input[type="text"][maxlength="255"]`).Length())

		reason, exists := htmlDoc.doc.Find(`#comment-form input[name="close_reason"]`).Attr("value")
		require.True(t, exists, "The template has changed")
		action, exists := htmlDoc.doc.Find("#comment-form").Attr("action")
		require.True(t, exists, "The template has changed")
		session.MakeRequest(t, NewRequestWithValues(t, "POST", action, map[string]string{"status": "close", "close_reason": reason}), http.StatusOK)
	}

	t.Run("an issue starts on completed", func(t *testing.T) {
		closeFromPage(t, "/user2/repo1/issues/1", "Close as completed", "Close as completed with comment",
			[]string{"completed", "not_planned", "duplicate", "other"}, []string{"Completed", "Not planned", "Duplicate", "Other"})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
		assert.True(t, issue.IsClosed)
		assert.Equal(t, issues_model.CloseReasonCompleted, issue.CloseReason)
	})

	t.Run("a pull request starts on not planned and cannot be completed", func(t *testing.T) {
		closeFromPage(t, "/user2/repo1/pulls/3", "Close as not planned", "Close as not planned with comment",
			[]string{"not_planned", "duplicate", "other"}, []string{"Not planned", "Duplicate", "Other"})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
		assert.True(t, issue.IsClosed)
		assert.Equal(t, issues_model.CloseReasonNotPlanned, issue.CloseReason)
	})

	t.Run("a closed issue only offers reopening", func(t *testing.T) {
		htmlDoc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues/4"), http.StatusOK).Body)
		assert.Equal(t, "Reopen Issue", strings.TrimSpace(htmlDoc.doc.Find("#status-button .status-button-text").Text()))
		assert.Zero(t, htmlDoc.doc.Find("#comment-form .ui.buttons .ui.dropdown").Length())
		assert.Zero(t, htmlDoc.doc.Find(`#comment-form input[name="close_reason"]`).Length())
		assert.Zero(t, htmlDoc.doc.Find(`#comment-form [data-close-reason-popup], #comment-form input[data-close-reason]`).Length())
	})
}

func TestIssueTimelineCloseReason(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

	// closeWith opens a new issue and closes it through the page's form; closeEntry reads its close event from a fresh page
	closeWith := func(t *testing.T, fields map[string]string) (issue *issues_model.Issue, closeEntry func() *goquery.Selection) {
		issueURL := testNewIssue(t, session, "user2", "repo1", "Timeline check", "Description")
		action, exists := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", issueURL), http.StatusOK).Body).doc.Find("#comment-form").Attr("action")
		require.True(t, exists, "The template has changed")
		fields["status"] = "close"
		session.MakeRequest(t, NewRequestWithValues(t, "POST", action, fields), http.StatusOK)

		index, err := strconv.ParseInt(path.Base(issueURL), 10, 64)
		require.NoError(t, err)
		issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: index})
		require.True(t, issue.IsClosed)
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: issue.ID, Type: issues_model.CommentTypeClose})
		return issue, func() *goquery.Selection {
			return NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", issueURL), http.StatusOK).Body).doc.Find("#" + comment.HashTag())
		}
	}
	mainLine := func(entry *goquery.Selection) string {
		return strings.Join(strings.Fields(entry.Find(".comment-text-line").First().Text()), " ")
	}

	t.Run("each reason names itself, in bold like a lock reason", func(t *testing.T) {
		for reason, want := range map[string]string{"completed": "completed", "not_planned": "not planned"} {
			_, closeEntry := closeWith(t, map[string]string{"close_reason": reason})
			entry := closeEntry()
			assert.Contains(t, mainLine(entry), "closed this as "+want)
			assert.Equal(t, want, entry.Find(".comment-text-line strong").Text())
			assert.Zero(t, entry.Find(".detail").Length())
		}
	})

	t.Run("a close without a reason reads as before", func(t *testing.T) {
		_, closeEntry := closeWith(t, map[string]string{})
		entry := closeEntry()
		assert.Contains(t, mainLine(entry), "closed this issue")
		assert.Zero(t, entry.Find(".comment-text-line strong, .detail").Length())
	})

	t.Run("a duplicate links to the original on its own line", func(t *testing.T) {
		_, closeEntry := closeWith(t, map[string]string{"close_reason": "duplicate", "close_duplicate_index": "1"})
		entry := closeEntry()
		assert.Contains(t, mainLine(entry), "closed this as a duplicate")
		link := entry.Find(".detail a")
		assert.Equal(t, "/user2/repo1/issues/1", link.AttrOr("href", ""))
		assert.Equal(t, "#1 issue1", strings.Join(strings.Fields(link.Text()), " "))
	})

	t.Run("other text is shown as plain text", func(t *testing.T) {
		text := "<b>not bold</b> **not bold either**"
		_, closeEntry := closeWith(t, map[string]string{"close_reason": "other", "close_reason_text": text})
		entry := closeEntry()
		assert.Contains(t, mainLine(entry), "closed this")
		assert.Equal(t, text, entry.Find(".detail .comment-text-line").Text())
		assert.Zero(t, entry.Find(".detail b, .detail strong").Length(), "neither HTML nor Markdown is rendered")
	})

	t.Run("reopening keeps the reason on the earlier close", func(t *testing.T) {
		issue, closeEntry := closeWith(t, map[string]string{"close_reason": "not_planned"})
		session.MakeRequest(t, NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/%d/comments", issue.Index), map[string]string{"status": "reopen"}), http.StatusOK)
		assert.False(t, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issue.ID}).IsClosed)
		assert.Contains(t, mainLine(closeEntry()), "closed this as not planned")
	})

	t.Run("a deleted original is left out", func(t *testing.T) {
		targetURL := testNewIssue(t, session, "user2", "repo1", "Original", "Description")
		targetIndex := path.Base(targetURL)
		_, closeEntry := closeWith(t, map[string]string{"close_reason": "duplicate", "close_duplicate_index": targetIndex})
		index, err := strconv.ParseInt(targetIndex, 10, 64)
		require.NoError(t, err)
		target := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: index})
		require.NoError(t, issue_service.DeleteIssue(t.Context(), unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}), target))

		entry := closeEntry() // the page still loads
		assert.Contains(t, mainLine(entry), "closed this as a duplicate")
		assert.Zero(t, entry.Find(".detail").Length())
	})

	t.Run("an original the viewer can't read is left out", func(t *testing.T) {
		_, closeEntry := closeWith(t, map[string]string{"close_reason": "duplicate", "close_duplicate_index": "3"}) // #3 is a pull request
		assert.Equal(t, 1, closeEntry().Find(".detail a").Length())
		require.NoError(t, repo_service.UpdateRepositoryUnits(t.Context(), repo, nil, []unit.Type{unit.TypePullRequests}))

		entry := closeEntry()
		assert.Contains(t, mainLine(entry), "closed this as a duplicate")
		assert.Zero(t, entry.Find(".detail").Length())
	})
}

func TestIssueCommentDelete(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")
	comment1 := "Test comment 1"
	commentID := testIssueAddComment(t, session, issueURL, comment1, "")
	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: commentID})
	assert.Equal(t, comment1, comment.Content)

	// Using the ID of a comment that does not belong to the repository must fail
	req := NewRequest(t, "POST", fmt.Sprintf("/%s/%s/comments/%d/delete", "user5", "repo4", commentID))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "POST", fmt.Sprintf("/%s/%s/comments/%d/delete", "user2", "repo1", commentID))
	session.MakeRequest(t, req, http.StatusOK)
	unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: commentID})
}

func TestIssueCommentUpdate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")
	comment1 := "Test comment 1"
	commentID := testIssueAddComment(t, session, issueURL, comment1, "")

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: commentID})
	assert.Equal(t, comment1, comment.Content)

	modifiedContent := comment.Content + "MODIFIED"

	// Using the ID of a comment that does not belong to the repository must fail
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/comments/%d", "user5", "repo4", commentID), map[string]string{
		"content": modifiedContent,
	})
	session.MakeRequest(t, req, http.StatusNotFound)

	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/comments/%d", "user2", "repo1", commentID), map[string]string{
		"content": modifiedContent,
	})
	session.MakeRequest(t, req, http.StatusOK)

	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: commentID})
	assert.Equal(t, modifiedContent, comment.Content)
}

func TestIssueCommentUpdateSimultaneously(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")
	comment1 := "Test comment 1"
	commentID := testIssueAddComment(t, session, issueURL, comment1, "")

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: commentID})
	assert.Equal(t, comment1, comment.Content)

	modifiedContent := comment.Content + "MODIFIED"

	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/comments/%d", "user2", "repo1", commentID), map[string]string{
		"content": modifiedContent,
	})
	session.MakeRequest(t, req, http.StatusOK)

	modifiedContent = comment.Content + "2"

	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/comments/%d", "user2", "repo1", commentID), map[string]string{
		"content": modifiedContent,
	})
	session.MakeRequest(t, req, http.StatusBadRequest)

	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/comments/%d", "user2", "repo1", commentID), map[string]string{
		"content":         modifiedContent,
		"content_version": "1",
	})
	session.MakeRequest(t, req, http.StatusOK)

	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: commentID})
	assert.Equal(t, modifiedContent, comment.Content)
	assert.Equal(t, 2, comment.ContentVersion)
}

func TestIssueReaction(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	issueURL := testNewIssue(t, session, "user2", "repo1", "Title", "Description")

	req := NewRequestWithValues(t, "POST", path.Join(issueURL, "/reactions/react"), map[string]string{
		"content": "8ball",
	})
	session.MakeRequest(t, req, http.StatusInternalServerError)
	req = NewRequestWithValues(t, "POST", path.Join(issueURL, "/reactions/react"), map[string]string{
		"content": "eyes",
	})
	session.MakeRequest(t, req, http.StatusOK)
	req = NewRequestWithValues(t, "POST", path.Join(issueURL, "/reactions/unreact"), map[string]string{
		"content": "eyes",
	})
	session.MakeRequest(t, req, http.StatusOK)
}

func TestIssueCrossReference(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Issue that will be referenced
	_, issueBase := testIssueWithBean(t, "user2", 1, "Title", "Description")

	// Ref from issue title
	issueRefURL, issueRef := testIssueWithBean(t, "user2", 1, fmt.Sprintf("Title ref #%d", issueBase.Index), "Description")
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    1,
		RefIssueID:   issueRef.ID,
		RefCommentID: 0,
		RefIsPull:    false,
		RefAction:    references.XRefActionNone,
	})

	// Edit title, neuter ref
	testIssueChangeInfo(t, "user2", issueRefURL, "title", "Title no ref")
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    1,
		RefIssueID:   issueRef.ID,
		RefCommentID: 0,
		RefIsPull:    false,
		RefAction:    references.XRefActionNeutered,
	})

	// Ref from issue content
	issueRefURL, issueRef = testIssueWithBean(t, "user2", 1, "TitleXRef", fmt.Sprintf("Description ref #%d", issueBase.Index))
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    1,
		RefIssueID:   issueRef.ID,
		RefCommentID: 0,
		RefIsPull:    false,
		RefAction:    references.XRefActionNone,
	})

	// Edit content, neuter ref
	testIssueChangeInfo(t, "user2", issueRefURL, "content", "Description no ref")
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    1,
		RefIssueID:   issueRef.ID,
		RefCommentID: 0,
		RefIsPull:    false,
		RefAction:    references.XRefActionNeutered,
	})

	// Ref from a comment
	session := loginUser(t, "user2")
	commentID := testIssueAddComment(t, session, issueRefURL, fmt.Sprintf("Adding ref from comment #%d", issueBase.Index), "")
	comment := &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    1,
		RefIssueID:   issueRef.ID,
		RefCommentID: commentID,
		RefIsPull:    false,
		RefAction:    references.XRefActionNone,
	}
	unittest.AssertExistsAndLoadBean(t, comment)

	// Ref from a different repository
	_, issueRef = testIssueWithBean(t, "user12", 10, "TitleXRef", fmt.Sprintf("Description ref user2/repo1#%d", issueBase.Index))
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:      issueBase.ID,
		RefRepoID:    10,
		RefIssueID:   issueRef.ID,
		RefCommentID: 0,
		RefIsPull:    false,
		RefAction:    references.XRefActionNone,
	})
}

func testIssueWithBean(t *testing.T, user string, repoID int64, title, content string) (string, *issues_model.Issue) {
	session := loginUser(t, user)
	issueURL := testNewIssue(t, session, user, fmt.Sprintf("repo%d", repoID), title, content)
	indexStr := issueURL[strings.LastIndexByte(issueURL, '/')+1:]
	index, err := strconv.Atoi(indexStr)
	assert.NoError(t, err, "Invalid issue href: %s", issueURL)
	issue := &issues_model.Issue{RepoID: repoID, Index: int64(index)}
	unittest.AssertExistsAndLoadBean(t, issue)
	return issueURL, issue
}

func testIssueChangeInfo(t *testing.T, user, issueURL, info, value string) {
	session := loginUser(t, user)
	req := NewRequestWithValues(t, "POST", path.Join(issueURL, info), map[string]string{
		info: value,
	})
	_ = session.MakeRequest(t, req, http.StatusOK)
}

func TestIssueRedirect(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	// Test external tracker where style not set (shall default numeric)
	req := NewRequest(t, "GET", "/org26/repo_external_tracker/issues/1")
	resp := session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "https://tracker.com/org26/repo_external_tracker/issues/1", test.RedirectURL(resp))

	// Test external tracker with numeric style
	req = NewRequest(t, "GET", "/org26/repo_external_tracker_numeric/issues/1")
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "https://tracker.com/org26/repo_external_tracker_numeric/issues/1", test.RedirectURL(resp))

	// Test external tracker with alphanumeric style (for a pull request)
	req = NewRequest(t, "GET", "/org26/repo_external_tracker_alpha/issues/1")
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/org26/repo_external_tracker_alpha/pulls/1", test.RedirectURL(resp))

	// test to check that the PR redirection works if the issue unit is disabled
	// repo1 is a normal repository with issue unit enabled, visit issue 2(which is a pull request)
	// will redirect to pulls
	req = NewRequest(t, "GET", "/user2/repo1/issues/2")
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/user2/repo1/pulls/2", test.RedirectURL(resp))

	// by the way, test PR redirection
	req = NewRequest(t, "GET", "/user2/repo1/pulls/1")
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/user2/repo1/issues/1", test.RedirectURL(resp))

	// disable issue unit
	repoUnit := unittest.AssertExistsAndLoadBean(t, &repo_model.RepoUnit{RepoID: 1, Type: unit.TypeIssues})
	_, err := db.DeleteByID[repo_model.RepoUnit](t.Context(), repoUnit.ID)
	assert.NoError(t, err)

	// even if the issue unit is disabled, visiting an issue which is a pull request
	// will still redirect to pull request
	req = NewRequest(t, "GET", "/user2/repo1/issues/2")
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/user2/repo1/pulls/2", test.RedirectURL(resp))
}

func TestSearchIssues(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	expectedIssueCount := min(20, setting.UI.IssuePagingNum) // 20 is from the fixtures

	link, _ := url.Parse("/issues/search")
	req := NewRequest(t, "GET", link.String())
	resp := session.MakeRequest(t, req, http.StatusOK)
	apiIssues := DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, expectedIssueCount)

	since := "2000-01-01T00:50:01+00:00" // 946687801
	before := time.Unix(999307200, 0).Format(time.RFC3339)
	query := url.Values{}
	query.Add("since", since)
	query.Add("before", before)
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 11)
	query.Del("since")
	query.Del("before")

	query.Add("state", "closed")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	query.Set("state", "all")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Equal(t, "22", resp.Header().Get("X-Total-Count"))
	assert.Len(t, apiIssues, 20)

	query.Add("limit", "5")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Equal(t, "22", resp.Header().Get("X-Total-Count"))
	assert.Len(t, apiIssues, 5)

	query = url.Values{"assigned": {"true"}, "state": {"all"}}
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	query = url.Values{"milestones": {"milestone1"}, "state": {"all"}}
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 1)

	query = url.Values{"milestones": {"milestone1,milestone3"}, "state": {"all"}}
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	query = url.Values{"owner": {"user2"}} // user
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 8)

	query = url.Values{"owner": {"org3"}} // organization
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 5)

	query = url.Values{"owner": {"org3"}, "team": {"team1"}} // organization + team
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)
}

func TestSearchIssuesWithLabels(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	expectedIssueCount := min(20, setting.UI.IssuePagingNum) // 20 is from the fixtures

	session := loginUser(t, "user1")
	link, _ := url.Parse("/issues/search")
	query := url.Values{}
	var apiIssues []*api.Issue

	link.RawQuery = query.Encode()
	req := NewRequest(t, "GET", link.String())
	resp := session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, expectedIssueCount)

	query.Add("labels", "label1")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	// multiple labels
	query.Set("labels", "label1,label2")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	// an org label
	query.Set("labels", "orglabel4")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 1)

	// org and repo label
	query.Set("labels", "label2,orglabel4")
	query.Add("state", "all")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)

	// org and repo label which share the same issue
	query.Set("labels", "label1,orglabel4")
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = session.MakeRequest(t, req, http.StatusOK)
	apiIssues = DecodeJSON(t, resp, []*api.Issue{})
	assert.Len(t, apiIssues, 2)
}

func TestGetIssueInfo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	assert.NoError(t, issue.LoadAttributes(t.Context()))
	assert.Equal(t, int64(1019307200), int64(issue.DeadlineUnix))
	assert.Equal(t, api.StateOpen, issue.State())

	session := loginUser(t, owner.Name)

	urlStr := fmt.Sprintf("/%s/%s/issues/%d/info", owner.Name, repo.Name, issue.Index)
	req := NewRequest(t, "GET", urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)
	respStruct := DecodeJSON(t, resp, &struct {
		ConvertedIssue api.Issue
		RenderedLabels template.HTML
	}{})

	assert.Equal(t, issue.ID, respStruct.ConvertedIssue.ID)
	assert.Contains(t, string(respStruct.RenderedLabels), `"labels-list"`)
}

func TestUpdateIssueDeadline(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issueBefore := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
	repoBefore := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issueBefore.RepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repoBefore.OwnerID})
	assert.NoError(t, issueBefore.LoadAttributes(t.Context()))
	assert.Equal(t, "2002-04-20", issueBefore.DeadlineUnix.FormatDate())
	assert.Equal(t, api.StateOpen, issueBefore.State())

	session := loginUser(t, owner.Name)
	urlStr := fmt.Sprintf("/%s/%s/issues/%d/deadline", owner.Name, repoBefore.Name, issueBefore.Index)

	req := NewRequestWithValues(t, "POST", urlStr, map[string]string{"deadline": "2022-04-06"})
	session.MakeRequest(t, req, http.StatusOK)
	issueAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
	assert.Equal(t, "2022-04-06", issueAfter.DeadlineUnix.FormatDate())

	req = NewRequestWithValues(t, "POST", urlStr, map[string]string{"deadline": ""})
	session.MakeRequest(t, req, http.StatusOK)
	issueAfter = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
	assert.True(t, issueAfter.DeadlineUnix.IsZero())
}

func TestIssueDueDateHighlight(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	deadlineURL := fmt.Sprintf("/%s/%s/issues/%d/deadline", owner.Name, repo.Name, issue.Index)
	issueURL := fmt.Sprintf("/%s/%s/issues/%d", owner.Name, repo.Name, issue.Index)

	setDeadlineAndGetDueDate := func(t *testing.T, deadline string) *goquery.Selection {
		req := NewRequestWithValues(t, "POST", deadlineURL, map[string]string{"deadline": deadline})
		session.MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", issueURL)
		resp := session.MakeRequest(t, req, http.StatusOK)
		return NewHTMLParser(t, resp.Body).Find(".due-date")
	}

	t.Run("near due", func(t *testing.T) {
		tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		dueDate := setDeadlineAndGetDueDate(t, tomorrow)
		assert.True(t, dueDate.HasClass("tw-text-warning-text"))
		assert.False(t, dueDate.HasClass("tw-text-red"))
		tooltip, _ := dueDate.Attr("data-tooltip-content")
		assert.Equal(t, "Due soon", tooltip)
	})

	t.Run("overdue", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		dueDate := setDeadlineAndGetDueDate(t, yesterday)
		assert.True(t, dueDate.HasClass("tw-text-red"))
		assert.False(t, dueDate.HasClass("tw-text-warning-text"))
		tooltip, _ := dueDate.Attr("data-tooltip-content")
		assert.Equal(t, "Overdue", tooltip)
	})

	t.Run("far future", func(t *testing.T) {
		nextMonth := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
		dueDate := setDeadlineAndGetDueDate(t, nextMonth)
		assert.False(t, dueDate.HasClass("tw-text-red"))
		assert.False(t, dueDate.HasClass("tw-text-warning-text"))
	})

	// leave the issue without a deadline for other tests relying on fixture state
	req := NewRequestWithValues(t, "POST", deadlineURL, map[string]string{"deadline": ""})
	session.MakeRequest(t, req, http.StatusOK)
}

func TestUpdateIssueRefByPoster(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user4 is a non-admin, non-collaborator on user2/repo1.
	// They create an issue, making them the poster.
	posterSession := loginUser(t, "user4")
	issueURL := testNewIssue(t, posterSession, "user2", "repo1", "Poster ref test", "body")
	refURL := issueURL + "/ref"

	// The poster (non-collaborator) must be able to update the ref.
	req := NewRequestWithValues(t, "POST", refURL, map[string]string{"ref": "refs/heads/main"})
	posterSession.MakeRequest(t, req, http.StatusOK)

	// A different non-collaborator non-poster must be forbidden.
	otherSession := loginUser(t, "user5")
	req = NewRequestWithValues(t, "POST", refURL, map[string]string{"ref": "refs/heads/main"})
	otherSession.MakeRequest(t, req, http.StatusForbidden)
}

func TestIssueRefSelectorEnabledForPoster(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user4 creates an issue in user2/repo1 (user4 has no write permission there).
	posterSession := loginUser(t, "user4")
	issueURL := testNewIssue(t, posterSession, "user2", "repo1", "Ref selector test", "body")

	resp := posterSession.MakeRequest(t, NewRequest(t, "GET", issueURL), http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// The branch selector must not carry the "disabled" CSS class for the poster.
	sel := htmlDoc.Find(".branch-selector-dropdown")
	assert.Equal(t, 1, sel.Length())
	assert.False(t, sel.HasClass("disabled"), "branch selector should be enabled for the issue poster")
	// The update-ref URL must be present so JS can send the POST request.
	_, hasURL := sel.Attr("data-url-update-issueref")
	assert.True(t, hasURL, "data-url-update-issueref must be set for the issue poster")
}

func TestIssueRefSelectorEnabledForNewIssue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user4 (non-collaborator on user2/repo1) must see an enabled ref selector
	// when creating a new issue.
	session := loginUser(t, "user4")
	req := NewRequest(t, "GET", "/user2/repo1/issues/new")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	sel := htmlDoc.Find(".branch-selector-dropdown")
	assert.Equal(t, 1, sel.Length())
	assert.False(t, sel.HasClass("disabled"), "branch selector should be enabled on the new issue form")
}

func TestIssueReferenceURL(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})

	req := NewRequest(t, "GET", fmt.Sprintf("%s/issues/%d", repo.Link(), issue.Index))
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// the "reference" uses relative URLs, then JS code will convert them to absolute URLs for current origin, in case users are using multiple domains
	ref, _ := htmlDoc.Find(`.timeline-item.comment.issue-content-comment .reference-issue`).Attr("data-reference")
	assert.Equal(t, "/user2/repo1/issues/1#issue-1", ref)

	ref, _ = htmlDoc.Find(`.timeline-item.comment:not(.issue-content-comment) .reference-issue`).Attr("data-reference")
	assert.Equal(t, "/user2/repo1/issues/1#issuecomment-2", ref)
}
