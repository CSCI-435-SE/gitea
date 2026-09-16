# Suggest similar issues while an issue is being written

**Status:** approved, ready for implementation plan
**Branch:** `feat/similar-issue-suggestions`
**Upstream issue:** "Display similar issues when creating an issue"

## Problem

When someone files an issue, Gitea offers no hint that the same report already exists. The
reporter finds out only when a maintainer closes it as a duplicate, so the cost lands on
maintainers and the reporter's write-up is wasted. GitLab addresses this by listing related
issues under the title field as the title is typed.

The ask is to show the 5 most similar existing issues below the title input on the new-issue
form.

## What already exists

- `GET /{owner}/{repo}/issues/suggestions?q=` (`routers/web/repo/issue_suggestions.go`) returns
  5 issues and is consumed by the `#`-autocomplete in the comment editor
  (`web_src/js/features/comp/TextExpander.ts`).
- Its matching is a SQL `LIKE` against the **title only** (`models/issues/issue.go:532`); a
  comment there records that content search was removed because `LIKE` over content produced
  unusable results.
- A real full-text issue indexer exists (`modules/indexer/issues/`), with bleve, elasticsearch,
  meilisearch and db backends, indexing title and content. The default backend is bleve
  (`modules/setting/indexer.go:37`).

So the delivery mechanism is nearly free; the open question is matching quality.

## Scope

In scope:

- A panel under the title input on the new-issue form **and** the compare / new-pull-request
  form, both of which render `templates/repo/issue/new_form.tmpl`.
- Full-text candidate retrieval through the existing issue indexer, re-ranked by title
  similarity in Go.
- The new-issue form searches issues (open and closed); the new-PR form searches pull requests.

Out of scope:

- **Embedding / vector similarity.** True semantic matching needs a model or an external
  service, a vector store, a backfill job and configuration for air-gapped instances. That is
  its own multi-PR project. This spec delivers token-level similarity and does not claim more.
- **Relevance sorting inside the indexer.** All four backends force a date sort when `SortBy`
  is empty (`bleve.go:302`, `elasticsearch.go:218`, `meilisearch.go:232`) and `internal.SortBy`
  has no relevance value. Adding one touches shared search infrastructure used by every issue
  search page; it is a worthwhile follow-up, not part of this change.
- **Changing `/issues/suggestions`.** That endpoint feeds `#`-autocomplete, where typing `#123`
  and prefix matching matter. Re-pointing it at fuzzy full-text search would be a silent
  regression for a different feature.
- Cross-repository suggestions, including a fork's parent repository.

## Design

### Retrieval and ranking

New service in `services/issue/similar.go`:

```go
func FindSimilarIssues(ctx context.Context, repo *repo_model.Repository,
    isPull optional.Option[bool], title string, limit int) ([]*structs.Issue, error)
```

1. Return an empty result, with no query issued, when the title has fewer than 3 characters
   after trimming surrounding whitespace.
2. Over-fetch candidates, one indexer query per query token, unioned by issue ID.

   The default bleve backend builds its keyword query with `MatchQueryOperatorAnd`
   (`modules/indexer/internal/bleve/query.go:37`), so every term of the keyword must appear in
   the document. Passing a whole typed title as one keyword therefore matches almost nothing.
   Meilisearch is lenient here, but bleve is the default, so candidate generation cannot rely
   on backend-specific leniency.

   Instead: normalize the title into tokens (below), take the 3 most distinctive ones (longest
   first, ties broken by order of appearance), and issue one `issue_indexer.SearchIssues` call
   per token with `Keyword: token`, `RepoIDs: []int64{repo.ID}`, `IsPull: isPull`,
   `SearchMode: indexer.SearchModeFuzzy`, `SortBy: issue_indexer.SortByUpdatedDesc` and a page
   size of 20. Union the returned IDs, preserving first-seen order, capped at 60.

   This bounds the work at 3 indexer queries per 300ms typing pause and behaves identically on
   every backend: a candidate is any issue sharing at least one distinctive word.
3. Load the candidate issues and re-rank them with the pure scorer below, keeping the top
   `limit` (5).
4. Project each result to the lightweight `structs.Issue` shape `GetSuggestion` already
   produces — ID, Index, Title, State, pull-request meta — plus `HTMLURL` for the link.

The scorer lives beside it and takes no context and no database handle:

```go
func tokenizeTitle(title string) []string
func queryTokens(tokens []string, limit int) []string
func titleSimilarity(a, b string) float64
func rankBySimilarity(title string, cands []*issues_model.Issue, limit int) []*issues_model.Issue
```

Tokenizing lowercases, strips punctuation, splits on whitespace, and drops tokens of 2 runes or
fewer along with a short built-in English stopword list defined in this package (the, and, for,
with, are, when, while, not, this, that). Shorter stopwords need no entry because the length
filter already removes them. No new dependency and no configuration knob. The score is the Dice
coefficient over the resulting token sets, `2*|A and B| / (|A| + |B|)`. Ties break on
`updated_unix` descending.

No score threshold is applied. The indexer has already decided these documents match; a
candidate that matched on body text with an unrelated title simply scores 0, sorts last, and
surfaces only when fewer than 5 title matches exist.

### Route

Registered next to the existing suggestions route (`routers/web/web.go:1318`), inside the same
`optSignIn`, `context.RepoAssignment`, `reqRepoIssuesOrPullsReader` group:

```
GET /{owner}/{repo}/issues/similar?q=<title>&is_pull=<bool>
```

Handler `routers/web/repo/issue_similar.go`. It intersects the requested `is_pull` with what the
viewer may actually read, reusing the `CanRead(unit.TypeIssues)` / `CanRead(unit.TypePullRequests)`
logic from `issue_suggestions.go:19`. If the requested type is not readable it returns an empty
list rather than an error, so the panel degrades to invisible instead of revealing which units
exist.

The response is a JSON array of the projected issues, capped at 5.

### Template

In `new_form.tmpl`, directly after the title `field` block — so it renders between the title
input and the description editor on both the `.Fields` (issue template) branch and the plain
branch:

```html
<div class="issue-similar-suggestions" hidden
     data-global-init="initRepoIssueSimilarSuggestions"
     data-search-url="{{.RepoLink}}/issues/similar"
     data-is-pull="{{if .PageIsComparePull}}true{{else}}false{{end}}"
     data-locale-heading="...">
</div>
```

The heading string is `repo.issues.similar_pulls` on the compare page and
`repo.issues.similar_issues` otherwise.

### Browser behaviour

New `web_src/js/features/repo-issue-similar.ts`, registered through `registerGlobalInitFunc`
(`web_src/js/modules/observer.ts`), the pattern already used across `repo-issue.ts`.

- Listens on `#issue_title`, debounced 300ms, minimum 3 characters.
- Holds a single `AbortController`; each new keystroke aborts the in-flight request, so a slow
  earlier response cannot repaint stale results over a newer one.
- Requests through `web_src/js/modules/fetch.ts`. On abort, network failure or non-OK status the
  panel hides silently — this is an assist and must never block filing an issue.
- Renders a heading plus up to 5 rows: state icon via `svg()`, `#index`, and the title. Each row
  is an `<a target="_blank" rel="noopener">` so clicking never discards the draft.
- The panel is `hidden` for 0 results or a too-short title. The results container carries
  `aria-live="polite"` so the appearance of suggestions is announced rather than silent.
- Layout uses `flex-text-block` and `tw-` utilities per AGENTS.md, with new rules in
  `web_src/css/repo/issue-similar.css` only if the existing helpers are insufficient.

### Strings

Two new keys in `options/locale/locale_en-US.json`: `repo.issues.similar_issues`
("Similar issues") and `repo.issues.similar_pulls` ("Similar pull requests"). English only;
other locales arrive through the normal translation workflow.

## Testing

`services/issue/similar_test.go` — unit tests for `titleSimilarity` and `rankBySimilarity`:

- a near-identical title outranks a partial match;
- stopwords and short tokens are ignored, so "the app crashes on login" matches
  "app crash at login";
- empty, whitespace-only and punctuation-only input score 0 and do not panic;
- equal scores break on `updated_unix` descending;
- the limit is honoured when more than 5 candidates score above 0.

These are pure functions: no database, no context, no indexer.

`FindSimilarIssues` itself is deliberately not unit-tested. Outside a running server
`globalIndexer` is the dummy indexer, whose `Search` returns `"indexer is not ready"`
(`modules/indexer/issues/internal/indexer.go:46`). Keeping that function thin and pushing the
logic into pure helpers is what makes the logic testable at all.

`tests/e2e/issue-similar.test.ts` — Playwright, mirroring `tests/e2e/issue-popup.test.ts`: open
the new-issue page, type a title matching a fixture issue, assert the panel appears and links to
the expected issue, then clear the title and assert the panel hides. Budgeted under 2 seconds per
AGENTS.md.

## Documentation

This change touches areas covered by `.claude-students/docs/routers-web.md`,
`.claude-students/docs/services-issue.md` and `.claude-students/docs/frontend-js.md`. Any of
those found inaccurate while implementing is fixed in the same PR, and editing a `docs/*.md` also
means regenerating its `guide/*.md` twin. `./.claude-students/check.sh` runs before push.

## Follow-ups this spec deliberately leaves open

1. Relevance sorting in the indexer backends, which would remove the over-fetch-and-re-rank step
   and improve candidate quality for repositories with many matches.
2. Embedding-based similarity behind the same route and UI, which the service boundary already
   accommodates: only `FindSimilarIssues` would change.
