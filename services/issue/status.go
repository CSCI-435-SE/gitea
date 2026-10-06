// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/log"
	notify_service "gitea.dev/services/notify"
)

// CloseIssue closes an issue with the default reason for its kind: completed for an issue, not planned for a
// pull request. It is for closes that give no reason: commit keywords, merged pull requests closing the issues
// they reference, deleted branches, and API requests without one. Callers with a reason use CloseIssueWithReason.
func CloseIssue(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, commitID string) error {
	return CloseIssueWithReason(ctx, issue, doer, commitID, issues_model.CloseReasonOptions{Reason: issues_model.DefaultCloseReason(issue.IsPull)})
}

// CloseIssueWithReason closes an issue and records why; an invalid reason leaves the issue open.
func CloseIssueWithReason(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, commitID string, reason issues_model.CloseReasonOptions) error {
	var comment *issues_model.Comment
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		var err error
		comment, err = issues_model.CloseIssue(ctx, issue, doer, reason)
		if err != nil {
			if issues_model.IsErrDependenciesLeft(err) {
				if _, err := issues_model.FinishIssueStopwatch(ctx, doer, issue); err != nil {
					log.Error("Unable to stop stopwatch for issue[%d]#%d: %v", issue.ID, issue.Index, err)
				}
			}
			return err
		}

		_, err = issues_model.FinishIssueStopwatch(ctx, doer, issue)
		return err
	}); err != nil {
		return err
	}

	notify_service.IssueChangeStatus(ctx, doer, commitID, issue, comment, true)

	return nil
}

// ReopenIssue reopen an issue.
// FIXME: If some issues dependent this one are closed, should we also reopen them?
func ReopenIssue(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, commitID string) error {
	comment, err := issues_model.ReopenIssue(ctx, issue, doer)
	if err != nil {
		return err
	}

	notify_service.IssueChangeStatus(ctx, doer, commitID, issue, comment, false)

	return nil
}
