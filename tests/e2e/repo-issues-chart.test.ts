import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, apiDeleteRepo, assertNoJsError, randomString} from './utils.ts';

test('activity issues chart renders and switches range', async ({page}) => {
  const repoName = `e2e-issues-chart-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);
  await apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'Backlog item'});

  await page.goto(`/${user}/${repoName}/activity`);
  await page.locator('.flex-container-nav').getByRole('link', {name: 'Issues'}).click();
  await page.waitForURL(`**/${user}/${repoName}/activity/issues`);

  const chart = page.locator('#repo-issues-chart');
  await expect(chart.locator('canvas')).toBeVisible();

  // the picker defaults to three months and reslices the chart in the browser
  const ranges = chart.locator('.menu .item');
  await expect(ranges).toHaveCount(4);
  await expect(chart.locator('.menu .item.active')).toHaveText('Last 3 months');
  await ranges.filter({hasText: 'All time'}).click();
  await expect(chart.locator('.menu .item.active')).toHaveText('All time');
  await expect(chart.locator('canvas')).toBeVisible();

  await assertNoJsError(page);
  await apiDeleteRepo(page.request, user, repoName);
});

test('activity issues chart shows an empty state without issues', async ({page}) => {
  const repoName = `e2e-issues-chart-empty-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);

  await page.goto(`/${user}/${repoName}/activity/issues`);
  const chart = page.locator('#repo-issues-chart');
  await expect(chart).toContainText('This repository has no issues yet.');
  await expect(chart.locator('canvas')).toBeHidden();
  // no data means no range to pick between
  await expect(chart.locator('.menu .item')).toHaveCount(0);

  await apiDeleteRepo(page.request, user, repoName);
});
