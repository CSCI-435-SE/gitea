# Mid-Sprint Check-in: @arjunsb26

## Question 1: What did you implement so far in the sprint, and what remains?

My main work so far is #14, the milestone burndown chart, which was carried over from Sprint 1. PR #77 merged on 10-09. The milestone page now draws a burndown with an ideal line and a projected finish date. It marks where scope was added or removed, and hovering over a day lists the items that changed on it. A checkbox, saved in the URL, adds pull requests to the chart. The history is rebuilt from timeline comments, so it needed no migration. The PR has Go unit, integration, Vitest and e2e tests. After review I fixed stale responses when the toggle is clicked quickly, and a projection that was measured from the wrong day.

What remains is all of my Sprint 2 load. I have no issues on the Sprint 2 milestone yet. I need to pick two medium issues, write their specs with a design decision, get them peer-reviewed, and then implement them with Tech Debt sections. I also need to draw the UML diagrams for one of them, file my AI logs, and review a teammate's PR.

## Question 2: What is one thing you learned about the codebase or about software engineering practice?

I learned that a permission check on a route doesn't always mean the data behind it is safe to show. The milestone pages let in anyone who can read issues *or* pull requests, so adding my chart endpoint to the same route group was not enough. A viewer who could read issues but not pull requests would still have seen pull request titles in the chart. The server now counts and lists a kind only if the viewer can read it, and two integration tests turn off each unit and check that nothing leaks. I confirmed that both tests fail when the check is removed, because a test that never fails proves nothing.

I also learned that Gitea's timeline comments work as an event log. Replaying them lets you rebuild the past without storing daily snapshots, but the replay has to end on the database's current state. Otherwise the chart can disagree with the progress bar right above it.

## Question 3: What is one thing you would do differently?

I would check my work against the acceptance criteria before calling it done, not after. When I asked the AI whether #14 was complete, it found that the first version had missed three criteria: the scope markers, the hover details and the pull request toggle. Adding them late meant a second round of commits, more tests and another merge from main. If I had kept the criteria list open as a checklist while I worked, I would have caught those gaps on the first day.

I would also start my Sprint 2 issues sooner. Finishing #14 took the first part of the sprint, and I put off choosing new issues until it merged. That leaves the spec reviews, which depend on teammates' turnaround, crowded into the second week. Writing the specs while #14 was in review would have kept both moving at once.