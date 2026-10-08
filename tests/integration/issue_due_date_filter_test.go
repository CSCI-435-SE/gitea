// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueDueDateFilter(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))() // a Wednesday

	session := loginUser(t, "user2")
	setDeadline := func(index int64, date string) {
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/%d/deadline", index), map[string]string{"deadline": date})
		session.MakeRequest(t, req, http.StatusOK)
	}
	// user2/repo1 holds issue #1, closed issue #4 and pulls #2, #3 and #5, none with a deadline in the fixtures
	deadlines := map[int64]string{1: "2026-10-06", 2: "2026-10-07", 4: "2026-10-08", 5: "2026-10-30"}
	for index, date := range deadlines {
		setDeadline(index, date)
	}
	defer func() { // the issue indexer outlives the test database, so leave it as the fixtures had it
		for index := range deadlines {
			setDeadline(index, "")
		}
	}()

	list := func(t *testing.T, url string) (doc *HTMLDoc, links []string) {
		t.Helper()
		resp := session.MakeRequest(t, NewRequest(t, "GET", url), http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		doc.Find("#issue-list .list-item-large-title").Each(func(_ int, a *goquery.Selection) {
			links = append(links, a.AttrOr("href", ""))
		})
		return doc, links
	}

	t.Run("repo issues", func(t *testing.T) {
		_, links := list(t, "/user2/repo1/issues?state=all&due=overdue")
		assert.Equal(t, []string{"/user2/repo1/issues/1"}, links)

		_, links = list(t, "/user2/repo1/issues?state=all&due=soon")
		assert.Equal(t, []string{"/user2/repo1/issues/4"}, links, "a closed issue matches by its deadline alone")
	})

	t.Run("repo pulls", func(t *testing.T) {
		for due, expected := range map[string][]string{
			"today": {"/user2/repo1/pulls/2"},
			"week":  {"/user2/repo1/pulls/2"},
			"set":   {"/user2/repo1/pulls/2", "/user2/repo1/pulls/5"},
			"none":  {"/user2/repo1/pulls/3"},
			"bogus": {"/user2/repo1/pulls/2", "/user2/repo1/pulls/3", "/user2/repo1/pulls/5"},
		} {
			_, links := list(t, "/user2/repo1/pulls?due="+due)
			assert.ElementsMatch(t, expected, links, due)
		}

		doc, _ := list(t, "/user2/repo1/pulls?due=set")
		assert.Equal(t, "2 Open", strings.Join(strings.Fields(doc.Find(".small-menu-items .item").First().Text()), " "))
		assert.Equal(t, "Has a due date", strings.TrimSpace(doc.Find(".due-date-filter .active.item").Text()))
		doc.Find(".small-menu-items a").Each(func(_ int, a *goquery.Selection) {
			assert.Contains(t, a.AttrOr("href", ""), "due=set")
		})
		assert.Contains(t, doc.Find(`a[href*="sort=oldest"]`).AttrOr("href", ""), "due=set")
		assert.Equal(t, "set", doc.Find(`.issue-list-search input[name="due"]`).AttrOr("value", ""))
	})

	t.Run("dashboard", func(t *testing.T) {
		doc, links := list(t, "/pulls?due=today")
		assert.Equal(t, []string{"/user2/repo1/pulls/2"}, links)
		assert.Equal(t, 1, doc.Find(".due-date-filter").Length())

		_, links = list(t, "/pulls?due=none")
		assert.Contains(t, links, "/user2/repo1/pulls/3")
		assert.NotContains(t, links, "/user2/repo1/pulls/2")

		// a keyword search filters in the issue indexer, which has to have picked up the new deadline
		require.Eventually(t, func() bool {
			_, links := list(t, "/pulls?q=issue2&due=today")
			return slices.Equal([]string{"/user2/repo1/pulls/2"}, links)
		}, 5*time.Second, 50*time.Millisecond)

		doc, _ = list(t, "/org/org3/issues")
		assert.Equal(t, 1, doc.Find(".due-date-filter").Length())
	})

	t.Run("milestone page ignores the filter", func(t *testing.T) {
		doc, links := list(t, "/user2/repo1/milestone/1?due=none")
		assert.Equal(t, 0, doc.Find(".due-date-filter").Length())
		assert.Equal(t, []string{"/user2/repo1/pulls/2"}, links) // has a deadline, so a filter would have hidden it
	})
}
