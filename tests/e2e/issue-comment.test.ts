import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString} from './utils.ts';

test('comment on and close an issue', async ({page, request}) => {
  const repoName = `e2e-issue-comment-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: 'Comment test'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues/1`);

  const body = `e2e-comment-${randomString(8)}`;
  await page.getByPlaceholder('Leave a comment').fill(body);
  // exact match: the status button reads "Close as completed with comment" while the box has content, which substring-matches "Comment"
  await page.getByRole('button', {name: 'Comment', exact: true}).click();
  await expect(page.locator('.comment-body').filter({hasText: body})).toBeVisible();

  // wait for the form to re-initialize (the empty box disables the comment button); a close click
  // before that does a native submit which lands on a raw JSON page instead of reloading the issue
  await expect(page.getByRole('button', {name: 'Comment', exact: true})).toBeDisabled();
  await page.getByRole('button', {name: 'Close as completed'}).click();
  await expect(page.getByRole('button', {name: 'Reopen Issue'})).toBeVisible();
});

test('comment on a phone, beside the close button and its reason menu', async ({page, request}) => {
  const repoName = `e2e-issue-comment-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: 'Phone comment test'}),
    login(page),
  ]);
  await page.setViewportSize({width: 360, height: 800});
  await page.goto(`/${owner}/${repoName}/issues/1`);

  const body = `e2e-comment-${randomString(8)}`;
  await page.getByPlaceholder('Leave a comment').fill(body);
  // the click waits for nothing to cover the button: the reason menu's ▾ used to spill over it on narrow screens
  await page.getByRole('button', {name: 'Comment', exact: true}).click();
  await expect(page.locator('.comment-body').filter({hasText: body})).toBeVisible();
});

test('a comment the browser restores on Back is still named on the close button', async ({page, request}) => {
  const repoName = `e2e-issue-comment-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: 'Back test'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues/1`);

  const body = `e2e-comment-${randomString(8)}`;
  await page.getByPlaceholder('Leave a comment').fill(body);
  await page.goto(`/${owner}/${repoName}/issues`);
  await page.goBack();
  // Chromium puts the comment back after the page has started, without an event; the button must still say it will be posted
  await expect(page.getByPlaceholder('Leave a comment')).toHaveValue(body);
  await expect(page.getByRole('button', {name: 'Close as completed with comment'})).toBeVisible();
});
