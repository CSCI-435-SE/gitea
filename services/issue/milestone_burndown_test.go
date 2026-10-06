// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const burndownMilestone = 7

// at reads "2026-01-05 10:00" in UTC
func at(t *testing.T, value string) timeutil.TimeStamp {
	t.Helper()
	parsed, err := time.Parse(time.DateTime, value+":00")
	require.NoError(t, err)
	return timeutil.TimeStamp(parsed.Unix())
}

func openItem(id int64, created timeutil.TimeStamp) issues_model.MilestoneItem {
	return issues_model.MilestoneItem{ID: id, MilestoneID: burndownMilestone, CreatedUnix: created}
}

func closedItem(id int64, created, closed timeutil.TimeStamp) issues_model.MilestoneItem {
	return issues_model.MilestoneItem{ID: id, MilestoneID: burndownMilestone, CreatedUnix: created, IsClosed: true, ClosedUnix: closed}
}

// eventSeq gives every event a comment id in the order the test lists them
type eventSeq struct {
	events []issues_model.MilestoneEvent
}

func (s *eventSeq) add(issueID int64, typ issues_model.CommentType, when timeutil.TimeStamp, oldMilestone, newMilestone int64) *eventSeq {
	s.events = append(s.events, issues_model.MilestoneEvent{
		ID: int64(len(s.events) + 1), IssueID: issueID, Type: typ, CreatedUnix: when,
		OldMilestoneID: oldMilestone, MilestoneID: newMilestone,
	})
	return s
}

func (s *eventSeq) join(issueID int64, when timeutil.TimeStamp) *eventSeq {
	return s.add(issueID, issues_model.CommentTypeMilestone, when, 0, burndownMilestone)
}

func (s *eventSeq) leave(issueID int64, when timeutil.TimeStamp) *eventSeq {
	return s.add(issueID, issues_model.CommentTypeMilestone, when, burndownMilestone, burndownMilestone+1)
}

func (s *eventSeq) close(issueID int64, when timeutil.TimeStamp) *eventSeq {
	return s.add(issueID, issues_model.CommentTypeClose, when, 0, 0)
}

func (s *eventSeq) merge(issueID int64, when timeutil.TimeStamp) *eventSeq {
	return s.add(issueID, issues_model.CommentTypeMergePull, when, 0, 0)
}

func (s *eventSeq) reopen(issueID int64, when timeutil.TimeStamp) *eventSeq {
	return s.add(issueID, issues_model.CommentTypeReopen, when, 0, 0)
}

func remainingOf(points []BurndownPoint) []int {
	values := make([]int, 0, len(points))
	for _, p := range points {
		values = append(values, p.Remaining)
	}
	return values
}

func scopeOf(points []BurndownPoint) []int {
	values := make([]int, 0, len(points))
	for _, p := range points {
		values = append(values, p.Scope)
	}
	return values
}

func datesOf(points []BurndownPoint) []string {
	values := make([]string, 0, len(points))
	for _, p := range points {
		values = append(values, p.Date)
	}
	return values
}

func TestCalcMilestoneBurndown(t *testing.T) {
	jan := func(day int, hour string) timeutil.TimeStamp {
		return at(t, time.Date(2026, time.January, day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)+" "+hour)
	}
	endOfDay := func(day int) timeutil.TimeStamp { return jan(day, "23:59") + 59 }

	cases := []struct {
		name      string
		milestone issues_model.Milestone
		items     []issues_model.MilestoneItem
		events    *eventSeq
		now       timeutil.TimeStamp

		remaining []int
		scope     []int
		firstDate string
		check     func(t *testing.T, b *MilestoneBurndown)
	}{
		{
			name:      "no items",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			events:    &eventSeq{},
			now:       jan(9, "12:00"),
			remaining: []int{},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, BurndownEmpty, b.Status)
				assert.NotNil(t, b.Points, "an empty milestone still answers a list, so the page needs no null branch")
			},
		},
		{
			name:      "steady burn to done on the due date",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), DeadlineUnix: endOfDay(11)},
			items: []issues_model.MilestoneItem{
				closedItem(1, jan(5, "09:00"), jan(7, "10:00")),
				closedItem(2, jan(5, "09:00"), jan(9, "10:00")),
				closedItem(3, jan(5, "09:00"), jan(11, "10:00")),
			},
			events: (&eventSeq{}).join(1, jan(5, "10:00")).join(2, jan(5, "10:00")).join(3, jan(5, "10:00")).
				close(1, jan(7, "10:00")).close(2, jan(9, "10:00")).close(3, jan(11, "10:00")),
			now:       jan(11, "12:00"),
			remaining: []int{3, 3, 2, 2, 1, 1, 0},
			scope:     []int{3, 3, 3, 3, 3, 3, 3},
			firstDate: "2026-01-05",
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, BurndownDone, b.Status)
				assert.Equal(t, "2026-01-11", b.CompletedOn)
				assert.Equal(t, &BurndownIdeal{From: "2026-01-05", FromValue: 3, To: "2026-01-11"}, b.Ideal)
				assert.Equal(t, "2026-01-11", b.Deadline)
				assert.Empty(t, b.Projected)
			},
		},
		{
			name:      "scope creep raises the line but not the ideal",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), DeadlineUnix: endOfDay(15)},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00")), openItem(2, jan(5, "09:00")), openItem(3, jan(7, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).join(2, jan(5, "10:00")).join(3, jan(7, "10:00")),
			now:       jan(8, "12:00"),
			remaining: []int{2, 2, 3, 3},
			scope:     []int{2, 2, 3, 3},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, 2, b.Ideal.FromValue)
				assert.Equal(t, BurndownNotBurning, b.Status, "the net rate counts the creep, so this is no progress")
			},
		},
		{
			name:      "an item that leaves drops both lines and counts as net burn",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items: []issues_model.MilestoneItem{
				openItem(1, jan(5, "09:00")),
				{ID: 2, MilestoneID: burndownMilestone + 1, CreatedUnix: jan(5, "09:00")},
			},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).join(2, jan(5, "10:00")).leave(2, jan(7, "10:00")),
			now:       jan(9, "12:00"),
			remaining: []int{2, 2, 1, 1, 1},
			scope:     []int{2, 2, 1, 1, 1},
			check: func(t *testing.T, b *MilestoneBurndown) {
				// one item gone in four days: 0.25 a day, so the last item takes four more days
				assert.Equal(t, BurndownProjected, b.Status)
				assert.Equal(t, "2026-01-13", b.Projected)
				assert.Nil(t, b.DaysLate, "no due date, nothing to be late for")
				assert.Nil(t, b.Ideal)
			},
		},
		{
			name:      "a reopened item is open again until it closes",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(8, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).close(1, jan(6, "10:00")).reopen(1, jan(7, "10:00")).close(1, jan(8, "10:00")),
			now:       jan(8, "12:00"),
			remaining: []int{1, 0, 1, 0},
			scope:     []int{1, 1, 1, 1},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, "2026-01-08", b.CompletedOn, "done is the start of the final run of zeros, not the first zero")
			},
		},
		{
			name:      "a merged pull request burns down like a closed issue",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(6, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).merge(1, jan(6, "10:00")),
			now:       jan(6, "12:00"),
			remaining: []int{1, 0},
			scope:     []int{1, 1},
		},
		{
			name:      "an item added already closed widens scope only",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00")), closedItem(2, jan(1, "09:00"), jan(2, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).close(2, jan(2, "10:00")).join(2, jan(6, "10:00")),
			now:       jan(6, "12:00"),
			remaining: []int{1, 1},
			scope:     []int{1, 2},
		},
		{
			name:      "an item with no join comment joins when the milestone was created",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{openItem(1, jan(1, "09:00")), openItem(2, jan(6, "09:00"))},
			events:    &eventSeq{},
			now:       jan(6, "12:00"),
			remaining: []int{1, 2},
			scope:     []int{1, 2},
		},
		{
			name:      "an item whose first comment moves it out was in before then",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{{ID: 1, MilestoneID: burndownMilestone + 1, CreatedUnix: jan(1, "09:00")}},
			events:    (&eventSeq{}).leave(1, jan(6, "10:00")),
			now:       jan(7, "12:00"),
			remaining: []int{1, 0, 0},
			scope:     []int{1, 0, 0},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, BurndownEmpty, b.Status, "nothing is left in it")
			},
		},
		{
			name:      "a closed item with no close comment closes at closed_unix",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(6, "10:00")), openItem(2, jan(5, "09:00"))},
			events:    &eventSeq{},
			now:       jan(7, "12:00"),
			remaining: []int{2, 1, 1},
			scope:     []int{2, 2, 2},
		},
		{
			name:      "a closed item with no close time closes when it was created",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), 0)},
			events:    &eventSeq{},
			now:       jan(6, "12:00"),
			remaining: []int{0, 0},
			scope:     []int{1, 1},
		},
		{
			name:      "the row wins when the comments disagree with it",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			// closed, reopened, then closed again with no comment, as a migration can leave it
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(7, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).close(1, jan(5, "11:00")).reopen(1, jan(6, "10:00")),
			now:       jan(8, "12:00"),
			remaining: []int{0, 1, 0, 0},
			scope:     []int{1, 1, 1, 1},
		},
		{
			name:      "a milestone with no creation time starts at its first change",
			milestone: issues_model.Milestone{},
			items:     []issues_model.MilestoneItem{openItem(1, jan(3, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(4, "10:00")),
			now:       jan(5, "12:00"),
			remaining: []int{1, 1},
			firstDate: "2026-01-04",
		},
		{
			name:      "a closed milestone ends on the day it closed",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), IsClosed: true, ClosedDateUnix: jan(7, "10:00")},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00")), closedItem(2, jan(5, "09:00"), jan(6, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).join(2, jan(5, "10:00")).close(2, jan(6, "10:00")),
			now:       jan(20, "12:00"),
			remaining: []int{2, 1, 1},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, BurndownClosed, b.Status)
				assert.Empty(t, b.Projected)
			},
		},
		{
			name:      "a closed milestone runs on to changes made after it closed",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), IsClosed: true, ClosedDateUnix: jan(6, "10:00")},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(8, "10:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).close(1, jan(8, "10:00")),
			now:       jan(20, "12:00"),
			remaining: []int{1, 1, 1, 0},
		},
		{
			name:      "late projection against a due date",
			milestone: issues_model.Milestone{CreatedUnix: jan(1, "09:00"), DeadlineUnix: endOfDay(10)},
			items: []issues_model.MilestoneItem{
				closedItem(1, jan(1, "09:00"), jan(3, "10:00")), closedItem(2, jan(1, "09:00"), jan(5, "10:00")),
				openItem(3, jan(1, "09:00")), openItem(4, jan(1, "09:00")),
			},
			events: (&eventSeq{}).join(1, jan(1, "10:00")).join(2, jan(1, "10:00")).join(3, jan(1, "10:00")).join(4, jan(1, "10:00")).
				close(1, jan(3, "10:00")).close(2, jan(5, "10:00")),
			now:       jan(9, "12:00"),
			remaining: []int{4, 4, 3, 3, 2, 2, 2, 2, 2},
			check: func(t *testing.T, b *MilestoneBurndown) {
				// two closed in eight days: a quarter a day, so two more take eight days
				assert.Equal(t, BurndownProjected, b.Status)
				assert.Equal(t, "2026-01-17", b.Projected)
				assert.Equal(t, new(7), b.DaysLate)
			},
		},
		{
			name:      "the projection reads only the last two weeks",
			milestone: issues_model.Milestone{CreatedUnix: jan(1, "09:00"), DeadlineUnix: endOfDay(31)},
			items: []issues_model.MilestoneItem{
				closedItem(1, jan(1, "09:00"), jan(2, "10:00")), closedItem(2, jan(1, "09:00"), jan(2, "10:00")),
				closedItem(3, jan(1, "09:00"), jan(10, "10:00")), openItem(4, jan(1, "09:00")),
			},
			events: (&eventSeq{}).join(1, jan(1, "10:00")).join(2, jan(1, "10:00")).join(3, jan(1, "10:00")).join(4, jan(1, "10:00")).
				close(1, jan(2, "10:00")).close(2, jan(2, "10:00")).close(3, jan(10, "10:00")),
			now: jan(22, "12:00"),
			check: func(t *testing.T, b *MilestoneBurndown) {
				// the early pair is outside the window; one in fourteen days leaves one more for fourteen
				assert.Equal(t, "2026-02-05", b.Projected)
				assert.Equal(t, new(5), b.DaysLate)
			},
		},
		{
			name:      "an early projection is negative days late",
			milestone: issues_model.Milestone{CreatedUnix: jan(1, "09:00"), DeadlineUnix: endOfDay(31)},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(1, "09:00"), jan(2, "10:00")), openItem(2, jan(1, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(1, "10:00")).join(2, jan(1, "10:00")).close(1, jan(2, "10:00")),
			now:       jan(3, "12:00"),
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, "2026-01-05", b.Projected)
				assert.Equal(t, new(-26), b.DaysLate)
			},
		},
		{
			name:      "one day of history is too little to project",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")),
			now:       jan(6, "12:00"),
			remaining: []int{1, 1},
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Equal(t, BurndownInsufficientData, b.Status)
			},
		},
		{
			name:      "a due date on or before the first day of work draws no ideal line",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), DeadlineUnix: endOfDay(5)},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")),
			now:       jan(8, "12:00"),
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Nil(t, b.Ideal)
				assert.Equal(t, "2026-01-05", b.Deadline, "it is still the due date, just in the past")
			},
		},
		{
			name:      "a due date in the year 9999 is not drawn",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00"), DeadlineUnix: 253370764800},
			items:     []issues_model.MilestoneItem{closedItem(1, jan(5, "09:00"), jan(6, "10:00")), openItem(2, jan(5, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).join(2, jan(5, "10:00")).close(1, jan(6, "10:00")),
			now:       jan(8, "12:00"),
			check: func(t *testing.T, b *MilestoneBurndown) {
				assert.Nil(t, b.Ideal)
				assert.Empty(t, b.Deadline)
				assert.Nil(t, b.DaysLate)
				assert.Equal(t, BurndownProjected, b.Status)
			},
		},
		{
			name:      "an item from another milestone that never joined is ignored",
			milestone: issues_model.Milestone{CreatedUnix: jan(5, "09:00")},
			items:     []issues_model.MilestoneItem{openItem(1, jan(5, "09:00"))},
			events:    (&eventSeq{}).join(1, jan(5, "10:00")).close(99, jan(5, "11:00")),
			now:       jan(5, "12:00"),
			remaining: []int{1},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.milestone
			m.ID = burndownMilestone
			b := CalcMilestoneBurndown(&m, c.items, c.events.events, time.Unix(int64(c.now), 0), time.UTC)

			if c.remaining != nil {
				assert.Equal(t, c.remaining, remainingOf(b.Points))
			}
			if c.scope != nil {
				assert.Equal(t, c.scope, scopeOf(b.Points))
			}
			if c.firstDate != "" {
				assert.Equal(t, c.firstDate, b.Points[0].Date)
			}
			if c.check != nil {
				c.check(t, b)
			}

			// the last point always agrees with the rows, which is what the progress bar counts
			if len(b.Points) > 0 {
				var open, inScope int
				for _, item := range c.items {
					if item.MilestoneID == burndownMilestone {
						inScope++
						if !item.IsClosed {
							open++
						}
					}
				}
				last := b.Points[len(b.Points)-1]
				assert.Equal(t, open, last.Remaining, "remaining today")
				assert.Equal(t, inScope, last.Scope, "scope today")
			}
		})
	}
}

func TestCalcMilestoneBurndownTimezone(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	m := &issues_model.Milestone{ID: burndownMilestone, CreatedUnix: at(t, "2026-01-04 12:00")}
	items := []issues_model.MilestoneItem{closedItem(1, at(t, "2026-01-04 12:00"), at(t, "2026-01-06 02:00"))}
	// 02:00 UTC on the 6th is still the evening of the 5th in New York
	events := (&eventSeq{}).join(1, at(t, "2026-01-04 13:00")).close(1, at(t, "2026-01-06 02:00")).events
	now := time.Unix(int64(at(t, "2026-01-07 12:00")), 0)

	utc := CalcMilestoneBurndown(m, items, events, now, time.UTC)
	assert.Equal(t, []string{"2026-01-04", "2026-01-05", "2026-01-06", "2026-01-07"}, datesOf(utc.Points))
	assert.Equal(t, []int{1, 1, 0, 0}, remainingOf(utc.Points))
	assert.Equal(t, "2026-01-06", utc.CompletedOn)

	local := CalcMilestoneBurndown(m, items, events, now, newYork)
	assert.Equal(t, []string{"2026-01-04", "2026-01-05", "2026-01-06", "2026-01-07"}, datesOf(local.Points))
	assert.Equal(t, []int{1, 0, 0, 0}, remainingOf(local.Points))
	assert.Equal(t, "2026-01-05", local.CompletedOn)

	t.Run("DSTEnds", func(t *testing.T) {
		// New York falls back on 2026-11-01, a 25-hour day that must still be exactly one point
		m := &issues_model.Milestone{ID: burndownMilestone, CreatedUnix: at(t, "2026-10-30 16:00")}
		items := []issues_model.MilestoneItem{openItem(1, at(t, "2026-10-30 16:00"))}
		events := (&eventSeq{}).join(1, at(t, "2026-10-30 17:00")).events
		b := CalcMilestoneBurndown(m, items, events, time.Unix(int64(at(t, "2026-11-03 16:00")), 0), newYork)
		assert.Equal(t, []string{"2026-10-30", "2026-10-31", "2026-11-01", "2026-11-02", "2026-11-03"}, datesOf(b.Points))
	})

	t.Run("DSTStarts", func(t *testing.T) {
		// and springs forward on 2026-03-08, a 23-hour day
		m := &issues_model.Milestone{ID: burndownMilestone, CreatedUnix: at(t, "2026-03-06 17:00"), DeadlineUnix: at(t, "2026-03-11 03:59") + 59}
		items := []issues_model.MilestoneItem{openItem(1, at(t, "2026-03-06 17:00"))}
		events := (&eventSeq{}).join(1, at(t, "2026-03-06 18:00")).events
		b := CalcMilestoneBurndown(m, items, events, time.Unix(int64(at(t, "2026-03-09 16:00")), 0), newYork)
		assert.Equal(t, []string{"2026-03-06", "2026-03-07", "2026-03-08", "2026-03-09"}, datesOf(b.Points))
		// a due date stored as 23:59:59 New York time is that day, not the next one in UTC
		assert.Equal(t, "2026-03-10", b.Deadline)
	})
}
