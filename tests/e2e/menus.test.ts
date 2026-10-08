import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString, assertNoJsError} from './utils.ts';

test('profile menu is operable by keyboard', async ({page}) => {
  await login(page);
  await page.goto('/');
  const trigger = page.getByRole('button', {name: 'Profile and Settings…'});
  const menu = page.getByRole('menu', {name: 'Profile and Settings…'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', {name: 'Profile', exact: true})).toBeFocused();

  const menuBox = (await menu.boundingBox())!;
  expect(menuBox.x + menuBox.width).toBeLessThanOrEqual(page.viewportSize()!.width);

  await page.keyboard.press('End');
  await expect(page.getByRole('menuitem', {name: 'Sign Out'})).toBeFocused();
  await page.keyboard.press('Home');
  await expect(page.getByRole('menuitem', {name: 'Profile', exact: true})).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  for (const name of ['Starred', 'Subscriptions', 'Settings']) {
    await page.keyboard.press('s');
    await expect(page.getByRole('menuitem', {name})).toBeFocused();
  }
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/\/user\/settings$/);
  await assertNoJsError(page);
});

test('profile menu is reached by Tab', async ({page, browserName}) => {
  test.skip(browserName === 'webkit', 'macOS WebKit, like Safari, skips buttons on Tab by default'); // eslint-disable-line playwright/no-skipped-test
  await login(page);
  await page.goto('/');
  await page.getByRole('button', {name: 'Create…'}).focus();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('button', {name: 'Profile and Settings…'})).toBeFocused();
});

test('navbar tooltip does not cover an open menu', async ({page}) => {
  await page.clock.install();
  await login(page);
  await page.goto('/');
  const trigger = page.getByRole('button', {name: 'Create…'});
  const tooltip = page.locator('.tippy-box').filter({hasText: 'Create…'});

  await trigger.hover();
  await expect(tooltip).toBeVisible();
  await trigger.click();
  await expect(tooltip).toBeHidden();
  await page.getByRole('menuitem', {name: 'New Repository'}).hover();
  await trigger.hover();
  await page.clock.runFor(300); // past the tooltip's show delay
  await expect(tooltip).toBeHidden();
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
});

test('footer theme selector has an accessible name', async ({page}) => {
  await page.goto('/');
  await expect(page.getByRole('menu', {name: 'Theme'})).toBeVisible();
  await assertNoJsError(page);
});

test('comment menu is operable by keyboard', async ({page, request}) => {
  const repoName = `e2e-comment-menu-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: 'Comment menu test', body: 'the issue body'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues/1`);
  const trigger = page.getByRole('button', {name: 'More Operations'}).first();

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', {name: 'Copy Link'})).toBeFocused();
  await page.keyboard.press('Enter');
  // the browser's clipboard permission decides the text
  const tooltip = page.locator('.tippy-box').filter({hasText: /Copied|Copy failed/});
  await expect(tooltip).toBeVisible();
  const [tipBox, triggerBox] = [(await tooltip.boundingBox())!, (await trigger.boundingBox())!];
  expect(Math.abs(tipBox.y - triggerBox.y)).toBeLessThan(80);
  expect(Math.abs(tipBox.x - triggerBox.x)).toBeLessThan(300);
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowUp');
  await page.keyboard.press('Escape');
  await expect(page.getByRole('menuitem', {name: 'Copy Link'})).toBeHidden();
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('e');
  await expect(page.getByRole('menuitem', {name: 'Edit'})).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('.edit-content-zone textarea')).toHaveValue('the issue body');
  await assertNoJsError(page);
});
