import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import type {Page} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, apiDeleteRepo, apiHeaders, assertNoJsError, baseUrl, randomString} from './utils.ts';

// these selectors follow Fomantic's dropdown markup; update them if #25 moves the label picker off Fomantic
const labelPicker = (page: Page) => page.locator('.issue-sidebar-combo:has(input[name="label_ids"]) > .ui.dropdown');

async function withLabeledIssue(page: Page, labels: string[], run: () => Promise<void>) {
  const repoName = `e2e-label-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName, autoInit: false})]);
  try {
    await Promise.all([
      ...labels.map((name) => page.request.post(`${baseUrl()}/api/v1/repos/${user}/${repoName}/labels`, {
        headers: apiHeaders(), data: {name, color: '#ee0701'},
      })),
      apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'Label keyboard test'}),
    ]);
    await page.goto(`/${user}/${repoName}/issues/1`);
    await run();
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
}

test('issue labels can be selected and deselected by keyboard', async ({page}) => {
  await withLabeledIssue(page, ['bug', 'docs'], async () => {
    const dropdown = labelPicker(page);
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
  });
});

// separate from the test above because a label change replaces the whole sidebar, which drops focus
test('issue label picker exposes its combobox state', async ({page}) => {
  await page.clock.install();
  await withLabeledIssue(page, ['bug'], async () => {
    const dropdown = labelPicker(page);
    const menu = dropdown.locator('> .menu');

    await expect(dropdown).toHaveAttribute('role', 'combobox');
    await expect(dropdown).toHaveAttribute('aria-expanded', 'false'); // before focus, which opens the menu
    await expect(menu).toHaveAttribute('id', (await dropdown.getAttribute('aria-controls'))!);

    await dropdown.focus();
    await expect(menu).toBeVisible();
    await expect(dropdown).toHaveAttribute('aria-expanded', 'true');

    await page.keyboard.press('ArrowDown');
    await expect(dropdown).toHaveAttribute('aria-activedescendant', /.+/);
    const activeId = (await dropdown.getAttribute('aria-activedescendant'))!;
    await expect(menu.locator(`[id="${activeId}"]`)).toHaveAttribute('role', 'option');

    await page.clock.runFor(200); // the same delayed Escape as the test below, so its setup is checked here
    await page.keyboard.press('Escape');
    await expect(menu).toBeHidden();
    await expect(dropdown).toBeFocused();
    await assertNoJsError(page);
  });
});

test('issue label picker reports collapsed after Escape', async ({page}) => {
  test.fail(); // bug: https://github.com/CSCI-435-SE/gitea/issues/104
  await page.clock.install();
  await withLabeledIssue(page, ['bug'], async () => {
    const dropdown = labelPicker(page);
    await dropdown.focus();
    await expect(dropdown).toHaveAttribute('aria-expanded', 'true');
    await page.clock.runFor(200); // past the refresh that focus schedules, which otherwise hides the bug
    await page.keyboard.press('Escape');
    await expect(dropdown).toHaveAttribute('aria-expanded', 'false', {timeout: 1000}); // keeps the expected failure under 2s
  });
});

test('repo topics can be added and removed by keyboard', async ({page}) => {
  const repoName = `e2e-topic-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  // the topic editor needs a non-empty repo; "alpha" is seeded so typing only one topic keeps the test short
  await Promise.all([login(page), (async () => {
    await apiCreateRepo(page.request, {name: repoName});
    await page.request.put(`${baseUrl()}/api/v1/repos/${user}/${repoName}/topics`, {headers: apiHeaders(), data: {topics: ['alpha']}});
  })()]);
  try {
    await page.goto(`/${user}/${repoName}`);
    await page.locator('#manage_topic').click(); // focuses the topic search input
    const liveRegion = page.locator('body > [role="status"]');
    await expect(page.locator('#topic_edit .ui.label .delete.icon')).toHaveAttribute('aria-label', /alpha/);

    await page.keyboard.type('beta');
    await expect(page.locator('#topic_edit .menu > .item', {hasText: 'beta'}).first()).toBeVisible();
    await page.keyboard.press('Enter');
    await expect(liveRegion).toHaveText('Selected "beta"'); // the existing "alpha" is not announced
    await expect(page.locator('#topic_edit input.search')).toBeFocused(); // Enter must not also click "Save"

    await page.keyboard.press('Backspace'); // in an empty search, Backspace removes the last label
    await expect(liveRegion).toHaveText('Deselected "beta"');
    await page.locator('#save_topic').click();
    await expect(page.locator('#repo-topics')).toContainText('alpha');
    await expect(page.locator('#repo-topics')).not.toContainText('beta');
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
});
