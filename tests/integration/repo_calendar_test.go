// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestRepoCalendar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()

	session := loginUser(t, "user2")
	setDeadline := func(index int64, date string) {
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/issues/%d/deadline", index), map[string]string{"deadline": date})
		session.MakeRequest(t, req, http.StatusOK)
	}
	// user2/repo1 holds issue #1 and pull #2, neither with a deadline in the fixtures
	deadlines := map[int64]string{1: "2026-10-06", 2: "2026-10-30"}
	for index, date := range deadlines {
		setDeadline(index, date)
	}
	defer func() { // the issue indexer outlives the test database, so leave it as the fixtures had it
		for index := range deadlines {
			setDeadline(index, "")
		}
	}()

	t.Run("shows issues and pulls on their days", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		var links []string
		doc.Find(".repo-calendar-day:not(.is-other-month) .repo-calendar-entry").Each(func(_ int, a *goquery.Selection) {
			links = append(links, a.AttrOr("href", ""))
		})
		assert.ElementsMatch(t, []string{"/user2/repo1/issues/1", "/user2/repo1/pulls/2"}, links)
		// a timezone bug would shift an entry into the neighbouring day cell
		assert.Equal(t, 1, doc.Find(`.repo-calendar-day:has(time[datetime="2026-10-06"]) a.repo-calendar-entry[href="/user2/repo1/issues/1"]`).Length())
		assert.Equal(t, 1, doc.Find(`.repo-calendar-day:has(time[datetime="2026-10-30"]) a.repo-calendar-entry[href="/user2/repo1/pulls/2"]`).Length())
		assert.Equal(t, 1, doc.Find(".repo-calendar-entry.is-pull").Length())
		assert.Equal(t, 1, doc.Find(".repo-calendar-day.is-today").Length())
	})

	t.Run("a crowded day folds extra entries into +N more", func(t *testing.T) {
		crowded := []int64{1, 2, 4, 5} // issue 1, pull 2, closed issue 4, pull 5
		for _, index := range crowded {
			setDeadline(index, "2026-10-14")
		}
		defer func() {
			for _, index := range crowded {
				setDeadline(index, "")
			}
			setDeadline(1, deadlines[1]) // back to the shared setup
			setDeadline(2, deadlines[2])
		}()
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar?state=all"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		cell := doc.Find(`.repo-calendar-day:has(time[datetime="2026-10-14"])`)
		assert.Equal(t, 3, cell.ChildrenFiltered("a.repo-calendar-entry").Length())
		more := cell.ChildrenFiltered("details.repo-calendar-more")
		assert.Equal(t, 1, more.Length())
		assert.Equal(t, 1, more.Find(".repo-calendar-more-list a.repo-calendar-entry").Length())
	})

	t.Run("month navigation keeps the filters", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar?month=2026-09&state=all"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		assert.Zero(t, doc.Find(".repo-calendar-day:not(.is-other-month) .repo-calendar-entry").Length())
		next, _ := doc.Find(`.list-header a[href*="month=2026-10"]`).First().Attr("href") // the next-month arrow
		assert.Contains(t, next, "month=2026-10")
		assert.Contains(t, next, "state=all")
		assert.Equal(t, 1, doc.Find(".repo-calendar-empty").Length())
	})

	t.Run("the issue list links to the calendar", func(t *testing.T) {
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		assert.Equal(t, 1, doc.Find(`.list-header a[href="/user2/repo1/calendar"]`).Length())
	})

	t.Run("anonymous users", func(t *testing.T) {
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/calendar"), http.StatusOK)
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo2/calendar"), http.StatusNotFound) // private
	})
}
