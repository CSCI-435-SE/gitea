// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"gitea.dev/models/db"

	"xorm.io/xorm"
)

func AddReasonToNotification(x db.EngineMigration) error {
	type Notification struct {
		Reason int `xorm:"SMALLINT NOT NULL DEFAULT 0"` // 0 is "Other": why an existing notification was sent isn't known
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
		IgnoreConstrains:  true,
	}, new(Notification))
	return err
}
