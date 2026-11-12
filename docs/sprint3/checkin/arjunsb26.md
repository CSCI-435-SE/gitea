# Mid-Sprint Check-in: @arjunsb26

**Code PR:** [#164 — test(api): prove suggestions posted through the reviews API render as suggestions](https://github.com/CSCI-435-SE/gitea/pull/164) (open, ready for review)

## Question 1: What have you implemented so far, and what remains?

I took S7 (#160), which lets bots create and apply code suggestions through `/api/v1`. My PR covers AC1: a lint bot that posts a review through the existing reviews API gets suggestions with no API change. The integration test posts a review with a `suggestion` block on each side of the diff. It then checks that the Files and Conversation tabs mark only the new-side comment as a suggestion, using the line the reviewer saw. The PR is stacked on S1 (#163).

What remains is the apply endpoint (AC2–AC5), which waits on S3's service (#156), plus a second medium issue, mutation testing, and a review.

## Question 2: What is one thing you learned about testing in your project?

I learned that a passing test only proves something once you have seen it fail. My test passed on the first run. To check it, I swapped S1's two templates back to the `main` versions and ran it again. Both the Files tab and the Conversation tab subtests failed, which shows they really depend on the feature. I also took every expected value from the input the test writes itself (the README line and the suggestion text), not from what the code printed. A test that copies the code's own output would pass even if the code were wrong. Finally, Gitea's integration tests need a built `gitea` binary, because the git hooks call it.

## Question 3: What is one risk to finishing your work by Nov 19, and what will you do about it?

The biggest risk is the dependency chain. The rest of #160 calls S3's apply service, and S3 (#156) has no PR yet. My test PR also sits on top of S1's open PR, so it can't merge first. To manage this, I will agree on S3's function signature with @CarsonRackley this week, so I can write the API endpoint and its tests against it in parallel. I will also review S1 (#163) within 24 hours to help it merge. If S3 is still not open by Nov 16, I will move to S5's range anchoring (#158), which needs only a migration, so I still have a second medium issue to deliver.
