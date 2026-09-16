// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"time"

	"gitea.dev/modules/timeutil"
)

// NearDueDuration is the window before a deadline in which an open issue, pull request
// or milestone counts as near due. Deadlines are pinned to 23:59:59 local time by
// routers/common/deadline.go, so in practice this covers the next three calendar days.
const NearDueDuration = 72 * time.Hour

// calcDeadlineStatus computes the overdue / near-due state shared by Issue and Milestone.
// The two are mutually exclusive: a passed deadline is overdue, never near due.
func calcDeadlineStatus(deadlineUnix timeutil.TimeStamp, isClosed bool, closedUnix timeutil.TimeStamp) (isOverdue, isNearDue bool) {
	if deadlineUnix == 0 {
		return false, false
	}
	if isClosed {
		return closedUnix >= deadlineUnix, false
	}
	now := timeutil.TimeStampNow()
	if now >= deadlineUnix {
		return true, false
	}
	return false, deadlineUnix <= now.AddDuration(NearDueDuration)
}
