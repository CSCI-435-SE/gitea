# CodeRabbit — an AI reviewer for your pull requests

**GiCoffee · Gitea · CSCI 435 New-Tech Showcase 1**

- **Tutorial video:** [watch on YouTube](https://youtu.be/REPLACE_ME)
- **Tool's website:** <https://www.coderabbit.ai> · **Docs:** <https://docs.coderabbit.ai>
- **Version:** hosted GitHub App (no version number; used October 2026)

## The problem

With AI agents now writing and submitting code changes automatically, it has become harder to review changes in a timely manner, whether because human reviewers are bogged down by the number of PRs or because poorly prompted agents produce large commits.

CodeRabbit helps automate the review process: it digs into the changes and leaves comprehensive reviews on PRs. It doesn't take away control, though. It handles the nitty-gritty parts of the review so reviewers can focus on the main parts of the PR.

## Setup (about 10 minutes)

1. Use a public repo to get the free version, or sign up for a plan on their website (details below).
2. Sign in at <https://app.coderabbit.ai> with GitHub and install the CodeRabbit GitHub App on the repo. In an organization, an owner has to approve it (for us, the instructor approved the install).
3. Optionally, add a `.coderabbit.yaml` file to the repo root with rules CodeRabbit should follow when reviewing a PR.
4. Open a pull request.
5. **First result:** once the pull request is opened, CodeRabbit starts reviewing it automatically. Draft PRs are skipped by default, so comment `@coderabbitai review` to start one manually. A summary and inline comments arrive within a few minutes.

## Handy PR commands

| Comment on the PR | What happens |
|---|---|
| `@coderabbitai review` | Reviews new commits |
| `@coderabbitai full review` | Re-reviews the whole PR |
| `@coderabbitai rate limit` | Reviews left + refill time |
| `@coderabbitai help` | Lists every command |
| Reply in a comment thread | Ask any question |

## When to use it, and when not

- **Good fit:** public/open-source repos, student teams where PRs wait on reviewers, and keeping code quality up.
- **Free-plan limits:** new organizations start with a 14-day trial of the Advanced plan, which is what our demo runs on (10 reviews per hour). After that, public repos stay free but get fewer reviews per hour, and repos with fewer than 10 stars need reviews triggered manually.
- **It's AI:** some comments are wrong or nitpicky. Treat them as suggestions and keep human approval.
- **Private repos:** paid plans start at $24/developer/month (billed annually), and they have rate limits too.
- **Sensitive code:** your diff is sent to a third-party service. Don't use it on code you can't share.