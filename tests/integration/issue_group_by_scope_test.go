// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newExclusiveLabel creates an exclusive scoped label on user2/repo1 and returns its id.
func newExclusiveLabel(t *testing.T, sess *TestSession, name string, order int) int64 {
	t.Helper()
	req := NewRequestWithValues(t, "POST", "/user2/repo1/labels/new", map[string]string{
		"title":           name,
		"color":           "#112233",
		"exclusive":       "on",
		"exclusive_order": strconv.Itoa(order),
	})
	sess.MakeRequest(t, req, http.StatusOK) // NewLabel answers with ctx.JSONRedirect
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{RepoID: 1, Name: name})
	require.True(t, label.Exclusive, "%s must be exclusive, or its scope does not exist", name)
	return label.ID
}

func attachIssueLabel(t *testing.T, sess *TestSession, issueID, labelID int64) {
	t.Helper()
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/labels?issue_ids=%d", issueID), map[string]string{
		"action": "attach",
		"id":     strconv.FormatInt(labelID, 10),
	})
	sess.MakeRequest(t, req, http.StatusOK)
}

// pullListGroups returns, per rendered group, its summary text and the issue indexes inside it.
func pullListGroups(t *testing.T, sess *TestSession, query string) (summaries []string, indexes [][]string) {
	t.Helper()
	req := NewRequest(t, "GET", "/user2/repo1/pulls?"+query)
	resp := sess.MakeRequest(t, req, http.StatusOK)
	NewHTMLParser(t, resp.Body).doc.Find("details.issue-list-group").Each(func(_ int, group *goquery.Selection) {
		summaries = append(summaries, strings.Join(strings.Fields(group.Find("summary").Text()), " "))
		var groupIndexes []string
		group.Find(".item-body .index").Each(func(_ int, index *goquery.Selection) {
			groupIndexes = append(groupIndexes, strings.TrimSpace(index.Text()))
		})
		indexes = append(indexes, groupIndexes)
	})
	return summaries, indexes
}

// TestPullListGroupByLabelScope covers the "folder" view: ?group=<scope> renders the pull request
// list as one collapsible section per label in that exclusive scope, uncategorized last.
func TestPullListGroupByLabelScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	sess := loginUser(t, "user2")
	bugID := newExclusiveLabel(t, sess, "Kind/Bug", 1)
	featureID := newExclusiveLabel(t, sess, "Kind/Feature", 2)

	// repo1's open pull requests are #2, #3 and #5 (issue ids 2, 3 and 11)
	attachIssueLabel(t, sess, 3, bugID)     // #3
	attachIssueLabel(t, sess, 2, featureID) // #2
	// issue id 11 (#5) stays uncategorized

	t.Run("one group per label, uncategorized last", func(t *testing.T) {
		summaries, indexes := pullListGroups(t, sess, "group=Kind")
		require.Len(t, summaries, 3)
		assert.Contains(t, summaries[0], "Bug")
		assert.Contains(t, summaries[1], "Feature")
		assert.Contains(t, summaries[2], "Uncategorized")
		// exclusive order wins over the default newest-first ordering, which would put #5 first
		assert.Equal(t, [][]string{{"#3"}, {"#2"}, {"#5"}}, indexes)
	})

	t.Run("the group label is not repeated on every row in its group", func(t *testing.T) {
		req := NewRequest(t, "GET", "/user2/repo1/pulls?group=Kind")
		resp := sess.MakeRequest(t, req, http.StatusOK)
		// issue 2 (#2) carries label1 and orglabel4 besides Kind/Feature
		feature := NewHTMLParser(t, resp.Body).doc.Find("details.issue-list-group").Eq(1)
		assert.NotContains(t, feature.Find(".labels-list").Text(), "Feature")
		assert.Equal(t, 2, feature.Find(".labels-list .item").Length(), "its other labels still show")
	})

	t.Run("the group by dropdown offers the scope", func(t *testing.T) {
		req := NewRequest(t, "GET", "/user2/repo1/pulls")
		resp := sess.MakeRequest(t, req, http.StatusOK)
		assert.Positive(t, NewHTMLParser(t, resp.Body).doc.Find(`a[href*="group=Kind"]`).Length())
	})

	t.Run("an unknown scope falls back to the flat list", func(t *testing.T) {
		req := NewRequest(t, "GET", "/user2/repo1/pulls?group=NotAScope")
		resp := sess.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		assert.Equal(t, 0, htmlDoc.doc.Find("details.issue-list-group").Length())
		assert.Equal(t, 3, htmlDoc.doc.Find("#issue-list .item-body .index").Length())
	})

	t.Run("sort orders issues within a group", func(t *testing.T) {
		attachIssueLabel(t, sess, 11, bugID) // #5 joins the Kind/Bug folder

		_, indexes := pullListGroups(t, sess, "group=Kind&sort=oldest")
		assert.Equal(t, [][]string{{"#3", "#5"}, {"#2"}}, indexes)

		_, indexes = pullListGroups(t, sess, "group=Kind&sort=latest")
		assert.Equal(t, [][]string{{"#5", "#3"}, {"#2"}}, indexes)
	})
}
