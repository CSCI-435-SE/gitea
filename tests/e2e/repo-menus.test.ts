import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Page} from '@playwright/test';
import {login, apiCreateRepo, apiCreateFile, apiCreateBranch, apiEditRepo, apiHeaders, baseUrl, randomString, assertNoJsError} from './utils.ts';

const owner = env.GITEA_TEST_E2E_USER;

// the archive handler POSTs to prepare the archive before downloading it
function waitForArchivePost(page: Page, suffix: string) {
  return page.waitForRequest((req) => req.method() === 'POST' && new URL(req.url()).pathname.endsWith(suffix));
}

test('code tab add file menu is operable by keyboard', async ({page, request}) => {
  const repoName = `e2e-add-file-menu-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  await page.goto(`/${owner}/${repoName}`);
  const trigger = page.getByRole('button', {name: 'Add File'});
  const menu = page.getByRole('menu', {name: 'Add File'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', {name: 'New File'})).toBeFocused();
  await page.keyboard.press('End');
  await expect(page.getByRole('menuitem', {name: 'Apply Patch'})).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(trigger).toHaveAttribute('aria-expanded', 'false');
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('u');
  await expect(page.getByRole('menuitem', {name: 'Upload File'})).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(new RegExp(`/${repoName}/_upload/main/$`));
  await assertNoJsError(page);
});

test('code tab add file menu stays shut in an archived repo', async ({page, request}) => {
  const repoName = `e2e-add-file-archived-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  await apiEditRepo(request, owner, repoName, {archived: true});
  await page.goto(`/${owner}/${repoName}`);
  const trigger = page.getByRole('button', {name: 'Add File'});

  await expect(trigger).toBeDisabled(); // a disabled button takes neither focus nor clicks
  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'false');
  await expect(page.getByRole('menu', {name: 'Add File'})).toBeHidden();
});

test('code tab folder menu is operable by keyboard', async ({page, request}) => {
  const repoName = `e2e-folder-menu-${randomString(8)}`;
  await apiCreateRepo(request, {name: repoName});
  await Promise.all([apiCreateFile(request, owner, repoName, 'docs/readme.md', 'docs'), login(page)]);
  await page.goto(`/${owner}/${repoName}/src/branch/main/docs`);
  const trigger = page.getByRole('button', {name: 'More Operations'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', {name: 'Copy Permalink'})).toBeFocused();
  await page.keyboard.press('End');
  await expect(page.getByRole('menuitem', {name: 'Delete Directory'})).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('menu', {name: 'More Operations'})).toBeHidden();
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  // the browser's clipboard permission decides the text
  await expect(page.locator('.tippy-box').filter({hasText: /Copied|Copy failed/})).toBeVisible();

  await trigger.focus();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('ArrowDown');
  await expect(page.getByRole('menuitem', {name: 'Download directory as ZIP'})).toBeFocused();
  const archivePost = waitForArchivePost(page, '/archive/main.zip');
  await page.keyboard.press('Enter');
  await archivePost;
  await assertNoJsError(page);
});

test('branch download menus are named and operable by keyboard', async ({page, request}) => {
  const repoName = `e2e-branch-menu-${randomString(8)}`;
  await apiCreateRepo(request, {name: repoName});
  await Promise.all([apiCreateBranch(request, owner, repoName, 'feature'), login(page)]);
  await page.goto(`/${owner}/${repoName}/branches`);

  for (const branch of ['main', 'feature']) {
    const trigger = page.getByRole('button', {name: `Download Branch "${branch}"`});
    await expect(trigger).toHaveAttribute('aria-haspopup', 'menu');
    await expect(page.getByRole('menu', {name: `Download Branch "${branch}"`, includeHidden: true})).toHaveAttribute('id', (await trigger.getAttribute('aria-controls'))!);
  }

  const trigger = page.getByRole('button', {name: 'Download Branch "feature"'});
  const triggerBox = (await trigger.boundingBox())!;
  await trigger.focus();
  await page.keyboard.press('ArrowUp');
  await expect(page.getByRole('menuitem', {name: 'TAR.GZ'})).toBeFocused();
  // the fixed table's cells clip by default, and focusing a clipped item scrolls the cell
  expect(await trigger.boundingBox()).toEqual(triggerBox);
  const menuBox = (await page.getByRole('menu', {name: 'Download Branch "feature"'}).boundingBox())!;
  expect(menuBox.y).toBeGreaterThanOrEqual(triggerBox.y + triggerBox.height);
  await expect(page.getByRole('menuitem', {name: 'ZIP'})).toBeInViewport();
  const archivePost = waitForArchivePost(page, '/archive/feature.tar.gz');
  await page.keyboard.press('Enter');
  await archivePost;
  await assertNoJsError(page);
});

test('commit operations menu opens its dialogs by keyboard', async ({page, request}) => {
  const repoName = `e2e-commit-menu-${randomString(8)}`;
  await apiCreateRepo(request, {name: repoName});
  const [branchResp] = await Promise.all([
    request.get(`${baseUrl()}/api/v1/repos/${owner}/${repoName}/branches/main`, {headers: apiHeaders()}),
    login(page),
  ]);
  const commitID = (await branchResp.json()).commit.id;
  await page.goto(`/${owner}/${repoName}/commit/${commitID}`);
  const trigger = page.getByRole('button', {name: 'Operations'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', {name: 'Create branch'})).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(page.getByRole('menuitem', {name: 'Create tag'})).toBeFocused();
  await page.keyboard.press('Enter');
  const tagInput = page.locator('#create-tag-modal input[name=new_branch_name]');
  await expect(tagInput).toBeVisible();
  await tagInput.pressSequentially('v1.0-rc');
  await expect(tagInput).toHaveValue('v1.0-rc');
  await page.locator('#create-tag-modal .cancel.button').click();
  await expect(tagInput).toBeHidden();

  await trigger.focus();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  const branchInput = page.locator('#create-branch-modal input[name=new_branch_name]');
  await expect(branchInput).toBeVisible();
  await branchInput.pressSequentially('release-1');
  await expect(branchInput).toHaveValue('release-1');
  await assertNoJsError(page);
});
