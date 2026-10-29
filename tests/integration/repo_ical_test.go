// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"
	"time"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/test"
	"gitea.dev/modules/timeutil"
	"gitea.dev/routers/web/feed"
	feed_service "gitea.dev/services/feed"
	"gitea.dev/tests"

	ics "github.com/arran4/golang-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoICalFeeds(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer timeutil.MockSet(time.Date(2026, 10, 7, 12, 0, 0, 0, setting.DefaultUILocation))()

	deadline := func(date string) timeutil.TimeStamp {
		day, err := time.ParseInLocation("2006-01-02", date, setting.DefaultUILocation)
		require.NoError(t, err)
		return timeutil.TimeStamp(day.Add(24*time.Hour - time.Second).Unix())
	}
	// public user2/repo1: issue #1 (id 1), pull #2 (id 2), closed issue #4 (id 5); direct DB writes skip the
	// issue indexer, which outlives the test database
	for id, date := range map[int64]string{1: "2026-10-06", 2: "2026-10-30", 5: "2026-10-14"} {
		require.NoError(t, issues_model.UpdateIssueCols(t.Context(), &issues_model.Issue{ID: id, DeadlineUnix: deadline(date)}, "deadline_unix"))
	}
	_, err := db.GetEngine(t.Context()).ID(1).Cols("deadline_unix").Update(&issues_model.Milestone{DeadlineUnix: deadline("2026-10-14")})
	require.NoError(t, err)

	fetch := func(t *testing.T, url string) *ics.Calendar {
		resp := MakeRequest(t, NewRequest(t, "GET", url), http.StatusOK)
		assert.Equal(t, "text/calendar; charset=utf-8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=300", resp.Header().Get("Cache-Control"))
		cal, err := ics.ParseCalendar(resp.Body)
		require.NoError(t, err)
		return cal
	}
	uids := func(cal *ics.Calendar) []string {
		var ids []string
		for _, e := range cal.Events() {
			ids = append(ids, e.Id())
		}
		return ids
	}
	uid := func(kind string, id int64) string { return feed_service.ICalUID(kind, id, setting.Domain) } // the production scheme

	t.Run("milestones with a due date", func(t *testing.T) {
		cal := fetch(t, "/user2/repo1/milestones.ics")
		assert.Equal(t, []string{uid("milestone", 1)}, uids(cal)) // milestone 2 has no date, milestone 3 is closed
		e := cal.Events()[0]
		assert.Equal(t, "milestone1 (user2/repo1)", e.GetProperty(ics.ComponentPropertySummary).Value)
		assert.Equal(t, "20261014", e.GetProperty(ics.ComponentPropertyDtStart).Value)
	})

	t.Run("open issues and pulls by default", func(t *testing.T) {
		cal := fetch(t, "/user2/repo1/issues.ics")
		assert.ElementsMatch(t, []string{uid("issue", 1), uid("issue", 2)}, uids(cal))
	})

	t.Run("state=all adds closed issues", func(t *testing.T) {
		cal := fetch(t, "/user2/repo1/issues.ics?state=all")
		assert.ElementsMatch(t, []string{uid("issue", 1), uid("issue", 2), uid("issue", 5)}, uids(cal))
	})

	t.Run("the cap keeps the closest deadline", func(t *testing.T) {
		defer test.MockVariableValue(&feed.ICalMaxItems, 1)()
		cal := fetch(t, "/user2/repo1/issues.ics")
		assert.Equal(t, []string{uid("issue", 1)}, uids(cal)) // Oct 6 is closer to Oct 7 than Oct 30
	})

	t.Run("a private repo is hidden even from its owner", func(t *testing.T) {
		session := loginUser(t, "user2")
		for _, url := range []string{"/user2/repo2/issues.ics", "/user2/repo2/milestones.ics"} {
			MakeRequest(t, NewRequest(t, "GET", url), http.StatusNotFound)
			session.MakeRequest(t, NewRequest(t, "GET", url), http.StatusNotFound)
		}
	})

	t.Run("disabled feeds", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Other.EnableFeed, false)()
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues.ics"), http.StatusNotFound)
		MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/milestones.ics"), http.StatusNotFound)
	})

	t.Run("subscribe links only where the feed works", func(t *testing.T) {
		for page, link := range map[string]string{
			"/user2/repo1/milestones": "/user2/repo1/milestones.ics",
			"/user2/repo1/issues":     "/user2/repo1/issues.ics",
		} {
			doc := NewHTMLParser(t, MakeRequest(t, NewRequest(t, "GET", page), http.StatusOK).Body)
			assert.Equal(t, 1, doc.Find(`.list-header a[href="`+link+`"]`).Length(), page)
		}
		session := loginUser(t, "user2")
		for _, page := range []string{"/user2/repo2/milestones", "/user2/repo2/issues"} {
			doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", page), http.StatusOK).Body)
			assert.Zero(t, doc.Find(`a[href$=".ics"]`).Length(), page) // the private feed would 404
		}
	})

	t.Run("sign-in required hides both feeds", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.RequireSignInViewStrict, true)()
		session := loginUser(t, "user2") // a signed-in viewer must not get a public-cacheable feed either
		for _, url := range []string{"/user2/repo1/issues.ics", "/user2/repo1/milestones.ics"} {
			session.MakeRequest(t, NewRequest(t, "GET", url), http.StatusNotFound)
		}
		for _, page := range []string{"/user2/repo1/issues", "/user2/repo1/milestones"} {
			doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", page), http.StatusOK).Body)
			assert.Zero(t, doc.Find(`a[href$=".ics"]`).Length(), page)
		}
	})

	t.Run("expensive blocking hides only the issues feed", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.BlockAnonymousAccessExpensive, true)()
		session := loginUser(t, "user2")
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues.ics"), http.StatusNotFound)
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/milestones.ics"), http.StatusOK)
		doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/issues"), http.StatusOK).Body)
		assert.Zero(t, doc.Find(`a[href$="issues.ics"]`).Length())
		doc = NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/milestones"), http.StatusOK).Body)
		assert.Equal(t, 1, doc.Find(`.list-header a[href="/user2/repo1/milestones.ics"]`).Length())
	})

	// must stay last: it deletes repo1's pull request unit
	t.Run("pulls are left out when the pull request unit is off", func(t *testing.T) {
		_, err := db.DeleteByBean(t.Context(), &repo_model.RepoUnit{RepoID: 1, Type: unit.TypePullRequests})
		require.NoError(t, err)
		cal := fetch(t, "/user2/repo1/issues.ics?state=all")
		assert.ElementsMatch(t, []string{uid("issue", 1), uid("issue", 5)}, uids(cal))
	})
}
