import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Page} from '@playwright/test';
import {apiCreateBranch, apiCreateFile, apiCreateRepo, apiEditRepo, assertNoJsError, login, randomString} from './utils.ts';

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
  expect(page.url()).toMatch(new RegExp(`${repoLink}$`));
  expect(page.context().pages()).toHaveLength(1); // no external tab opened
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
