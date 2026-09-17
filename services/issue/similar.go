// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"
	"unicode/utf8"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/modules/container"
	"gitea.dev/modules/indexer"
	issue_indexer "gitea.dev/modules/indexer/issues"
	"gitea.dev/modules/log"
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
	// similarMaxCandidates caps the unioned candidate pool that gets re-ranked. Deliberately
	// similarMaxQueryTokens * similarCandidatesPerToken: that is what makes the inner loop's
	// break only ever trigger while draining the last token's page, never wasting a query.
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
	if utf8.RuneCountInString(title) < similarMinTitleLen || len(tokens) == 0 {
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
			// Best-effort: one failed token still leaves the others' candidates worth ranking.
			log.Debug("SearchIssues failed for token %q in repo %d: %v", token, repo.ID, err)
			continue
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
	// HTMLURL below needs issue.Repo: GetIssuesByIDs only queries the issue table, and
	// LoadPullRequests only loads pull_request rows, so neither populates it.
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
