// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"gitea.dev/modules/util"
)

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

// CloseReasonTextMaxLength matches the VARCHAR(255) close_reason_text column, counted in characters.
const CloseReasonTextMaxLength = 255

// AllowedCloseReasons lists the reasons a person can pick when closing.
// A pull request is completed by merging, so it cannot be closed as completed.
func AllowedCloseReasons(isPull bool) []CloseReason {
	if isPull {
		return []CloseReason{CloseReasonNotPlanned, CloseReasonDuplicate, CloseReasonOther}
	}
	return []CloseReason{CloseReasonCompleted, CloseReasonNotPlanned, CloseReasonDuplicate, CloseReasonOther}
}

// CloseReasonOptions is the reason a person gives when closing an issue or pull request.
type CloseReasonOptions struct {
	Reason CloseReason
	Text   string // only for CloseReasonOther
}

// Validate checks the options for an issue or pull request.
// CloseReasonNone is always valid, so close paths that give no reason keep working.
func (opts CloseReasonOptions) Validate(isPull bool) error {
	if opts.Reason != CloseReasonNone && !slices.Contains(AllowedCloseReasons(isPull), opts.Reason) {
		return ErrCloseReasonNotAllowed{Reason: opts.Reason, IsPull: isPull}
	}

	hasText := strings.TrimSpace(opts.Text) != "" // whitespace-only counts as no text
	if opts.Reason != CloseReasonOther {
		if hasText {
			return ErrInvalidCloseReasonText{Reason: opts.Reason, Detail: "text is only allowed with the other reason"}
		}
		return nil
	}
	if !hasText {
		return ErrInvalidCloseReasonText{Reason: opts.Reason, Detail: "text is empty"}
	}
	if utf8.RuneCountInString(opts.Text) > CloseReasonTextMaxLength {
		return ErrInvalidCloseReasonText{Reason: opts.Reason, Detail: fmt.Sprintf("text is longer than %d characters", CloseReasonTextMaxLength)}
	}
	return nil
}

// ErrCloseReasonNotAllowed represents a reason that is out of range, or not allowed for the issue or pull request.
type ErrCloseReasonNotAllowed struct {
	Reason CloseReason
	IsPull bool
}

// IsErrCloseReasonNotAllowed checks if an error is a ErrCloseReasonNotAllowed.
func IsErrCloseReasonNotAllowed(err error) bool {
	_, ok := err.(ErrCloseReasonNotAllowed)
	return ok
}

func (err ErrCloseReasonNotAllowed) Error() string {
	return fmt.Sprintf("close reason is not allowed for this %s [reason: %d]", util.Iif(err.IsPull, "pull request", "issue"), err.Reason)
}

func (err ErrCloseReasonNotAllowed) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrInvalidCloseReasonText represents close reason text that is empty, too long, or sent with a reason other than CloseReasonOther.
type ErrInvalidCloseReasonText struct {
	Reason CloseReason
	Detail string
}

// IsErrInvalidCloseReasonText checks if an error is a ErrInvalidCloseReasonText.
func IsErrInvalidCloseReasonText(err error) bool {
	_, ok := err.(ErrInvalidCloseReasonText)
	return ok
}

func (err ErrInvalidCloseReasonText) Error() string {
	return fmt.Sprintf("invalid close reason text: %s [reason: %d]", err.Detail, err.Reason)
}

func (err ErrInvalidCloseReasonText) Unwrap() error {
	return util.ErrInvalidArgument
}
