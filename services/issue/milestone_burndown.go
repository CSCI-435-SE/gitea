// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"cmp"
	"context"
	"math"
	"slices"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
)

// BurndownStatus says what the chart can tell about when a milestone will finish.
type BurndownStatus string

const (
	BurndownEmpty            BurndownStatus = "empty"             // nothing has ever been in the milestone
	BurndownDone             BurndownStatus = "done"              // every item in it is closed
	BurndownClosed           BurndownStatus = "closed"            // the milestone was closed with items still open
	BurndownProjected        BurndownStatus = "projected"         // a finish date could be extrapolated
	BurndownNotBurning       BurndownStatus = "not_burning"       // remaining work has not fallen over the window
	BurndownInsufficientData BurndownStatus = "insufficient_data" // too few days of history to extrapolate
)

const (
	burndownWindowDays     = 14  // the projection extrapolates the net burn rate over at most this many days
	burndownMinHistoryDays = 2   // a rate over fewer days is noise, not a trend
	burndownHorizonDays    = 365 // a due date further out than this past the chart is not drawn
)

// BurndownPoint is the state of a milestone at the end of one day.
type BurndownPoint struct {
	Date      string `json:"date"`      // "YYYY-MM-DD" in the instance zone, so every viewer sees the same day
	Remaining int    `json:"remaining"` // items in the milestone and open
	Scope     int    `json:"scope"`     // items in the milestone, open or closed
	Added     int    `json:"added"`     // items that joined the milestone this day, after its first day of work
	Removed   int    `json:"removed"`   // items that left the milestone this day, after its first day of work

	Changes []BurndownChange `json:"changes,omitempty"` // what moved the counts this day, in order; omitted on quiet days
}

// BurndownChangeKind is how one item changed the milestone's counts.
type BurndownChangeKind string

const (
	BurndownChangeClosed   BurndownChangeKind = "closed" // a merge counts as a close
	BurndownChangeReopened BurndownChangeKind = "reopened"
	BurndownChangeAdded    BurndownChangeKind = "added"
	BurndownChangeRemoved  BurndownChangeKind = "removed"
)

// BurndownChange is one item that changed the milestone's counts on a day.
type BurndownChange struct {
	Index  int64              `json:"index"`
	Title  string             `json:"title"`
	IsPull bool               `json:"isPull"`
	Kind   BurndownChangeKind `json:"kind"`
}

// BurndownOptions says which kinds of item to count. A caller must leave out any kind the viewer cannot read.
type BurndownOptions struct {
	Issues bool
	Pulls  bool
}

// BurndownIdeal is the straight line from the first day with work to zero on the due date.
type BurndownIdeal struct {
	From      string `json:"from"`
	FromValue int    `json:"fromValue"`
	To        string `json:"to"` // the due date; the line always ends at zero
}

// MilestoneBurndown is everything the milestone page charts.
type MilestoneBurndown struct {
	Points      []BurndownPoint `json:"points"`
	Deadline    string          `json:"deadline"` // empty when there is no due date, or it is too far out to draw
	Ideal       *BurndownIdeal  `json:"ideal"`
	Status      BurndownStatus  `json:"status"`
	Projected   string          `json:"projected"`   // set only when Status is projected
	DaysLate    *int            `json:"daysLate"`    // projected finish minus the due date; negative is early
	CompletedOn string          `json:"completedOn"` // set only when Status is done
}

type burndownState struct {
	inScope bool
	open    bool
}

type burndownDelta struct {
	at        int64
	remaining int
	scope     int
	item      *issues_model.MilestoneItem
}

func (d burndownDelta) moves() bool {
	return d.remaining != 0 || d.scope != 0
}

// kind names the change for the day's list; a change of scope outranks a close, since an item that
// joins already closed was added, not closed. It is empty when the counts did not move.
func (d burndownDelta) kind() BurndownChangeKind {
	switch {
	case d.scope > 0:
		return BurndownChangeAdded
	case d.scope < 0:
		return BurndownChangeRemoved
	case d.remaining < 0:
		return BurndownChangeClosed
	case d.remaining > 0:
		return BurndownChangeReopened
	}
	return ""
}

func (s burndownState) counts() (remaining, scope int) {
	if s.inScope {
		scope = 1
		if s.open {
			remaining = 1
		}
	}
	return remaining, scope
}

// diff is the change in the milestone's counts when an item moves from s to next
func (s burndownState) diff(next burndownState, at int64) burndownDelta {
	r0, s0 := s.counts()
	r1, s1 := next.counts()
	return burndownDelta{at: at, remaining: r1 - r0, scope: s1 - s0}
}

// replayMilestoneItem turns one item's comments into changes to the milestone's counts. Missing history
// is filled in so the replay always ends where the database is now: an item in the milestone with no
// comment saying it joined (migrated issues have none) joins when it was created or the milestone was,
// and a closed item with no close comment closes at its closed_unix.
func replayMilestoneItem(item issues_model.MilestoneItem, events []issues_model.MilestoneEvent, milestoneID, milestoneCreated int64) []burndownDelta {
	events = slices.Clone(events) // synthetic events are appended below
	created := int64(item.CreatedUnix)
	var firstScope *issues_model.MilestoneEvent
	hasClose := false
	for i := range events {
		switch events[i].Type {
		case issues_model.CommentTypeMilestone:
			if firstScope == nil {
				firstScope = &events[i]
			}
		case issues_model.CommentTypeClose, issues_model.CommentTypeMergePull:
			hasClose = true
		}
	}

	// synthetic events take id 0 so they sort before a real comment in the same second
	joinAt := max(created, milestoneCreated)
	if firstScope == nil && item.MilestoneID == milestoneID {
		events = append(events, issues_model.MilestoneEvent{Type: issues_model.CommentTypeMilestone, MilestoneID: milestoneID, CreatedUnix: timeutil.TimeStamp(joinAt)})
	} else if firstScope != nil && firstScope.MilestoneID != milestoneID {
		// its first recorded move is out of the milestone, so it was in it before then
		joinAt = min(joinAt, int64(firstScope.CreatedUnix))
		events = append(events, issues_model.MilestoneEvent{Type: issues_model.CommentTypeMilestone, MilestoneID: milestoneID, CreatedUnix: timeutil.TimeStamp(joinAt)})
	}
	if item.IsClosed && !hasClose {
		// a close time of zero predates the item, so clamp it as the Activity issues chart does
		closedAt := max(int64(item.ClosedUnix), created)
		events = append(events, issues_model.MilestoneEvent{Type: issues_model.CommentTypeClose, CreatedUnix: timeutil.TimeStamp(closedAt)})
	}
	slices.SortFunc(events, issues_model.CompareMilestoneEvents)

	state := burndownState{open: true}
	lastAt := created
	deltas := make([]burndownDelta, 0, len(events)+1)
	for _, e := range events {
		next := state
		switch e.Type {
		case issues_model.CommentTypeMilestone:
			if e.MilestoneID == milestoneID {
				next.inScope = true
			} else if e.OldMilestoneID == milestoneID {
				next.inScope = false
			}
		case issues_model.CommentTypeClose, issues_model.CommentTypeMergePull:
			next.open = false
		case issues_model.CommentTypeReopen:
			next.open = true
		}
		// a close or reopen while the item is outside the milestone moves no count; recording it would
		// stretch a closed milestone's chart to a change that never touched it
		if d := state.diff(next, int64(e.CreatedUnix)); d.moves() {
			deltas = append(deltas, d)
			lastAt = max(lastAt, d.at)
		}
		state = next
	}

	// the comments can still disagree with the row, for instance after a reopen whose close was never
	// recorded, and the row is the truth the progress bar shows, so end on it
	want := burndownState{inScope: item.MilestoneID == milestoneID, open: !item.IsClosed}
	if state != want {
		at := lastAt
		if state.open && !want.open {
			at = max(at, int64(item.ClosedUnix))
		}
		if d := state.diff(want, at); d.moves() {
			deltas = append(deltas, d)
		}
	}
	return deltas
}

// civilDay counts calendar days, so differences between dates ignore DST and zone offsets
func civilDay(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix() / (24 * 60 * 60)
}

func startOfDay(unix int64, loc *time.Location) time.Time {
	t := time.Unix(unix, 0).In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// CalcMilestoneBurndown charts a milestone from its items and their history. It reads no clock and no
// global zone, so it is tested with a fixed now and loc.
//
// Every item it is given counts, whatever its close reason; which kinds it is given is the caller's choice
// (see BurndownOptions).
func CalcMilestoneBurndown(m *issues_model.Milestone, items []issues_model.MilestoneItem, events []issues_model.MilestoneEvent, now time.Time, loc *time.Location) *MilestoneBurndown {
	result := &MilestoneBurndown{Points: []BurndownPoint{}, Status: BurndownEmpty}
	if len(items) == 0 {
		return result
	}

	byItem := make(map[int64][]issues_model.MilestoneEvent, len(items))
	for _, e := range events {
		byItem[e.IssueID] = append(byItem[e.IssueID], e)
	}
	var deltas []burndownDelta
	for i := range items {
		for _, d := range replayMilestoneItem(items[i], byItem[items[i].ID], m.ID, int64(m.CreatedUnix)) {
			d.item = &items[i]
			deltas = append(deltas, d)
		}
	}
	slices.SortStableFunc(deltas, func(a, b burndownDelta) int { return cmp.Compare(a.at, b.at) })

	start, end := int64(m.CreatedUnix), now.Unix()
	if start == 0 { // fixtures and some legacy rows have no creation time
		start = end
		if len(deltas) > 0 {
			start = deltas[0].at
		}
	}
	if m.IsClosed && m.ClosedDateUnix > 0 {
		// stop at the close, unless items moved after it: the last point must match today's counters
		end = int64(m.ClosedDateUnix)
		if len(deltas) > 0 {
			end = max(end, deltas[len(deltas)-1].at)
		}
		end = min(end, now.Unix())
	}
	end = max(end, start)

	firstDay, lastDay := startOfDay(start, loc), startOfDay(end, loc)
	remaining, scope, next := 0, 0, 0
	for day := firstDay; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1).Unix()
		point := BurndownPoint{Date: day.Format(time.DateOnly)}
		for ; next < len(deltas) && deltas[next].at < dayEnd; next++ {
			remaining += deltas[next].remaining
			scope += deltas[next].scope
			if deltas[next].scope > 0 {
				point.Added++
			} else if deltas[next].scope < 0 {
				point.Removed++
			}
			if kind := deltas[next].kind(); kind != "" {
				item := deltas[next].item
				point.Changes = append(point.Changes, BurndownChange{Index: item.Index, Title: item.Title, IsPull: item.IsPull, Kind: kind})
			}
		}
		point.Remaining, point.Scope = remaining, scope
		result.Points = append(result.Points, point)
	}
	points := result.Points
	last := len(points) - 1

	// what the milestone holds on its first day of work is its plan, not a change to it, so only
	// later days are marked; otherwise every chart would open on an "added" marker
	from := slices.IndexFunc(points, func(p BurndownPoint) bool { return p.Scope > 0 })
	for i := 0; i <= from; i++ {
		points[i].Added, points[i].Removed = 0, 0
		points[i].Changes = slices.DeleteFunc(points[i].Changes, func(c BurndownChange) bool {
			return c.Kind == BurndownChangeAdded || c.Kind == BurndownChangeRemoved
		})
		if len(points[i].Changes) == 0 {
			points[i].Changes = nil // keep quiet days out of the JSON
		}
	}

	var deadlineDay time.Time
	if m.DeadlineUnix > 0 {
		deadlineDay = startOfDay(int64(m.DeadlineUnix), loc)
		if civilDay(deadlineDay)-civilDay(lastDay) > burndownHorizonDays {
			deadlineDay = time.Time{} // the fixtures' year 9999, say: drawing it would flatten the chart
		} else {
			result.Deadline = deadlineDay.Format(time.DateOnly)
		}
	}

	if from >= 0 && !deadlineDay.IsZero() {
		fromDay := firstDay.AddDate(0, 0, from)
		if points[from].Remaining > 0 && deadlineDay.After(fromDay) {
			result.Ideal = &BurndownIdeal{From: points[from].Date, FromValue: points[from].Remaining, To: result.Deadline}
		}
	}

	switch {
	case points[last].Scope == 0:
		result.Status = BurndownEmpty
	case points[last].Remaining == 0:
		result.Status = BurndownDone
		done := last
		for done > 0 && points[done-1].Remaining == 0 && points[done-1].Scope > 0 {
			done--
		}
		result.CompletedOn = points[done].Date
	case m.IsClosed:
		result.Status = BurndownClosed
	// history counts from the first day with work: days before it hold nothing to burn, and a window
	// reaching back into them would read the work arriving as work growing
	case last-from < burndownMinHistoryDays:
		result.Status = BurndownInsufficientData
	default:
		window := min(burndownWindowDays, last-from)
		rate := float64(points[last-window].Remaining-points[last].Remaining) / float64(window)
		if rate <= 0 {
			result.Status = BurndownNotBurning
			break
		}
		projected := lastDay.AddDate(0, 0, int(math.Ceil(float64(points[last].Remaining)/rate)))
		result.Status = BurndownProjected
		result.Projected = projected.Format(time.DateOnly)
		if !deadlineDay.IsZero() {
			late := int(civilDay(projected) - civilDay(deadlineDay))
			result.DaysLate = &late
		}
	}
	return result
}

// GetMilestoneBurndown charts a milestone in the instance's zone, the zone its due date is stored in,
// counting only the kinds of item opts asks for.
func GetMilestoneBurndown(ctx context.Context, m *issues_model.Milestone, opts BurndownOptions) (*MilestoneBurndown, error) {
	scopeEvents, err := issues_model.GetMilestoneScopeEvents(ctx, m.RepoID, m.ID)
	if err != nil {
		return nil, err
	}
	movedIDs := make([]int64, 0, len(scopeEvents))
	for _, e := range scopeEvents {
		movedIDs = append(movedIDs, e.IssueID)
	}
	slices.Sort(movedIDs)
	items, err := issues_model.GetMilestoneItems(ctx, m.RepoID, m.ID, slices.Compact(movedIDs))
	if err != nil {
		return nil, err
	}
	// a left-out kind is not counted and not listed, so a viewer never learns of items they cannot read
	items = slices.DeleteFunc(items, func(item issues_model.MilestoneItem) bool {
		return item.IsPull && !opts.Pulls || !item.IsPull && !opts.Issues
	})

	itemIDs := make([]int64, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	stateEvents, err := issues_model.GetMilestoneStateEvents(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	return CalcMilestoneBurndown(m, items, append(scopeEvents, stateEvents...), time.Now(), setting.DefaultUILocation), nil
}
