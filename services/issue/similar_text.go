// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/container"
)

// similarityStopwords are dropped before two titles are compared: they appear in almost every
// issue title and so carry no signal about what the issue is about. Stopwords of two characters
// or fewer are omitted here because the length filter in tokenizeTitle already removes them.
var similarityStopwords = container.SetOf(
	"the", "and", "for", "with",
	"are", "when", "while", "not", "this", "that",
)

// similarityFloor is the lowest score worth showing. Dice is the shared fraction of two token
// sets, so 0.3 means roughly a third of the words match; below that the indexer most likely
// matched on body or comment text the reader never sees, and a row would be noise under a
// heading promising similar issues.
const similarityFloor = 0.3

// tokenizeTitle lowercases a title, splits it on anything that is not a letter or digit, and
// drops stopwords and tokens of two characters or fewer. It returns nil when nothing is left.
func tokenizeTitle(title string) []string {
	fields := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var tokens []string
	for _, f := range fields {
		if utf8.RuneCountInString(f) <= 2 || similarityStopwords.Contains(f) {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}

// queryTokens picks at most limit distinct tokens to send to the indexer, longest first: longer
// words are rarer, so they make better candidate filters. Each token costs one indexer query,
// which is why the count is capped.
func queryTokens(tokens []string, limit int) []string {
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
		distinct = distinct[:max(limit, 0)]
	}
	return distinct
}

// diceSimilarity scores how alike two token sets are, from 0 (nothing in common) to 1 (same
// set), using the Dice coefficient. Shared by titleSimilarity and rankBySimilarity so the latter
// can tokenize the typed title once and compare it against many candidates.
func diceSimilarity(setA, setB container.Set[string]) float64 {
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

// titleSimilarity scores how alike two titles are, from 0 (nothing in common) to 1 (same token
// set), using the Dice coefficient over their token sets.
func titleSimilarity(a, b string) float64 {
	setA := container.SetOf(tokenizeTitle(a)...)
	setB := container.SetOf(tokenizeTitle(b)...)
	return diceSimilarity(setA, setB)
}

// rankBySimilarity orders candidates by how closely their titles match the typed title, most
// similar first, and returns at most limit of them. Equal scores are broken by the most
// recently updated issue, which is the more likely duplicate of the two.
//
// Candidates scoring below similarityFloor are dropped: the indexer matches on title, body, or
// comment text, so a low title score likely means the match came from prose the reader never
// sees, and an unrelated row is worse than no row for a feature the reader did not ask for.
func rankBySimilarity(title string, cands []*issues_model.Issue, limit int) []*issues_model.Issue {
	type scored struct {
		issue *issues_model.Issue
		score float64
	}

	querySet := container.SetOf(tokenizeTitle(title)...)

	scoredCands := make([]scored, 0, len(cands))
	for _, c := range cands {
		score := diceSimilarity(querySet, container.SetOf(tokenizeTitle(c.Title)...))
		if score < similarityFloor {
			continue
		}
		scoredCands = append(scoredCands, scored{issue: c, score: score})
	}

	sort.SliceStable(scoredCands, func(i, j int) bool {
		if scoredCands[i].score != scoredCands[j].score {
			return scoredCands[i].score > scoredCands[j].score
		}
		return scoredCands[i].issue.UpdatedUnix > scoredCands[j].issue.UpdatedUnix
	})

	ranked := make([]*issues_model.Issue, len(scoredCands))
	for i, s := range scoredCands {
		ranked[i] = s.issue
	}
	if limit < len(ranked) {
		ranked = ranked[:max(limit, 0)]
	}
	return ranked
}
