# AI Logs

This folder holds every AI session for each sprint, one folder per student. The course rules are on
the [AI Log Instructions](https://csci-435-se.github.io/ai-logs/) page and the current sprint page
at <https://csci-435-se.github.io/>. This README is the team's shared convention for following them.

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
- **Tool:** one of `claude-code`, `claude-web`, `claude-desktop`, `claude-vscode`, `cursor`,
  `gemini-cli`, `codex-cli`, `copilot`, `chatgpt`, `gemini-web`, `other-<name>`.
- **Slug:** lowercase words joined with hyphens that say what the session did. Put the issue number
  first when there is one: `issue-52-impl`, `issue-52-spec`, `pr-61-review`, `sprint1-report`.
  Replace auto-generated slugs such as `local-command-caveat...`.
- **Formats:** `.md` is preferred; `.txt` and `.json` are also accepted. No `.docx`, `.pdf`, `.html`
  or share links.

## Content

- The **full session**: timestamps, every prompt and every AI response. Don't hand-write or trim
  it. Capture it with [SpecStory](https://specstory.com/) (`specstory run claude`), or run
  `specstory sync` afterwards to back-fill from Claude Code's local history.
- Log **every** session, including ones that went nowhere. Grading is on how complete the record
  is, not on whether AI helped.
- Re-sync and re-copy a log once the session is finished. A copy taken mid-session is incomplete.

## Where logs are referenced

- **Issue comments only.** Each contributor posts their own `### AI Assistance — @username` comment
  on the issue, with full `https://github.com/...` links to their logs. Format:
  [`.claude-students/AI-ASSISTANCE.md`](../.claude-students/AI-ASSISTANCE.md).
- **Not** in PR descriptions or PR comments. The PR has a short AI-use note with no log link.
- No AI used? Post an issue comment saying so, and keep your folder with its `.gitkeep`.
- The sprint report lists each member's tools, session count, folder link, and 2–3 sentences on
  lessons learned.
