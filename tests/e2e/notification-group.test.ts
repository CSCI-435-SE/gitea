import {test, expect} from '@playwright/test';
import {apiCreateRepo, apiCreateIssue, apiCreateUser, apiDeleteUser, apiUserHeaders, loginUser, randomString} from './utils.ts';

test('collapse a repository section by keyboard and keep it collapsed when the list re-renders', async ({page, request}) => {
  const owner = `e2e-group-${randomString(8)}`; // a fresh user, so no other test's notifications are listed
  await apiCreateUser(request, owner);
  await Promise.all(['alpha', 'beta'].map((name) => apiCreateRepo(request, {name, headers: apiUserHeaders(owner)})));
  // the admin's issues notify the owner, who watches their own repos
  await Promise.all([
    apiCreateIssue(request, {owner, repo: 'alpha', title: 'Alpha notification'}),
    apiCreateIssue(request, {owner, repo: 'beta', title: 'Beta notification'}),
    loginUser(page, owner),
  ]);

  const groups = page.locator('#notification_table .notification-group');
  await expect(async () => { // notifications are written by a background queue
    await page.goto('/notifications?group=repo');
    await expect(groups).toHaveCount(2, {timeout: 500});
  }).toPass();

  const alpha = groups.filter({hasText: `${owner}/alpha`});
  const beta = groups.filter({hasText: `${owner}/beta`});
  const alphaToggle = alpha.locator('.notification-group-toggle');
  const betaToggle = beta.locator('.notification-group-toggle');

  await alphaToggle.focus();
  await page.keyboard.press('Enter');
  await expect(alphaToggle).toHaveAttribute('aria-expanded', 'false');
  await expect(alpha.locator('.notifications-item')).toBeHidden();
  await page.keyboard.press('Space');
  await expect(alphaToggle).toHaveAttribute('aria-expanded', 'true');
  await expect(alpha.locator('.notifications-item')).toBeVisible();

  await betaToggle.click();
  await expect(betaToggle).toHaveAttribute('aria-expanded', 'false');

  // pinning a row re-renders the whole list in place, like a live refresh
  const alphaRow = alpha.locator('.notifications-item');
  await alphaRow.hover();
  await Promise.all([
    page.waitForResponse((response) => response.url().includes('/notifications/status')),
    alphaRow.locator('button[value="pin"]').click(),
  ]);
  await expect(alpha.locator('.notifications-item .octicon-pin.tw-text-blue')).toBeVisible();
  await expect(page).toHaveURL(/group=repo/);
  await expect(betaToggle).toHaveAttribute('aria-expanded', 'false');
  await expect(beta.locator('.notifications-item')).toBeHidden();

  await apiDeleteUser(request, owner);
});
