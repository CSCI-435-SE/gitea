import {test, expect} from '@playwright/test';
import {apiCreateRepo, apiCreateIssue, apiCreateUser, apiDeleteUser, apiUserHeaders, loginUser, randomString} from './utils.ts';

test('mark selected notifications as read in one action', async ({page, request}) => {
  const owner = `e2e-notif-${randomString(8)}`; // a fresh user, so no other test's notifications are listed
  const repo = 'notified';
  await apiCreateUser(request, owner);
  await apiCreateRepo(request, {name: repo, headers: apiUserHeaders(owner)});
  // the admin's issues notify the owner, who watches their own repo
  await Promise.all([
    apiCreateIssue(request, {owner, repo, title: 'First bulk notification'}),
    apiCreateIssue(request, {owner, repo, title: 'Second bulk notification'}),
    loginUser(page, owner),
  ]);

  const rows = page.locator('#notification_table .notifications-item');
  await expect(async () => { // notifications are written by a background queue
    await page.goto('/notifications');
    await expect(rows).toHaveCount(2, {timeout: 500});
  }).toPass();

  const bulkActions = page.locator('.notification-bulk-actions');
  await expect(bulkActions).toBeHidden();
  await rows.locator('.notification-checkbox').first().check();
  await rows.locator('.notification-checkbox').last().check();
  await expect(bulkActions).toContainText('2 selected');

  await Promise.all([
    page.waitForResponse((response) => response.url().includes('/notifications/bulk')),
    bulkActions.getByRole('button', {name: 'Mark as read'}).click(),
  ]);
  await expect(page.locator('.flash-message')).toContainText('Marked 2 notifications as read.');
  await expect(rows).toHaveCount(0);
  await expect(page.locator('.notification-bulk-bar')).toHaveCount(0);

  await apiDeleteUser(request, owner);
});
