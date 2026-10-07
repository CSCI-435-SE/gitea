import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, apiCreateMilestone, apiDeleteRepo, apiHeaders, assertNoJsError, baseUrl, randomString} from './utils.ts';

test('milestone burndown charts remaining work against the ideal line', async ({page}) => {
  const repoName = `e2e-burndown-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);
  const dueOn = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
  const {id} = await apiCreateMilestone(page.request, {owner: user, repo: repoName, title: 'Sprint', dueOn});
  const [first] = await Promise.all([
    apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'First', milestone: id}),
    apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'Second', milestone: id}),
  ]);
  const closed = await page.request.patch(`${baseUrl()}/api/v1/repos/${user}/${repoName}/issues/${first.index}`, {
    headers: apiHeaders(), data: {state: 'closed'},
  });
  expect(closed.ok()).toBeTruthy();

  await page.goto(`/${user}/${repoName}/milestone/${id}`);
  const chart = page.locator('#milestone-burndown-chart');
  await expect(chart.locator('canvas')).toBeVisible();
  // one day of history is too little to project, and the summary says so in words
  await expect(chart.locator('.milestone-burndown-summary')).toContainText('two days of history');
  await expect(chart.locator('canvas')).toHaveAttribute('aria-label', /two days of history/);

  // the list under the chart starts on the latest day with changes and links each item
  const changes = chart.locator('.milestone-burndown-changes');
  await expect(changes).toContainText(`Closed: #${first.index} First`);
  await expect(changes.getByRole('link', {name: `#${first.index} First`})).toHaveAttribute('href', `/${user}/${repoName}/issues/${first.index}`);

  // the pull request toggle is off by default and lives in the URL, so a shared link keeps it
  const includePulls = chart.getByRole('checkbox', {name: 'Include pull requests'});
  await expect(includePulls).not.toBeChecked();
  await includePulls.check();
  await expect(page).toHaveURL(/[?&]include_pulls=1/);
  await page.reload();
  await expect(chart.getByRole('checkbox', {name: 'Include pull requests'})).toBeChecked();
  await chart.getByRole('checkbox', {name: 'Include pull requests'}).uncheck();
  await expect(page).not.toHaveURL(/include_pulls/);

  // the chart folds away under its heading and comes back
  const heading = chart.locator('summary');
  await heading.click();
  await expect(chart.locator('canvas')).toBeHidden();
  await heading.click();
  await expect(chart.locator('canvas')).toBeVisible();

  await assertNoJsError(page);
  await apiDeleteRepo(page.request, user, repoName);
});

test('milestone burndown shows an empty state without items', async ({page}) => {
  const repoName = `e2e-burndown-empty-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);
  const {id} = await apiCreateMilestone(page.request, {owner: user, repo: repoName, title: 'Empty'});

  await page.goto(`/${user}/${repoName}/milestone/${id}`);
  const chart = page.locator('#milestone-burndown-chart');
  await expect(chart).toContainText('This milestone has no issues or pull requests yet.');
  await expect(chart.locator('canvas')).toBeHidden();

  await apiDeleteRepo(page.request, user, repoName);
});
