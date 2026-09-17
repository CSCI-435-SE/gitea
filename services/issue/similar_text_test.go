// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
)

func Test_tokenizeTitle(t *testing.T) {
	cases := []struct {
		name     string
		title    string
		expected []string
	}{
		{"lowercases and splits", "Login Page Crashes", []string{"login", "page", "crashes"}},
		{"strips punctuation", "login: page crashes!", []string{"login", "page", "crashes"}},
		{"drops stopwords", "the app crashes on login", []string{"app", "crashes", "login"}},
		{"drops short tokens", "an ui bug in the app", []string{"bug", "app"}},
		{"collapses whitespace", "  login\tpage  ", []string{"login", "page"}},
		{"keeps digits", "error 500 on login", []string{"error", "500", "login"}},
		{"empty input", "", nil},
		{"punctuation only", "!!! ??? ...", nil},
		{"stopwords only", "the and or to of", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, tokenizeTitle(c.title))
		})
	}
}

func Test_queryTokens(t *testing.T) {
	cases := []struct {
		name     string
		tokens   []string
		max      int
		expected []string
	}{
		{
			name:     "longest tokens win",
			tokens:   []string{"bug", "authentication", "crashes", "app"},
			max:      3,
			expected: []string{"authentication", "crashes", "bug"},
		},
		{
			name:     "ties keep original order",
			tokens:   []string{"delta", "alpha", "gamma"},
			max:      2,
			expected: []string{"delta", "alpha"},
		},
		{
			name:     "fewer tokens than max",
			tokens:   []string{"login"},
			max:      3,
			expected: []string{"login"},
		},
		{
			name:     "duplicates are removed",
			tokens:   []string{"login", "login", "crash"},
			max:      3,
			expected: []string{"login", "crash"},
		},
		{"no tokens", nil, 3, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, queryTokens(c.tokens, c.max))
		})
	}
}

func Test_titleSimilarity(t *testing.T) {
	t.Run("identical titles score 1", func(t *testing.T) {
		assert.InDelta(t, 1.0, titleSimilarity("login page crashes", "login page crashes"), 0.0001)
	})

	t.Run("stopwords and word forms are ignored", func(t *testing.T) {
		// "the", "on" and "at" drop out, leaving {app, crashes, login} vs {app, crash, login}
		score := titleSimilarity("the app crashes on login", "app crash at login")
		assert.Greater(t, score, 0.5)
		assert.Less(t, score, 1.0)
	})

	t.Run("nothing in common scores 0", func(t *testing.T) {
		assert.Zero(t, titleSimilarity("login page crashes", "update readme formatting"))
	})

	t.Run("closer title scores higher", func(t *testing.T) {
		typed := "login page crashes on submit"
		closer := titleSimilarity(typed, "login page crashes")
		further := titleSimilarity(typed, "login button colour is wrong")
		assert.Greater(t, closer, further)
	})

	t.Run("empty input scores 0 without panicking", func(t *testing.T) {
		assert.Zero(t, titleSimilarity("", "login page crashes"))
		assert.Zero(t, titleSimilarity("login page crashes", ""))
		assert.Zero(t, titleSimilarity("", ""))
		assert.Zero(t, titleSimilarity("!!!", "login"))
	})

	t.Run("duplicate words do not inflate the score", func(t *testing.T) {
		assert.InDelta(t, 1.0, titleSimilarity("login login page", "login page"), 0.0001)
	})
}

func Test_rankBySimilarity(t *testing.T) {
	newIssue := func(id int64, title string, updated int64) *issues_model.Issue {
		return &issues_model.Issue{ID: id, Title: title, UpdatedUnix: timeutil.TimeStamp(updated)}
	}

	t.Run("orders by similarity, not by input order", func(t *testing.T) {
		// Query tokens: {login, page, crashes, submitting}.
		cands := []*issues_model.Issue{
			// Tokens {login, page, redesign, proposal}: shares login+page = 2 of 4, Dice = 2*2/(4+4) = 0.5.
			newIssue(1, "login page redesign proposal", 100),
			// Tokens {login, page, crashes, submit}: shares login+page+crashes = 3 of 4, Dice = 2*3/(4+4) = 0.75.
			newIssue(2, "login page crashes on submit", 100),
		}
		ranked := rankBySimilarity("login page crashes when submitting", cands, 5)
		assert.Len(t, ranked, 2)
		assert.EqualValues(t, 2, ranked[0].ID)
		assert.EqualValues(t, 1, ranked[1].ID)
	})

	t.Run("equal scores break on most recently updated", func(t *testing.T) {
		cands := []*issues_model.Issue{
			newIssue(1, "login page crashes", 100),
			newIssue(2, "login page crashes", 300),
			newIssue(3, "login page crashes", 200),
		}
		ranked := rankBySimilarity("login page crashes", cands, 5)
		assert.Len(t, ranked, 3)
		assert.EqualValues(t, 2, ranked[0].ID)
		assert.EqualValues(t, 3, ranked[1].ID)
		assert.EqualValues(t, 1, ranked[2].ID)
	})

	t.Run("honours the limit", func(t *testing.T) {
		var cands []*issues_model.Issue
		for i := int64(1); i <= 8; i++ {
			cands = append(cands, newIssue(i, "login page crashes", 100+i))
		}
		ranked := rankBySimilarity("login page crashes", cands, 5)
		// All 8 tie on score, so the tie-break decides: newest first, truncated after sorting.
		ids := make([]int64, 0, len(ranked))
		for _, issue := range ranked {
			ids = append(ids, issue.ID)
		}
		assert.Equal(t, []int64{8, 7, 6, 5, 4}, ids)
	})

	t.Run("candidates below the floor are dropped", func(t *testing.T) {
		cands := []*issues_model.Issue{
			newIssue(1, "update readme formatting", 500),
			newIssue(2, "login page crashes", 100),
		}
		// Issue 1 shares no tokens with the query (score 0) and is dropped despite being newer;
		// issue 2 is an exact token match (score 1.0) and is kept. Score beats recency.
		ranked := rankBySimilarity("login page crashes", cands, 5)
		assert.Len(t, ranked, 1)
		assert.EqualValues(t, 2, ranked[0].ID)
	})

	t.Run("a partial match survives the floor while a weak one is dropped", func(t *testing.T) {
		// Query tokens: {login, page, crashes, submit}.
		cands := []*issues_model.Issue{
			// Tokens {login, page, updated, recently}: shares login+page = 2 of 4, Dice = 2*2/(4+4) = 0.5.
			newIssue(1, "login page updated recently", 100),
			// Tokens {login, button, colour, wrong}: shares only login = 1 of 4, Dice = 2*1/(4+4) = 0.25.
			newIssue(2, "login button colour wrong", 100),
		}
		ranked := rankBySimilarity("login page crashes submit", cands, 5)
		assert.Len(t, ranked, 1)
		assert.EqualValues(t, 1, ranked[0].ID)
	})

	t.Run("no candidates returns nothing", func(t *testing.T) {
		assert.Empty(t, rankBySimilarity("login page crashes", nil, 5))
	})
}
