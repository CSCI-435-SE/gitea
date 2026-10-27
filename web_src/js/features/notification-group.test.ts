import {initNotificationGroups} from './notification-group.ts';
import {createElementFromHTML} from '../utils/dom.ts';

// the grouped markup notification_div.tmpl renders; the collapsed state is module-wide, so each test uses its own repo IDs
function createGroupedTable(repoIDs: string[]) {
  const table = createElementFromHTML<HTMLElement>(`
    <div id="notification_table">
      ${repoIDs.map((id) => `
        <div class="notification-group" data-repo-id="${id}">
          <div class="notification-group-header">
            <button type="button" class="notification-group-toggle" aria-expanded="true" aria-controls="rows-${id}"></button>
            <a href="/owner/repo${id}">owner/repo${id}</a>
            <span class="ui label">1 unread</span>
          </div>
          <div class="notification-group-rows" id="rows-${id}"><div class="notifications-item"></div></div>
        </div>`).join('')}
    </div>`);
  initNotificationGroups(table);
  const group = (id: string) => {
    const el = table.querySelector<HTMLElement>(`.notification-group[data-repo-id="${CSS.escape(id)}"]`)!;
    return {
      toggle: el.querySelector<HTMLButtonElement>('.notification-group-toggle')!,
      link: el.querySelector<HTMLAnchorElement>('a')!,
      label: el.querySelector<HTMLElement>('.ui.label')!,
      rows: el.querySelector<HTMLElement>('.notification-group-rows')!,
    };
  };
  return {group};
}

test('#117 AC5 the header button collapses and expands its section', () => {
  const {group} = createGroupedTable(['1', '2']);
  group('1').toggle.click(); // Enter and Space on a button also fire click
  expect(group('1').toggle.getAttribute('aria-expanded')).toBe('false');
  expect(group('1').rows.classList.contains('tw-hidden')).toBe(true);
  expect(group('2').toggle.getAttribute('aria-expanded')).toBe('true'); // other sections are untouched

  group('1').label.click(); // the rest of the header toggles too
  expect(group('1').toggle.getAttribute('aria-expanded')).toBe('true');
  expect(group('1').rows.classList.contains('tw-hidden')).toBe(false);
});

test('#117 the repository link in the header does not toggle the section', () => {
  const {group} = createGroupedTable(['3']);
  group('3').link.addEventListener('click', (e) => e.preventDefault()); // happy-dom would try to navigate
  group('3').link.click();
  expect(group('3').toggle.getAttribute('aria-expanded')).toBe('true');
});

test('#117 AC5 a collapsed section stays collapsed when the list is refreshed in place', () => {
  const before = createGroupedTable(['4', '5']);
  before.group('4').toggle.click();

  const after = createGroupedTable(['4', '5']); // the live refresh swaps in freshly rendered markup
  expect(after.group('4').toggle.getAttribute('aria-expanded')).toBe('false');
  expect(after.group('4').rows.classList.contains('tw-hidden')).toBe(true);
  expect(after.group('5').toggle.getAttribute('aria-expanded')).toBe('true');
});
