# AI Logs

This folder holds every AI session for each sprint, one folder per student. The course rules are on
the [AI Log Instructions](https://csci-435-se.github.io/ai-logs/) page and the current sprint page
at <https://csci-435-se.github.io/>. This README is the team's shared convention for following them.

## What to log

Every AI session tied to this project: code, debugging, tests, reviews, drafting issues and PRs,
design discussion, the sprint report. That includes sessions that went nowhere. Purely academic
questions unrelated to the project don't need logging.

## Structure

```
ai-logs/
  sprint1/
    <github-username>/                          ← exact GitHub username, matching case
      .gitkeep                                  ← the folder must exist even if you used no AI
      YYYY-MM-DD_<tool>_<short-slug>.md
      YYYY-MM-DD_<tool>_<short-slug>/           ← attachments (images, etc.), optional
```

Team folders: `ab-wm`, `arjunsb26`, `CarsonRackley`, `controlled-opposition`, `Jack-Donohue`,
`vvchopra`.

## Naming

`YYYY-MM-DD_<tool>_<short-slug>.md`

- **Date:** the day the session started.
- **Sprint folder:** the sprint that date falls in, not the sprint the work is for.
- **Tool:** one of the course identifiers: `claude-code`, `claude-web`, `claude-desktop`, `cursor`,
  `gemini-cli`, `codex-cli`, `copilot`, `chatgpt`, `other-<name>`. Claude Code run inside VS Code is
  `claude-code`.
- **Slug:** lowercase words joined with hyphens that say what the session did. Put the issue number
  first when there is one: `issue-52-impl`, `issue-52-spec`, `pr-61-review`, `sprint1-report`.
  Replace auto-generated slugs such as `local-command-caveat...`.
- **Formats:** `.md` is preferred; `.txt` and `.json` are also accepted. No `.docx`, `.pdf`, `.html`
  or share links.

## Capturing a session

| Tool | How |
| --- | --- |
| Claude Code (terminal or VS Code), other agentic CLIs | Start it with [SpecStory](https://specstory.com/): `specstory run claude`, from the repo root. Forgot? Run `specstory sync` afterwards; it back-fills from Claude Code's local history. Then copy from `.specstory/history/` (git-ignored) into your folder and rename it to the convention. |
| Web chat (claude.ai, ChatGPT, Gemini, ...) | Install a browser chat exporter and export the session as Markdown right after it ends. Include any files you pasted in as context. |
| Claude Desktop | SpecStory doesn't capture it. Copy the conversation into a `.md` file by hand. |

At the end of each sprint, run `specstory sync` once more and re-copy. A log copied mid-session is
incomplete.

## Content

- The **tool and AI model** used (SpecStory writes both).
- The **complete** prompt and response history. For agentic tools, that includes file edits and
  commands (SpecStory includes them). Logs with missing responses or truncated histories are
  **invalid**. Don't hand-write, summarise or trim a log; only rename it.
- Attachments (screenshots, snippets) go in a folder with the log's name minus `.md`.
- Log **every** session, including ones that went nowhere. Grading is on how complete the record
  is, not on whether AI helped.

## Where logs are referenced

- **Issue comments only.** Each contributor posts their own `### AI Assistance — @username` comment
  on the issue, with full `https://github.com/...` links to their logs. Format:
  [`.claude-students/AI-ASSISTANCE.md`](../.claude-students/AI-ASSISTANCE.md).
- **Not** in PR descriptions or PR comments. The PR has a short AI-use note with no log link. The
  AI Log Instructions page says "Issue/PR comments", but the Sprint 1 page narrows this to issue
  comments only, and the sprint page wins.
- Each teammate posts their own comment, including reviewers who used AI to review.
- No AI used? Post an issue comment saying so, and keep your folder with its `.gitkeep`.
- The sprint report lists each member's tools, session count, folder link, and 2–3 sentences on
  lessons learned.
