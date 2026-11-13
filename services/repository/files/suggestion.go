// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"context"
	"errors"
	"fmt"
	"strings"

	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/gitrepo"
)

var (
	// ErrSuggestionNotApplicable means the comment has no suggestion that can be applied: not a new-side code comment
	// on this pull request, or not exactly one suggestion block (#156 AC7)
	ErrSuggestionNotApplicable = errors.New("the comment has no suggestion to apply")
	// ErrSuggestionPullClosed means the pull request is closed or merged, or has no head branch to commit to (#156 AC5)
	ErrSuggestionPullClosed = errors.New("the pull request is closed or its head branch is gone")
	// ErrSuggestionPermission means the doer cannot push to the head branch (#156 AC2)
	ErrSuggestionPermission = errors.New("the doer cannot push to the head branch")
	// ErrSuggestionHeadMoved means the head branch has new commits since the page was loaded (#156 AC4)
	ErrSuggestionHeadMoved = errors.New("the head branch has moved")
	// ErrSuggestionOutdated means the commented line no longer reads as it did when the suggestion was written (#156 AC3)
	ErrSuggestionOutdated = errors.New("the suggestion is outdated")
	// ErrSuggestionNoChange means applying the suggestion would leave the file as it is (#156 AC10)
	ErrSuggestionNoChange = errors.New("the suggestion makes no changes")
)

// ErrSuggestionProtectedBranch means branch protection stops the doer from pushing to the head branch (#156 AC6)
type ErrSuggestionProtectedBranch struct {
	Branch string
}

func (err ErrSuggestionProtectedBranch) Error() string {
	return fmt.Sprintf("branch %q is protected", err.Branch)
}

// CanApplySuggestions checks what does not depend on the comment: the pull request is open, has a head branch
// (AGit pull requests have none) and the doer can push to it. It returns the head branch's commit ID.
func CanApplySuggestions(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) (string, error) {
	if doer == nil {
		return "", ErrSuggestionPermission
	}
	if err := pr.LoadIssue(ctx); err != nil {
		return "", err
	}
	if pr.HasMerged || pr.Issue.IsClosed || pr.Flow != issues_model.PullRequestFlowGithub {
		return "", ErrSuggestionPullClosed
	}
	if err := pr.LoadHeadRepo(ctx); err != nil {
		return "", err
	}
	if pr.HeadRepo == nil {
		return "", ErrSuggestionPullClosed // the fork was deleted
	}
	headPerm, err := access_model.GetDoerRepoPermission(ctx, pr.HeadRepo, doer)
	if err != nil {
		return "", err
	}
	// includes "Allow edits from maintainers" on fork pull requests
	if !issues_model.CanMaintainerWriteToBranch(ctx, headPerm, pr.HeadBranch, doer) {
		return "", ErrSuggestionPermission
	}
	headGitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, pr.HeadRepo)
	if err != nil {
		return "", err
	}
	defer closer.Close()
	headCommitID, err := headGitRepo.GetBranchCommitID(pr.HeadBranch)
	if git.IsErrNotExist(err) {
		return "", ErrSuggestionPullClosed // the head branch was deleted
	}
	return headCommitID, err
}

// ApplySuggestion commits the comment's single suggestion to the pull request's head branch, replacing the
// commented line. expectedHeadSHA is the head commit the page showed; message may be empty for the default.
func ApplySuggestion(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest, comment *issues_model.Comment, expectedHeadSHA, message string) error {
	original := comment.SuggestionOriginalLine()
	if comment.IssueID != pr.IssueID || !original.Has() {
		return ErrSuggestionNotApplicable
	}
	blocks := comment.SuggestionBlocks(ctx)
	if len(blocks) != 1 {
		return ErrSuggestionNotApplicable
	}

	headCommitID, err := CanApplySuggestions(ctx, doer, pr)
	if err != nil {
		return err
	}
	if headCommitID != expectedHeadSHA {
		return ErrSuggestionHeadMoved
	}

	// the web editor's check, so the user gets our message rather than the pre-receive hook's output
	protectedBranch, err := git_model.GetFirstMatchProtectedBranchRule(ctx, pr.HeadRepo.ID, pr.HeadBranch)
	if err != nil {
		return err
	}
	if protectedBranch != nil {
		protectedBranch.Repo = pr.HeadRepo
		canPush := protectedBranch.CanUserPush(ctx, doer)
		if !canPush && protectedBranch.UnprotectedFilePatterns != "" {
			canPush = protectedBranch.IsUnprotectedFile(protectedBranch.GetUnprotectedFilePatterns(), comment.TreePath)
		}
		if !canPush {
			return ErrSuggestionProtectedBranch{Branch: pr.HeadBranch}
		}
	}

	headGitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, pr.HeadRepo)
	if err != nil {
		return err
	}
	defer closer.Close()
	headCommit, err := headGitRepo.GetCommit(headCommitID)
	if err != nil {
		return err
	}
	blob, err := headCommit.GetBlobByPath(comment.TreePath)
	if git.IsErrNotExist(err) {
		return ErrSuggestionOutdated // deleted or renamed since the review
	} else if err != nil {
		return err
	}
	content, err := blob.GetBlobContent(blob.Size())
	if err != nil {
		return err
	}
	newContent, err := applySuggestionToContent(content, int(comment.Line), original.Value(), blocks[0])
	if err != nil {
		return err
	}

	if err := comment.LoadPoster(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(message) == "" {
		message = comment.SuggestionCommitMessage()
	}
	// customer question 1 on #151 is open; the spec's default: the doer authors and commits, the reviewer is credited
	if comment.Poster.ID != doer.ID {
		sig := comment.Poster.NewGitSig()
		message = fmt.Sprintf("%s\n\n%s: %s <%s>", strings.TrimRight(message, "\n"), git.CoAuthoredByTrailer, sig.Name, sig.Email)
	}

	_, err = ChangeRepoFiles(ctx, pr.HeadRepo, doer, &ChangeRepoFilesOptions{
		LastCommitID: expectedHeadSHA,
		OldBranch:    pr.HeadBranch,
		NewBranch:    pr.HeadBranch,
		Message:      message,
		Files: []*ChangeRepoFile{{
			Operation:     "update",
			TreePath:      comment.TreePath,
			FromTreePath:  comment.TreePath,
			ContentReader: strings.NewReader(newContent),
		}},
	})
	if IsErrCommitIDDoesNotMatch(err) {
		return ErrSuggestionHeadMoved // someone pushed between our check and the commit
	}
	return err
}

// applySuggestionToContent replaces line lineNum (1-based) of content with the suggestion's lines, the way
// #154 draws the diff: "" deletes the line, "\n" leaves one blank line, otherwise one line per suggestion line.
// original is the line the suggestion was written against; content and original may use CRLF.
func applySuggestionToContent(content string, lineNum int, original, suggestion string) (string, error) {
	lines := strings.SplitAfter(content, "\n") // every element but the last keeps its "\n"
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1] // content ended with a newline
	}
	idx := lineNum - 1
	if idx < 0 || idx >= len(lines) {
		return "", ErrSuggestionOutdated
	}

	target := lines[idx]
	ending := ""
	switch {
	case strings.HasSuffix(target, "\r\n"):
		ending = "\r\n"
	case strings.HasSuffix(target, "\n"):
		ending = "\n"
	}
	if strings.TrimSuffix(target, ending) != strings.TrimSuffix(original, "\r") {
		return "", ErrSuggestionOutdated
	}

	var replacement string
	if suggestion != "" {
		innerEnding := ending
		if innerEnding == "" { // the last line has no newline, so new lines in between take the file's own
			innerEnding = "\n"
			if strings.Contains(content, "\r\n") {
				innerEnding = "\r\n"
			}
		}
		newLines := strings.Split(strings.TrimSuffix(suggestion, "\n"), "\n")
		replacement = strings.Join(newLines, innerEnding) + ending
	}

	before := strings.Join(lines[:idx], "")
	if replacement == "" && ending == "" && idx > 0 {
		// deleting a last line that has no newline: the line before it becomes last and loses its newline,
		// so the file still ends without one (#156 AC11)
		prev := lines[idx-1]
		before = strings.Join(lines[:idx-1], "") + strings.TrimSuffix(strings.TrimSuffix(prev, "\n"), "\r")
	}
	result := before + replacement + strings.Join(lines[idx+1:], "")
	if result == content {
		return "", ErrSuggestionNoChange
	}
	return result, nil
}
