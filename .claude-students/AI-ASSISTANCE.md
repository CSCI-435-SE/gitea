# AI assistance disclosure — required format

Course rule, not an upstream Gitea one. Every contributor discloses their own AI use per issue,
**even when no AI was used**. Source: the course AI Log Instructions page and `CONTRIBUTING.md`
(AI Contribution Policy). Two separate places, both required:

## 1. A comment on the issue

Posted by each contributor on the **issue** they worked, once their work is done. One comment per
person — never edit someone else's. Add later sessions by editing your own table.

```markdown
### AI Assistance — @your-github-username

**Role in this issue:** Implementation

| # | Tool | Log | What AI helped with |
|---|---|---|---|
| 1 | claude-code | [2026-09-14_claude-code_issue-1.md](ai-logs/sprint0/YourName/2026-09-14_claude-code_issue-1.md) | Confirmed the files found during concept location, drafted the template change and tests, ran them |

**Attachments:** [2026-09-14_claude-code_issue-1/](ai-logs/sprint0/YourName/2026-09-14_claude-code_issue-1/)
```

- **Role** is one of Implementation, Code review, Testing, Other — pick the ones that apply and
  delete the rest, do not leave the slash list in place.
- **Tool** is the tool's name, e.g. `claude-code`, `claude-web`.
- **Log** links the filed log under `ai-logs/sprint<N>/<github-username>/`, named
  `YYYY-MM-DD_<tool>_<short-slug>.md`. The folder is the **GitHub username**, matching case.
- **Attachments** only when the log has a companion folder; otherwise drop the line.
- **No AI used?** Keep the heading and role line and write `No AI used` in place of the table.

## 2. A short section in the pull request description

Disclosure belongs in the PR too, because reviewers read the PR, not the issue:

```markdown
## AI assistance

Claude Code (claude-opus-5) drafted the handler and tests following the existing register/delete
code; I reviewed every line, tested manually, and can explain each change.
AI log: [2026-09-14_claude-code_issue-1.md](https://github.com/CSCI-435-SE/gitea/blob/main/ai-logs/sprint0/YourName/2026-09-14_claude-code_issue-1.md)
```

Name the model, say what AI did and what you did, and link the log.

## Rules that make a disclosure wrong

- **Placeholders left in**: `@your-github-username` or the full `Implementation / Code review /
  Testing / Other` list means the comment was pasted unread.
- **A log link that 404s.** Until the log is merged to `main`, link the branch that holds it; switch
  to the `main` link once merged.
- **A log folder that is not your GitHub username** — `ai-logs/sprint0/jdonohue/` is wrong when the
  account is `Jack-Donohue`.
- **Claiming work AI did as your own, or the reverse.** The claim must match the log.
- **AI-written replies to reviewer questions on your own issue or PR.** `CONTRIBUTING.md` forbids it:
  the questions are for you.

## For AI agents drafting an issue or PR

Include the section above in every issue and PR description you draft, filled in from the session
you actually ran — never with placeholders. You may draft the wording, but say plainly that the
human must review and edit it before posting, and leave the issue comment for them to post under
their own account.
