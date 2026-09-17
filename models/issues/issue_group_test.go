// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"

	"github.com/stretchr/testify/assert"
)

func scopedTestLabel(id int64, name string, exclusive bool, order int) *issues_model.Label {
	return &issues_model.Label{ID: id, Name: name, Exclusive: exclusive, ExclusiveOrder: order}
}

func TestIssueList_GroupByExclusiveLabelScope(t *testing.T) {
	kindBug := scopedTestLabel(1, "Kind/Bug", true, 1)
	kindFeature := scopedTestLabel(2, "Kind/Feature", true, 2)
	// the Advanced.yaml "Reviewed/Invalid" + "Reviewed/Won't Fix" case: same order, different labels
	kindTiedHigh := scopedTestLabel(3, "Kind/Docs", true, 3)
	kindTiedLow := scopedTestLabel(4, "Kind/Testing", true, 3)
	kindUnordered := scopedTestLabel(5, "Kind/Chore", true, 0)
	kindNotExclusive := scopedTestLabel(6, "Kind/Security", false, 1)
	kindSubScope := scopedTestLabel(7, "Kind/UI/Bug", true, 1)
	otherScope := scopedTestLabel(8, "Priority/High", true, 1)

	issue := func(id int64, labels ...*issues_model.Label) *issues_model.Issue {
		return &issues_model.Issue{ID: id, Labels: labels}
	}
	// group is described as its label ID (0 for the uncategorised group) plus the issue IDs in it
	type wantGroup struct {
		labelID  int64
		issueIDs []int64
	}

	cases := []struct {
		name   string
		scope  string
		issues issues_model.IssueList
		want   []wantGroup
	}{
		{
			name:  "empty list",
			scope: "Kind",
		},
		{
			name:   "empty scope collapses to a single uncategorised group",
			scope:  "",
			issues: issues_model.IssueList{issue(1, kindBug), issue(2)},
			want:   []wantGroup{{labelID: 0, issueIDs: []int64{1, 2}}},
		},
		{
			name:   "no issue carries a label in the scope",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1), issue(2, otherScope)},
			want:   []wantGroup{{labelID: 0, issueIDs: []int64{1, 2}}},
		},
		{
			name:   "ordered labels first, uncategorised last",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1), issue(2, kindFeature), issue(3, kindBug)},
			want: []wantGroup{
				{labelID: kindBug.ID, issueIDs: []int64{3}},
				{labelID: kindFeature.ID, issueIDs: []int64{2}},
				{labelID: 0, issueIDs: []int64{1}},
			},
		},
		{
			name:   "labels sharing an exclusive order tie-break on label ID",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1, kindTiedLow), issue(2, kindTiedHigh)},
			want: []wantGroup{
				{labelID: kindTiedHigh.ID, issueIDs: []int64{2}},
				{labelID: kindTiedLow.ID, issueIDs: []int64{1}},
			},
		},
		{
			name:   "unordered label sorts after ordered ones but before uncategorised",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1, kindUnordered), issue(2), issue(3, kindFeature)},
			want: []wantGroup{
				{labelID: kindFeature.ID, issueIDs: []int64{3}},
				{labelID: kindUnordered.ID, issueIDs: []int64{1}},
				{labelID: 0, issueIDs: []int64{2}},
			},
		},
		{
			name:   "a non-exclusive label has no scope",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1, kindNotExclusive)},
			want:   []wantGroup{{labelID: 0, issueIDs: []int64{1}}},
		},
		{
			name:   "a sub-scoped label belongs to its own scope, not the parent",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1, kindSubScope)},
			want:   []wantGroup{{labelID: 0, issueIDs: []int64{1}}},
		},
		{
			name:   "two labels in one scope group the issue exactly once",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(1, kindFeature, kindBug)},
			want:   []wantGroup{{labelID: kindBug.ID, issueIDs: []int64{1}}},
		},
		{
			name:   "input order is preserved inside a group",
			scope:  "Kind",
			issues: issues_model.IssueList{issue(3, kindBug), issue(1, kindBug), issue(2, kindBug)},
			want:   []wantGroup{{labelID: kindBug.ID, issueIDs: []int64{3, 1, 2}}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups := c.issues.GroupByExclusiveLabelScope(c.scope)

			var got []wantGroup
			total := 0
			for _, group := range groups {
				var labelID int64
				if group.Label != nil {
					labelID = group.Label.ID
				}
				issueIDs := make([]int64, 0, len(group.Issues))
				for _, issue := range group.Issues {
					issueIDs = append(issueIDs, issue.ID)
				}
				got = append(got, wantGroup{labelID: labelID, issueIDs: issueIDs})
				total += len(group.Issues)
			}

			assert.Equal(t, c.want, got)
			assert.Len(t, c.issues, total, "every issue must appear in exactly one group")
		})
	}
}
