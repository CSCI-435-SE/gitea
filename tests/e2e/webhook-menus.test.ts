import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, randomString, assertNoJsError} from './utils.ts';

const owner = env.GITEA_TEST_E2E_USER;

test('add webhook menu adds a webhook by keyboard', async ({page, request}) => {
  const repoName = `e2e-add-webhook-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  await page.goto(`/${owner}/${repoName}/settings/hooks`);
  const trigger = page.getByRole('button', {name: 'Add Webhook'});
  const menu = page.getByRole('menu', {name: 'Add Webhook'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', {name: 'Gitea'})).toBeFocused();
  await page.keyboard.press('End');
  await expect(page.getByRole('menuitem', {name: 'Packagist'})).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('d');
  await expect(page.getByRole('menuitem', {name: 'Discord'})).toBeFocused();
  await page.keyboard.press('d');
  await expect(page.getByRole('menuitem', {name: 'DingTalk'})).toBeFocused();
  // the button sits at the header's right edge, so the menu must open leftwards from it
  const triggerBox = (await trigger.boundingBox())!;
  const menuBox = (await menu.boundingBox())!;
  expect(menuBox.x + menuBox.width).toBeCloseTo(triggerBox.x + triggerBox.width, 0);

  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(new RegExp(`/${repoName}/settings/hooks/dingtalk/new$`));
  const hookUrl = `https://example.com/${randomString(8)}`;
  await page.locator('#payload_url').fill(hookUrl);
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(new RegExp(`/${repoName}/settings/hooks$`));
  await expect(page.getByText(hookUrl)).toBeVisible();
  await assertNoJsError(page);
});

test('webhook type switcher is named and switches type by keyboard', async ({page, request}) => {
  const repoName = `e2e-webhook-type-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  await page.goto(`/${owner}/${repoName}/settings/hooks/dingtalk/new`);
  const trigger = page.getByRole('button', {name: 'Webhook type: DingTalk'});

  await expect(page.getByRole('menu', {name: 'Webhook type: DingTalk', includeHidden: true})).toHaveAttribute('id', (await trigger.getAttribute('aria-controls'))!);
  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', {name: 'Gitea'})).toBeFocused();
  await page.keyboard.press('s');
  await expect(page.getByRole('menuitem', {name: 'Slack'})).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(new RegExp(`/${repoName}/settings/hooks/slack/new$`));
  await expect(page.getByRole('button', {name: 'Webhook type: Slack'})).toBeVisible();
  await assertNoJsError(page);
});
