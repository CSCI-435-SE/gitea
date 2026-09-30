// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"gitea.dev/models/db"

	"xorm.io/xorm"
)

func AddCloseReasonToIssue(x db.EngineMigration) error {
	type Issue struct {
		CloseReason           int    `xorm:"SMALLINT NOT NULL DEFAULT 0"`
		CloseReasonText       string `xorm:"VARCHAR(255) NOT NULL DEFAULT ''"`
		CloseDuplicateIssueID int64  `xorm:"NOT NULL DEFAULT 0"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
		IgnoreConstrains:  true,
	}, new(Issue))
	return err
}
