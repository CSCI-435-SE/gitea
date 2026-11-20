// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"strings"
	"testing"

	"gitea.dev/models/db"
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySuggestionToContent(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		line       int
		original   string
		suggestion string
		expected   string
		err        error
	}{
		// AC1, AC12: only the commented line changes, as #154's diff shows it
		{"AC1 replaces only the commented line", "a\nb\nc\n", 2, "b", "B\n", "a\nB\nc\n", nil},
		{"AC1 first line", "a\nb\n", 1, "a", "A\n", "A\nb\n", nil},
		{"AC12 empty block deletes the line", "a\nb\nc\n", 2, "b", "", "a\nc\n", nil},
		{"AC12 one blank line replaces it with a blank line", "a\nb\nc\n", 2, "b", "\n", "a\n\nc\n", nil},
		{"AC12 several lines replace it in order", "a\nb\nc\n", 2, "b", "x\ny\n", "a\nx\ny\nc\n", nil},

		// AC11: line endings and the final newline are kept
		{"AC11 CRLF file keeps CRLF", "a\r\nb\r\nc\r\n", 2, "b\r", "B\n", "a\r\nB\r\nc\r\n", nil},
		{"AC11 several lines in a CRLF file all end in CRLF", "a\r\nb\r\nc\r\n", 2, "b\r", "x\ny\n", "a\r\nx\r\ny\r\nc\r\n", nil},
		{"AC11 last line without a final newline stays without one", "a\nb", 2, "b", "B\n", "a\nB", nil},
		{"AC11 several lines on a last line without a final newline", "a\nb", 2, "b", "x\ny\n", "a\nx\ny", nil},
		{"AC11 several lines on a CRLF last line without a final newline", "a\r\nb", 2, "b", "x\ny\n", "a\r\nx\r\ny", nil},
		{"AC11 deleting a last line without a final newline", "a\nb", 2, "b", "", "a", nil},
		{"AC11 deleting a last line that has a final newline", "a\nb\n", 2, "b", "", "a\n", nil},

		// AC3: the line no longer reads as the suggestion expects
		{"AC3 line changed since the review", "a\nX\nc\n", 2, "b", "B\n", "", ErrSuggestionOutdated},
		{"AC3 trailing whitespace counts as a change", "a\nb \nc\n", 2, "b", "B\n", "", ErrSuggestionOutdated},
		{"AC3 line number past the end of the file", "a\nb\n", 3, "c", "C\n", "", ErrSuggestionOutdated},
		{"AC3 empty file", "", 1, "a", "A\n", "", ErrSuggestionOutdated},

		// AC10: a suggestion identical to the line commits nothing
		{"AC10 same text makes no change", "a\nb\n", 2, "b", "b\n", "", ErrSuggestionNoChange},
		{"AC10 same text in a CRLF file makes no change", "a\r\nb\r\n", 2, "b\r", "b\n", "", ErrSuggestionNoChange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := applySuggestionToContent(c.content, c.line, nil, c.original, c.suggestion)
			assert.ErrorIs(t, err, c.err)
			assert.Equal(t, c.expected, result)
		})
	}

	// AC3 with the lines the reviewer saw above the commented one, as #179's review asked
	moved := []struct {
		name     string
		content  string
		line     int
		above    []string
		original string
		expected string
		err      error
	}{
		{"AC1 the lines above still match", "a\nb\nc\n", 3, []string{"a", "b"}, "c", "a\nb\nC\n", nil},
		{"AC1 the lines above match in a CRLF file", "a\r\nb\r\n", 2, []string{"a\r"}, "b\r", "a\r\nB\r\n", nil},
		// a "}" reviewed on line 4, below "b"; two lines pushed above it put the "}" below "a" on line 4
		{"AC3 the line moved onto an identical one", "x\ny\na\n}\nb\n}\n", 4, []string{"b"}, "}", "", ErrSuggestionOutdated},
		{"AC3 a line above changed", "a\nB\nc\n", 3, []string{"a", "b"}, "c", "", ErrSuggestionOutdated},
		{"AC3 the line moved too near the top for its lines above", "b\nc\n", 1, []string{"a"}, "b", "", ErrSuggestionOutdated},
	}
	for _, c := range moved {
		t.Run(c.name, func(t *testing.T) {
			result, err := applySuggestionToContent(c.content, c.line, c.above, c.original, strings.ToUpper(strings.TrimSuffix(c.original, "\r"))+"\n")
			assert.ErrorIs(t, err, c.err)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestSuggestionCommitMessage(t *testing.T) {
	reviewer := &user_model.User{ID: 5, Name: "ana", Email: "ana@example.com"}
	doer := &user_model.User{ID: 2, Name: "bo", Email: "bo@example.com"}
	comment := func(poster *user_model.User) *issues_model.Comment {
		return &issues_model.Comment{PosterID: poster.ID, Poster: poster, TreePath: "main.go"}
	}

	// AC1, and #156 Decision 3's default for customer question 1: the reviewer is credited as a co-author
	assert.Equal(t, "Apply suggestion from @ana on main.go\n\nCo-authored-by: ana <ana@example.com>", suggestionCommitMessage([]*issues_model.Comment{comment(reviewer)}, doer, ""))
	assert.Equal(t, "Use err\n\nCo-authored-by: ana <ana@example.com>", suggestionCommitMessage([]*issues_model.Comment{comment(reviewer)}, doer, "Use err\n"))
	// nobody to credit: the doer's own suggestion, or a reviewer whose account was deleted (#179 review)
	assert.Equal(t, "Apply suggestion from @bo on main.go", suggestionCommitMessage([]*issues_model.Comment{comment(doer)}, doer, "  "))
	assert.Equal(t, "Use err", suggestionCommitMessage([]*issues_model.Comment{comment(user_model.NewGhostUser())}, doer, "Use err"))

	// #159 AC1: several suggestions get "Apply <N> suggestions from code review", and each reviewer other than the
	// applier is credited once, in batch order
	carl := &user_model.User{ID: 6, Name: "carl", Email: "carl@example.com"}
	batch := []*issues_model.Comment{comment(reviewer), comment(doer), comment(carl), comment(reviewer), comment(user_model.NewGhostUser())}
	assert.Equal(t, "Apply 5 suggestions from code review\n\nCo-authored-by: ana <ana@example.com>\nCo-authored-by: carl <carl@example.com>", suggestionCommitMessage(batch, doer, ""))
	assert.Equal(t, "Tidy up\n\nCo-authored-by: ana <ana@example.com>\nCo-authored-by: carl <carl@example.com>", suggestionCommitMessage(batch, doer, "Tidy up"))
	assert.Equal(t, "Apply 2 suggestions from code review", suggestionCommitMessage([]*issues_model.Comment{comment(doer), comment(doer)}, doer, ""))
}

// #159 AC9 and AC2: one file's suggestions are applied bottom-up, and two that share a line are refused
func TestApplySuggestionsToContent(t *testing.T) {
	edit := func(line int64, original, suggestion string) suggestionEdit {
		return suggestionEdit{
			comment:    &issues_model.Comment{Type: issues_model.CommentTypeCode, Line: line, TreePath: "main.go", Patch: "@@ -1,1 +1,1 @@\n+" + original},
			original:   original,
			suggestion: suggestion,
		}
	}
	const content = "l1\nl2\nl3\nl4\nl5\nl6\n"

	t.Run("AC9 one line becomes three above, a line deleted below: the same as applying them one at a time", func(t *testing.T) {
		got, err := applySuggestionsToContent(content, []suggestionEdit{edit(2, "l2", "a\nb\nc\n"), edit(5, "l5", "")})
		require.NoError(t, err)
		assert.Equal(t, "l1\na\nb\nc\nl3\nl4\nl6\n", got)
		// the order the suggestions were selected in doesn't matter
		got, err = applySuggestionsToContent(content, []suggestionEdit{edit(5, "l5", ""), edit(2, "l2", "a\nb\nc\n")})
		require.NoError(t, err)
		assert.Equal(t, "l1\na\nb\nc\nl3\nl4\nl6\n", got)
	})

	t.Run("neighbouring lines both apply", func(t *testing.T) {
		got, err := applySuggestionsToContent(content, []suggestionEdit{edit(3, "l3", "x\n"), edit(4, "l4", "y\n")})
		require.NoError(t, err)
		assert.Equal(t, "l1\nl2\nx\ny\nl5\nl6\n", got)
	})

	t.Run("AC2 two suggestions on the same line overlap", func(t *testing.T) {
		_, err := applySuggestionsToContent(content, []suggestionEdit{edit(3, "l3", "x\n"), edit(3, "l3", "y\n")})
		assert.ErrorIs(t, err, ErrSuggestionOverlap)
	})

	t.Run("AC2 a range that covers another suggestion's line overlaps", func(t *testing.T) {
		_, err := applySuggestionsToContent(content, []suggestionEdit{edit(4, "l3\nl4", "x\n"), edit(3, "l3", "y\n")})
		assert.ErrorIs(t, err, ErrSuggestionOverlap)
	})

	t.Run("AC3 an outdated suggestion is named", func(t *testing.T) {
		_, err := applySuggestionsToContent(content, []suggestionEdit{edit(2, "l2", "x\n"), edit(5, "changed", "y\n")})
		var at ErrSuggestionAt
		require.ErrorAs(t, err, &at)
		assert.ErrorIs(t, err, ErrSuggestionOutdated)
		assert.EqualValues(t, 5, at.Comment.Line)
	})
}

// The checks before the commit (#156), against fixture pull request 2: user2/repo1, head branch2, whose README.md
// has "And change for branch2" on line 5. The commit itself needs git hooks, so it is in the integration test.
func TestApplySuggestion(t *testing.T) {
	unittest.PrepareTestEnv(t)
	const branch2Head = "985f0301dba5e7b34be866819cd15ad3d8f508ee" // tests/gitea-repositories-meta/user2/repo1.git
	const masterHead = "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})  // owns repo1
	reader := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // cannot push to repo1
	loadPR := func(t *testing.T, id int64) *issues_model.PullRequest {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: id})
		require.NoError(t, pr.LoadIssue(t.Context()))
		return pr
	}
	suggest := func(pr *issues_model.PullRequest, treePath, original, content string) *issues_model.Comment {
		return &issues_model.Comment{
			Type: issues_model.CommentTypeCode, IssueID: pr.IssueID, PosterID: 4, TreePath: treePath, Line: 5,
			Patch: "@@ -4,1 +4,2 @@\n \n+" + original, Content: content,
		}
	}
	const fix = "```suggestion\nAnd a change for branch2\n```"

	t.Run("AC1 someone who can push gets the head branch's commit", func(t *testing.T) {
		head, err := CanApplySuggestions(t.Context(), owner, loadPR(t, 2))
		require.NoError(t, err)
		assert.Equal(t, branch2Head, head)
	})

	cannotApply := []struct {
		name   string
		doer   *user_model.User
		change func(pr *issues_model.PullRequest)
		err    error
	}{
		{"AC2 signed out", nil, nil, ErrSuggestionPermission},
		{"AC2 cannot push", reader, nil, ErrSuggestionPermission},
		{"AC2 nobody can push to an archived head repository", owner, func(pr *issues_model.PullRequest) {
			pr.HeadRepo = &repo_model.Repository{ID: repo.ID, OwnerID: repo.OwnerID, Name: repo.Name, IsArchived: true}
		}, ErrSuggestionPermission},
		{"AC5 closed", owner, func(pr *issues_model.PullRequest) { pr.Issue.IsClosed = true }, ErrSuggestionPullClosed},
		{"AC5 merged", owner, func(pr *issues_model.PullRequest) { pr.HasMerged = true }, ErrSuggestionPullClosed},
		{"AC5 AGit flow", owner, func(pr *issues_model.PullRequest) { pr.Flow = issues_model.PullRequestFlowAGit }, ErrSuggestionPullClosed},
		{"AC5 head branch deleted", owner, func(pr *issues_model.PullRequest) { pr.HeadBranch = "deleted-branch" }, ErrSuggestionPullClosed},
	}
	for _, c := range cannotApply {
		t.Run(c.name, func(t *testing.T) {
			pr := loadPR(t, 2)
			if c.change != nil {
				c.change(pr)
			}
			_, err := CanApplySuggestions(t.Context(), c.doer, pr)
			assert.ErrorIs(t, err, c.err)
			// ApplySuggestion makes the same check before anything else that touches the branch
			assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), c.doer, pr, suggest(pr, "README.md", "And change for branch2", fix), branch2Head, "")), c.err)
		})
	}

	refused := []struct {
		name    string
		comment func(pr *issues_model.PullRequest) *issues_model.Comment
		head    string
		err     error
	}{
		{"AC7 a comment from another pull request", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.IssueID = 2
			return c
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 an old-side comment", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.Line = -5
			return c
		}, branch2Head, ErrSuggestionNotApplicable},
		{"a range suggestion until #189 applies ranges", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.CommentMetaData = &issues_model.CommentMetaData{StartLine: 4} // lines 4-5, both still as the patch shows them
			return c
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 no suggestion block", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", "looks good")
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC7 two suggestion blocks", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", fix+"\nor\n"+fix)
		}, branch2Head, ErrSuggestionNotApplicable},
		{"AC4 the page showed an older head", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", fix)
		}, masterHead, ErrSuggestionHeadMoved},
		{"AC3 the file was deleted or renamed", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "gone.md", "And change for branch2", fix)
		}, branch2Head, ErrSuggestionOutdated},
		{"AC3 the line changed", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And an older change for branch2", fix)
		}, branch2Head, ErrSuggestionOutdated},
		{"AC3 the line reads the same but the line above it changed", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.Patch = "@@ -4,1 +4,2 @@\n Not a blank line\n+And change for branch2" // line 4 at branch2 is blank
			return c
		}, branch2Head, ErrSuggestionOutdated},
		{"AC3 a comment marked outdated", func(pr *issues_model.PullRequest) *issues_model.Comment {
			c := suggest(pr, "README.md", "And change for branch2", fix)
			c.Invalidated = true
			return c
		}, branch2Head, ErrSuggestionOutdated},
		{"AC10 the suggestion is the line as it is", func(pr *issues_model.PullRequest) *issues_model.Comment {
			return suggest(pr, "README.md", "And change for branch2", "```suggestion\nAnd change for branch2\n```")
		}, branch2Head, ErrSuggestionNoChange},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			pr := loadPR(t, 2)
			assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), owner, pr, c.comment(pr), c.head, "")), c.err)
		})
	}

	t.Run("a suggestion in someone else's pending review does not exist for the doer", func(t *testing.T) {
		pending := &issues_model.Review{Type: issues_model.ReviewTypePending, ReviewerID: 4, IssueID: 3}
		require.NoError(t, db.Insert(t.Context(), pending))
		pr := loadPR(t, 2)
		c := suggest(pr, "README.md", "And change for branch2", fix)
		c.ReviewID = pending.ID
		assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), owner, pr, c, branch2Head, "")), ErrSuggestionNotApplicable)
	})

	t.Run("the reviewer may apply a suggestion from their own pending review", func(t *testing.T) {
		pending := &issues_model.Review{Type: issues_model.ReviewTypePending, ReviewerID: owner.ID, IssueID: 3}
		require.NoError(t, db.Insert(t.Context(), pending))
		pr := loadPR(t, 2)
		c := suggest(pr, "README.md", "And an older change for branch2", fix) // outdated, so it stops before committing
		c.PosterID, c.ReviewID = owner.ID, pending.ID
		assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), owner, pr, c, branch2Head, "")), ErrSuggestionOutdated)
	})

	t.Run("AC6 a protected branch", func(t *testing.T) {
		rule := &git_model.ProtectedBranch{RepoID: repo.ID, RuleName: "branch2", CanPush: false}
		require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, rule, git_model.WhitelistOptions{}))
		defer func() { require.NoError(t, git_model.DeleteProtectedBranch(t.Context(), repo, rule.ID)) }()

		pr := loadPR(t, 2)
		_, err := ApplySuggestion(t.Context(), owner, pr, suggest(pr, "README.md", "And change for branch2", fix), branch2Head, "")
		var errProtected ErrSuggestionProtectedBranch
		require.ErrorAs(t, err, &errProtected)
		assert.Equal(t, "branch2", errProtected.Branch)

		// like the web editor, a file matching the rule's unprotected patterns may still be changed,
		// so this one gets past protection and stops at its outdated line instead
		rule.UnprotectedFilePatterns = "README.md"
		require.NoError(t, git_model.UpdateProtectBranch(t.Context(), repo, rule, git_model.WhitelistOptions{}))
		_, err = ApplySuggestion(t.Context(), owner, pr, suggest(pr, "README.md", "And an older change for branch2", fix), branch2Head, "")
		assert.ErrorIs(t, err, ErrSuggestionOutdated)
	})

	// #159: a batch is refused as a whole before anything is committed
	batch := func(pr *issues_model.PullRequest, n int) []*issues_model.Comment {
		comments := make([]*issues_model.Comment, n)
		for i := range comments {
			comments[i] = suggest(pr, "README.md", "And change for branch2", fix)
			comments[i].ID = int64(1000 + i)
		}
		return comments
	}

	t.Run("AC12 an empty batch", func(t *testing.T) {
		assert.ErrorIs(t, errOf(ApplySuggestions(t.Context(), owner, loadPR(t, 2), nil, branch2Head, "")), ErrSuggestionBatchEmpty)
	})

	t.Run("AC14 a batch of 51", func(t *testing.T) {
		pr := loadPR(t, 2)
		assert.ErrorIs(t, errOf(ApplySuggestions(t.Context(), owner, pr, batch(pr, 51), branch2Head, "")), ErrSuggestionBatchTooLarge)
	})

	t.Run("AC13 the same suggestion twice", func(t *testing.T) {
		pr := loadPR(t, 2)
		comments := batch(pr, 2)
		comments[1].ID = comments[0].ID
		assert.ErrorIs(t, errOf(ApplySuggestions(t.Context(), owner, pr, comments, branch2Head, "")), ErrSuggestionBatchDuplicate)
	})

	t.Run("AC3 an outdated suggestion in a batch is named, and a single Apply keeps #156's bare error", func(t *testing.T) {
		pr := loadPR(t, 2)
		comments := batch(pr, 2)
		comments[1].Invalidated = true
		_, err := ApplySuggestions(t.Context(), owner, pr, comments, branch2Head, "")
		var at ErrSuggestionAt
		require.ErrorAs(t, err, &at)
		assert.ErrorIs(t, err, ErrSuggestionOutdated)
		assert.Same(t, comments[1], at.Comment)

		_, err = ApplySuggestion(t.Context(), owner, pr, comments[1], branch2Head, "")
		assert.Equal(t, ErrSuggestionOutdated, err)
	})

	t.Run("AC2 two suggestions on the same line", func(t *testing.T) {
		pr := loadPR(t, 2)
		assert.ErrorIs(t, errOf(ApplySuggestions(t.Context(), owner, pr, batch(pr, 2), branch2Head, "")), ErrSuggestionOverlap)
	})

	// #157 AC2: an applied suggestion is refused as applied before any of #156's other refusals
	applied := func(c *issues_model.Comment) *issues_model.Comment {
		c.SuggestionAppliedByID, c.SuggestionAppliedCommit = owner.ID, masterHead
		return c
	}
	alreadyApplied := []struct {
		name   string
		doer   *user_model.User
		change func(pr *issues_model.PullRequest, c *issues_model.Comment)
		head   string
	}{
		{"AC2 already applied", owner, nil, branch2Head},
		{"AC2 already applied wins over a closed pull request", owner, func(pr *issues_model.PullRequest, _ *issues_model.Comment) { pr.Issue.IsClosed = true }, branch2Head},
		{"AC2 already applied wins over no push permission", reader, nil, branch2Head},
		{"AC2 already applied wins over a moved head", owner, nil, masterHead},
		{"AC2 already applied wins over an outdated comment", owner, func(_ *issues_model.PullRequest, c *issues_model.Comment) { c.Invalidated = true }, branch2Head},
	}
	for _, c := range alreadyApplied {
		t.Run(c.name, func(t *testing.T) {
			pr := loadPR(t, 2)
			comment := applied(suggest(pr, "README.md", "And change for branch2", fix))
			if c.change != nil {
				c.change(pr, comment)
			}
			assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), c.doer, pr, comment, c.head, "")), ErrSuggestionAlreadyApplied)
		})
	}

	t.Run("AC2 a comment that has nothing to apply is still not found once applied", func(t *testing.T) {
		pr := loadPR(t, 2)
		comment := applied(suggest(pr, "README.md", "And change for branch2", fix+"\nor\n"+fix))
		assert.ErrorIs(t, errOf(ApplySuggestion(t.Context(), owner, pr, comment, branch2Head, "")), ErrSuggestionNotApplicable)
	})

	t.Run("#159 AC10 an applied suggestion in a batch is refused and named", func(t *testing.T) {
		pr := loadPR(t, 2)
		comments := batch(pr, 2)
		applied(comments[1])
		_, err := ApplySuggestions(t.Context(), owner, pr, comments, branch2Head, "")
		var at ErrSuggestionAt
		require.ErrorAs(t, err, &at)
		assert.ErrorIs(t, err, ErrSuggestionAlreadyApplied)
		assert.Same(t, comments[1], at.Comment)
	})
}

// errOf drops ApplySuggestion's commit SHA for the cases that only check its error
func errOf(_ string, err error) error {
	return err
}
