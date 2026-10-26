import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Page} from '@playwright/test';
import {apiCreateBranch, apiCreateFile, apiCreateIssue, apiCreatePR, apiCreateRepo, apiEditRepo, apiHeaders, assertNoJsError, baseUrl, login, randomString} from './utils.ts';

async function createRepo(page: Page) {
  const repoName = `e2e-shortcut-${randomString(8)}`;
  await apiCreateRepo(page.request, {name: repoName});
  await apiCreateFile(page.request, env.GITEA_TEST_E2E_USER, repoName, 'other.txt', 'other file');
  return `/${env.GITEA_TEST_E2E_USER}/${repoName}`;
}

// presses a shortcut that leaves the page, and waits until the next page can take one
async function goWith(page: Page, keys: string, url: RegExp) {
  for (const key of keys.split(' ')) await page.keyboard.press(key);
  await expect(page).toHaveURL(url);
  await page.waitForLoadState();
}

test('s and t focus their boxes, and Escape clears them', async ({page}) => {
  await page.goto(await createRepo(page));
  const search = page.locator('.repo-home-sidebar-top input[name="q"]');
  const searchHint = page.locator('.repo-home-sidebar-top kbd');
  const goToFile = page.locator('.repo-file-search-container input');
  const goToFileHint = page.locator('.repo-file-search-container kbd');
  await expect(goToFileHint).toHaveAttribute('aria-keyshortcuts', 't'); // the Vue box has mounted and registered
  await expect(searchHint).toHaveAttribute('data-shortcut-section', 'Repository');
  await expect(goToFileHint).toHaveAttribute('data-shortcut-section', 'Repository');

  await page.keyboard.press('s');
  await expect(search).toBeFocused();
  await expect(searchHint).toBeHidden();
  await page.keyboard.type('st');
  await expect(search).toHaveValue('st');
  await page.keyboard.press('Escape');
  await expect(search).toHaveValue('');
  await expect(search).not.toBeFocused();
  await expect(searchHint).toBeVisible();

  await page.keyboard.press('t');
  await expect(goToFile).toBeFocused();
  await expect(goToFileHint).toBeHidden();
  await assertNoJsError(page);
});

test('a key the language menu uses for type-ahead runs no shortcut', async ({page}) => {
  await page.goto(await createRepo(page));
  await expect(page.locator('.repo-home-sidebar-top kbd')).toHaveAttribute('aria-keyshortcuts', 's'); // s is a live shortcut here
  await page.locator('aria-menu:has(.language-menu) > button').focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('s'); // jumps to a language starting with "s", such as Suomi
  await expect(page.locator('.language-menu .item:focus')).toHaveText(/^s/i);
  await expect(page.locator('.repo-home-sidebar-top input[name="q"]')).not.toBeFocused();
  await assertNoJsError(page);
});

test('t keeps working after moving through the file tree', async ({page}) => {
  const repoLink = await createRepo(page);
  await page.goto(`${repoLink}/src/branch/main/README.md`); // the file tree is shown by default
  const goToFile = page.locator('.repo-file-search-container input');
  const goToFileHint = page.locator('.repo-file-search-container kbd');

  for (const [file, content] of [['other.txt', 'other file'], ['README.md', 'e2e-shortcut-']]) {
    await page.locator(`#view-file-tree .tree-item[title="${file}"]`).click();
    await expect(page.locator('.file-view')).toContainText(content); // the page content was swapped
    await expect(goToFileHint).toHaveAttribute('aria-keyshortcuts', 't'); // the new hint has registered
    await page.keyboard.press('t');
    await expect(goToFile).toBeFocused();
    await page.keyboard.press('Escape');
  }
  await assertNoJsError(page);
});

test('g shortcuts go to the repository tabs, and g c to the default branch', async ({page}) => {
  const repoLink = await createRepo(page);
  const [, owner, repoName] = repoLink.split('/');
  await apiCreateBranch(page.request, owner, repoName, 'feature');
  await page.goto(`${repoLink}/src/branch/feature`);
  const issuesTab = page.locator('[data-shortcut-keys="g i"]');
  await expect(issuesTab).toHaveAttribute('data-shortcut-description', 'Issues'); // the tab's text also holds its open count
  await expect(issuesTab).toHaveAttribute('data-shortcut-section', 'Repository');
  await expect(page.locator('[data-shortcut-keys="g c"]')).toHaveAttribute('href', `${repoLink}/src/branch/feature`);

  await goWith(page, 'g c', new RegExp(`${repoLink}$`));
  await goWith(page, 'g i', new RegExp(`${repoLink}/issues$`));
  await goWith(page, 'g p', new RegExp(`${repoLink}/pulls$`));
  await goWith(page, 'g a', new RegExp(`${repoLink}/actions$`));
  await goWith(page, 'g w', new RegExp(`${repoLink}/wiki$`));
  await assertNoJsError(page);
});

test('g shortcuts are missing for tabs that are not shown or that leave Gitea', async ({page}) => {
  const repoLink = await createRepo(page);
  const [, owner, repoName] = repoLink.split('/');
  await apiEditRepo(page.request, owner, repoName, {
    has_pull_requests: false,
    has_actions: false,
    has_issues: true,
    external_tracker: {external_tracker_url: 'https://tracker.example.com/'},
    has_wiki: true,
    external_wiki: {external_wiki_url: 'https://wiki.example.com/'},
  });
  await page.goto(repoLink);
  await expect(page.locator('.overflow-menu-items a[href="https://tracker.example.com/"]')).toBeVisible();
  await expect(page.locator('.overflow-menu-items a[href="https://wiki.example.com/"]')).toBeVisible();
  await expect(page.locator('[data-shortcut-keys^="g "]')).toHaveCount(1); // only g c

  for (const keys of ['g i', 'g p', 'g a', 'g w']) {
    for (const key of keys.split(' ')) await page.keyboard.press(key);
  }
  await page.keyboard.press('s'); // a live shortcut still runs after them
  await expect(page.locator('.repo-home-sidebar-top input[name="q"]')).toBeFocused();
  await assertNoJsError(page);
});

test('g shortcuts work for tabs moved into the "more" menu', async ({page}) => {
  await page.setViewportSize({width: 420, height: 800});
  const repoLink = await createRepo(page);
  await page.goto(repoLink);
  await expect(page.locator('.overflow-menu-popup [data-shortcut-keys="g w"]')).toBeAttached(); // the Wiki tab no longer fits
  await goWith(page, 'g w', new RegExp(`${repoLink}/wiki$`));
  await assertNoJsError(page);
});

test('g n goes to notifications when signed in, outside any repository', async ({page}) => {
  await page.goto('/explore/repos');
  await expect(page.locator('[data-shortcut-keys="g n"]')).toHaveCount(0); // signed out
  await login(page);
  await page.goto('/explore/repos');
  const bell = page.locator('[data-shortcut-keys="g n"]');
  await expect(bell).toHaveCount(1); // the bell is in the page twice, only one copy declares it
  await expect(bell).toHaveAttribute('data-shortcut-description', 'Notifications');
  await expect(bell).toHaveAttribute('data-shortcut-section', 'General');
  await expect(page.locator('[data-shortcut-keys^="g "]')).toHaveCount(1); // no repository shortcuts outside a repository
  await goWith(page, 'g n', /\/notifications$/);
  await assertNoJsError(page);
});

test('c starts a new issue or pull request from the lists', async ({page}) => {
  await login(page); // the New Pull Request button is greyed out for visitors
  const repoLink = await createRepo(page);
  await page.goto(`${repoLink}/issues`);
  const newIssue = page.locator('[data-shortcut-keys="c"]');
  await expect(newIssue).toHaveAttribute('data-shortcut-description', 'New Issue');
  await expect(newIssue).toHaveAttribute('data-shortcut-section', 'Issues');
  await goWith(page, 'g c', new RegExp(`${repoLink}$`)); // c as the second key of g c runs only g c

  await page.goto(`${repoLink}/issues`);
  await goWith(page, 'c', new RegExp(`${repoLink}/issues/new$`));

  await page.goto(`${repoLink}/pulls`);
  const newPull = page.locator('[data-shortcut-keys="c"]');
  await expect(newPull).toHaveAttribute('data-shortcut-description', 'New Pull Request');
  await expect(newPull).toHaveAttribute('data-shortcut-section', 'Issues');
  const compareLink = (await newPull.getAttribute('href'))!;
  await goWith(page, 'c', new RegExp(`${compareLink.replaceAll('.', '\\.')}$`));

  const [, owner, repoName] = repoLink.split('/');
  await apiCreateFile(page.request, owner, repoName, '.gitea/ISSUE_TEMPLATE/bug.md', '---\nname: Bug\nabout: Something broke\n---\n');
  await page.goto(`${repoLink}/issues`);
  await goWith(page, 'c', new RegExp(`${repoLink}/issues/new/choose$`)); // with issue templates, the button leads to the chooser
  await assertNoJsError(page);
});

test('c typed into a list filter picks its item instead of starting a new issue', async ({page}) => {
  await login(page); // the Type filter is only shown when signed in
  const repoLink = await createRepo(page);
  let newIssueRequests = 0;
  await page.route('**/issues/new', (route) => {
    newIssueRequests++;
    return route.abort();
  });
  await page.goto(`${repoLink}/issues`);
  const typeFilter = page.locator('#issue-filters .ui.dropdown', {hasText: 'Type'});
  await typeFilter.focus();
  await page.keyboard.press('c');
  await expect(typeFilter.locator('.item.selected')).toHaveText('Created by you'); // the filter used the key
  expect(newIssueRequests).toBe(0);
  await assertNoJsError(page);

  await typeFilter.blur();
  await page.keyboard.press('c'); // outside the filter, c still starts a new issue
  await expect.poll(() => newIssueRequests).toBe(1);
});

test('a list filter with its own search box still takes typing', async ({page}) => {
  await login(page);
  const repoLink = await createRepo(page);
  const [, owner, repoName] = repoLink.split('/');
  await page.request.post(`${baseUrl()}/api/v1/repos/${owner}/${repoName}/labels`, {
    headers: apiHeaders(),
    data: {name: 'cobalt', color: '#0000ff'}, // the Label filter is disabled without labels
  });
  await page.goto(`${repoLink}/issues`);
  const labelFilter = page.locator('#issue-filters .label-filter');
  await labelFilter.click();
  const search = labelFilter.locator('.menu input[type="text"]');
  await search.click();
  await page.keyboard.type('cob'); // its box sits inside the menu, so these letters are not the dropdown's own
  await expect(search).toHaveValue('cob');
  await assertNoJsError(page);
});

test('c is missing where there is nothing to create', async ({page}) => {
  const repoLink = await createRepo(page);
  await page.goto(`${repoLink}/pulls`);
  await expect(page.locator('.new-pr-button.disabled')).toBeVisible(); // signed out, so greyed out
  await expect(page.locator('[data-shortcut-keys="c"]')).toHaveCount(0);

  const [, owner, repoName] = repoLink.split('/');
  await apiEditRepo(page.request, owner, repoName, {archived: true});
  await page.goto(`${repoLink}/issues`);
  await expect(page.locator('[data-shortcut-keys="c"]')).toHaveCount(0);
  await page.goto(`${repoLink}/pulls`);
  await expect(page.getByRole('link', {name: 'Compare commits'})).toBeVisible(); // in the create button's place
  await expect(page.locator('[data-shortcut-keys="c"]')).toHaveCount(0);
  await assertNoJsError(page);
});

// a pull request with one commit; returns its "Files changed" tab
async function createPullRequest(page: Page) {
  const repoLink = await createRepo(page);
  const [, owner, repoName] = repoLink.split('/');
  await apiCreateFile(page.request, owner, repoName, 'feature.txt', 'feature', {newBranch: 'feature'});
  const index = await apiCreatePR(page.request, owner, repoName, 'feature', 'main', 'picker');
  return `${repoLink}/pulls/${index}/files`;
}

test('c opens the commit picker on the files changed tab', async ({page}) => {
  await page.goto(await createPullRequest(page));
  await expect(page.locator('#diff-commit-select')).toHaveAttribute('data-shortcut-section', 'Pull Requests');
  const button = page.getByRole('button', {name: 'Filter by commit'});
  const menu = page.locator('.diff-commit-selector .menu');
  const showAll = page.getByRole('menuitem', {name: /^Show all commits/});
  await expect(button).toBeVisible(); // the picker has mounted and declared c

  await page.keyboard.press('c');
  await expect(menu).toBeVisible();
  await expect(showAll).toBeFocused(); // as after a click
  await page.keyboard.press('ArrowDown');
  await expect(showAll).not.toBeFocused();
  await expect(menu.locator(':focus')).toHaveCount(1); // the arrow keys move within the menu
  await page.keyboard.press('ArrowUp');
  await expect(showAll).toBeFocused();
  await page.keyboard.press('c');
  await expect(menu).toBeVisible(); // c never closes it
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(button).toBeFocused();
  await assertNoJsError(page);
});

test('Escape closes the commit picker that c opened while its commits load', async ({page}) => {
  let release!: () => void;
  const held = new Promise<void>((resolve) => release = resolve);
  await page.route('**/commits/list', async (route) => {
    await held;
    await route.continue();
  });
  await page.goto(await createPullRequest(page));
  const button = page.getByRole('button', {name: 'Filter by commit'});
  const menu = page.locator('.diff-commit-selector .menu');
  await expect(button).toBeVisible();

  await page.keyboard.press('c');
  await expect(menu.locator('.is-loading')).toBeVisible(); // the commits are still on their way
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(button).toBeFocused(); // c put focus there, so Escape reached the picker

  const loaded = page.waitForResponse('**/commits/list');
  release();
  await loaded;
  await expect(menu).toBeHidden(); // the late commits don't reopen it
  await assertNoJsError(page);
});

// an issue whose description is taller than the window, so the reply box starts out of view
async function createIssue(page: Page) {
  const repoLink = await createRepo(page);
  const [, owner, repoName] = repoLink.split('/');
  const filler = Array.from({length: 30}, (_, i) => `filler paragraph ${i}`).join('\n\n');
  await apiCreateIssue(page.request, {owner, repo: repoName, title: 'Quote test', body: `first line\n\nsecond line\n\n${filler}`});
  return {issueLink: `${repoLink}/issues/1`, owner, repoName};
}

test('r quotes the selected text into the reply box, or moves to it', async ({page}) => {
  await page.setViewportSize({width: 1280, height: 600});
  const [{issueLink}] = await Promise.all([createIssue(page), login(page)]);
  await page.goto(issueLink);
  const replyBox = page.getByPlaceholder('Leave a comment');
  await expect(page.locator('#comment-form')).toHaveAttribute('data-shortcut-section', 'Conversation');
  await expect(replyBox).not.toBeInViewport();

  await page.locator('.issue-content-left .render-content.markup').first().evaluate((markup) => {
    const range = document.createRange();
    range.setStart(markup.querySelector('p')!.firstChild!, 0);
    range.setEnd(markup.querySelectorAll('p')[1].firstChild!, 6);
    window.getSelection()!.addRange(range);
  });
  await page.keyboard.press('r');
  await expect(replyBox).toBeFocused();
  await expect(replyBox).toBeInViewport();
  await expect(replyBox).toHaveValue(/^> first line\n(> \n)?> second\n/);
  await page.keyboard.type('reply');
  await expect(replyBox).toHaveValue(/\n\nreply$/); // the cursor was after the quote, and r typed in the box is a letter
  const typed = await replyBox.inputValue();

  await replyBox.blur();
  await page.evaluate(() => window.scrollTo(0, 0));
  await expect(replyBox).not.toBeInViewport();
  await page.keyboard.press('r'); // nothing selected
  await expect(replyBox).toBeFocused();
  await expect(replyBox).toBeInViewport();
  await expect(replyBox).toHaveValue(typed);
  await assertNoJsError(page);
});

test('r does nothing without a reply box', async ({page}) => {
  const {issueLink, owner, repoName} = await createIssue(page);
  await page.goto(issueLink); // signed out
  await expect(page.locator('#comment-form')).toHaveCount(0);
  await page.keyboard.press('r');
  await assertNoJsError(page);

  await apiEditRepo(page.request, owner, repoName, {archived: true});
  await login(page);
  await page.goto(issueLink);
  await expect(page.locator('#comment-form')).toHaveCount(0);
  await page.keyboard.press('r');
  await assertNoJsError(page);
});

test('the Quote Reply menu item still quotes the whole comment, and leaves the Preview tab', async ({page}) => {
  const [{issueLink}] = await Promise.all([createIssue(page), login(page)]);
  await page.goto(issueLink);
  const replyBox = page.getByPlaceholder('Leave a comment');
  await replyBox.fill('draft');
  await page.locator('#comment-form [data-tab-for="markdown-previewer"]').click();
  await expect(replyBox).toBeHidden();

  await page.locator('.issue-content-left .comment-header .context-menu').first().click();
  await page.getByRole('menuitem', {name: 'Quote Reply'}).click();
  await expect(replyBox).toBeFocused();
  await expect(replyBox).toHaveValue(/^draft\n\n> first line\n> \n> second line\n> \n> filler paragraph 0\n/); // the raw Markdown
  await assertNoJsError(page);
});
