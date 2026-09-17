// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"cmp"
	"math"
	"slices"
)

// IssueLabelGroup is a set of issues sharing one label inside an exclusive label scope.
// Label is nil for the issues that carry no label in the scope.
type IssueLabelGroup struct {
	Label  *Label
	Issues IssueList
}

// groupLabelSortKey ranks a label inside its scope. An unset exclusive order (0) sorts last,
// mirroring the COALESCE in applyGroupByLabelScope.
func groupLabelSortKey(l *Label) (int, int64) {
	order := l.ExclusiveOrder
	if order == 0 {
		order = math.MaxInt32
	}
	return order, l.ID
}

// GroupByExclusiveLabelScope partitions issues by the label each one carries inside the given
// exclusive label scope, so "Kind" groups them into Kind/Bug, Kind/Feature and so on. Issues
// carrying no label in the scope end up in a trailing group whose Label is nil.
//
// Labels must already be loaded (LoadAttributes or LoadLabels); an issue with unloaded labels
// looks uncategorised.
//
// Groups are ordered by exclusive order then label ID, and empty groups are not emitted. That
// ordering is deliberately the same rule applyGroupByLabelScope applies in SQL, so a group split
// across a page boundary resumes in the same position on the next page.
func (issues IssueList) GroupByExclusiveLabelScope(scope string) []IssueLabelGroup {
	if len(issues) == 0 {
		return nil
	}
	if scope == "" {
		return []IssueLabelGroup{{Issues: issues}}
	}

	labelled := make(map[int64]*IssueLabelGroup)
	var order []*Label
	var unlabelled IssueList
	for _, issue := range issues {
		label := scopedLabel(issue, scope)
		if label == nil {
			unlabelled = append(unlabelled, issue)
			continue
		}
		group, ok := labelled[label.ID]
		if !ok {
			group = &IssueLabelGroup{Label: label}
			labelled[label.ID] = group
			order = append(order, label)
		}
		group.Issues = append(group.Issues, issue)
	}

	slices.SortStableFunc(order, func(a, b *Label) int {
		aOrder, aID := groupLabelSortKey(a)
		bOrder, bID := groupLabelSortKey(b)
		if c := cmp.Compare(aOrder, bOrder); c != 0 {
			return c
		}
		return cmp.Compare(aID, bID)
	})

	groups := make([]IssueLabelGroup, 0, len(order)+1)
	for _, label := range order {
		groups = append(groups, *labelled[label.ID])
	}
	if len(unlabelled) > 0 {
		groups = append(groups, IssueLabelGroup{Issues: unlabelled})
	}
	return groups
}

// scopedLabel returns the issue's label in the given exclusive scope. Inconsistent data can leave
// an issue with several, so the lowest-ranked one wins and the issue is only ever grouped once.
func scopedLabel(issue *Issue, scope string) *Label {
	var found *Label
	for _, label := range issue.Labels {
		if label.ExclusiveScope() != scope {
			continue
		}
		if found == nil {
			found = label
			continue
		}
		foundOrder, foundID := groupLabelSortKey(found)
		labelOrder, labelID := groupLabelSortKey(label)
		if labelOrder < foundOrder || (labelOrder == foundOrder && labelID < foundID) {
			found = label
		}
	}
	return found
}
