import {test, expect} from '@playwright/test';
import {assertNoJsError} from './utils.ts';

// covers what happy-dom can't: native button activation on Space keyup and the browser's own Tab move
test('aria-menu is operable by keyboard', async ({page}) => {
  await page.goto('/devtest/aria-menu');
  const trigger = page.getByRole('button', {name: 'Text items'});
  const menu = page.getByRole('menu').filter({has: page.getByRole('menuitem', {name: 'Apple'})});
  const output = page.locator('#devtest-aria-menu-output');

  await trigger.focus();
  await page.keyboard.press('Enter');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', {name: 'Apple'})).toBeFocused();

  await page.keyboard.press('ArrowUp');
  await expect(page.getByRole('menuitem', {name: 'Grape'})).toBeFocused();
  await page.keyboard.press('b');
  await expect(page.getByRole('menuitem', {name: 'Banana'})).toBeFocused();

  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();

  // Space on a native button must open the menu and not re-toggle it shut on keyup
  await page.keyboard.press(' ');
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', {name: 'Apple'})).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press(' ');
  await expect(output).toHaveText('Banana');
  await expect(trigger).toHaveAttribute('aria-expanded', 'false');
  await expect(trigger).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Tab');
  await expect(menu).toBeHidden();
  await expect(trigger).not.toBeFocused(); // the browser's Tab moved on from the trigger

  await assertNoJsError(page);
});

test('aria-menu Tab moves to the next button', async ({page, browserName}) => {
  test.skip(browserName === 'webkit', 'macOS WebKit, like Safari, skips buttons on Tab by default'); // eslint-disable-line playwright/no-skipped-test
  await page.goto('/devtest/aria-menu');
  await page.getByRole('button', {name: 'Text items'}).focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('Tab');
  await expect(page.getByRole('button', {name: 'Items with icons'})).toBeFocused();
});

test('aria-menu is operable by mouse', async ({page}) => {
  await page.goto('/devtest/aria-menu');
  const trigger = page.getByRole('button', {name: 'With a divider and header'});
  await trigger.click();
  await page.getByRole('menuitem', {name: 'Stars'}).click();
  await expect(page.locator('#devtest-aria-menu-output')).toHaveText('Stars');
  await expect(trigger).toHaveAttribute('aria-expanded', 'false');

  await trigger.click();
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await page.getByRole('heading', {name: 'aria-menu'}).click();
  await expect(trigger).toHaveAttribute('aria-expanded', 'false');
  await assertNoJsError(page);
});
