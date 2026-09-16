# Issue/PR Hover Preview Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the existing issue/PR hover popup appear on every link that points at an issue or pull request, instead of only on markdown cross-references.

**Architecture:** The popup, its Vue component and its backend endpoint already exist and are unchanged. The work replaces the handler's narrow "does this link have the `.ref-issue` class" check with a pure, unit-tested predicate that attaches to any issue-shaped link unless it opts out or points at the current page. Two supporting fixes make global attachment safe: composing the info URL from parsed href parts rather than reusing `link.pathname`, and caching failed lookups so unreadable links stop refetching.

**Tech Stack:** TypeScript, Vue 3, tippy.js, vitest (happy-dom), Playwright, Go html/template.

**Spec:** `docs/superpowers/specs/2026-09-15-issue-hover-preview-coverage-design.md`

---

## Background for the implementer

You need to know four things about this codebase before starting.

**The feature already exists, narrowly.** `web_src/js/features/ref-issue.ts` registers one delegated `mouseover` listener on `document`. When the cursor lands on a matching link it waits 300ms, fetches issue JSON, mounts `ContextPopup.vue` into a detached div, and hands that div to tippy as popup content. You are changing *which links match*, plus two bugs that only appear once more links match. You are not changing the popup's appearance, its timing, or the backend.

**`parseIssueHref` is the parser you build on.** It lives at `web_src/js/utils.ts:54` and is already unit-tested at `web_src/js/utils.test.ts:36`. Given any issue or PR URL it returns `{ownerName, repoName, pathType, indexString}`, where `pathType` is the literal string `issues` or `pulls`. It handles absolute URLs, query strings, hashes, and app sub-paths. On a non-issue URL every field is `undefined`, so `if (!ownerName)` is the "this isn't an issue link" check. Note it *strips* the app sub-path, so any URL you rebuild from its output must re-add `window.config.appSubUrl`.

**Gitea treats a PR as an issue with the same index.** `/owner/repo/issues/5` and `/owner/repo/pulls/5` are the same object. This is why the self-reference check compares owner, repo and index but deliberately ignores `pathType` — otherwise hovering a `pulls` link while on the `issues` URL for the same PR would pop a preview of the page you're already reading.

**Run commands from the repository root.** Unit tests: `pnpm exec vitest <filter>`. Lint: `make lint-js`. E2E: `GITEA_TEST_E2E_FLAGS=<filepath> make test-e2e`. Per `AGENTS.md`: no trailing whitespace, Conventional Commits, an `Assisted-by:` trailer on every commit, and never a `Co-Authored-By:` or `Signed-off-by:` trailer.

**Substitute your own identity in commit trailers.** Every commit command below ends with the literal `Assisted-by: AGENT_NAME:MODEL_VERSION`. That is `AGENTS.md`'s documented format, not text to copy verbatim — replace both halves with your agent name and model version before committing.

**Keep imports in one statement per module.** Tasks 1–3 each add tests to the same file. Where a task says to append a test that imports from a module already imported, merge the new name into the existing `import` statement rather than adding a second one — `make lint-js` rejects duplicate imports from the same path.

## File structure

| File | Responsibility | Action |
| --- | --- | --- |
| `web_src/js/features/ref-issue.ts` | Decide which links get a popup; fetch and cache issue info; mount the popup | Modify |
| `web_src/js/features/ref-issue.test.ts` | Unit-test the two pure functions and the caching behaviour | Create |
| `templates/repo/issue/sidebar/issue_dependencies.tmpl` | Render sidebar dependency links | Modify (2 lines) |
| `tests/e2e/issue-popup.test.ts` | One end-to-end proof the popup appears on an issue list | Create |

`ref-issue.ts` is ~75 lines and gains ~30. That is well within the size where a single focused file is correct; do not split it.

---

### Task 1: Compose the info URL from parsed href parts

The current code requests `${link.pathname}/info`. That works only because `.ref-issue` links are always canonical issue URLs. Once every issue-shaped link is a candidate, links like `/owner/repo/pulls/1/files` (the PR diff tab) and `/owner/repo/issues/3/attachments` would request `/owner/repo/pulls/1/files/info`, which does not exist. Build the URL from the parsed parts instead.

**Files:**
- Modify: `web_src/js/features/ref-issue.ts`
- Test: `web_src/js/features/ref-issue.test.ts` (create)

- [ ] **Step 1: Write the failing test**

Create `web_src/js/features/ref-issue.test.ts` with exactly this content:

```ts
import {buildIssueInfoUrl} from './ref-issue.ts';

test('buildIssueInfoUrl', () => {
  expect(buildIssueInfoUrl('/owner/repo/issues/1')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('/owner/repo/pulls/2')).toEqual('/owner/repo/pulls/2/info');

  // non-canonical links must resolve to the canonical info endpoint, not to a sub-path
  expect(buildIssueInfoUrl('/owner/repo/pulls/1/files')).toEqual('/owner/repo/pulls/1/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/3/attachments')).toEqual('/owner/repo/issues/3/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/1?query=x')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/1#issuecomment-4')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('https://example.com/owner/repo/issues/1')).toEqual('/owner/repo/issues/1/info');

  expect(buildIssueInfoUrl('/owner/repo/issues')).toEqual(null);
  expect(buildIssueInfoUrl('/owner/repo')).toEqual(null);
  expect(buildIssueInfoUrl('')).toEqual(null);
});

test('buildIssueInfoUrl with appSubUrl', () => {
  const oldSubUrl = window.config.appSubUrl;
  window.config.appSubUrl = '/sub';
  try {
    // parseIssueHref strips the sub-path, so the rebuilt URL must re-add it
    expect(buildIssueInfoUrl('/sub/owner/repo/issues/1')).toEqual('/sub/owner/repo/issues/1/info');
  } finally {
    window.config.appSubUrl = oldSubUrl;
  }
});
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
pnpm exec vitest ref-issue
```

Expected: FAIL. The error names `buildIssueInfoUrl` as not exported from `./ref-issue.ts`.

- [ ] **Step 3: Write the implementation**

In `web_src/js/features/ref-issue.ts`, add this exported function immediately after the `issueInfoCache` declaration (after the line `const issueInfoCache = new Map<string, IssueInfo>();`):

```ts
// builds the canonical info endpoint from a link's parts, because the link may point at a
// sub-path such as /pulls/1/files where appending /info would 404
export function buildIssueInfoUrl(href: string): string | null {
  const {ownerName, repoName, pathType, indexString} = parseIssueHref(href);
  if (!ownerName) return null;
  return `${window.config.appSubUrl}/${ownerName}/${repoName}/${pathType}/${indexString}/info`;
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
pnpm exec vitest ref-issue
```

Expected: PASS, 2 tests.

- [ ] **Step 5: Commit**

```bash
git add web_src/js/features/ref-issue.ts web_src/js/features/ref-issue.test.ts
git commit -m "feat(ui): build issue info URL from parsed href parts

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 2: Extract the attachment predicate

Replace the inline class-and-container check with a pure function. Pure means testable: the caller passes the current page's path rather than the function reading `window.location`, so a test can simulate being on any page.

**Files:**
- Modify: `web_src/js/features/ref-issue.ts`
- Test: `web_src/js/features/ref-issue.test.ts`

- [ ] **Step 1: Write the failing test**

Append to `web_src/js/features/ref-issue.test.ts`, merging the import into the existing first line so it reads `import {buildIssueInfoUrl, shouldAttachIssuePopup} from './ref-issue.ts';`:

```ts
function makeLink(html: string): HTMLAnchorElement {
  const container = document.createElement('div');
  container.innerHTML = html;
  return container.querySelector('a')!;
}

test('shouldAttachIssuePopup attaches to issue-shaped links', () => {
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/1">#1</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/pulls/2">#2</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a class="ref-issue" href="/owner/repo/issues/1">#1</a>'), '/other/repo/issues/9')).toBe(true);
});

test('shouldAttachIssuePopup ignores links that are not issues', () => {
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo">repo</a>'), '/other/repo/issues/9')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues">issues</a>'), '/other/repo/issues/9')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/explore/repos">explore</a>'), '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup ignores external issue references', () => {
  // these point at Jira/Redmine, where no /info endpoint exists
  const link = makeLink('<a class="ref-external-issue" href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup honours the opt-out attribute', () => {
  const onLink = makeLink('<a data-issue-popup="off" href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(onLink, '/other/repo/issues/9')).toBe(false);

  const container = document.createElement('div');
  container.innerHTML = '<div data-issue-popup="off"><a href="/owner/repo/issues/1">#1</a></div>';
  expect(shouldAttachIssuePopup(container.querySelector('a')!, '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup ignores links to the current page', () => {
  const link = makeLink('<a href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/owner/repo/issues/1')).toBe(false);

  // a PR is the same object under both path types, so /pulls/1 on the /issues/1 page is still self
  const pullLink = makeLink('<a href="/owner/repo/pulls/1">#1</a>');
  expect(shouldAttachIssuePopup(pullLink, '/owner/repo/issues/1')).toBe(false);

  // a different index in the same repo is a real reference
  const otherLink = makeLink('<a href="/owner/repo/issues/2">#2</a>');
  expect(shouldAttachIssuePopup(otherLink, '/owner/repo/issues/1')).toBe(true);

  // the same index in a different repo is a real reference
  const otherRepoLink = makeLink('<a href="/other/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(otherRepoLink, '/owner/repo/issues/1')).toBe(true);
});
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
pnpm exec vitest ref-issue
```

Expected: FAIL. The error names `shouldAttachIssuePopup` as not exported.

- [ ] **Step 3: Write the implementation**

In `web_src/js/features/ref-issue.ts`, add this exported function directly below `buildIssueInfoUrl`:

```ts
// decides whether a link deserves a hover preview; kept pure so every surface is unit-testable
export function shouldAttachIssuePopup(link: HTMLAnchorElement, currentPath: string): boolean {
  const target = parseIssueHref(link.getAttribute('href')!);
  if (!target.ownerName) return false;
  if (link.classList.contains('ref-external-issue')) return false;
  if (link.closest('[data-issue-popup="off"]')) return false;
  if (getAttachedTippyInstance(link)) return false;

  // previewing the page you are already reading is useless; path type is ignored because a pull
  // request is reachable at both /issues/{index} and /pulls/{index}
  const current = parseIssueHref(currentPath);
  return !(current.ownerName === target.ownerName &&
    current.repoName === target.repoName &&
    current.indexString === target.indexString);
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
pnpm exec vitest ref-issue
```

Expected: PASS, 7 tests.

- [ ] **Step 5: Commit**

```bash
git add web_src/js/features/ref-issue.ts web_src/js/features/ref-issue.test.ts
git commit -m "feat(ui): extract issue popup attachment predicate

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 3: Cache failed lookups

`issueInfoCache` stores only successes. A link the viewer cannot read refetches on every single hover. `GetIssueInfo` (`routers/web/repo/issue.go:244`) returns 404 for a deleted issue, for a repository whose Issues or Pull Requests unit is disabled, and for one the viewer cannot read. Those links become common once every issue-shaped link is a candidate, so store a negative entry and fail fast on repeat hovers.

**Files:**
- Modify: `web_src/js/features/ref-issue.ts`
- Test: `web_src/js/features/ref-issue.test.ts`

- [ ] **Step 1: Write the failing test**

Append to `web_src/js/features/ref-issue.test.ts`. Two placement rules: merge the first import into the existing one so it reads `import {buildIssueInfoUrl, shouldAttachIssuePopup, getIssueInfo, issueInfoCache} from './ref-issue.ts';`, and put the `vi.mock` block directly below the imports at the top of the file rather than mid-file — vitest hoists it there regardless, and leaving it inline misleads the next reader about when it takes effect:

```ts
import {GET} from '../modules/fetch.ts';

vi.mock('../modules/fetch.ts', () => ({
  GET: vi.fn(),
}));

test('getIssueInfo caches successful responses', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockResolvedValue({
    ok: true,
    json: async () => ({convertedIssue: {number: 1}, renderedLabels: ''}),
  } as unknown as Response);

  const first = await getIssueInfo('/owner/repo/issues/1/info');
  const second = await getIssueInfo('/owner/repo/issues/1/info');
  expect(first).toBe(second);
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});

test('getIssueInfo caches failures so repeat hovers do not refetch', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockReset();
  vi.mocked(GET).mockResolvedValue({ok: false, statusText: 'Not Found'} as unknown as Response);

  await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
  await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});

test('getIssueInfo caches network errors', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockReset();
  vi.mocked(GET).mockRejectedValue(new Error('network down'));

  await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
  await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
pnpm exec vitest ref-issue
```

Expected: FAIL. `getIssueInfo` and `issueInfoCache` are not exported yet.

- [ ] **Step 3: Write the implementation**

In `web_src/js/features/ref-issue.ts`, replace the cache declaration and the whole `getIssueInfo` function with:

```ts
// null marks a lookup that failed, so unreadable or missing issues are not refetched on every hover
export const issueInfoCache = new Map<string, IssueInfo | null>();

export async function getIssueInfo(url: string): Promise<IssueInfo> {
  if (issueInfoCache.has(url)) {
    const cached = issueInfoCache.get(url);
    if (!cached) throw new Error('issue info previously failed to load');
    return cached;
  }
  let resp: Response;
  try {
    resp = await GET(url);
  } catch (err) {
    issueInfoCache.set(url, null);
    throw err;
  }
  if (!resp.ok) {
    issueInfoCache.set(url, null);
    throw new Error(resp.statusText || 'Unknown network error');
  }
  const data = await resp.json();
  issueInfoCache.set(url, data);
  return data;
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
pnpm exec vitest ref-issue
```

Expected: PASS, 10 tests.

- [ ] **Step 5: Commit**

```bash
git add web_src/js/features/ref-issue.ts web_src/js/features/ref-issue.test.ts
git commit -m "fix(ui): cache failed issue info lookups

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 4: Wire the predicate into the handler

Now make the handler use the three pieces. This is the task that actually changes user-visible behaviour.

**Files:**
- Modify: `web_src/js/features/ref-issue.ts`

- [ ] **Step 1: Replace the popup-showing function**

In `web_src/js/features/ref-issue.ts`, change the first line of `showRefIssuePopup` to use the composed URL. Replace:

```ts
  const [data, {default: ContextPopup}] = await Promise.all([
    getIssueInfo(`${link.pathname}/info`),
    import('../components/ContextPopup.vue'),
  ]);
```

with:

```ts
  const infoUrl = buildIssueInfoUrl(link.getAttribute('href')!);
  if (!infoUrl) return;
  const [data, {default: ContextPopup}] = await Promise.all([
    getIssueInfo(infoUrl),
    import('../components/ContextPopup.vue'),
  ]);
```

Leave the rest of the function — the `link.title = ''` line and the whole `createTippy` call — exactly as it is. The `link.title = ''` line suppresses a `.commit-summary` ancestor's native browser tooltip and is still needed.

- [ ] **Step 2: Replace the attachment check**

In `initRefIssueContextPopup`, replace these three lines:

```ts
    if (!parseIssueHref(link.getAttribute('href')!).ownerName) return;
    if (!link.classList.contains('ref-issue') && !link.closest('[data-ref-issue-container]')) return;
    if (getAttachedTippyInstance(link)) return;
```

with this single line:

```ts
    if (!shouldAttachIssuePopup(link, window.location.pathname)) return;
```

Leave the selector on the line above and everything below — the `data-ref-issue-popup` marker, the 300ms timer, the `mouseleave` cancel, the try/catch — unchanged.

- [ ] **Step 3: Remove the now-unused `.ref-external-issue` clause from the selector**

The predicate now handles external refs, so the selector no longer needs to. Change:

```ts
  const selector = 'a[href]:not([data-ref-issue-popup]):not(.ref-external-issue)';
```

to:

```ts
  const selector = 'a[href]:not([data-ref-issue-popup])';
```

- [ ] **Step 4: Fix the imports**

`parseIssueHref` and `getAttachedTippyInstance` are now used only inside the two new functions, which are in the same file, so both imports stay. Confirm the top of the file still reads:

```ts
import {parseIssueHref} from '../utils.ts';
import {GET} from '../modules/fetch.ts';
import {createApp} from 'vue';
import {createTippy, getAttachedTippyInstance} from '../modules/tippy.ts';
import {addDelegatedEventListener} from '../utils/dom.ts';
import type {Issue} from '../types.ts';
```

- [ ] **Step 5: Run the unit tests and the linter**

```bash
pnpm exec vitest ref-issue
```

Expected: PASS, 10 tests.

```bash
make lint-js
```

Expected: no errors. If eslint reports an unused import, remove only the import it names.

- [ ] **Step 6: Commit**

```bash
git add web_src/js/features/ref-issue.ts
git commit -m "feat(ui): show issue hover previews on all issue links

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 5: Unblock the sidebar dependency links

Sidebar dependency links carry `data-tooltip-content`, which means they already have a tippy instance attached. The predicate's `getAttachedTippyInstance` guard therefore skips them and the popup silently never appears. The popup shows the title, state, body and labels, so it strictly supersedes that tooltip — remove the attribute.

**Files:**
- Modify: `templates/repo/issue/sidebar/issue_dependencies.tmpl` (lines 25 and 59)

- [ ] **Step 1: Edit both dependency title links**

Both lines currently read (line 25 inside the `BlockingDependencies` range, line 59 inside `BlockedByDependencies` — the two lines are byte-identical):

```html
<a class="muted issue-dependency-title gt-ellipsis" href="{{.Issue.Link}}" data-tooltip-content="#{{.Issue.Index}} {{.Issue.Title | ctx.RenderUtils.RenderEmoji}}">
```

Change both to:

```html
<a class="muted issue-dependency-title gt-ellipsis" href="{{.Issue.Link}}">
```

Do not touch the `data-tooltip-content` on the sibling `<div class="tw-text-xs gt-ellipsis">` below each link — that one shows the repository name, is not an issue link, and keeps its tooltip.

- [ ] **Step 2: Verify exactly two lines changed**

```bash
git diff --stat templates/repo/issue/sidebar/issue_dependencies.tmpl
```

Expected: `2 insertions(+), 2 deletions(-)`.

- [ ] **Step 3: Check for trailing whitespace**

```bash
git diff --check
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add templates/repo/issue/sidebar/issue_dependencies.tmpl
git commit -m "fix(ui): let hover previews attach to dependency links

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 6: End-to-end proof on the issue list

One e2e test, not one per surface — the unit tests already cover the decision logic, and `AGENTS.md` asks for sub-2s e2e runtime. This test proves the whole chain works in a real browser: delegated listener, predicate, fetch, Vue mount, tippy render.

**Files:**
- Create: `tests/e2e/issue-popup.test.ts`

- [ ] **Step 1: Write the test**

Create `tests/e2e/issue-popup.test.ts` with exactly this content:

```ts
import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString} from './utils.ts';

test('hover preview on the issue list', async ({page, request}) => {
  const repoName = `e2e-issue-popup-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  const title = `Hover preview target ${randomString(8)}`;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title, body: 'Body of the hovered issue.'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues`);

  await page.getByRole('link', {name: title}).hover();

  // the popup is a tippy dialog mounted outside the list; it renders the issue body
  const popup = page.locator('[data-tippy-root] [role="dialog"], [role="dialog"]').first();
  await expect(popup).toBeVisible();
  await expect(popup).toContainText('Body of the hovered issue.');
});
```

- [ ] **Step 2: Run the test**

```bash
GITEA_TEST_E2E_FLAGS=tests/e2e/issue-popup.test.ts make test-e2e
```

Expected: PASS, 1 test.

If it fails on the popup locator, open `web_src/js/modules/tippy.ts` and check what `createTippy` renders for `role: 'dialog'`, then narrow the locator to match. Do not lengthen the timeout to make it pass — the popup has a fixed 300ms delay and Playwright's auto-waiting covers that.

- [ ] **Step 3: Commit**

```bash
git add tests/e2e/issue-popup.test.ts
git commit -m "test(e2e): cover issue hover preview on the issue list

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

### Task 7: Full verification

Automated tests do not prove coverage across surfaces, because the whole point of the change is that popups now appear in places no test enumerates. Verify by hand.

**Files:** none

- [ ] **Step 1: Run the full JS test suite and linter**

```bash
pnpm exec vitest
```

Expected: PASS, no regressions. Pay attention to `utils.test.ts`, which covers `parseIssueHref`.

```bash
make lint-js
```

Expected: no errors.

- [ ] **Step 2: Start a local instance and walk the surfaces**

Popups must appear on all of these:

- [ ] Issue list: an issue title
- [ ] PR list: a pull request title
- [ ] Issue list: the `#N` index link on a row
- [ ] Issue list: the comment-count link on a row
- [ ] Project board: an issue card title
- [ ] Issue sidebar: a dependency link (requires an issue with a dependency)
- [ ] Commit list: an issue reference inside a commit message
- [ ] An issue comment body: a markdown `#N` cross-reference (regression — worked before)
- [ ] Repository activity tab and dashboard feed (regression — worked before)

Popups must NOT appear on:

- [ ] The issue's own title or `#N` index while viewing that issue's page
- [ ] A link to an issue in a repository you cannot read (it should fail silently, with no popup and no repeated network requests — confirm in the Network tab that a second hover fires no request)

- [ ] **Step 3: Update the context docs if anything is now wrong**

`AGENTS.md` requires fixing any `.claude-students` doc this change makes inaccurate. Check `.claude-students/docs/frontend-js.md` for statements about hover previews or `ref-issue.ts`:

```bash
grep -n "ref-issue\|popup\|hover" .claude-students/docs/frontend-js.md
```

If it describes the old `.ref-issue`-only behaviour, correct it, and regenerate its twin at `.claude-students/guide/frontend-js.md` per `.claude-students/MAINTENANCE.md`. If grep returns nothing, no doc change is needed.

- [ ] **Step 4: Run the context doc check**

```bash
./.claude-students/check.sh
```

Expected: PASS.

- [ ] **Step 5: Commit any doc changes**

Skip if Step 3 found nothing.

```bash
git add .claude-students/
git commit -m "docs: update frontend-js context for popup coverage

Assisted-by: AGENT_NAME:MODEL_VERSION"
```

---

## Out of scope

Do not implement these; they are separate work on the same upstream issue:

- **User hovercards** for `@mentions` and avatars. Needs a new backend endpoint and a new Vue component.
- **Richer popup content** — the Open/Closed/Merged state pill, assignee note, comment count. Needs a visual design pass first.

`ContextPopup.vue` must not be modified by this plan. If you find yourself editing it, stop — you are out of scope.
