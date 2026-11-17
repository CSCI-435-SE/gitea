# Mid-Sprint Check-in: @controlled-opposition

Code PR: [#162 feat(pulls): add a "Suggest change" button to the code comment form (#155)](https://github.com/CSCI-435-SE/gitea/pull/162), merged on 11-12.

## Question 1: What have you implemented so far, and what remains?

I finished #155, which is S2 of our team feature #151 (code suggestions in PR reviews). PR #162 adds a "Suggest change" button to the comment form on the new side of a diff. Clicking it inserts a `suggestion` block that holds the line being commented on. If the line already contains backticks, the block uses a longer fence so the line doesn't close it early, and the button never puts one block inside another. Unit and E2E tests cover all twelve acceptance criteria, and the PR merged today.

I still have two things to do. #155 is a small issue, so I need a second issue to reach the load of two medium issues. I plan to take one of our Deferred issues (#4, #19, #27, #36 or #37), break it into sub-issues and own one of them. I also have to run D6 mutation testing on #162's files with StrykerJS.

## Question 2: What is one thing you learned about testing in your project?

Putting the tricky logic in pure functions made it much easier to test. Most of #155 is string handling: how many backticks the fence needs, where to insert the block when the cursor is already inside one, and what to do when a selection crosses a fence. I wrote this as four functions that take text and cursor positions and return a result, with no DOM involved. The unit tests check every edge case directly, and I took the expected values from the acceptance criteria and the CommonMark spec. That left the two E2E tests with less to do. They only check that the button appears and works in split view, unified view and on lines shown by expanding the diff, and that it doesn't appear on the old side. When a unit test fails, it names the function that broke.

## Question 3: What is one risk to finishing your work by Nov 19, and what will you do about it?

My biggest risk is running out of time. My second issue hasn't been written yet. Before I can code it, it needs a spec and a spec review, and after that it needs a PR that a teammate reviews, all within one week and alongside D6. If the spec review takes a day or two, I won't start coding until around Nov 16. To avoid that, I'll pick the Deferred issue and post its sub-issue breakdown and my spec by Nov 13, and ask a teammate to review the spec the same day. I'll keep my sub-issue small enough to finish in about two days. While I wait for the review, I'll run StrykerJS on #162, so D6 isn't left for the last days of the sprint.
