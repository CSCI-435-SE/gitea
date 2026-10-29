// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

const burndownLocalePrefix = "repo.milestones.burndown."

// MilestoneProjection is the one-line verdict a milestone list row shows. It reuses the chart summary's
// locale keys, so the list and the chart can't word the same projection differently.
type MilestoneProjection struct {
	Key1       string // the singular key; the same as KeyN when the sentence has no day count
	KeyN       string
	Date       string // the projected finish, "YYYY-MM-DD"; empty when the sentence names no date
	Days       int    // days after or before the due date, set only when WithDays
	WithDays   bool   // the sentence takes a day count after the date
	Late       bool
	NotBurning bool
}

// NewMilestoneProjection picks the chart summary's sentence for a burndown, mirroring burndownSummary in
// web_src/js/components/RepoMilestoneBurndown.utils.ts. It returns nil for a status the list leaves out.
func NewMilestoneProjection(b *MilestoneBurndown) *MilestoneProjection {
	switch b.Status {
	case BurndownNotBurning:
		return &MilestoneProjection{Key1: burndownLocalePrefix + "status_not_burning", KeyN: burndownLocalePrefix + "status_not_burning", NotBurning: true}
	case BurndownProjected:
		p := &MilestoneProjection{Date: b.Projected}
		switch {
		case b.DaysLate == nil: // no due date, or one too far out to draw
			p.Key1, p.KeyN = burndownLocalePrefix+"projected", burndownLocalePrefix+"projected"
		case *b.DaysLate == 0:
			p.Key1, p.KeyN = burndownLocalePrefix+"projected_on_due_date", burndownLocalePrefix+"projected_on_due_date"
		case *b.DaysLate > 0:
			p.Key1, p.KeyN = burndownLocalePrefix+"projected_late_1", burndownLocalePrefix+"projected_late_n"
			p.Days, p.WithDays, p.Late = *b.DaysLate, true, true
		default:
			p.Key1, p.KeyN = burndownLocalePrefix+"projected_on_time_1", burndownLocalePrefix+"projected_on_time_n"
			p.Days, p.WithDays = -*b.DaysLate, true
		}
		return p
	}
	return nil
}
