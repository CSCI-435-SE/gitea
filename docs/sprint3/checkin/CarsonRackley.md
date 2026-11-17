# Mid-Sprint Check-in: @CarsonRackley

**Code PR:** [#161 — feat(notifications): badge tab icon and opt-in desktop alerts (#118)](https://github.com/CSCI-435-SE/gitea/pull/161) (open, ready for review)

## Question 1: What have you implemented so far, and what remains?

I implemented #118, my Sprint 2 carryover, in PR #161. The tab icon gets a dot while there are unread notifications, and Actions run status still wins on a run page. Desktop notifications are opt-in per browser, show only a count, and fire once across all open tabs, only when no Gitea tab is visible. The shared worker picks which tab alerts. Each AC has a unit test, and I checked them manually in the browser.

What remains: fix the frontend CI failure, merge `main` in, and get the PR approved and merged. My second issue, #156 (apply a suggestion as a commit), needs its spec revised after review before I start coding. I also still need to do mutation testing on #118 with StrykerJS.

## Question 2: What is one thing you learned about testing in your project?

I learned that tests passing on my machine doesn't mean they are independent. All 405 frontend tests passed locally, but in CI an existing favicon test failed: it expected the default icon and got my notification dot. Gitea's Vitest config sets `isolate: false`, so test files that run in the same worker share module state. My notification tests turned the dot on and never turned it off, and whether the favicon test ran after them depended on how files were split across workers. The fix was to reset the badge state before each favicon test and after each notification test. Now, any test that changes module-level state has to clean it up, because another file may run next with that state still set.

## Question 3: What is one risk to finishing your work by Nov 19, and what will you do about it?

The biggest risk is #156. Its spec review asked for changes, two of its ACs depend on customer questions on #151 that are still unanswered, and it builds on #154, which was merged, reverted and is back as an open PR. @arjunsb26's API work (#160) also waits on my service. To manage this, I will revise the spec and ask for re-review by Nov 14, and ask the customer on #151 for answers now. I will agree on the `ApplySuggestion` signature with Arjun this week, and write the service and its tests first, since they don't need #154's rendering. If there are no customer answers by Nov 16, I will pick a default, say so on the issue, and test against it.
