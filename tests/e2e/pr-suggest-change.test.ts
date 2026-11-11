import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Locator, Page} from '@playwright/test';
import {apiCreateFile, apiCreatePR, apiCreateRepo, apiHeaders, baseUrl, login, randomString} from './utils.ts';

const owner = env.GITEA_TEST_E2E_USER;
const base64 = (s: string) => Buffer.from(s, 'utf8').toString('base64');

// line 4 changes (old side and new side), line 5 holds a fence, line 6 is empty; expected blocks follow #155's ACs
const before = 'package main\n\nfunc main() {\n\told := 1\n}\n';
const after = 'package main\n\nfunc main() {\n\tx := 1\n\ts := "```js"\n\n}\n';

async function openCommentForm(page: Page, side: 'left' | 'right', line: number): Promise<Locator> {
  const plus = page.locator(`.add-code-comment[data-side="${side}"][data-idx="${line}"]`).first();
  const row = plus.locator('xpath=ancestor::tr');
  await row.hover();
  await plus.click();
  return row.locator(`xpath=following-sibling::tr[1]`).locator(`.add-comment-${side} form`);
}

test('suggest change button in the code comment form', async ({page, request}) => {
  const repo = `e2e-suggest-${randomString(8)}`;
  await apiCreateRepo(request, {name: repo});
  await apiCreateFile(request, owner, repo, 'main.go', before);
  expect((await request.post(`${baseUrl()}/api/v1/repos/${owner}/${repo}/contents`, {
    headers: apiHeaders(),
    data: {new_branch: 'feat', files: [{operation: 'upload', path: 'main.go', content: base64(after)}]},
  })).ok()).toBe(true);
  const index = await apiCreatePR(request, owner, repo, 'feat', 'main', 'suggest');
  const filesUrl = `/${owner}/${repo}/pulls/${index}/files`;
  await login(page);

  // unified view, added line holding a 3-backtick run (AC1, AC3, AC4, AC10)
  await page.goto(`${filesUrl}?style=unified`);
  let form = await openCommentForm(page, 'right', 5);
  const textarea = form.locator('textarea[name="content"]');
  const suggest = form.getByRole('button', {name: 'Suggest change'});
  await expect(suggest).toBeVisible();
  await suggest.click();
  const block = '````suggestion\n\ts := "```js"\n````\n';
  await expect(textarea).toHaveValue(block);
  await expect(textarea).toBeFocused();
  const cursor = '````suggestion\n\ts := "```js"'.length;
  expect(await textarea.evaluate((el: HTMLTextAreaElement) => [el.selectionStart, el.selectionEnd])).toEqual([cursor, cursor]);

  // a second click, cursor still in the first block, adds a second block after it and leaves the first as is (AC12)
  await suggest.click();
  const content = `${block}${block}`;
  await expect(textarea).toHaveValue(content);
  expect(await textarea.evaluate((el: HTMLTextAreaElement) => el.selectionStart)).toEqual(block.length + cursor);

  // the stored comment is exactly the textarea text (AC11), with the CRLF newlines every form submit sends (HTML spec)
  await form.getByRole('button', {name: 'Add single comment'}).click();
  await expect(page.locator('.conversation-holder .comment-body')).toBeVisible();
  const reviewsResp = await request.get(`${baseUrl()}/api/v1/repos/${owner}/${repo}/pulls/${index}/reviews`, {headers: apiHeaders()});
  expect(reviewsResp.ok()).toBe(true);
  const commentsResp = await request.get(`${baseUrl()}/api/v1/repos/${owner}/${repo}/pulls/${index}/reviews/${(await reviewsResp.json())[0].id}/comments`, {headers: apiHeaders()});
  expect(commentsResp.ok()).toBe(true);
  const comments = await commentsResp.json();
  expect(comments[0].body).toEqual(content.replaceAll('\n', '\r\n'));

  // a reply in an existing conversation has no button (AC7)
  await page.locator('.conversation-holder .comment-form-reply').click();
  await expect(page.locator('.conversation-holder form textarea[name="content"]')).toBeVisible();
  await expect(page.locator('.conversation-holder form').getByRole('button', {name: 'Suggest change'})).toBeHidden();

  // old-side (deleted) line has no button (AC6)
  form = await openCommentForm(page, 'left', 4);
  await expect(form.locator('textarea[name="content"]')).toBeVisible();
  await expect(form.getByRole('button', {name: 'Suggest change'})).toBeHidden();

  // split view, empty added line on a closed PR (AC1, AC5, AC9)
  expect((await request.patch(`${baseUrl()}/api/v1/repos/${owner}/${repo}/pulls/${index}`, {headers: apiHeaders(), data: {state: 'closed'}})).ok()).toBe(true);
  await page.goto(`${filesUrl}?style=split`);
  form = await openCommentForm(page, 'right', 6);
  await form.getByRole('button', {name: 'Suggest change'}).click();
  await expect(form.locator('textarea[name="content"]')).toHaveValue('```suggestion\n\n```\n');

  // signed out: no code comment form to hold the button (AC8)
  await page.context().clearCookies();
  await page.goto(filesUrl);
  await expect(page.locator('.diff-file-box')).toBeVisible();
  await expect(page.locator('.add-code-comment')).toHaveCount(0);
});

test('suggest change on a line revealed by expand', async ({page, request}) => {
  const repo = `e2e-suggest-expand-${randomString(8)}`;
  await apiCreateRepo(request, {name: repo});
  const lines = Array.from({length: 12}, (_, i) => (i === 1 ? '  keep 2' : `line ${i + 1}`));
  await apiCreateFile(request, owner, repo, 'a.txt', `${lines.join('\n')}\n`);
  lines[11] = 'changed 12'; // the hunk shows lines 9-12, so line 2 is hidden behind the expander
  expect((await request.post(`${baseUrl()}/api/v1/repos/${owner}/${repo}/contents`, {
    headers: apiHeaders(),
    data: {new_branch: 'feat', files: [{operation: 'upload', path: 'a.txt', content: base64(`${lines.join('\n')}\n`)}]},
  })).ok()).toBe(true);
  const index = await apiCreatePR(request, owner, repo, 'feat', 'main', 'suggest expand');
  await login(page);
  await page.goto(`/${owner}/${repo}/pulls/${index}/files?style=unified`);

  await page.locator('.code-expander-button').first().click();
  await expect(page.locator('tr.line-expanded .add-code-comment[data-side="right"][data-idx="2"]')).toHaveCount(1);
  const form = await openCommentForm(page, 'right', 2);
  await form.getByRole('button', {name: 'Suggest change'}).click();
  await expect(form.locator('textarea[name="content"]')).toHaveValue('```suggestion\n  keep 2\n```\n'); // AC1, expanded line
});
