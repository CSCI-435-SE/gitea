import {initNotificationBulkActions, loadSelection, saveSelection, selectedCount, setSelected} from './notification-bulk.ts';
import {createElementFromHTML} from '../utils/dom.ts';

// the markup notification_div.tmpl renders for a page of a view
function createNotificationPage(view: string, viewCount: number, ids: string[]) {
  const root = createElementFromHTML<HTMLElement>(`
    <div id="notification_div" data-view-key="${view}" data-view-count="${viewCount}" data-bulk-url="/notifications/bulk">
      <div class="notification-bulk-bar">
        <input type="checkbox" class="notification-checkbox-all">
        <div class="notification-bulk-actions tw-hidden">
          <span><span class="notification-selected-count">0</span> selected</span>
          <a class="notification-select-all-view tw-hidden" href="#">Select all</a>
          <button class="notification-bulk-action" data-action="mark_as_read">Mark as read</button>
        </div>
      </div>
      <div id="notification_table">
        ${ids.map((id) => `<div class="notifications-item"><input type="checkbox" class="notification-checkbox" data-notification-id="${id}"></div>`).join('')}
      </div>
    </div>`);
  initNotificationBulkActions(root);
  const q = <T extends HTMLElement>(selector: string) => root.querySelector<T>(selector)!;
  const row = (id: string) => q<HTMLInputElement>(`.notification-checkbox[data-notification-id="${id}"]`);
  const tick = (el: HTMLInputElement, checked: boolean) => {
    el.checked = checked;
    el.dispatchEvent(new Event('change'));
  };
  return {
    row, tick,
    selectPage: q<HTMLInputElement>('.notification-checkbox-all'),
    actions: q('.notification-bulk-actions'),
    count: () => q('.notification-selected-count').textContent,
    selectAllView: q('.notification-select-all-view'),
  };
}

// tests run concurrently and share session storage, so each one uses its own view, which drops any other test's selection

test('a selection is kept for the same view and dropped for another', () => {
  saveSelection({view: '/n?t=keep', ids: ['1', '2'], all: false});
  expect(loadSelection('/n?t=keep').ids).toEqual(['1', '2']); // another page of the view
  expect(loadSelection('/n?t=keep-read')).toEqual({view: '/n?t=keep-read', ids: [], all: false});
  expect(loadSelection('/n?t=keep').ids).toEqual([]); // switching view cleared it
});

test('an empty selection is not stored', () => {
  saveSelection({view: '/n?t=empty', ids: ['1'], all: false});
  saveSelection({view: '/n?t=empty', ids: [], all: false});
  expect(loadSelection('/n?t=empty').ids).toEqual([]);
});

test('unusable session storage gives an empty selection', () => {
  vi.stubGlobal('sessionStorage', {getItem: () => { throw new Error('blocked') }, removeItem: () => { throw new Error('blocked') }});
  const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  expect(loadSelection('/n?t=blocked')).toEqual({view: '/n?t=blocked', ids: [], all: false});
  errorSpy.mockRestore();
  vi.unstubAllGlobals();
});

test('setSelected adds, removes and leaves "all in view" mode', () => {
  let sel = setSelected({view: 'v', ids: ['1'], all: true}, ['2', '3'], true);
  expect(sel).toEqual({view: 'v', ids: ['1', '2', '3'], all: false});
  sel = setSelected(sel, ['1', '3'], false);
  expect(sel.ids).toEqual(['2']);
  expect(selectedCount(sel, 50)).toEqual(1);
  expect(selectedCount({...sel, all: true}, 50)).toEqual(50);
});

test('the actions show only while something is selected', () => {
  const page = createNotificationPage('/n?t=actions', 2, ['1', '2']);
  expect(page.actions.classList.contains('tw-hidden')).toBe(true);
  page.tick(page.row('1'), true);
  expect(page.actions.classList.contains('tw-hidden')).toBe(false);
  expect(page.count()).toEqual('1');
  expect(page.selectPage.indeterminate).toBe(true);
  page.tick(page.row('1'), false);
  expect(page.actions.classList.contains('tw-hidden')).toBe(true);
});

test('selections on another page of the view are counted and restored', () => {
  const page1 = createNotificationPage('/n?t=pages', 4, ['1', '2']);
  page1.tick(page1.row('1'), true);
  const page2 = createNotificationPage('/n?t=pages', 4, ['3', '4']);
  page2.tick(page2.row('3'), true);
  expect(page2.count()).toEqual('2');
  const back = createNotificationPage('/n?t=pages', 4, ['1', '2']);
  expect(back.row('1').checked).toBe(true);
  expect(back.row('2').checked).toBe(false);
});

test('selecting the page offers every notification in the view', () => {
  const page = createNotificationPage('/n?t=all', 30, ['1', '2']);
  page.tick(page.selectPage, true);
  expect(page.count()).toEqual('2');
  expect(page.selectAllView.classList.contains('tw-hidden')).toBe(false);
  page.selectAllView.click();
  expect(page.count()).toEqual('30');
  expect(page.selectAllView.classList.contains('tw-hidden')).toBe(true);
  page.tick(page.row('2'), false); // unticking one falls back to the rest of the page
  expect(page.count()).toEqual('1');
  page.tick(page.selectPage, false);
  expect(page.actions.classList.contains('tw-hidden')).toBe(true);
});
