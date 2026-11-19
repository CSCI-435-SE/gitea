import {collectRangeRows, getSelectedRange, initDiffRangeSelect} from './repo-diff-range.ts';
import {showErrorToast} from '../modules/toast.ts';

// a unified diff: new-side lines carry data-line-num on td.lines-num-new, removed lines have it empty
function row(cls: string, newLine = '') {
  return `<tr class="${cls}"><td class="lines-num lines-num-new" data-line-num="${newLine}"></td></tr>`;
}

function rowsOf(html: string) {
  document.body.innerHTML = `<table><tbody>${html}</tbody></table>`;
  return Array.from(document.querySelectorAll<HTMLElement>('tr'));
}

const lineNums = (rows: HTMLElement[] | null) => rows?.map((tr) => tr.querySelector('td')!.getAttribute('data-line-num'));

test('collectRangeRows', () => {
  const trs = rowsOf(row('same-code', '18') + row('del-code') + row('add-code', '19') + row('add-comment') + row('same-code', '20'));
  // #158 AC1: lines 18-20, skipping the removed line and the comment row
  expect(lineNums(collectRangeRows(trs[0], trs[4]))).toEqual(['18', '19', '20']);
  // #158 AC2: an upward selection gives the same range
  expect(lineNums(collectRangeRows(trs[4], trs[0]))).toEqual(['18', '19', '20']);
  // #158 AC3: one line
  expect(lineNums(collectRangeRows(trs[2], trs[2]))).toEqual(['19']);
});

test('collectRangeRows refuses a range outside one hunk', () => {
  // #158 AC10: crossing a hunk header
  const trs = rowsOf(row('same-code', '18') + row('tag-code') + row('same-code', '40'));
  expect(collectRangeRows(trs[0], trs[2])).toBeNull();
  // #158 AC10: expanded context lines
  const expanded = rowsOf(row('same-code line-expanded', '17') + row('same-code', '18'));
  expect(collectRangeRows(expanded[0], expanded[1])).toBeNull();
});

vi.mock('../modules/toast.ts', () => ({showErrorToast: vi.fn()}));

// a unified diff with both line-number cells, inside the element that carries the refusal message
function diffRows(html: string) {
  document.body.innerHTML = `<div data-range-invalid="msg"><table><tbody>${html}</tbody></table></div>`;
  return Array.from(document.querySelectorAll<HTMLElement>('tr'));
}

function diffRow(cls: string, oldLine: string, newLine: string) {
  return `<tr class="${cls}"><td class="lines-num lines-num-old" data-line-num="${oldLine}"></td><td class="lines-num lines-num-new" data-line-num="${newLine}"></td></tr>`;
}

const cell = (side: 'old' | 'new', line: number) => document.querySelector<HTMLElement>(`td.lines-num-${side}[data-line-num="${CSS.escape(String(line))}"]`)!;
const click = (el: HTMLElement, shiftKey = false) => el.dispatchEvent(new MouseEvent('click', {bubbles: true, shiftKey}));
const press = (el: HTMLElement, key: string, shiftKey = false) => el.dispatchEvent(new KeyboardEvent('keydown', {bubbles: true, key, shiftKey}));
const selectedLines = () => Array.from(document.querySelectorAll('tr.range-selected'), (tr) => tr.querySelector('.lines-num-new')!.getAttribute('data-line-num'));

initDiffRangeSelect();

const hunk = diffRow('same-code', '18', '18') + diffRow('del-code', '19', '') + diffRow('add-code', '', '19') + diffRow('same-code', '20', '20');

test('click then shift-click selects a range', () => {
  const trs = diffRows(hunk);
  click(cell('new', 18));
  expect(selectedLines()).toEqual(['18']);
  click(cell('new', 20), true);
  // #158 AC1: lines 18-20, the removed line is not part of the range
  expect(selectedLines()).toEqual(['18', '19', '20']);
  expect(getSelectedRange(trs[3])).toEqual({rows: [trs[0], trs[2], trs[3]], start: 18, end: 20});
  expect(getSelectedRange(trs[2])).toBeNull(); // the form opens on the range's last line only
});

test('Enter and Shift+Enter select a range from the keyboard', () => {
  diffRows(hunk);
  press(cell('new', 19), 'Enter');
  press(cell('new', 20), 'a', true); // other keys do nothing
  expect(selectedLines()).toEqual(['19']);
  press(cell('new', 20), 'Enter', true);
  // #158 AC14
  expect(selectedLines()).toEqual(['19', '20']);
});

test('one line is not a range', () => {
  const trs = diffRows(hunk);
  click(cell('new', 18));
  // #158 AC3
  expect(getSelectedRange(trs[0])).toBeNull();
});

test('a range across sides or hunks is refused', () => {
  diffRows(hunk + diffRow('tag-code', '', '') + diffRow('same-code', '40', '40'));
  // #158 AC11: old side to new side
  click(cell('old', 18));
  expect(selectedLines()).toEqual([]);
  click(cell('new', 20), true);
  expect(showErrorToast).toHaveBeenLastCalledWith('msg');
  expect(selectedLines()).toEqual([]);

  // #158 AC10: across a hunk header
  vi.mocked(showErrorToast).mockClear();
  click(cell('new', 18));
  click(cell('new', 40), true);
  expect(showErrorToast).toHaveBeenCalledWith('msg');
  expect(selectedLines()).toEqual([]);

  // the refused range drops its start, so a later shift-click starts a new one
  click(cell('new', 20), true);
  expect(selectedLines()).toEqual(['20']);
});
