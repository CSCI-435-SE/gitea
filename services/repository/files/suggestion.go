// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/container"
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
	// ErrSuggestionBatchEmpty means a batch holds no suggestions (#159 AC12)
	ErrSuggestionBatchEmpty = errors.New("the batch holds no suggestions")
	// ErrSuggestionBatchTooLarge means a batch holds more than MaxSuggestionBatch suggestions (#159 AC14)
	ErrSuggestionBatchTooLarge = errors.New("the batch holds too many suggestions")
	// ErrSuggestionBatchDuplicate means a batch holds the same suggestion twice (#159 AC13)
	ErrSuggestionBatchDuplicate = errors.New("the batch holds a suggestion twice")
	// ErrSuggestionOverlap means two suggestions in a batch change the same line (#159 AC2)
	ErrSuggestionOverlap = errors.New("two suggestions in the batch change the same line")
)

// MaxSuggestionBatch is how many suggestions one batch may apply (#159 AC14)
const MaxSuggestionBatch = 50

// ErrSuggestionAt names the suggestion a batch was refused for, so the page can say which one (#159 AC3)
type ErrSuggestionAt struct {
	Comment *issues_model.Comment
	Err     error
}

func (err ErrSuggestionAt) Error() string {
	return fmt.Sprintf("%s:%d: %v", err.Comment.TreePath, err.Comment.Line, err.Err)
}

func (err ErrSuggestionAt) Unwrap() error {
	return err.Err
}

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
	if pr.HeadRepo.IsArchived {
		return "", ErrSuggestionPermission // nobody may push to an archived fork; the route only checks the base repo
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
	err := ApplySuggestions(ctx, doer, pr, []*issues_model.Comment{comment}, expectedHeadSHA, message)
	if at, ok := errors.AsType[ErrSuggestionAt](err); ok {
		return at.Err // one suggestion needs no name, so #156's messages stay as they were
	}
	return err
}

// suggestionEdit is one suggestion to apply: its comment, the line(s) it was written against and its text
type suggestionEdit struct {
	comment    *issues_model.Comment
	original   string
	suggestion string
}

// suggestionEditFor checks that the comment holds a suggestion this doer may apply (#156 AC7)
func suggestionEditFor(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest, comment *issues_model.Comment) (suggestionEdit, error) {
	original := comment.SuggestionOriginalLine()
	if comment.IssueID != pr.IssueID || !original.Has() {
		return suggestionEdit{}, ErrSuggestionNotApplicable
	}
	if comment.RangeStartLine() > 0 {
		return suggestionEdit{}, ErrSuggestionNotApplicable // applying a range is #189; until then the templates show no Apply on one
	}
	blocks := comment.SuggestionBlocks(ctx)
	if len(blocks) != 1 {
		return suggestionEdit{}, ErrSuggestionNotApplicable
	}
	// a pending review's comments are visible only to its reviewer (FetchCodeComments, the API), so for anyone
	// else the suggestion doesn't exist; applying it would publish a draft into the branch history
	if err := comment.LoadReview(ctx); err != nil {
		return suggestionEdit{}, err
	}
	if comment.Review != nil && comment.Review.Type == issues_model.ReviewTypePending && (doer == nil || comment.Review.ReviewerID != doer.ID) {
		return suggestionEdit{}, ErrSuggestionNotApplicable
	}
	return suggestionEdit{comment: comment, original: original.Value(), suggestion: blocks[0]}, nil
}

// ApplySuggestions commits the comments' suggestions to the pull request's head branch as one commit, or nothing
// if any of them can't be applied (#159). expectedHeadSHA is the head commit the page showed; message may be empty
// for the default.
func ApplySuggestions(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest, comments []*issues_model.Comment, expectedHeadSHA, message string) error {
	if len(comments) == 0 {
		return ErrSuggestionBatchEmpty
	}
	if len(comments) > MaxSuggestionBatch {
		return ErrSuggestionBatchTooLarge
	}
	seen := make(container.Set[int64], len(comments))
	edits := make([]suggestionEdit, 0, len(comments))
	for _, comment := range comments {
		if !seen.Add(comment.ID) {
			return ErrSuggestionBatchDuplicate
		}
		edit, err := suggestionEditFor(ctx, doer, pr, comment)
		if err != nil {
			return err
		}
		edits = append(edits, edit)
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
		if !protectedBranch.CanUserPush(ctx, doer) {
			for _, edit := range edits {
				if protectedBranch.UnprotectedFilePatterns == "" || !protectedBranch.IsUnprotectedFile(protectedBranch.GetUnprotectedFilePatterns(), edit.comment.TreePath) {
					return ErrSuggestionProtectedBranch{Branch: pr.HeadBranch}
				}
			}
		}
	}

	for _, edit := range edits {
		if edit.comment.Invalidated {
			// the page hides Apply on outdated comments, but a page loaded earlier may still post
			return ErrSuggestionAt{Comment: edit.comment, Err: ErrSuggestionOutdated}
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
	var paths []string // in batch order, so the commit is the same however often it is retried
	byPath := map[string][]suggestionEdit{}
	for _, edit := range edits {
		if _, ok := byPath[edit.comment.TreePath]; !ok {
			paths = append(paths, edit.comment.TreePath)
		}
		byPath[edit.comment.TreePath] = append(byPath[edit.comment.TreePath], edit)
	}
	files := make([]*ChangeRepoFile, 0, len(paths))
	for _, path := range paths {
		blob, err := headCommit.GetBlobByPath(path)
		if git.IsErrNotExist(err) {
			return ErrSuggestionAt{Comment: byPath[path][0].comment, Err: ErrSuggestionOutdated} // deleted or renamed since the review
		} else if err != nil {
			return err
		}
		content, err := blob.GetBlobContent(blob.Size())
		if err != nil {
			return err
		}
		newContent, err := applySuggestionsToContent(content, byPath[path])
		if err != nil {
			return err
		}
		files = append(files, &ChangeRepoFile{Operation: "update", TreePath: path, FromTreePath: path, ContentReader: strings.NewReader(newContent)})
	}

	for _, edit := range edits {
		if err := edit.comment.LoadPoster(ctx); err != nil {
			return err
		}
	}
	_, err = ChangeRepoFiles(ctx, pr.HeadRepo, doer, &ChangeRepoFilesOptions{
		LastCommitID: expectedHeadSHA,
		OldBranch:    pr.HeadBranch,
		NewBranch:    pr.HeadBranch,
		Message:      suggestionCommitMessage(comments, doer, message),
		Files:        files,
	})
	if IsErrCommitIDDoesNotMatch(err) {
		return ErrSuggestionHeadMoved // someone pushed between our check and the commit
	}
	return err
}

// applySuggestionsToContent applies one file's suggestions bottom-up, so each edit leaves the line numbers of the
// ones above it as they were (#159 AC9). Two suggestions that share a line are refused (#159 AC2).
func applySuggestionsToContent(content string, edits []suggestionEdit) (string, error) {
	edits = slices.Clone(edits)
	slices.SortFunc(edits, func(a, b suggestionEdit) int { return cmp.Compare(b.comment.Line, a.comment.Line) })
	for i := 1; i < len(edits); i++ {
		if edits[i].comment.Line >= edits[i-1].startLine() {
			return "", ErrSuggestionOverlap
		}
	}
	for _, edit := range edits {
		var err error
		content, err = applySuggestionToContent(content, int(edit.comment.Line), edit.comment.SuggestionContextLines(), edit.original, edit.suggestion)
		if err != nil {
			return "", ErrSuggestionAt{Comment: edit.comment, Err: err}
		}
	}
	return content, nil
}

// startLine is the first line the suggestion replaces: its comment's line, or the start of the range it covers
func (edit suggestionEdit) startLine() int64 {
	return edit.comment.Line - int64(strings.Count(edit.original, "\n"))
}

// suggestionCommitMessage is the doer's message, or the default, crediting each reviewer once (#156 AC1, #159 AC1).
// The comments' posters must be loaded.
func suggestionCommitMessage(comments []*issues_model.Comment, doer *user_model.User, message string) string {
	if strings.TrimSpace(message) == "" {
		if len(comments) == 1 {
			message = comments[0].SuggestionCommitMessage()
		} else {
			message = fmt.Sprintf("Apply %d suggestions from code review", len(comments))
		}
	}
	// customer question 1 on #151 is open; the spec's default: the doer authors and commits, the reviewers are credited
	credited := make(container.Set[int64])
	var trailers []string
	for _, comment := range comments {
		if comment.Poster.ID == doer.ID || comment.Poster.IsGhost() || !credited.Add(comment.Poster.ID) { // a deleted reviewer has no address to credit
			continue
		}
		sig := comment.Poster.NewGitSig()
		trailers = append(trailers, fmt.Sprintf("%s: %s <%s>", git.CoAuthoredByTrailer, sig.Name, sig.Email))
	}
	if len(trailers) == 0 {
		return message
	}
	return strings.TrimRight(message, "\n") + "\n\n" + strings.Join(trailers, "\n")
}

// applySuggestionToContent replaces line lineNum (1-based) of content with the suggestion's lines, the way
// #154 draws the diff: "" deletes the line, "\n" leaves one blank line, otherwise one line per suggestion line.
// original is the line the suggestion was written against, and above the lines the reviewer saw right above it;
// content and both of those may use CRLF.
func applySuggestionToContent(content string, lineNum int, above []string, original, suggestion string) (string, error) {
	lines := strings.SplitAfter(content, "\n") // every element but the last keeps its "\n"
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1] // content ended with a newline
	}
	idx := lineNum - 1
	if idx < len(above) || idx >= len(lines) { // also refuses lineNum < 1, and a line that moved too near the top
		return "", ErrSuggestionOutdated
	}
	// blame-based invalidation misses a line that moved onto an identical one, like another "}"; its neighbours don't
	for i, want := range above {
		if strings.TrimSuffix(strings.TrimSuffix(lines[idx-len(above)+i], "\n"), "\r") != strings.TrimSuffix(want, "\r") {
			return "", ErrSuggestionOutdated
		}
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
