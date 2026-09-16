// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
)

func TestIssueDeadlineStatus(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	defer timeutil.MockSet(now)()
	nowTS := timeutil.TimeStamp(now.Unix())

	cases := []struct {
		name    string
		issue   issues_model.Issue
		overdue bool
		nearDue bool
	}{
		{"no deadline", issues_model.Issue{}, false, false},
		{"far future", issues_model.Issue{DeadlineUnix: nowTS.AddDuration(96 * time.Hour)}, false, false},
		{"just outside window", issues_model.Issue{DeadlineUnix: nowTS.AddDuration(issues_model.NearDueDuration).Add(1)}, false, false},
		{"exactly at window edge", issues_model.Issue{DeadlineUnix: nowTS.AddDuration(issues_model.NearDueDuration)}, false, true},
		{"inside window", issues_model.Issue{DeadlineUnix: nowTS.AddDuration(time.Hour)}, false, true},
		{"deadline is now", issues_model.Issue{DeadlineUnix: nowTS}, true, false},
		{"overdue", issues_model.Issue{DeadlineUnix: nowTS.Add(-1)}, true, false},
		{"closed before deadline", issues_model.Issue{IsClosed: true, ClosedUnix: nowTS.Add(-3600), DeadlineUnix: nowTS.AddDuration(time.Hour)}, false, false},
		{"closed after deadline", issues_model.Issue{IsClosed: true, ClosedUnix: nowTS, DeadlineUnix: nowTS.Add(-3600)}, true, false},
		{"closed, no deadline", issues_model.Issue{IsClosed: true, ClosedUnix: nowTS}, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.overdue, c.issue.IsOverdue())
			assert.Equal(t, c.nearDue, c.issue.IsNearDue())
			assert.False(t, c.issue.IsOverdue() && c.issue.IsNearDue(), "overdue and near-due must be mutually exclusive")
		})
	}
}

func TestMilestoneDeadlineStatus(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	defer timeutil.MockSet(now)()
	nowTS := timeutil.TimeStamp(now.Unix())

	load := func(m *issues_model.Milestone) *issues_model.Milestone {
		m.AfterLoad()
		return m
	}

	noDeadline := load(&issues_model.Milestone{})
	assert.False(t, noDeadline.IsOverdue)
	assert.False(t, noDeadline.IsNearDue)
	assert.Empty(t, noDeadline.DeadlineString)

	soon := load(&issues_model.Milestone{DeadlineUnix: nowTS.AddDuration(time.Hour)})
	assert.False(t, soon.IsOverdue)
	assert.True(t, soon.IsNearDue)
	assert.NotEmpty(t, soon.DeadlineString)

	overdue := load(&issues_model.Milestone{DeadlineUnix: nowTS.Add(-1)})
	assert.True(t, overdue.IsOverdue)
	assert.False(t, overdue.IsNearDue)

	far := load(&issues_model.Milestone{DeadlineUnix: nowTS.AddDuration(96 * time.Hour)})
	assert.False(t, far.IsOverdue)
	assert.False(t, far.IsNearDue)

	closedInTime := load(&issues_model.Milestone{IsClosed: true, ClosedDateUnix: nowTS.Add(-3600), DeadlineUnix: nowTS.AddDuration(time.Hour)})
	assert.False(t, closedInTime.IsOverdue)
	assert.False(t, closedInTime.IsNearDue)

	closedLate := load(&issues_model.Milestone{IsClosed: true, ClosedDateUnix: nowTS, DeadlineUnix: nowTS.Add(-3600)})
	assert.True(t, closedLate.IsOverdue)
	assert.False(t, closedLate.IsNearDue)

	// NumOpenIssues is still computed as before.
	m := load(&issues_model.Milestone{NumIssues: 5, NumClosedIssues: 2})
	assert.Equal(t, 3, m.NumOpenIssues)
}
