// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

// CloseReason records why an issue or pull request was closed.
// It is stored as an integer, so existing values must never be renumbered.
type CloseReason int

const (
	CloseReasonNone       CloseReason = iota // 0
	CloseReasonCompleted                     // 1
	CloseReasonNotPlanned                    // 2
	CloseReasonDuplicate                     // 3
	CloseReasonOther                         // 4
)
