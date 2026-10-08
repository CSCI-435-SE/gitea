// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"
	"time"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/optional"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDueDateFilter(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	wednesday := time.Date(2026, 10, 7, 15, 30, 0, 0, loc)
	local := func(unix int64) string { // "" for an unset bound
		if unix == 0 {
			return ""
		}
		return time.Unix(unix, 0).In(loc).Format(time.DateTime)
	}

	cases := []struct {
		name          string
		due           string
		now           time.Time
		after, before string
		hasDeadline   optional.Option[bool]
	}{
		{"overdue is anything due up to now", "overdue", wednesday, "", "2026-10-07 15:30:00", optional.Some(true)},
		{"soon is the near-due badge window", "soon", wednesday, "2026-10-07 15:30:01", local(wednesday.Add(issues_model.NearDueDuration).Unix()), optional.Some(true)},
		{"today is the local calendar day", "today", wednesday, "2026-10-07 00:00:00", "2026-10-07 23:59:59", optional.Some(true)},
		{"today on the 25-hour day daylight saving ends", "today", time.Date(2026, 11, 1, 12, 0, 0, 0, loc), "2026-11-01 00:00:00", "2026-11-01 23:59:59", optional.Some(true)},
		{"week runs Monday to Sunday", "week", wednesday, "2026-10-05 00:00:00", "2026-10-11 23:59:59", optional.Some(true)},
		{"week on its first second", "week", time.Date(2026, 10, 5, 0, 0, 0, 0, loc), "2026-10-05 00:00:00", "2026-10-11 23:59:59", optional.Some(true)},
		{"week on its last second", "week", time.Date(2026, 10, 11, 23, 59, 59, 0, loc), "2026-10-05 00:00:00", "2026-10-11 23:59:59", optional.Some(true)},
		{"week across the end of daylight saving", "week", time.Date(2026, 10, 28, 9, 0, 0, 0, loc), "2026-10-26 00:00:00", "2026-11-01 23:59:59", optional.Some(true)},
		{"set has no bounds", "set", wednesday, "", "", optional.Some(true)},
		{"none matches undated issues", "none", wednesday, "", "", optional.Some(false)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := ParseDueDateFilter(c.due, c.now)
			assert.Equal(t, c.due, f.Name)
			assert.Equal(t, c.after, local(f.AfterUnix))
			assert.Equal(t, c.before, local(f.BeforeUnix))
			assert.Equal(t, c.hasDeadline, f.HasDeadline)
		})
	}

	t.Run("unknown values turn the filter off", func(t *testing.T) {
		for _, due := range []string{"", "bogus", "Overdue"} {
			assert.Equal(t, DueDateFilter{}, ParseDueDateFilter(due, wednesday), due)
		}
	})

	t.Run("apply copies the bounds", func(t *testing.T) {
		opts := &issues_model.IssuesOptions{}
		ParseDueDateFilter("today", wednesday).Apply(opts)
		assert.Equal(t, "2026-10-07 00:00:00", local(opts.DeadlineAfterUnix))
		assert.Equal(t, "2026-10-07 23:59:59", local(opts.DeadlineBeforeUnix))
		assert.Equal(t, optional.Some(true), opts.HasDeadline)
	})
}
