// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	issues_model "gitea.dev/models/issues"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/util"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
)

// apiCloseReason is the close reason an API request gives in close_reason, close_reason_text and close_duplicate_of.
type apiCloseReason struct {
	name  string
	opts  issues_model.CloseReasonOptions
	given bool // without any of the fields, a close records the default reason
}

func newAPICloseReason(name, text string, duplicateOf int64) apiCloseReason {
	return apiCloseReason{
		name:  name,
		opts:  issues_model.CloseReasonOptions{Reason: issues_model.AsCloseReason(name), Text: text, DuplicateIndex: duplicateOf},
		given: name != "" || strings.TrimSpace(text) != "" || duplicateOf != 0, // Validate treats whitespace-only text as none
	}
}

// editCloses reports whether an edit's state field closes an open item.
func editCloses(state *string, issue *issues_model.Issue) bool {
	return state != nil && api.StateType(*state) == api.StateClosed && !issue.IsClosed
}

// check answers 422 and returns false when the reason can't be used. It runs before the request writes anything,
// so a rejected request leaves the item as it was. index is 0 for an item not created yet.
func (r apiCloseReason) check(ctx *context.APIContext, closes, isPull bool, index int64) bool {
	if !r.given {
		return true
	}
	if !closes { // changing the reason of a closed item is not supported
		ctx.APIError(http.StatusUnprocessableEntity, "close_reason, close_reason_text and close_duplicate_of are only allowed when the request closes an open issue or pull request")
		return false
	}
	if err := r.opts.Validate(isPull); err != nil {
		msg := err.Error()
		if issues_model.IsErrCloseReasonNotAllowed(err) { // its own message shows the stored number
			names := make([]string, 0, 4)
			for _, allowed := range issues_model.AllowedCloseReasons(isPull) {
				names = append(names, allowed.String())
			}
			msg = fmt.Sprintf("close_reason %q is not allowed for %s, use one of: %s", r.name, util.Iif(isPull, "a pull request", "an issue"), strings.Join(names, ", "))
		}
		var textErr issues_model.ErrInvalidCloseReasonText // its own message shows the stored number too
		if errors.As(err, &textErr) {
			msg = "close_reason_text: " + textErr.Detail
		}
		ctx.APIError(http.StatusUnprocessableEntity, msg)
		return false
	}
	if _, err := r.opts.DuplicateIssueID(ctx, ctx.Repo.Repository.ID, index); err != nil {
		if issues_model.IsErrInvalidCloseDuplicate(err) {
			ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		} else {
			ctx.APIErrorInternal(err)
		}
		return false
	}
	return true
}

// closeIssue closes the item with the request's reason, or the default one when it gave none,
// answering the error itself; it returns false when it did.
func (r apiCloseReason) closeIssue(ctx *context.APIContext, issue *issues_model.Issue) bool {
	var err error
	if r.given {
		err = issue_service.CloseIssueWithReason(ctx, issue, ctx.Doer, "", r.opts)
	} else {
		err = issue_service.CloseIssue(ctx, issue, ctx.Doer, "")
	}
	switch {
	case err == nil:
		return true
	case issues_model.IsErrDependenciesLeft(err):
		ctx.APIError(http.StatusPreconditionFailed, fmt.Sprintf("cannot close this %s because it still has open dependencies", util.Iif(issue.IsPull, "pull request", "issue")))
	case errors.Is(err, issues_model.ErrIssueAlreadyChanged): // a concurrent change won
		ctx.APIError(http.StatusConflict, err.Error())
	case errors.Is(err, util.ErrInvalidArgument): // check already ran; only a concurrent change gets here
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
	default:
		ctx.APIErrorInternal(err)
	}
	return false
}
