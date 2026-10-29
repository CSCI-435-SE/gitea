// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMilestoneProjection(t *testing.T) {
	days := func(n int) *int { return &n }
	projected := func(daysLate *int) *MilestoneBurndown {
		return &MilestoneBurndown{Status: BurndownProjected, Projected: "2026-11-12", DaysLate: daysLate}
	}
	key := func(name string) string { return burndownLocalePrefix + name }

	cases := []struct {
		name     string
		burndown *MilestoneBurndown
		want     *MilestoneProjection
	}{
		{"LateOneDay", projected(days(1)), &MilestoneProjection{
			Key1: key("projected_late_1"), KeyN: key("projected_late_n"), Date: "2026-11-12", Days: 1, WithDays: true, Late: true,
		}},
		{"LateDays", projected(days(4)), &MilestoneProjection{
			Key1: key("projected_late_1"), KeyN: key("projected_late_n"), Date: "2026-11-12", Days: 4, WithDays: true, Late: true,
		}},
		{"OnDueDate", projected(days(0)), &MilestoneProjection{
			Key1: key("projected_on_due_date"), KeyN: key("projected_on_due_date"), Date: "2026-11-12",
		}},
		{"EarlyOneDay", projected(days(-1)), &MilestoneProjection{
			Key1: key("projected_on_time_1"), KeyN: key("projected_on_time_n"), Date: "2026-11-12", Days: 1, WithDays: true,
		}},
		{"EarlyDays", projected(days(-6)), &MilestoneProjection{
			Key1: key("projected_on_time_1"), KeyN: key("projected_on_time_n"), Date: "2026-11-12", Days: 6, WithDays: true,
		}},
		// also what the burndown reports for a due date past the chart's horizon
		{"NoDueDate", projected(nil), &MilestoneProjection{
			Key1: key("projected"), KeyN: key("projected"), Date: "2026-11-12",
		}},
		{"NotBurning", &MilestoneBurndown{Status: BurndownNotBurning}, &MilestoneProjection{
			Key1: key("status_not_burning"), KeyN: key("status_not_burning"), NotBurning: true,
		}},
		{"Empty", &MilestoneBurndown{Status: BurndownEmpty}, nil},
		{"InsufficientData", &MilestoneBurndown{Status: BurndownInsufficientData}, nil},
		{"Done", &MilestoneBurndown{Status: BurndownDone, CompletedOn: "2026-11-01"}, nil},
		{"Closed", &MilestoneBurndown{Status: BurndownClosed}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, NewMilestoneProjection(c.burndown))
		})
	}
}
