import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, apiDeleteRepo, apiHeaders, baseUrl, randomString} from './utils.ts';

test('issue labels can be selected and deselected by keyboard', async ({page}) => {
  const repoName = `e2e-label-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName, autoInit: false})]);
  try {
    await Promise.all([
      ...['bug', 'docs'].map((name) => page.request.post(`${baseUrl()}/api/v1/repos/${user}/${repoName}/labels`, {
        headers: apiHeaders(), data: {name, color: '#ee0701'},
      })),
      apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'Label keyboard test'}),
    ]);
    await page.goto(`/${user}/${repoName}/issues/1`);
    const dropdown = page.locator('.issue-sidebar-combo:has(input[name="label_ids"]) > .ui.dropdown');
    const bug = dropdown.locator('.scrolling.menu > .item', {hasText: 'bug'});
    const liveRegion = page.locator('body > [role="status"]');

    await dropdown.focus(); // the patch opens the menu on focus
    await expect(bug).toHaveAttribute('role', 'option');
    await expect(bug).toHaveAttribute('aria-selected', 'false');
    // arrow down from the highlighted item (or from before the first one) to "bug"
    const presses = await dropdown.evaluate((el) => {
      const items = Array.from(el.querySelectorAll('.scrolling.menu > .item:not(.filtered, .tw-hidden)'));
      return items.findIndex((item) => item.textContent.includes('bug')) - items.findIndex((item) => item.classList.contains('selected'));
    });
    for (let i = 0; i < presses; i++) await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(bug).toHaveAttribute('aria-selected', 'true');
    await expect(liveRegion).toHaveText('Selected "bug"');
    await expect(dropdown).toBeFocused(); // the menu stays open for the next pick

    await page.keyboard.press('Enter');
    await expect(bug).toHaveAttribute('aria-selected', 'false');
    await expect(liveRegion).toHaveText('Deselected "bug"');

    await page.keyboard.press('Enter');
    await page.keyboard.press('Escape'); // closing the menu saves the selection
    await expect(page.locator('.issue-sidebar-combo:has(input[name="label_ids"]) .ui.list')).toContainText('bug');
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
});

test('repo topics can be added and removed by keyboard', async ({page}) => {
  const repoName = `e2e-topic-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);
  try {
    await page.goto(`/${user}/${repoName}`);
    await page.locator('#manage_topic').click(); // focuses the topic search input
    const liveRegion = page.locator('body > [role="status"]');
    for (const topic of ['alpha', 'beta']) {
      await page.keyboard.type(topic);
      await expect(page.locator('#topic_edit .menu > .item', {hasText: topic}).first()).toBeVisible();
      await page.keyboard.press('Enter');
      await expect(liveRegion).toHaveText(`Selected "${topic}"`);
      await expect(page.locator('#topic_edit input.search')).toBeFocused(); // Enter must not also click "Save"
    }
    await expect(page.locator('#topic_edit .ui.label .delete.icon').first()).toHaveAttribute('aria-label', /alpha/);
    await page.keyboard.press('Backspace'); // in an empty search, Backspace removes the last label
    await expect(liveRegion).toHaveText('Deselected "beta"');
    await page.locator('#save_topic').click();
    await expect(page.locator('#repo-topics')).toContainText('alpha');
    await expect(page.locator('#repo-topics')).not.toContainText('beta');
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
});
