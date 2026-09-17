# Similar Issue Suggestions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show the 5 most similar existing issues under the title input while someone writes a new issue (or pull request), so duplicates are noticed before they are filed.

**Architecture:** A new `services/issue/similar.go` turns the typed title into tokens, asks the existing issue indexer for candidates one token at a time, unions them, and re-ranks by a pure Dice-coefficient title similarity. A new web route `GET /{owner}/{repo}/issues/similar` exposes it as JSON. A small TypeScript module debounces keystrokes on `#issue_title` and renders the results into a panel declared in `new_form.tmpl`.

**Tech Stack:** Go 1.x, XORM, Gitea's issue indexer (`modules/indexer/issues`, bleve by default), Go templates, TypeScript with Vite, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-16-similar-issue-suggestions-design.md`

---

## Background the engineer needs

Read these before starting. They are short.

- `.claude-students/docs/services-issue.md` — what belongs in `services/issue` vs `models/issues`.
- `.claude-students/docs/routers-web.md` — how web routes and their guard chains are declared.
- `.claude-students/docs/frontend-js.md` — the `data-global-init` pattern used to attach behaviour to DOM.

Three facts that shape the whole design:

1. **The indexer's keyword match is AND, not OR.** The default bleve backend sets
   `q.Operator = query.MatchQueryOperatorAnd` (`modules/indexer/internal/bleve/query.go:37`), so
   every word of a keyword must appear in the document. Passing a whole sentence as one keyword
   returns almost nothing. That is why this plan searches one token at a time and unions the
   results.
2. **The indexer cannot sort by relevance.** Every backend forces a date sort when `SortBy` is
   empty (`modules/indexer/issues/bleve/bleve.go:302`). Ranking therefore happens in Go, after
   retrieval.
3. **Outside a running server, the indexer is a dummy that errors.** `globalIndexer` starts as
   `dummyIndexer`, whose `Search` returns `"indexer is not ready"`
   (`modules/indexer/issues/internal/indexer.go:46`). So `FindSimilarIssues` cannot be unit
   tested — all testable logic lives in pure functions, and the wiring is covered by the e2e
   test in Task 8.

## File structure

**Create:**

| File | Responsibility |
| --- | --- |
| `services/issue/similar_text.go` | Pure text functions: tokenize, pick query tokens, score similarity, rank candidates. No context, no DB. |
| `services/issue/similar_text_test.go` | Unit tests for every function in `similar_text.go`. |
| `services/issue/similar.go` | `FindSimilarIssues`: indexer retrieval + projection to API structs. Thin. |
| `routers/web/repo/issue_similar.go` | HTTP handler: parse params, clamp permissions, emit JSON. |
| `web_src/js/features/repo-issue-similar.ts` | Debounce, fetch, render, hide. |
| `web_src/js/features/repo-issue-similar.test.ts` | Vitest coverage of the render/hide logic. |
| `web_src/css/repo/issue-similar.css` | Panel styling. |
| `tests/e2e/issue-similar.test.ts` | End-to-end: type a title, see the panel. |

**Modify:**

| File | Change |
| --- | --- |
| `routers/web/web.go:1318` | Register the new route beside `/issues/suggestions`. |
| `templates/repo/issue/new_form.tmpl:19` | Add the panel element after the title field. |
| `options/locale/locale_en-US.json` | Two new strings. |
| `web_src/js/index.ts` | Import and call the new init function. |
| `web_src/css/index.css` | Import the new stylesheet. |

The text functions are split into their own file deliberately: they are the only part that can
be unit tested, and keeping them free of `context.Context` and database handles is what makes
that true. Do not merge them into `similar.go`.

## Conventions this repo enforces

- New `.go` files start with a copyright header carrying the current year (2026).
- No trailing whitespace anywhere.
- Conventional Commits for every commit; `test` type for test-only commits.
- Add `Assisted-by: Claude Code:claude-opus-5` as a trailer on each commit.
- **Never commit without explicit human approval.** Each "Commit" step below means: show the
  user the command and wait. Do not run it yourself unless they say to.
- In TypeScript use `!` rather than `?.`/`??` when a value is known to exist.
- For CSS layout prefer `flex-*` helpers over per-child `tw-ml-*`/`tw-mr-*`.

---

### Task 1: Tokenizing a title

**Files:**
- Create: `services/issue/similar_text.go`
- Create: `services/issue/similar_text_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/issue/similar_text_test.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

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
		{"empty input", "", nil},
		{"punctuation only", "!!! ??? ...", nil},
		{"stopwords only", "the and or to of", nil},
		{"keeps digits", "error 500 on login", []string{"error", "500", "login"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, tokenizeTitle(c.title))
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test -run '^Test_tokenizeTitle$' ./services/issue/
```

Expected: FAIL — `undefined: tokenizeTitle`.

- [ ] **Step 3: Write the implementation**

Create `services/issue/similar_text.go`:

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"gitea.dev/modules/container"
)

// similarityStopwords are dropped before two titles are compared: they appear in almost every
// issue title and so carry no signal about what the issue is about. Stopwords of two characters
// or fewer need no entry here, because the length filter already removes them.
var similarityStopwords = container.SetOf(
	"the", "and", "for", "with", "are", "when", "while", "not", "this", "that",
)

// tokenizeTitle lowercases a title, splits it on anything that is not a letter or digit, and
// drops stopwords and tokens of two characters or fewer. It returns nil when nothing is left.
func tokenizeTitle(title string) []string {
	fields := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var tokens []string
	for _, f := range fields {
		// Count runes, not bytes, so a short word in a non-Latin script is treated like a short
		// word in English.
		if utf8.RuneCountInString(f) <= 2 || similarityStopwords.Contains(f) {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test -run '^Test_tokenizeTitle$' ./services/issue/
```

Expected: PASS (9 subtests).

- [ ] **Step 5: Format and lint**

```bash
gofmt -w services/issue/similar_text.go services/issue/similar_text_test.go
```

Expected: no output. (`make fmt` and `make lint-go` are the project-wide equivalents; `make`
may not be installed locally, in which case run `gofmt` directly as above.)

- [ ] **Step 6: Commit** — show this to the user and wait for approval

```bash
git add services/issue/similar_text.go services/issue/similar_text_test.go && git commit -m "feat(issue): add title tokenizer for similarity matching

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: Choosing the query tokens

Only the most distinctive tokens are sent to the indexer, because each one costs a query.
Longest-first is the heuristic: longer words are rarer and more specific than short ones.

**Files:**
- Modify: `services/issue/similar_text.go`
- Modify: `services/issue/similar_text_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/issue/similar_text_test.go`:

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test -run '^Test_queryTokens$' ./services/issue/
```

Expected: FAIL — `undefined: queryTokens`.

- [ ] **Step 3: Write the implementation**

Add to `services/issue/similar_text.go` (and add `"sort"` to the import block):

```go
// queryTokens picks at most limit distinct tokens to send to the indexer, longest first: longer
// words are rarer, so they make better candidate filters. Each token costs one indexer query,
// which is why the count is capped. Length is counted in runes, so a short word in a non-Latin
// script is not mistaken for a long one.
func queryTokens(tokens []string, limit int) []string {
	// Built in input order, so sort.SliceStable keeps equal-length tokens in the order they were
	// typed without any explicit tie-break.
	var distinct []string
	seen := make(container.Set[string], len(tokens))
	for _, t := range tokens {
		if seen.Add(t) {
			distinct = append(distinct, t)
		}
	}

	sort.SliceStable(distinct, func(i, j int) bool {
		// Count runes, not bytes, so a short word in a non-Latin script is not mistaken for a long one.
		return utf8.RuneCountInString(distinct[i]) > utf8.RuneCountInString(distinct[j])
	})

	if limit < len(distinct) {
		distinct = distinct[:max(limit, 0)] // max() so a negative limit cannot panic the slice
	}
	return distinct
}
```

Note: `container.Set[T].Add` returns true when the value was not already present
(`modules/container/set.go:19`), which is what makes the dedupe one line. Keep `var distinct
[]string` rather than `make([]string, 0, n)`: the "no tokens" case asserts a nil result, and
`assert.Equal` distinguishes nil from an empty non-nil slice.

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test -run '^Test_queryTokens$' ./services/issue/
```

Expected: PASS (5 subtests).

- [ ] **Step 5: Commit** — show this to the user and wait for approval

```bash
git add services/issue/similar_text.go services/issue/similar_text_test.go && git commit -m "feat(issue): pick distinctive query tokens for similarity search

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: Scoring two titles

Dice coefficient over token sets: `2 * |A and B| / (|A| + |B|)`. It ranges from 0 (nothing in
common) to 1 (identical token sets).

**Files:**
- Modify: `services/issue/similar_text.go`
- Modify: `services/issue/similar_text_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/issue/similar_text_test.go`:

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test -run '^Test_titleSimilarity$' ./services/issue/
```

Expected: FAIL — `undefined: titleSimilarity`.

- [ ] **Step 3: Write the implementation**

Add to `services/issue/similar_text.go`:

```go
// titleSimilarity scores how alike two titles are, from 0 (nothing in common) to 1 (same token
// set), using the Dice coefficient over their token sets.
func titleSimilarity(a, b string) float64 {
	setA := container.SetOf(tokenizeTitle(a)...)
	setB := container.SetOf(tokenizeTitle(b)...)
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}

	shared := 0
	for token := range setA {
		if setB.Contains(token) {
			shared++
		}
	}
	return 2 * float64(shared) / float64(len(setA)+len(setB))
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test -run '^Test_titleSimilarity$' ./services/issue/
```

Expected: PASS (6 subtests).

If "stopwords and word forms are ignored" fails, check the expectation arithmetic: the token
sets are `{app, crashes, login}` and `{app, crash, login}`, sharing 2 of 3 each, so the score is
`2*2/6 = 0.667`, which satisfies `> 0.5` and `< 1.0`.

- [ ] **Step 5: Commit** — show this to the user and wait for approval

```bash
git add services/issue/similar_text.go services/issue/similar_text_test.go && git commit -m "feat(issue): score title similarity with a Dice coefficient

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: Ranking candidates

**Files:**
- Modify: `services/issue/similar_text.go`
- Modify: `services/issue/similar_text_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/issue/similar_text_test.go` (and add these imports to the file's import
block: `issues_model "gitea.dev/models/issues"` and `"gitea.dev/modules/timeutil"`):

```go
func Test_rankBySimilarity(t *testing.T) {
	newIssue := func(id int64, title string, updated int64) *issues_model.Issue {
		return &issues_model.Issue{ID: id, Title: title, UpdatedUnix: timeutil.TimeStamp(updated)}
	}

	t.Run("orders by similarity, not by input order", func(t *testing.T) {
		cands := []*issues_model.Issue{
			newIssue(1, "login button colour is wrong", 100),
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

	t.Run("zero-scoring candidates rank last but are kept", func(t *testing.T) {
		cands := []*issues_model.Issue{
			newIssue(1, "update readme formatting", 500),
			newIssue(2, "login page crashes", 100),
		}
		ranked := rankBySimilarity("login page crashes", cands, 5)
		assert.Len(t, ranked, 2)
		assert.EqualValues(t, 2, ranked[0].ID)
		assert.EqualValues(t, 1, ranked[1].ID)
	})

	t.Run("no candidates returns nothing", func(t *testing.T) {
		assert.Empty(t, rankBySimilarity("login page crashes", nil, 5))
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test -run '^Test_rankBySimilarity$' ./services/issue/
```

Expected: FAIL — `undefined: rankBySimilarity`.

- [ ] **Step 3: Write the implementation**

Add to `services/issue/similar_text.go` (and add `issues_model "gitea.dev/models/issues"` to
its import block):

```go
// rankBySimilarity orders candidates by how closely their titles match the typed title, most
// similar first, and returns at most limit of them. Equal scores are broken by the most
// recently updated issue, which is the more likely duplicate of the two.
//
// Candidates that score zero are kept rather than dropped: the indexer already decided they
// match, so one that matched on its body instead of its title simply sorts last and only
// surfaces when there are fewer than limit title matches.
func rankBySimilarity(title string, cands []*issues_model.Issue, limit int) []*issues_model.Issue {
	type scored struct {
		issue *issues_model.Issue
		score float64
	}

	scoredCands := make([]scored, 0, len(cands))
	for _, c := range cands {
		scoredCands = append(scoredCands, scored{issue: c, score: titleSimilarity(title, c.Title)})
	}

	// Truncation mirrors queryTokens, so both "sorted list, capped by limit" functions read alike.
	sort.SliceStable(scoredCands, func(i, j int) bool {
		if scoredCands[i].score != scoredCands[j].score {
			return scoredCands[i].score > scoredCands[j].score
		}
		return scoredCands[i].issue.UpdatedUnix > scoredCands[j].issue.UpdatedUnix
	})

	result := make([]*issues_model.Issue, 0, len(scoredCands))
	for _, s := range scoredCands {
		result = append(result, s.issue)
	}

	if limit < len(result) {
		result = result[:max(limit, 0)] // max() so a negative limit cannot panic the slice
	}
	return result
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test -run '^Test_rankBySimilarity$' ./services/issue/
```

Expected: PASS (5 subtests).

- [ ] **Step 5: Run the whole package's similarity tests together**

```bash
go test -run '^Test_(tokenizeTitle|queryTokens|titleSimilarity|rankBySimilarity)$' ./services/issue/
```

Expected: PASS, well under a second. No database is touched by any of these.

- [ ] **Step 6: Commit** — show this to the user and wait for approval

```bash
git add services/issue/similar_text.go services/issue/similar_text_test.go && git commit -m "feat(issue): rank similar-issue candidates by title similarity

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: The service function

This is the part that talks to the indexer. It has no unit test, on purpose — see fact 3 in the
background section. Keep it thin; if you find yourself adding logic here, it belongs in
`similar_text.go` where it can be tested.

**Files:**
- Create: `services/issue/similar.go`

- [ ] **Step 1: Write the implementation**

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/modules/container"
	"gitea.dev/modules/indexer"
	issue_indexer "gitea.dev/modules/indexer/issues"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/structs"
)

const (
	// similarMinTitleLen is the shortest title worth searching for: below this the results are
	// noise and every keystroke would cost queries.
	similarMinTitleLen = 3
	// similarMaxQueryTokens caps how many indexer queries one lookup costs.
	similarMaxQueryTokens = 3
	// similarCandidatesPerToken is the page size of each per-token indexer query.
	similarCandidatesPerToken = 20
	// similarMaxCandidates caps the unioned candidate pool that gets re-ranked.
	similarMaxCandidates = 60
)

// FindSimilarIssues returns up to limit issues in the repository whose titles are most similar
// to the given title, most similar first.
//
// Candidates come from the issue indexer, one query per distinctive token, because the default
// bleve backend requires every term of a keyword to match: a whole sentence as a single keyword
// would match nothing. The union is then re-ranked in Go, because no indexer backend can sort
// by relevance.
func FindSimilarIssues(ctx context.Context, repo *repo_model.Repository, isPull optional.Option[bool], title string, limit int) ([]*structs.Issue, error) {
	tokens := queryTokens(tokenizeTitle(title), similarMaxQueryTokens)
	if len(title) < similarMinTitleLen || len(tokens) == 0 {
		return []*structs.Issue{}, nil
	}

	seen := make(container.Set[int64], similarMaxCandidates)
	candidateIDs := make([]int64, 0, similarMaxCandidates)
	for _, token := range tokens {
		ids, _, err := issue_indexer.SearchIssues(ctx, &issue_indexer.SearchOptions{
			Keyword:    token,
			RepoIDs:    []int64{repo.ID},
			IsPull:     isPull,
			SearchMode: indexer.SearchModeFuzzy,
			SortBy:     issue_indexer.SortByUpdatedDesc,
			Paginator:  &db.ListOptions{Page: 1, PageSize: similarCandidatesPerToken},
		})
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if len(candidateIDs) >= similarMaxCandidates {
				break
			}
			if seen.Add(id) {
				candidateIDs = append(candidateIDs, id)
			}
		}
	}

	if len(candidateIDs) == 0 {
		return []*structs.Issue{}, nil
	}

	candidates, err := issues_model.GetIssuesByIDs(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}
	// HTMLURL dereferences issue.Repo, which neither GetIssuesByIDs nor LoadPullRequests loads.
	if _, err := candidates.LoadRepositories(ctx); err != nil {
		return nil, err
	}
	if err := candidates.LoadPullRequests(ctx); err != nil {
		return nil, err
	}

	ranked := rankBySimilarity(title, candidates, limit)

	results := make([]*structs.Issue, 0, len(ranked))
	for _, issue := range ranked {
		result := &structs.Issue{
			ID:      issue.ID,
			Index:   issue.Index,
			Title:   issue.Title,
			State:   issue.State(),
			HTMLURL: issue.HTMLURL(ctx),
		}
		if issue.IsPull && issue.PullRequest != nil {
			result.PullRequest = &structs.PullRequestMeta{
				HasMerged:        issue.PullRequest.HasMerged,
				IsWorkInProgress: issue.PullRequest.IsWorkInProgress(ctx),
			}
		}
		results = append(results, result)
	}
	return results, nil
}
```

This mirrors the projection in `services/issue/suggestion.go:53-68`; read that file alongside
this one. Two details verified against source rather than assumed: `Issue.HTMLURL` takes a
`context.Context` (`models/issues/issue.go:392`), and it dereferences `issue.Repo`, which
neither `GetIssuesByIDs` nor `LoadPullRequests` populates — only `IssueList.LoadRepositories`
does (`models/issues/issue_list.go:36`, which assigns `issue.Repo` and returns
`(RepositoryList, error)`). Without that call this nil-panics on the first request.

- [ ] **Step 2: Verify it compiles**

```bash
go build ./services/issue/
```

Expected: no output.

- [ ] **Step 3: Verify the existing package tests still pass**

```bash
go test ./services/issue/
```

Expected: PASS. This runs `Test_Suggestion` and friends, which use the test database via
`services/issue/main_test.go`.

- [ ] **Step 4: Format**

```bash
gofmt -w services/issue/similar.go
```

Expected: no output.

- [ ] **Step 5: Commit** — show this to the user and wait for approval

```bash
git add services/issue/similar.go && git commit -m "feat(issue): add FindSimilarIssues backed by the issue indexer

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: The web route

**Files:**
- Create: `routers/web/repo/issue_similar.go`
- Modify: `routers/web/web.go:1318`

- [ ] **Step 1: Write the handler**

Create `routers/web/repo/issue_similar.go`. Read
`routers/web/repo/issue_suggestions.go` first — this deliberately follows its shape, including
how it decides what the viewer may read.

```go
// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"gitea.dev/models/unit"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/structs"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
)

// similarIssuesLimit is how many suggestions the new-issue form shows.
const similarIssuesLimit = 5

// SimilarIssues returns issues whose titles resemble the title being typed on the new
// issue/pull request form, so the author can spot a duplicate before filing one.
func SimilarIssues(ctx *context.Context) {
	wantPull := ctx.FormBool("is_pull")

	// Only search what this viewer may read: someone who cannot see pull requests must not
	// learn about them through suggestions. An empty list rather than an error keeps the panel
	// invisible instead of surfacing a failure on a form that is working fine.
	canRead := ctx.Repo.Permission.CanRead(unit.TypeIssues)
	if wantPull {
		canRead = ctx.Repo.Permission.CanRead(unit.TypePullRequests)
	}
	if !canRead {
		ctx.JSON(http.StatusOK, []*structs.Issue{})
		return
	}

	similar, err := issue_service.FindSimilarIssues(ctx, ctx.Repo.Repository,
		optional.Some(wantPull), ctx.FormString("q"), similarIssuesLimit)
	if err != nil {
		ctx.ServerError("FindSimilarIssues", err)
		return
	}

	ctx.JSON(http.StatusOK, similar)
}
```

- [ ] **Step 2: Register the route**

In `routers/web/web.go`, find this group (around line 1313-1320):

```go
	m.Group("/{username}/{reponame}", func() {
		m.Get("/comments/{id}/attachments", repo.GetCommentAttachments)
		m.Get("/labels", repo.RetrieveLabelsForList, repo.Labels)
		m.Get("/milestones", repo.Milestones)
		m.Get("/milestone/{id}", repo.MilestoneIssuesAndPulls)
		m.Get("/issues/suggestions", repo.IssueSuggestions)
	}, optSignIn, context.RepoAssignment, reqRepoIssuesOrPullsReader)
```

Add one line after the suggestions route:

```go
		m.Get("/issues/similar", repo.SimilarIssues)
```

The guard chain (`optSignIn`, `context.RepoAssignment`, `reqRepoIssuesOrPullsReader`) is
inherited from the group — do not add your own.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./routers/...
```

Expected: no output.

- [ ] **Step 4: Verify the route by hand**

Start the server from the repo root (this works because the work dir is the repo, so assets
resolve on disk):

```bash
go build -o gitea.exe && ./gitea.exe web
```

Then in another shell, against any repo you can read (replace owner/repo):

```bash
curl -s 'http://localhost:3000/owner/repo/issues/similar?q=login%20page%20crashes&is_pull=false'
```

Expected: a JSON array, `[]` or up to 5 objects each with `id`, `number`, `title`, `state` and
`html_url`. A 404 means the route did not register; a 500 means the indexer errored — check the
server log.

- [ ] **Step 5: Commit** — show this to the user and wait for approval

```bash
git add routers/web/repo/issue_similar.go routers/web/web.go && git commit -m "feat(issue): add the similar issues web route

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 7: The form panel, strings and browser behaviour

**Files:**
- Modify: `templates/repo/issue/new_form.tmpl` (after the title field, around line 19)
- Modify: `options/locale/locale_en-US.json`
- Create: `web_src/js/features/repo-issue-similar.ts`
- Create: `web_src/js/features/repo-issue-similar.test.ts`
- Create: `web_src/css/repo/issue-similar.css`
- Modify: `web_src/js/index.ts`
- Modify: `web_src/css/index.css`

- [ ] **Step 1: Add the two strings**

In `options/locale/locale_en-US.json`, find the `repo.issues.*` keys and add, in alphabetical
position among its neighbours:

```json
  "repo.issues.similar_issues": "Similar issues",
  "repo.issues.similar_pulls": "Similar pull requests",
```

Match the surrounding key style exactly — open the file and copy the shape of an adjacent
`repo.issues.` entry rather than assuming flat or nested keys.

- [ ] **Step 2: Add the panel to the form**

In `templates/repo/issue/new_form.tmpl`, the title field block ends with `</div>` on the line
before `{{if .Fields}}`. Insert this immediately after that closing `</div>`, so the panel
renders between the title input and the description editor on both the templated
(`{{if .Fields}}`) and plain branches:

```html
					<div class="issue-similar-suggestions" hidden
							data-global-init="initRepoIssueSimilarSuggestions"
							data-search-url="{{.RepoLink}}/issues/similar"
							data-is-pull="{{if .PageIsComparePull}}true{{else}}false{{end}}"
							data-locale-heading="{{if .PageIsComparePull}}{{ctx.Locale.Tr "repo.issues.similar_pulls"}}{{else}}{{ctx.Locale.Tr "repo.issues.similar_issues"}}{{end}}"
					></div>
```

Use tabs for indentation, matching the file. Leave no trailing whitespace.

- [ ] **Step 3: Write the failing frontend test**

Create `web_src/js/features/repo-issue-similar.test.ts`:

```ts
import {renderSimilarIssues} from './repo-issue-similar.ts';

function makePanel(): HTMLElement {
  const panel = document.createElement('div');
  panel.setAttribute('data-locale-heading', 'Similar issues');
  panel.hidden = true;
  return panel;
}

describe('renderSimilarIssues', () => {
  test('renders a row per issue and reveals the panel', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [
      {number: 7, title: 'Login page crashes', state: 'open', html_url: '/o/r/issues/7'},
      {number: 9, title: 'Login is broken', state: 'closed', html_url: '/o/r/issues/9'},
    ]);

    expect(panel.hidden).toBeFalsy();
    expect(panel.textContent).toContain('Similar issues');
    const links = panel.querySelectorAll('a');
    expect(links).toHaveLength(2);
    expect(links[0].getAttribute('href')).toEqual('/o/r/issues/7');
    expect(links[0].textContent).toContain('Login page crashes');
    expect(links[0].getAttribute('target')).toEqual('_blank');
  });

  test('hides the panel when there are no results', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [{number: 7, title: 'x', state: 'open', html_url: '/o/r/issues/7'}]);
    expect(panel.hidden).toBeFalsy();

    renderSimilarIssues(panel, []);
    expect(panel.hidden).toBeTruthy();
    expect(panel.querySelectorAll('a')).toHaveLength(0);
  });

  test('escapes titles rather than injecting them as markup', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [
      {number: 1, title: '<img src=x onerror=alert(1)>', state: 'open', html_url: '/o/r/issues/1'},
    ]);
    expect(panel.querySelector('img')).toBeNull();
    expect(panel.textContent).toContain('<img src=x onerror=alert(1)>');
  });
});
```

- [ ] **Step 4: Run the test to verify it fails**

```bash
pnpm exec vitest repo-issue-similar
```

Expected: FAIL — cannot resolve `./repo-issue-similar.ts`.

- [ ] **Step 5: Write the frontend module**

Create `web_src/js/features/repo-issue-similar.ts`:

```ts
import {registerGlobalInitFunc} from '../modules/observer.ts';
import {GET} from '../modules/fetch.ts';
import {svg} from '../svg.ts';

const debounceMs = 300;
const minTitleLength = 3;

type SimilarIssue = {
  number: number,
  title: string,
  state: string,
  html_url: string,
};

// Exported for tests: rendering is the part worth asserting on, separately from the fetching.
export function renderSimilarIssues(panel: HTMLElement, issues: SimilarIssue[]): void {
  panel.textContent = '';
  if (!issues.length) {
    panel.hidden = true;
    return;
  }

  const heading = document.createElement('div');
  heading.classList.add('issue-similar-heading');
  heading.textContent = panel.getAttribute('data-locale-heading')!;
  panel.append(heading);

  const list = document.createElement('div');
  list.classList.add('issue-similar-list');
  list.setAttribute('aria-live', 'polite');

  for (const issue of issues) {
    const link = document.createElement('a');
    link.classList.add('issue-similar-item', 'flex-text-block');
    link.href = issue.html_url;
    link.target = '_blank';
    link.rel = 'noopener';

    const closed = issue.state === 'closed';
    const icon = document.createElement('span');
    icon.classList.add('text', closed ? 'purple' : 'green');
    icon.innerHTML = svg(closed ? 'octicon-issue-closed' : 'octicon-issue-opened');
    link.append(icon);

    const index = document.createElement('span');
    index.classList.add('text', 'light-3');
    index.textContent = `#${issue.number}`;
    link.append(index);

    // textContent, not innerHTML: issue titles are user input.
    const title = document.createElement('span');
    title.classList.add('gt-ellipsis');
    title.textContent = issue.title;
    link.append(title);

    list.append(link);
  }

  panel.append(list);
  panel.hidden = false;
}

function initSimilarIssuesPanel(panel: HTMLElement): void {
  const titleInput = document.querySelector<HTMLInputElement>('#issue_title');
  if (!titleInput) return;

  const searchURL = panel.getAttribute('data-search-url')!;
  const isPull = panel.getAttribute('data-is-pull')!;

  let timer: number | undefined;
  let abortController: AbortController | undefined;

  const search = async () => {
    const title = titleInput.value.trim();
    if (title.length < minTitleLength) {
      renderSimilarIssues(panel, []);
      return;
    }

    // A newer keystroke invalidates the request in flight, so a slow earlier response can never
    // repaint over a newer one.
    abortController?.abort();
    abortController = new AbortController();

    try {
      const params = new URLSearchParams({q: title, is_pull: isPull});
      const response = await GET(`${searchURL}?${params}`, {signal: abortController.signal});
      if (!response.ok) {
        renderSimilarIssues(panel, []);
        return;
      }
      renderSimilarIssues(panel, await response.json());
    } catch {
      // Aborted or offline. This is an assist, so it must never interrupt filing an issue.
      renderSimilarIssues(panel, []);
    }
  };

  titleInput.addEventListener('input', () => {
    window.clearTimeout(timer);
    timer = window.setTimeout(search, debounceMs);
  });
}

export function initRepoIssueSimilar(): void {
  registerGlobalInitFunc('initRepoIssueSimilarSuggestions', initSimilarIssuesPanel);
}
```

Before writing this, open `web_src/js/modules/fetch.ts` and confirm `GET`'s second argument
accepts a `signal` (it spreads `...other` into `fetch`, so it should). If it does not, drop the
`signal` and instead guard with a request counter: increment a module-level counter per request
and ignore any response whose counter is not the latest.

- [ ] **Step 6: Run the test to verify it passes**

```bash
pnpm exec vitest repo-issue-similar
```

Expected: PASS (3 tests).

- [ ] **Step 7: Wire the init function into the page**

In `web_src/js/index.ts`, add the import next to the other repo-issue imports (near line 17):

```ts
import {initRepoIssueSimilar} from './features/repo-issue-similar.ts';
```

and add `initRepoIssueSimilar,` to the array of init functions, next to `initRepoIssueList,`
(near line 130). It must be in that array: `registerGlobalInitFunc` throws if it is called after
`initGlobalSelectorObserver`, which runs last in that file.

- [ ] **Step 8: Add the stylesheet**

Create `web_src/css/repo/issue-similar.css`:

```css
.issue-similar-suggestions {
  margin-top: 8px;
  border: 1px solid var(--color-secondary);
  border-radius: var(--border-radius);
  padding: 8px 12px;
}

.issue-similar-heading {
  font-weight: var(--font-weight-medium);
  margin-bottom: 4px;
}

.issue-similar-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.issue-similar-item {
  color: var(--color-text);
}

.issue-similar-item:hover {
  color: var(--color-primary);
}
```

Then add the import to `web_src/css/index.css` alongside the other `./repo/` imports (near line
68, after `issue-list.css`):

```css
@import "./repo/issue-similar.css";
```

- [ ] **Step 9: Lint and typecheck the frontend**

```bash
pnpm exec eslint web_src/js/features/repo-issue-similar.ts web_src/js/features/repo-issue-similar.test.ts
```

Expected: no output. (`make lint-js` is the project-wide equivalent.)

```bash
pnpm exec vue-tsc --noEmit
```

Expected: no errors.

- [ ] **Step 10: See it work in a browser**

```bash
pnpm exec vite build
```

Then run the server and open `/{owner}/{repo}/issues/new` on a repo that has issues. Type three
or more words from an existing issue's title and confirm the panel appears under the title
input, lists up to 5 issues, and disappears when you clear the field. The frontend is served
from `public/assets`, so the vite build is mandatory — without it you are testing stale JS.

- [ ] **Step 11: Commit** — show this to the user and wait for approval

```bash
git add templates/repo/issue/new_form.tmpl options/locale/locale_en-US.json web_src/js/features/repo-issue-similar.ts web_src/js/features/repo-issue-similar.test.ts web_src/js/index.ts web_src/css/repo/issue-similar.css web_src/css/index.css && git commit -m "feat(issue): show similar issues under the new issue title

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 8: End-to-end coverage

**Files:**
- Create: `tests/e2e/issue-similar.test.ts`

- [ ] **Step 1: Write the test**

Read `tests/e2e/issue-popup.test.ts` first; this follows its structure and uses the same helpers
from `tests/e2e/utils.ts`.

```ts
import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString} from './utils.ts';

test('similar issues appear while typing a new issue title', async ({page, request}) => {
  const repoName = `e2e-issue-similar-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  const marker = randomString(8);
  const existingTitle = `Login page crashes on submit ${marker}`;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: existingTitle, body: 'Steps to reproduce.'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues/new`);

  await page.locator('#issue_title').fill(`Login page crashes when submitting ${marker}`);

  // The issue indexer consumes a queue, so a just-created issue takes a moment to become
  // searchable. The generous timeout is flake insurance, not expected runtime.
  const panel = page.locator('.issue-similar-suggestions');
  await expect(panel).toBeVisible({timeout: 10000});
  await expect(panel.getByRole('link', {name: existingTitle})).toBeVisible();

  await page.locator('#issue_title').fill('');
  await expect(panel).toBeHidden();
});
```

- [ ] **Step 2: Build what the e2e harness needs**

The harness runs the server from a temp work dir, so a plain build cannot find `options/locale`
and dies at startup. Build with bindata, and rebuild the frontend so the test exercises your
actual JS:

```bash
pnpm exec vite build
```

```bash
go generate -tags bindata ./modules/public/... ./modules/options/... ./modules/templates/... ./modules/migration/...
```

```bash
go build -tags bindata -o gitea-e2e.exe
```

- [ ] **Step 3: Run the test**

```bash
EXECUTABLE=gitea-e2e.exe PLAYWRIGHT_MODE=local bash ./tools/test-e2e.sh run --project=chromium tests/e2e/issue-similar.test.ts
```

Expected: 1 passed. (`GITEA_TEST_E2E_FLAGS='tests/e2e/issue-similar.test.ts' make test-e2e` is
the project-wide equivalent where `make` is available.)

If the panel never appears, check in this order: is `public/assets` freshly built; does
`curl` against the route return results (Task 6 step 4); is the issue indexed yet — raise the
timeout to 20000 once to distinguish an indexing race from a wiring bug, then put it back.

- [ ] **Step 4: Lint the test file**

```bash
pnpm exec eslint tests/e2e/issue-similar.test.ts
```

Expected: no output.

- [ ] **Step 5: Commit** — show this to the user and wait for approval

```bash
git add tests/e2e/issue-similar.test.ts && git commit -m "test(e2e): cover similar issue suggestions on the new issue form

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 9: Documentation and final verification

**Files:**
- Modify (only if inaccurate): `.claude-students/docs/services-issue.md`, `.claude-students/docs/routers-web.md`, `.claude-students/docs/frontend-js.md`
- Modify: the matching `.claude-students/guide/*.md` twin of any doc you edit

- [ ] **Step 1: Check the three context docs against what you built**

Read each of `.claude-students/docs/services-issue.md`, `.claude-students/docs/routers-web.md`
and `.claude-students/docs/frontend-js.md`. If anything in them is now wrong or was wrong while
you were working, fix it. Per `AGENTS.md`, editing a `docs/*.md` also requires regenerating its
`guide/*.md` twin — see `.claude-students/MAINTENANCE.md` for how.

If all three are accurate, change nothing and say so. Do not invent edits to satisfy the step.

- [ ] **Step 2: Run the repo's own doc check**

```bash
bash ./.claude-students/check.sh
```

Expected: passes. Fix anything it reports.

- [ ] **Step 3: Run the full Go checks for the packages you touched**

```bash
go build ./... && go test ./services/issue/ ./routers/web/repo/
```

Expected: PASS.

- [ ] **Step 4: Run the frontend checks**

```bash
pnpm exec vitest run repo-issue-similar && pnpm exec eslint web_src/js tests/e2e && pnpm exec vue-tsc --noEmit
```

Expected: all pass.

- [ ] **Step 5: Check the diff for trailing whitespace**

```bash
git diff origin/main -- . ':!*.svg' | grep -nE '^\+.* +$' || echo "clean"
```

Expected: `clean`.

- [ ] **Step 6: Commit any doc fixes** — show this to the user and wait for approval

```bash
git add .claude-students && git commit -m "docs: update context docs for similar issue suggestions

Assisted-by: Claude Code:claude-opus-5

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

- [ ] **Step 7: Hand back to the user**

Report: which tasks landed, the actual output of the test commands (not a summary of it), and
anything left undone. Then ask whether to push the branch and open a pull request. Do not push
or open a PR unprompted.

---

## Known limitations to state in the pull request description

Be upfront about these; a reviewer will find them anyway.

1. **This is lexical similarity, not semantic similarity.** "app crashes on login" matches
   "app crash at login", but "can't sign in" will not match "authentication broken". Embedding
   based matching is scoped out in the spec, with reasons.
2. **Ranking happens in Go over a capped candidate pool**, because no indexer backend can sort
   by relevance. On a repository where hundreds of issues share a common word, the true best
   match can fall outside the 60-candidate pool.
3. **Up to 3 indexer queries per typing pause.** Bounded and debounced, but it is not free on
   instances backed by elasticsearch.
4. **Newly created issues take a moment to appear**, because indexing runs off a queue.


---

## Corrections found during implementation

Every task below was implemented and reviewed twice (spec compliance, then code quality). These
are the places where this plan was WRONG and the shipped code deviates from it. The code is the
truth; this list exists so the deviations are not mistaken for drift.

**Task 1 — `normalizeTitle` renamed to `tokenizeTitle`.** The name promised a normalized title but
the function returns a token slice. Also switched the length filter from `len()` (bytes) to
`utf8.RuneCountInString`, so a two-rune word in a non-Latin script is treated like a two-letter
English one, and trimmed ten stopwords that the length filter already made unreachable.

**Task 2 — the plan's tie-break test fixture was arithmetically impossible.** It expected
`{"beta", "alpha"}` from longest-first ordering over words of length 4, 5 and 5. Replaced with
three same-length words so the case actually tests the tie-break. The implementer caught this and
correctly refused to bend the algorithm to fit it. The `max` parameter was also renamed to `limit`
(it shadowed the Go builtin), and a reviewer showed the wrapper struct's `order` field was
re-implementing a guarantee `sort.SliceStable` already provides.

**Task 5 — the plan's projection would have nil-panicked in production.** `Issue.HTMLURL` takes a
`context.Context` (the plan called it with no arguments) and dereferences `issue.Repo`, which
neither `GetIssuesByIDs` nor `LoadPullRequests` populates. `IssueList.LoadRepositories(ctx)` is
required before the projection loop. Nothing would have caught this before a human clicked a link,
because this function has no unit test. Separately, a failed per-token indexer query now logs and
continues instead of failing the whole lookup: this is a self-hiding assist, so one flaky query
should narrow the candidate pool, not return a 500.

**Task 6 — two idiom fixes.** `Permission.CanReadIssuesOrPulls(isPull)` already exists and is used
at 25+ call sites; the plan hand-rolled it. And `ctx.ServerError` renders the full HTML 500 template
into what is exclusively an XHR JSON endpoint — replaced with `log.Error` plus
`ctx.JSON(http.StatusInternalServerError, nil)`, the pattern already used elsewhere in the same
package.

**Task 7 — the plan invented CSS classes that do not exist.** `text`, `green`, `purple` and
`light-3` are not classes in this codebase. The real convention, from
`templates/shared/issueicon.tmpl`, is `tw-text-green` for open and `tw-text-red` for closed —
`tw-text-purple` is reserved for MERGED pull requests, not closed issues — and
`tw-text-text-light-3` for the `#number`. The plan also said to insert the locale keys in
alphabetical position; the file is not alphabetized, it is grouped by feature, so they went in the
`repo.issues.*` cluster instead.

Two review findings were fixed after the fact: the `aria-live` region was being destroyed and
recreated on every render, which means assistive tech would almost certainly never have announced
anything (it now persists across renders and only its children are replaced), and a bare `catch {}`
was swallowing genuine bugs alongside expected aborts.

**Task 8 — the plan's e2e test could not pass as written.** `locator.fill()` dispatches ONE input
event, so the frontend issues exactly one debounced fetch, which loses the race against the
asynchronous issue-indexer queue. Playwright's `toBeVisible` polling re-reads the DOM but never
re-triggers a search, so no timeout rescues it. The test now polls the `/issues/similar` endpoint
directly until the indexer has caught up, then drives the UI. The build step was also missing
`./modules/migration/...`, without which the bindata build fails.

**Final review — a similarity floor, an icon helper, and a query cap.** A whole-feature review ran
the finished algorithm against 5,223 real Gitea titles and found the panel showed five rows scoring
0.222 (one word in common) for a generic title, because nothing was ever filtered out. A 0.3 Dice
floor was added so the panel stays hidden when nothing is genuinely similar; the spec records the
reversal and why. The same review found the frontend hand-rolled an issue icon that two files say
must stay in sync with `getIssueIcon`/`getIssueColorClass`, so merged and draft pull requests
rendered as issue circles on the compare page even though the backend was already sending
`pull_request.merged` and `.draft` — it now uses the shared helpers and the shared `Issue` type. The
`q` parameter is now truncated to 255 runes, matching the form's own `maxlength`, and a pre-filled
title (from a `?title=` deep link or an issue template) now triggers the panel at init instead of
waiting for a keystroke.

## Known gaps, deliberately accepted

- **No unit test for the abort-on-newer-keystroke race.** A reviewer argued for one using
  `vi.useFakeTimers` and a mock fetch. Declined: it would require exporting an internal and
  substantial mock scaffolding, and the regression it guards against is a brief flash of stale
  suggestions. Recorded here rather than silently skipped.
- **`FindSimilarIssues` has no unit test at all**, by design — see fact 3 at the top of this plan.
  The e2e test is its only coverage.
