import {collectRangeRows} from './repo-diff-range.ts';

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
