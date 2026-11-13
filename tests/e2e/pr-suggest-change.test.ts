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

  // selected text is replaced by the block, which starts on a line of its own; the text before is kept (AC2)
  const splitTextarea = form.locator('textarea[name="content"]');
  await splitTextarea.fill('nit: rename this');
  await splitTextarea.press('End');
  for (const key of Array.from('rename this', () => 'Shift+ArrowLeft')) await splitTextarea.press(key);
  await form.getByRole('button', {name: 'Suggest change'}).click();
  await expect(splitTextarea).toHaveValue('nit: \n```suggestion\n\n```\n');

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

// #158: lines 2-4 change, so new-side 2-4 sit in one hunk after old-side 2-4
test('suggest a change across several lines', async ({page, request}) => {
  const repo = `e2e-suggest-range-${randomString(8)}`;
  await apiCreateRepo(request, {name: repo});
  await apiCreateFile(request, owner, repo, 'main.go', 'a\nb\nc\nd\ne\n');
  expect((await request.post(`${baseUrl()}/api/v1/repos/${owner}/${repo}/contents`, {
    headers: apiHeaders(),
    data: {new_branch: 'feat', files: [{operation: 'upload', path: 'main.go', content: base64('a\n\tB\nC\nD\ne\n')}]},
  })).ok()).toBe(true);
  const index = await apiCreatePR(request, owner, repo, 'feat', 'main', 'range');
  await login(page);
  await page.goto(`/${owner}/${repo}/pulls/${index}/files?style=unified`);
  const lineNum = (side: 'old' | 'new', line: number) => page.locator(`td.lines-num-${side}[data-line-num="${line}"]`).first();

  const message = 'A comment can only cover lines on one side of a single hunk.';

  // AC14: from the keyboard, Enter on line 2 and Shift+Enter on line 3 select both
  await lineNum('new', 2).focus();
  await page.keyboard.press('Enter');
  await lineNum('new', 3).focus();
  await page.keyboard.press('Shift+Enter');
  await expect(page.locator('tr.range-selected')).toHaveCount(2);

  // AC2: an upward selection, 4 then shift-click 2, gives lines 2-4 (AC1)
  await lineNum('new', 4).click();
  await lineNum('new', 2).click({modifiers: ['Shift']});
  const form = await openCommentForm(page, 'right', 4);
  await expect(form.locator('.code-comment-range')).toHaveText('Lines 2–4');

  // AC4: the block holds every line of the range, indentation kept
  const textarea = form.locator('textarea[name="content"]');
  await form.getByRole('button', {name: 'Suggest change'}).click();
  await expect(textarea).toHaveValue('```suggestion\n\tB\nC\nD\n```\n');

  // AC15: a range the server refuses shows its message and keeps the form
  const startLine = form.locator('input[name="start_line"]');
  await startLine.evaluate((el: HTMLInputElement) => { el.value = '9' });
  await form.getByRole('button', {name: 'Add single comment'}).click();
  await expect(page.getByText(message)).toBeVisible();
  await startLine.evaluate((el: HTMLInputElement) => { el.value = '2' });

  // AC1, AC5: the posted thread is headed with the range and shows 3 removed lines and 2 added
  await textarea.fill('```suggestion\nX\nY\n```');
  await form.getByRole('button', {name: 'Add single comment'}).click();
  const thread = page.locator('.conversation-holder');
  await expect(thread.locator('.code-comment-range')).toHaveText('Lines 2–4');
  await expect(thread.locator('.markup-suggestion-line.removed code')).toHaveText(['\tB', 'C', 'D']);
  await expect(thread.locator('.markup-suggestion-line.added code')).toHaveText(['X', 'Y']);

  // AC11: old side to new side is refused with a message
  await lineNum('old', 2).click();
  await lineNum('new', 4).click({modifiers: ['Shift']});
  await expect(page.getByText(message).last()).toBeVisible();
});
