# Broaden issue/PR hover preview coverage

**Status:** approved, ready for implementation plan
**Branch:** `feature/hoverissue`
**Upstream issue:** CSCI-435-SE/gitea#22 — "Show hover previews for issue, pull request, and user references"

## Problem

Links to issues and pull requests render as a bare `#123`, giving no indication of the item's
title or state. Following a lengthy discussion means opening each link in turn.

Gitea already implements the popup itself. `web_src/js/features/ref-issue.ts` attaches a tippy
popup on hover that renders `web_src/js/components/ContextPopup.vue` with data from
`GET /{owner}/{repo}/{issues|pulls}/{index}/info` (`routers/web/repo/issue.go:244`). It shows the
repository, creation date, state icon, title, a truncated body and the labels, behind a 300ms
hover delay with a per-URL cache.

The gap is coverage. The handler attaches only to links that carry the `.ref-issue` class
(cross-references rendered inside markdown) or that sit inside a `[data-ref-issue-container]`
element (`ref-issue.ts:53`). Exactly two templates in the repository set that container:
`templates/repo/activity.tmpl:8` and `templates/user/dashboard/feeds.tmpl:1`. Issue and PR list
titles, project board cards and sidebar dependency links get nothing.

## Scope

In scope: making the existing issue/PR popup appear wherever an issue or PR link appears.

Out of scope, tracked separately as follow-up work on the same upstream issue:

- **User hovercards** for `@mentions` and avatars. Mentions render as a bare
  `<a href="/username">` with no class hook (`modules/markup/html_mention.go:47`) and no user
  info endpoint exists. Needs a new endpoint and a new component.
- **Richer popup content** — the Open/Closed/Merged state pill, assignee note and comment count.
  This one deserves a visual design pass before implementation, since the popup layout is the
  deliverable.

This spec changes no popup pixels. `ContextPopup.vue` is reused as-is.

## Design

### The attachment predicate

The decision currently lives inline in the event handler. Extract it into a pure function so it
can be unit-tested without a DOM fixture per surface:

```ts
export function shouldAttachIssuePopup(link: HTMLAnchorElement): boolean
```

Returns true when `parseIssueHref` (`web_src/js/utils.ts:54`) yields an owner, repo and index,
and none of the following hold:

1. **Opt-out** — the link or an ancestor carries `data-issue-popup="off"`.
2. **External reference** — the link carries `.ref-external-issue`. These point at Jira, Redmine
   or similar, where no `/info` endpoint exists. Preserves today's behaviour.
3. **Self-reference** — the parsed owner, repo and index match the current page's. Previewing the
   page you are already on is useless; without this, the issue's own title and its `#N` index
   link both pop on their own page.
4. **Already attached** — `getAttachedTippyInstance(link)` is truthy. Preserves today's behaviour.

The handler in `initRefIssueContextPopup` calls the predicate in place of its current two-part
class-and-container check. Everything else in the handler — the 300ms timer, the `mouseleave`
cancel, the `data-ref-issue-popup` marker — is unchanged.

### Two correctness fixes required by global attachment

**Rebuild the info URL from parsed parts.** The current code requests `${link.pathname}/info`,
which is correct only because `.ref-issue` links are always canonical. Under global attachment,
`/owner/repo/pulls/1/files` (the diff tab) and `/owner/repo/issues/3/attachments` both parse as
issue references and would request a nonexistent `/files/info`. Compose the URL instead as
`{appSubUrl}/{ownerName}/{repoName}/{pathType}/{indexString}/info` from the `parseIssueHref`
result, re-adding `window.config.appSubUrl` since `parseIssueHref` strips it. `/info` is
registered under both `/issues/{index}` and `/pulls/{index}` (`routers/web/web.go:1298`, inside
`addIssuesPullsViewRoutes`), so one template serves both.

**Cache failures as well as successes.** `issueInfoCache` stores only successful responses, so a
link the viewer cannot read refetches on every hover. `GetIssueInfo` returns 404 for a deleted
issue, a repository whose Issues or Pull Requests unit is disabled, and one the viewer cannot
read (`routers/web/repo/issue.go:248-267`). These become common once every issue-shaped link is a
candidate. Store a negative entry on failure so repeat hovers are silent and make no request.

### Cross-repository links

The popup will now appear on links to issues in other repositories outside markdown. This is
correct and leaks nothing: `GetIssueInfo` resolves the target repository through
`context.RepoAssignment` and checks the viewer's read permission on the target's Issues or Pull
Requests unit before responding.

## Surfaces

| Surface | Link | Change needed |
| --- | --- | --- |
| Issue & PR lists | `templates/shared/issuelist.tmpl` lines 19, 41, 155 | None — predicate covers it |
| Project board cards | `templates/repo/issue/card.tmpl:17` | None — predicate covers it |
| Commit messages | markup-generated `.ref-issue` links | None — already works today |
| Issue sidebar dependencies | `templates/repo/issue/sidebar/issue_dependencies.tmpl` lines 25, 59 | Remove `data-tooltip-content` |

Issue and PR lists carry three links to the same issue per row — the title, the `#N` index and
the comment count. All three get a popup. This is acceptable: tippy shows one at a time and each
link has its own 300ms delay.

Commit messages already work. Issue references in commit messages pass through the markup
pipeline and emit `.ref-issue` links, which is why `ref-issue.ts:35` already clears `link.title`
to suppress the `.commit-summary` ancestor's native tooltip. No work; verify it still works.

### The tooltip collision

Sidebar dependency links carry `data-tooltip-content="#12 The title"`, which is an existing tippy
instance, so guard 4 skips them and the popup silently never appears. The popup shows the title,
state, body and labels, so it strictly supersedes that tooltip: remove the attribute from both
lines.

Generally: **any issue link that already has `data-tooltip-content` loses its popup silently.**
The two dependency lines are the only known instances on the target surfaces; an implementer
adding a surface later should check for this first rather than debug it.

## Unchanged

The 300ms hover delay, the tippy configuration (`top-start`, interactive, `role="dialog"`,
`interactiveBorder: 5`), `ContextPopup.vue` and its content, the `/info` endpoint and its
permission checks, and the Vue mount/unmount lifecycle.

## Testing

**Unit (vitest, preferred per AGENTS.md).** A new test file for `shouldAttachIssuePopup` covering:
a plain issue link attaches; a plain PR link attaches; a non-issue link does not; a
`.ref-external-issue` link does not; a link under `data-issue-popup="off"` does not; a link to the
current page does not; a link to a different index on the current page does. Plus a test that the
info URL is composed correctly for `/owner/repo/pulls/1/files` and under a non-empty `appSubUrl`.

Run with `pnpm exec vitest ref-issue`.

**E2E (playwright), one test.** On the repository issue list, hover the first title and assert the
popup appears with the issue title. One test only — the unit tests cover the decision logic, and
AGENTS.md asks for sub-2s e2e runtime.

Run with `GITEA_TEST_E2E_FLAGS=<filepath> make test-e2e`.

**Manual verification list.** Issue list title, PR list title, `#N` index link, comment count
link, project board card title, sidebar dependency link, commit message reference, markdown
cross-reference (regression), activity feed (regression), and the issue's own title on its own
page (must NOT pop).

## Risks

- **Over-firing on an unanticipated surface.** Global attachment means popups appear in places
  not enumerated here. The `data-issue-popup="off"` escape hatch handles any that turn out to be
  unwanted, without another logic change.
- **Silent tooltip collisions** on links not listed above. Mitigated by the manual verification
  list and the note in this spec.

## Files

- `web_src/js/features/ref-issue.ts` — extract predicate, rebuild info URL, cache failures
- `web_src/js/features/ref-issue.test.ts` — new
- `templates/repo/issue/sidebar/issue_dependencies.tmpl` — remove two `data-tooltip-content`
- `tests/e2e/issue-popup.test.ts` — new, one test

No Go changes. No migration. No new locale string.
