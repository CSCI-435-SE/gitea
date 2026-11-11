// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"
	"slices"

	"gitea.dev/models/db"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/container"
	"gitea.dev/modules/timeutil"
	"gitea.dev/modules/util"

	"xorm.io/builder"
)

// ErrDependencyExists represents a "DependencyAlreadyExists" kind of error.
type ErrDependencyExists struct {
	IssueID      int64
	DependencyID int64
}

// IsErrDependencyExists checks if an error is a ErrDependencyExists.
func IsErrDependencyExists(err error) bool {
	_, ok := err.(ErrDependencyExists)
	return ok
}

func (err ErrDependencyExists) Error() string {
	return fmt.Sprintf("issue dependency does already exist [issue id: %d, dependency id: %d]", err.IssueID, err.DependencyID)
}

func (err ErrDependencyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrDependencyNotExists represents a "DependencyAlreadyExists" kind of error.
type ErrDependencyNotExists struct {
	IssueID      int64
	DependencyID int64
}

// IsErrDependencyNotExists checks if an error is a ErrDependencyExists.
func IsErrDependencyNotExists(err error) bool {
	_, ok := err.(ErrDependencyNotExists)
	return ok
}

func (err ErrDependencyNotExists) Error() string {
	return fmt.Sprintf("issue dependency does not exist [issue id: %d, dependency id: %d]", err.IssueID, err.DependencyID)
}

func (err ErrDependencyNotExists) Unwrap() error {
	return util.ErrNotExist
}

// ErrCircularDependency represents a "DependencyCircular" kind of error.
type ErrCircularDependency struct {
	IssueID      int64
	DependencyID int64
}

// IsErrCircularDependency checks if an error is a ErrCircularDependency.
func IsErrCircularDependency(err error) bool {
	_, ok := err.(ErrCircularDependency)
	return ok
}

func (err ErrCircularDependency) Error() string {
	return fmt.Sprintf("circular dependencies exists (issues blocking each other in a loop) [issue id: %d, dependency id: %d]", err.IssueID, err.DependencyID)
}

func (err ErrCircularDependency) Unwrap() error {
	return util.ErrUnprocessableContent
}

// ErrDependenciesLeft represents an error where the issue you're trying to close still has dependencies left.
type ErrDependenciesLeft struct {
	IssueID int64
}

// IsErrDependenciesLeft checks if an error is a ErrDependenciesLeft.
func IsErrDependenciesLeft(err error) bool {
	_, ok := err.(ErrDependenciesLeft)
	return ok
}

func (err ErrDependenciesLeft) Error() string {
	return fmt.Sprintf("issue has open dependencies [issue id: %d]", err.IssueID)
}

// ErrUnknownDependencyType represents an error where an unknown dependency type was passed
type ErrUnknownDependencyType struct {
	Type DependencyType
}

func (err ErrUnknownDependencyType) Error() string {
	return fmt.Sprintf("unknown dependency type [type: %d]", err.Type)
}

func (err ErrUnknownDependencyType) Unwrap() error {
	return util.ErrInvalidArgument
}

// IssueDependency represents an issue dependency
type IssueDependency struct {
	ID           int64              `xorm:"pk autoincr"`
	UserID       int64              `xorm:"NOT NULL"`
	IssueID      int64              `xorm:"UNIQUE(issue_dependency) NOT NULL"`
	DependencyID int64              `xorm:"UNIQUE(issue_dependency) NOT NULL"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(IssueDependency))
}

// DependencyType Defines Dependency Type Constants
type DependencyType int

// Define Dependency Types
const (
	DependencyTypeBlockedBy DependencyType = iota
	DependencyTypeBlocking
)

// CreateIssueDependency creates a new dependency for an issue
func CreateIssueDependency(ctx context.Context, user *user_model.User, issue, dep *Issue) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		// Check if it already exists
		exists, err := issueDepExists(ctx, issue.ID, dep.ID)
		if err != nil {
			return err
		}
		if exists {
			return ErrDependencyExists{issue.ID, dep.ID}
		}
		// And if it would be circular
		circular, err := dependencyLoopExists(ctx, issue.ID, dep.ID)
		if err != nil {
			return err
		}
		if circular {
			return ErrCircularDependency{issue.ID, dep.ID}
		}

		if err := db.Insert(ctx, &IssueDependency{
			UserID:       user.ID,
			IssueID:      issue.ID,
			DependencyID: dep.ID,
		}); err != nil {
			return err
		}

		// Add comment referencing the new dependency
		return createIssueDependencyComment(ctx, user, issue, dep, true)
	})
}

// RemoveIssueDependency removes a dependency from an issue
func RemoveIssueDependency(ctx context.Context, user *user_model.User, issue, dep *Issue, depType DependencyType) (err error) {
	return db.WithTx(ctx, func(ctx context.Context) error {
		var issueDepToDelete IssueDependency

		switch depType {
		case DependencyTypeBlockedBy:
			issueDepToDelete = IssueDependency{IssueID: issue.ID, DependencyID: dep.ID}
		case DependencyTypeBlocking:
			issueDepToDelete = IssueDependency{IssueID: dep.ID, DependencyID: issue.ID}
		default:
			return ErrUnknownDependencyType{depType}
		}

		affected, err := db.GetEngine(ctx).Delete(&issueDepToDelete)
		if err != nil {
			return err
		}

		// If we deleted nothing, the dependency did not exist
		if affected <= 0 {
			return ErrDependencyNotExists{issue.ID, dep.ID}
		}

		// Add comment referencing the removed dependency
		return createIssueDependencyComment(ctx, user, issue, dep, false)
	})
}

// Check if the dependency already exists
func issueDepExists(ctx context.Context, issueID, depID int64) (bool, error) {
	return db.GetEngine(ctx).Where("(issue_id = ? AND dependency_id = ?)", issueID, depID).Exist(&IssueDependency{})
}

// dependencyLoopExists checks if depID already depends on issueID, directly or through other issues,
// so making issueID depend on depID would close a loop; issueID == depID is a loop of one.
func dependencyLoopExists(ctx context.Context, issueID, depID int64) (bool, error) {
	seen := container.SetOf(depID) // loops made before this check existed must not walk forever
	next := []int64{depID}
	for len(next) > 0 {
		if slices.Contains(next, issueID) {
			return true, nil
		}
		var depIDs []int64
		for batch := range slices.Chunk(next, db.DefaultMaxInSize) { // a long IN list can pass the database's parameter limit
			var batchDepIDs []int64
			if err := db.GetEngine(ctx).Table("issue_dependency").In("issue_id", batch).Cols("dependency_id").Find(&batchDepIDs); err != nil {
				return false, err
			}
			depIDs = append(depIDs, batchDepIDs...)
		}
		next = next[:0]
		for _, id := range depIDs {
			if seen.Add(id) {
				next = append(next, id)
			}
		}
	}
	return false, nil
}

// IssueNoDependenciesLeft checks if issue can be closed
func IssueNoDependenciesLeft(ctx context.Context, issue *Issue) (bool, error) {
	exists, err := db.GetEngine(ctx).
		Table("issue_dependency").
		Select("issue.*").
		Join("INNER", "issue", "issue.id = issue_dependency.dependency_id").
		Where("issue_dependency.issue_id = ?", issue.ID).
		And("issue.is_closed = ?", "0").
		Exist(&Issue{})

	return !exists, err
}

// MergeBlockedByDependencies checks if pr depends on an open issue or on a pull request that is not merged.
// Closing ignores whether a pull request was merged (IssueNoDependenciesLeft); merging must not.
func MergeBlockedByDependencies(ctx context.Context, pr *PullRequest) (bool, error) {
	return db.GetEngine(ctx).
		Table("issue_dependency").
		Select("issue.*").
		Join("INNER", "issue", "issue.id = issue_dependency.dependency_id").
		Join("LEFT", "pull_request", "pull_request.issue_id = issue.id").
		Where("issue_dependency.issue_id = ?", pr.IssueID).
		And(builder.Or(
			builder.Eq{"issue.is_closed": false},
			builder.Eq{"pull_request.has_merged": false}, // NULL for an issue, so only a pull request matches
		)).
		Exist(&Issue{})
}
