import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateFile, createProject, createProjectColumn, randomString, assertNoJsError, timeoutFactor} from './utils.ts';

const owner = env.GITEA_TEST_E2E_USER;

test('project column menu opens the edit dialog by keyboard without moving the column', async ({page, request}) => {
  const repoName = `e2e-column-menu-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  const {id} = await createProject(page, {owner, repo: repoName, title: 'Board'});
  for (const title of ['First', 'Second']) {
    await createProjectColumn(page.request, owner, repoName, String(id), title); // a web route, so it needs the signed-in session
  }
  await page.goto(`/${owner}/${repoName}/projects/${id}`);
  const columnIds = () => page.locator('#project-board > .project-column').evaluateAll((els) => els.map((el) => el.getAttribute('data-id')));
  const before = await columnIds();
  const column = page.locator('.project-column').filter({hasText: 'Second'});
  const trigger = column.getByRole('button', {name: 'More Operations'});

  // a mouse press on the menu sits inside the column's drag handle, and must open the menu instead
  await trigger.click();
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await page.keyboard.press('Escape');
  await expect(trigger).toBeFocused();

  await page.keyboard.press('Enter');
  await expect(column.getByRole('menuitem', {name: 'Edit Column'})).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('#project-column-modal-edit')).toBeVisible();
  await expect(page.locator('#project-column-modal-edit input[name="title"]')).toHaveValue('Second');
  expect(await columnIds()).toEqual(before);
  await assertNoJsError(page);
});

test('run row menu cancels a run by keyboard', async ({page, request}) => {
  test.setTimeout(30000 * timeoutFactor); // the run is created by a background queue, slower when browsers run in parallel
  const repoName = `e2e-run-menu-${randomString(8)}`;
  await Promise.all([apiCreateRepo(request, {name: repoName}), login(page)]);
  // with no runner the run stays waiting, so it can be cancelled
  await apiCreateFile(request, owner, repoName, '.gitea/workflows/wait.yml', 'on: push\njobs:\n  wait:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo hi\n');

  const run = page.locator('.run-list > .item').first();
  await expect(async () => {
    await page.goto(`/${owner}/${repoName}/actions`);
    await expect(run).toBeVisible({timeout: 1000});
  }).toPass({timeout: 20000 * timeoutFactor});
  const trigger = run.getByRole('button', {name: 'More Operations'});

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(run.getByRole('menuitem', {name: 'View workflow file'})).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(run.getByRole('menuitem', {name: 'Cancel workflow run'})).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('.run-list > .item').first()).toHaveAttribute('data-status', 'cancelled');
  await assertNoJsError(page);
});
