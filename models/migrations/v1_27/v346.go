// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"gitea.dev/models/db"

	"xorm.io/xorm"
)

func AddSuggestionAppliedToComment(x db.EngineMigration) error {
	type Comment struct {
		SuggestionAppliedByID   int64  `xorm:"NOT NULL DEFAULT 0"`
		SuggestionAppliedCommit string `xorm:"VARCHAR(64) NOT NULL DEFAULT ''"` // empty: the suggestion hasn't been applied
		SuggestionAppliedUnix   int64  `xorm:"NOT NULL DEFAULT 0"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
		IgnoreConstrains:  true,
	}, new(Comment))
	return err
}
