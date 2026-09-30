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

// CloseReasonUnknown is never stored: it stands for an unrecognised reason name, so that Validate rejects it.
const CloseReasonUnknown CloseReason = -1

// closeReasonNames are what forms and pages send instead of the stored numbers, so existing names must never change.
// CloseReasonNone has no name: an empty name means no reason.
var closeReasonNames = map[CloseReason]string{
	CloseReasonCompleted:  "completed",
	CloseReasonNotPlanned: "not_planned",
	CloseReasonDuplicate:  "duplicate",
	CloseReasonOther:      "other",
}

// String returns the reason's name, or "" for no reason.
func (r CloseReason) String() string {
	return closeReasonNames[r] // a map rather than CommentType's slice, so CloseReasonUnknown can't index out of range
}

// AsCloseReason returns the reason with the given name; an empty name means no reason,
// and an unknown one returns CloseReasonUnknown so that Validate rejects it.
func AsCloseReason(name string) CloseReason {
	if name == "" {
		return CloseReasonNone
	}
	for reason, reasonName := range closeReasonNames {
		if reasonName == name {
			return reason
		}
	}
	return CloseReasonUnknown
}

// CloseReasonTextMaxLength matches the VARCHAR(255) close_reason_text column and the close form's MaxSize(255)
// rule. Validate counts characters the same way MaxSize does, so paths that skip the form get the same limit.
// MSSQL's NVARCHAR counts UTF-16 units instead, so text with many emoji can pass this limit and still not fit.
const CloseReasonTextMaxLength = 255

// AllowedCloseReasons lists the reasons a person can pick when closing.
// A pull request is completed by merging, so it cannot be closed as completed.
func AllowedCloseReasons(isPull bool) []CloseReason {
	if isPull {
		return []CloseReason{CloseReasonNotPlanned, CloseReasonDuplicate, CloseReasonOther}
	}
	return []CloseReason{CloseReasonCompleted, CloseReasonNotPlanned, CloseReasonDuplicate, CloseReasonOther}
}

// DefaultCloseReason is the reason the close button starts on; consumers may still send any allowed reason, or none.
func DefaultCloseReason(isPull bool) CloseReason {
	if isPull {
		return CloseReasonNotPlanned
	}
	return CloseReasonCompleted
}

// CloseReasonOptions is the reason a person gives when closing an issue or pull request.
type CloseReasonOptions struct {
	Reason         CloseReason
	Text           string // only for CloseReasonOther
	DuplicateIndex int64  // only for CloseReasonDuplicate: the number of the issue it duplicates, in the same repository
}

// Validate checks the options for an issue or pull request, without looking anything up;
// SetIssueAsClosed checks that the duplicate target exists.
// No reason at all is valid, so close paths that give none keep working.
func (opts CloseReasonOptions) Validate(isPull bool) error {
	if opts.Reason != CloseReasonNone && !slices.Contains(AllowedCloseReasons(isPull), opts.Reason) {
		return ErrCloseReasonNotAllowed{Reason: opts.Reason, IsPull: isPull}
	}

	if opts.Reason == CloseReasonDuplicate {
		if opts.DuplicateIndex <= 0 {
			return ErrInvalidCloseDuplicate{Index: opts.DuplicateIndex, Detail: "no issue number given"}
		}
	} else if opts.DuplicateIndex != 0 {
		return ErrInvalidCloseDuplicate{Index: opts.DuplicateIndex, Detail: "an issue number is only allowed with the duplicate reason"}
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

// ErrInvalidCloseDuplicate represents a duplicate target that is missing, is the issue itself, or is not in the same repository.
type ErrInvalidCloseDuplicate struct {
	Index  int64
	Detail string
}

// IsErrInvalidCloseDuplicate checks if an error is a ErrInvalidCloseDuplicate.
func IsErrInvalidCloseDuplicate(err error) bool {
	_, ok := err.(ErrInvalidCloseDuplicate)
	return ok
}

func (err ErrInvalidCloseDuplicate) Error() string {
	return fmt.Sprintf("invalid duplicate target: %s [index: %d]", err.Detail, err.Index)
}

func (err ErrInvalidCloseDuplicate) Unwrap() error {
	return util.ErrInvalidArgument
}
