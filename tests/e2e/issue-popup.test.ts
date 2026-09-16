import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString} from './utils.ts';

test('hover preview on the issue list', async ({page, request}) => {
  const repoName = `e2e-issue-popup-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  const title = `Hover preview target ${randomString(8)}`;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title, body: 'Body of the hovered issue.'}),
    login(page),
  ]);
  await page.goto(`/${owner}/${repoName}/issues`);

  // the list title carries no .ref-issue class and sits in no popup container, so this only
  // works once the handler attaches to any issue-shaped link
  await page.getByRole('link', {name: title}).hover();

  // the popup is a tippy instance created with role "dialog"; assert on the role rather than
  // tippy's internal classes, which vary with the animation setting
  const popup = page.getByRole('dialog');
  await expect(popup).toBeVisible();
  await expect(popup).toContainText(title);
  await expect(popup).toContainText('Body of the hovered issue.');
});
