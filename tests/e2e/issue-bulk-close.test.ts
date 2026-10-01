import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString} from './utils.ts';

test('bulk close selected issues with the reason picked in the menu', async ({page, request}) => {
  const repoName = `e2e-bulk-close-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: 'First bulk close test'}),
    apiCreateIssue(request, {owner, repo: repoName, title: 'Second bulk close test'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues?state=open`);

  await page.locator('.issue-checkbox-all').check();
  const closeGroup = page.locator('#issue-actions [data-global-init="initIssueListCloseReason"]');
  await closeGroup.locator('.ui.dropdown').click();
  await closeGroup.locator('.menu .item', {hasText: 'Not planned'}).click();
  // the button takes the picked reason, and its request carries it; the page reloads once the server answers
  await Promise.all([
    page.waitForResponse((response) => response.url().endsWith('/issues/status')),
    closeGroup.getByRole('button', {name: 'Close as not planned'}).click(),
  ]);

  await page.goto(`/${owner}/${repoName}/issues?state=closed`);
  await expect(page.locator('#issue-list .item-leading .octicon-skip')).toHaveCount(2); // not planned's icon, not the red one of no reason
});
