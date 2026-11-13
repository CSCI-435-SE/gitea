import {addDelegatedEventListener} from '../utils/dom.ts';
import {showErrorToast} from '../modules/toast.ts';

let anchor: {tr: HTMLElement, side: string} | null = null;
let selected: HTMLElement[] = [];

function newLineNum(tr: Element): number {
  return Number(tr.querySelector('td.lines-num-new')?.getAttribute('data-line-num') || 0);
}

/** the new-side rows from `a` to `b` in either order, or null when they aren't in one hunk */
export function collectRangeRows(a: HTMLElement, b: HTMLElement): HTMLElement[] | null {
  const [first, last] = a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_PRECEDING ? [b, a] : [a, b];
  const rows: HTMLElement[] = [];
  for (let tr: Element | null = first; tr; tr = tr.nextElementSibling) {
    // a hunk header, or expanded context which belongs to no hunk
    if (tr.classList.contains('tag-code') || tr.classList.contains('line-expanded')) return null;
    if (newLineNum(tr)) rows.push(tr as HTMLElement); // skips removed lines and comment rows
    if (tr === last) return rows;
  }
  return null; // `last` isn't after `first` in this file
}

function setSelected(rows: HTMLElement[]) {
  for (const tr of selected) tr.classList.remove('range-selected');
  selected = rows;
  for (const tr of selected) tr.classList.add('range-selected');
}

/** the selected range, when `tr` is its last row and it has more than one line */
export function getSelectedRange(tr: Element): {rows: HTMLElement[], start: number, end: number} | null {
  if (selected.length < 2 || selected.at(-1) !== tr) return null;
  return {rows: selected, start: newLineNum(selected[0]), end: newLineNum(tr)};
}

function selectLine(td: HTMLElement, extend: boolean) {
  const tr = td.closest('tr')!;
  const side = td.classList.contains('lines-num-new') ? 'right' : 'left';
  if (!extend || !anchor) {
    anchor = {tr, side};
    setSelected(side === 'right' ? [tr] : []);
    return;
  }
  window.getSelection()?.removeAllRanges();
  const rows = anchor.side === 'right' && side === 'right' ? collectRangeRows(anchor.tr, tr) : null;
  if (!rows) {
    anchor = null;
    setSelected([]);
    showErrorToast(td.closest('[data-range-invalid]')!.getAttribute('data-range-invalid')!);
    return;
  }
  setSelected(rows);
}

// click a line number to start a range, shift-click another to end it; Enter / Shift+Enter do the same from the keyboard
export function initDiffRangeSelect() {
  const selector = '[data-range-invalid] td.lines-num[data-line-num]:not([data-line-num=""])';
  addDelegatedEventListener(document, 'click', selector, (td, e: MouseEvent) => selectLine(td, e.shiftKey));
  addDelegatedEventListener(document, 'keydown', selector, (td, e: KeyboardEvent) => {
    if (e.key !== 'Enter') return;
    e.preventDefault();
    selectLine(td, e.shiftKey);
  });
}
