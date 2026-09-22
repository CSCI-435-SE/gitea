# AI assistance disclosure — required format

Course rule, not an upstream Gitea one. Every contributor discloses their own AI use per issue,
**even when no AI was used**. Source: the course AI Log Instructions page and `CONTRIBUTING.md`
(AI Contribution Policy). Two separate places, both required. The rest of the PR and issue
format is in `COURSE-WORKFLOW.md`.

## 1. A comment on the issue

Posted by each contributor on the **issue** they worked, once their work is done. One comment per
person — never edit someone else's. Add later sessions by editing your own table.

```markdown
### AI Assistance — @your-github-username

**Role in this issue:** Implementation

| # | Tool | Log | What AI helped with |
|---|---|---|---|
| 1 | claude-code | [2026-09-24_claude-code_issue-52-impl.md](https://github.com/CSCI-435-SE/gitea/blob/main/ai-logs/sprint1/YourName/2026-09-24_claude-code_issue-52-impl.md) | Confirmed the files found during concept location, drafted the template change and tests, ran them |

**Attachments:** [2026-09-24_claude-code_issue-52-impl/](https://github.com/CSCI-435-SE/gitea/tree/main/ai-logs/sprint1/YourName/2026-09-24_claude-code_issue-52-impl)
```

- **Role** is one of Implementation, Code review, Testing, Other — pick the ones that apply and
  delete the rest, do not leave the slash list in place.
- **Tool** is the tool's name, e.g. `claude-code`, `claude-web`.
- **Log** is a full `https://github.com/...` link to the filed log. Relative paths break in issue
  comments. File naming is in `ai-logs/README.md`.
- **Attachments** only when the log has a companion folder; otherwise drop the line.
- **No AI used?** Keep the heading and role line and write `No AI used` in place of the table.

## 2. A short section in the pull request description — no log link

Reviewers read the PR, and the review rubric scores "AI transparency", so the PR says how AI was
used and how you verified it. It does **not** link or cite the AI log: from Sprint 1 on, logs are
referenced in **issue comments only**, never in PR descriptions or PR comments (Sprint 1 page, D5).

```markdown
## AI assistance

Claude Code (claude-opus-5) drafted the handler and tests following the existing register/delete
code; I reviewed every line, ran the tests, checked the page by hand, and can explain each change.
AI log: see the AI Assistance comment on the linked issue.
```

Name the model, say what AI did and what you did, and point at the issue comment for the log.

## Rules that make a disclosure wrong

- **Placeholders left in**: `@your-github-username` or the full `Implementation / Code review /
  Testing / Other` list means the comment was pasted unread.
- **A log link in a PR description or PR comment.** Log links live in the issue comment only.
- **A log link that 404s.** Until the log is merged to `main`, link the branch that holds it; switch
  to the `main` link once merged.
- **A log folder that is not your GitHub username** — `ai-logs/sprint1/jdonohue/` is wrong when the
  account is `Jack-Donohue`.
- **Claiming work AI did as your own, or the reverse.** The claim must match the log.
- **AI-written replies to reviewer questions on your own issue or PR.** `CONTRIBUTING.md` forbids it:
  the questions are for you.

## For AI agents drafting an issue or PR

Put section 2 in every PR description you draft, and draft the section 1 issue comment
separately, both filled in from the session you actually ran — never with placeholders. You may
draft the wording, but say plainly that the human must review and edit it before posting, and leave
the issue comment for them to post under their own account.
