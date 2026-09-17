// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"
	"time"

	repo_service "gitea.dev/services/repository"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoActivityIssuesChart(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("NavbarAndPage", func(t *testing.T) {
		req := NewRequest(t, "GET", "/user2/repo1/activity/issues")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		// the chart is mounted by web_src/js/features/issues-chart.ts on this id
		assert.Equal(t, 1, htmlDoc.doc.Find("#repo-issues-chart").Length())

		link := htmlDoc.doc.Find(`.ui.vertical.menu a[href="/user2/repo1/activity/issues"]`)
		require.Equal(t, 1, link.Length())
		assert.True(t, link.HasClass("active"))
	})

	t.Run("Data", func(t *testing.T) {
		req := NewRequest(t, "GET", "/user2/repo1/activity/issues/data")
		resp := MakeRequest(t, req, http.StatusOK)
		weeks := DecodeJSON(t, resp, []repo_service.IssueWeek{})

		require.NotEmpty(t, weeks)
		// repo1 has two issues and several pull requests; only the issues may be counted
		var opened, closed int
		for _, week := range weeks {
			opened += week.Opened
			closed += week.Closed
		}
		assert.Equal(t, 2, opened)
		assert.Equal(t, 1, closed)
		assert.Equal(t, 1, weeks[len(weeks)-1].Open)

		// weeks are contiguous and start on a Sunday, so the chart has no gaps
		for i, week := range weeks {
			require.Equal(t, time.Sunday, time.UnixMilli(week.Week).UTC().Weekday(), "week %d does not start on a Sunday", i)
			if i > 0 {
				require.Equal(t, int64(7*24*60*60*1000), week.Week-weeks[i-1].Week, "gap before week %d", i)
			}
		}
	})

	t.Run("RepoWithoutIssues", func(t *testing.T) {
		// user5/repo4 has the issues unit but no issues, so the endpoint answers null
		req := NewRequest(t, "GET", "/user5/repo4/activity/issues/data")
		resp := MakeRequest(t, req, http.StatusOK)
		assert.JSONEq(t, "null", resp.Body.String())
	})

	t.Run("IssuesUnitDisabled", func(t *testing.T) {
		// user13/repo11 enables the code unit only
		req := NewRequest(t, "GET", "/user13/repo11/activity/issues")
		MakeRequest(t, req, http.StatusNotFound)

		req = NewRequest(t, "GET", "/user13/repo11/activity/issues/data")
		MakeRequest(t, req, http.StatusNotFound)

		req = NewRequest(t, "GET", "/user13/repo11/activity")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		assert.Equal(t, 0, htmlDoc.doc.Find(`a[href="/user13/repo11/activity/issues"]`).Length())
	})

	t.Run("PrivateRepoHidden", func(t *testing.T) {
		// user2/repo2 is private and does have issues, so a signed-out reader may not count them
		req := NewRequest(t, "GET", "/user2/repo2/activity/issues/data")
		MakeRequest(t, req, http.StatusNotFound)
	})
}
