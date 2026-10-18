import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Page} from '@playwright/test';
import {apiCreateFile, apiCreateRepo, assertNoJsError, randomString} from './utils.ts';

async function createRepo(page: Page) {
  const repoName = `e2e-shortcut-${randomString(8)}`;
  await apiCreateRepo(page.request, {name: repoName});
  await apiCreateFile(page.request, env.GITEA_TEST_E2E_USER, repoName, 'other.txt', 'other file');
  return `/${env.GITEA_TEST_E2E_USER}/${repoName}`;
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
