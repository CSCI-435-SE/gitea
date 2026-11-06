import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {APIRequestContext} from '@playwright/test';
import {apiCreateFile, apiCreateIssue, apiCreatePR, apiCreateRepo, apiEditRepo, apiHeaders, baseUrl, login, randomString} from './utils.ts';

const owner = env.GITEA_TEST_E2E_USER;

// a repository whose issues can have dependencies, set here rather than left to the server's default
async function createRepoWithDependencies(request: APIRequestContext, name: string, extra: Record<string, unknown> = {}) {
  await apiCreateRepo(request, {name});
  await apiEditRepo(request, owner, name, {
    has_issues: true, // the API ignores internal_tracker without it
    internal_tracker: {enable_time_tracker: true, allow_only_contributors_to_track_time: true, enable_issue_dependencies: true},
    ...extra,
  });
}

async function createPull(request: APIRequestContext, repo: string, branch: string, title: string) {
  await apiCreateFile(request, owner, repo, `${branch}.txt`, `${branch}\n`, {newBranch: branch});
  return apiCreatePR(request, owner, repo, branch, 'main', title);
}

async function addDependency(request: APIRequestContext, repo: string, index: number, dependsOn: number) {
  const response = await request.post(`${baseUrl()}/api/v1/repos/${owner}/${repo}/issues/${index}/dependencies`, {
    headers: apiHeaders(),
    data: {owner, repo, index: dependsOn},
  });
  expect(response.ok()).toBeTruthy();
}

test('#144 AC1 AC2 the merge box names the blocking dependency and offers only Manually merged', async ({page, request}) => {
  const repo = `e2e-merge-deps-${randomString(8)}`;
  await createRepoWithDependencies(request, repo, {allow_manual_merge: true});
  const pull = await createPull(request, repo, 'blocked', 'Blocked pull');
  const {index: issue} = await apiCreateIssue(request, {owner, repo, title: 'Open dependency'});
  await Promise.all([addDependency(request, repo, pull, issue), login(page)]);

  await page.goto(`/${owner}/${repo}/pulls/${pull}`);
  const mergeBox = page.locator('.pull-merge-box');
  await expect(mergeBox.getByText('This pull request is blocked by its dependencies:')).toBeVisible();
  await expect(mergeBox.getByRole('link', {name: `#${issue} Open dependency`})).toHaveAttribute('href', `/${owner}/${repo}/issues/${issue}`);

  // wait for the merge form to load before checking what it offers, so an empty form cannot pass
  await expect(mergeBox.getByRole('button', {name: 'Manually merged'})).toBeVisible();
  await mergeBox.locator('.merge-button .dropdown').click();
  await expect(mergeBox.locator('.merge-button .menu .item')).toHaveText(['Manually merged']); // no other style, no "(When checks succeed)"
});

test('#144 AC5 the sidebar shows each dependency as Open, Closed or Merged', async ({page, request}) => {
  const repo = `e2e-dep-states-${randomString(8)}`;
  await createRepoWithDependencies(request, repo);
  const {index: subject} = await apiCreateIssue(request, {owner, repo, title: 'Subject'});
  const [{index: open}, closed, merged] = await Promise.all([
    apiCreateIssue(request, {owner, repo, title: 'Open issue'}),
    createPull(request, repo, 'closed', 'Closed pull'),
    createPull(request, repo, 'merged', 'Merged pull'),
  ]);
  const pulls = `${baseUrl()}/api/v1/repos/${owner}/${repo}/pulls`;
  expect((await request.patch(`${pulls}/${closed}`, {headers: apiHeaders(), data: {state: 'closed'}})).ok()).toBeTruthy();
  await expect(async () => { // the merge is refused until the new pull request's conflict check has finished
    expect((await request.post(`${pulls}/${merged}/merge`, {headers: apiHeaders(), data: {Do: 'merge'}})).ok()).toBeTruthy();
  }).toPass();
  await Promise.all([
    ...[open, closed, merged].map((dep) => addDependency(request, repo, subject, dep)),
    login(page),
  ]);

  await page.goto(`/${owner}/${repo}/issues/${subject}`);
  const entries = page.locator('.ui.depending .item.dependency');
  await expect(entries.filter({hasText: `#${open} Open issue`})).toContainText('· Open');
  await expect(entries.filter({hasText: `#${closed} Closed pull`})).toContainText('· Closed');
  await expect(entries.filter({hasText: `#${merged} Merged pull`})).toContainText('· Merged');
});
