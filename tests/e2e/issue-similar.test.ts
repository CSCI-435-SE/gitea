import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, randomString, timeoutFactor} from './utils.ts';

test('similar issues appear while typing a new issue title', async ({page, request}) => {
  const repoName = `e2e-issue-similar-${randomString(8)}`;
  const owner = env.GITEA_TEST_E2E_USER;
  const marker = randomString(8);
  const existingTitle = `Login page crashes on submit ${marker}`;
  const typedTitle = `Login page crashes when submitting ${marker}`;
  await apiCreateRepo(request, {name: repoName, autoInit: false});
  await Promise.all([
    apiCreateIssue(request, {owner, repo: repoName, title: existingTitle, body: 'Steps to reproduce.'}),
    login(page),
  ]);

  // The issue indexer consumes a queue, so a just-created issue takes a moment to become
  // searchable. The panel only ever fires one fetch per keystroke pause (no polling of its own),
  // so racing the indexer here would leave it stuck hidden for the rest of the test no matter how
  // long we waited on it afterwards. Poll the same endpoint the frontend calls until the indexer
  // catches up, before driving the browser at all.
  //
  // The marker is shared by both titles and, at 8 runes, is one of the three longest tokens sent
  // to the indexer, so it is not just this suite's usual isolation device: it also gives this
  // exact-match lookup a guaranteed hit. This is a whole-chain smoke test, not proof that the
  // ranking treats "submit" and "submitting" as related — there is no stemming, they are distinct
  // tokens, and that logic is covered by the unit tests in similar_text_test.go.
  await expect.poll(async () => {
    const response = await page.request.get(`/${owner}/${repoName}/issues/similar`, {
      params: {q: typedTitle, is_pull: 'false'},
    });
    return (await response.json()).length;
  }, {timeout: 10000 * timeoutFactor}).toBeGreaterThan(0);

  await page.goto(`/${owner}/${repoName}/issues/new`);
  await page.locator('#issue_title').fill(typedTitle);

  const panel = page.locator('.issue-similar-suggestions');
  await expect(panel).toBeVisible();
  await expect(panel.getByRole('link', {name: existingTitle})).toBeVisible();

  await page.locator('#issue_title').fill('');
  await expect(panel).toBeHidden();
});
