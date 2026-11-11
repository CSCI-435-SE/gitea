import {replaceTextareaSelection} from './comp/EditorMarkdown.ts';
import {showElem} from '../utils/dom.ts';
import type {ComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';

/** the backtick fence for a suggestion holding `line`: longer than any backtick run in it, at least 3 */
export function suggestionFence(line: string): string {
  let longest = 0;
  for (const run of line.match(/`+/g) ?? []) longest = Math.max(longest, run.length);
  return '`'.repeat(Math.max(3, longest + 1));
}

/** the text to insert after `textBefore` (the textarea text before the selection),
 *  and the offset in it where the cursor goes: the end of `line` inside the block */
export function buildSuggestionInsert(textBefore: string, line: string): {text: string, cursor: number} {
  const fence = suggestionFence(line);
  const prefix = textBefore && !textBefore.endsWith('\n') ? '\n' : ''; // the block starts on a line of its own
  const head = `${prefix}${fence}suggestion\n${line}`;
  return {text: `${head}\n${fence}\n`, cursor: head.length};
}

/** where to insert a block for a cursor at `pos`: after the fenced code block holding `pos`, if any, so a
 *  block is never nested in another (CommonMark: a closing fence uses the opening's character, is at least
 *  as long and has no info string; an unclosed block runs to the end) */
export function suggestionInsertPos(value: string, pos: number): number {
  let open: {char: string, len: number} | null = null;
  let lineStart = 0;
  while (true) {
    const newline = value.indexOf('\n', lineStart);
    const lineEnd = newline === -1 ? value.length : newline;
    const fence = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(value.slice(lineStart, lineEnd));
    if (!open) {
      if (pos <= lineStart) return pos;
      if (fence && !(fence[1][0] === '`' && fence[2].includes('`'))) {
        open = {char: fence[1][0], len: fence[1].length};
      } else if (pos <= lineEnd) {
        return pos;
      }
    } else if (fence?.[1][0] === open.char && fence[1].length >= open.len && !fence[2].trim()) {
      if (pos <= lineEnd) return newline === -1 ? value.length : newline + 1;
      open = null;
    }
    if (newline === -1) return open ? value.length : pos;
    lineStart = newline + 1;
  }
}

/** the range a block replaces for a selection from `start` to `end`: the selection itself (AC2), or, when
 *  either end is inside a fenced block, an empty range after that block, so no block is cut into */
export function suggestionInsertRange(value: string, start: number, end: number): [number, number] {
  const afterEnd = suggestionInsertPos(value, end);
  if (afterEnd !== end) return [afterEnd, afterEnd];
  const afterStart = suggestionInsertPos(value, start);
  if (afterStart !== start) return [afterStart, afterStart];
  return [start, end];
}

/** show the "Suggest change" button of a new code comment's editor, inserting a block with the text of `codeLine` */
export function initSuggestChangeButton(editor: ComboMarkdownEditor, codeLine: HTMLElement) {
  const button = editor.container.querySelector<HTMLElement>('.markdown-button-suggest-change');
  if (!button) return;
  showElem(button.closest('.markdown-toolbar-group')!);
  button.addEventListener('click', (e) => {
    e.preventDefault();
    const textarea = editor.textarea;
    const [start, end] = suggestionInsertRange(textarea.value, textarea.selectionStart, textarea.selectionEnd);
    textarea.setSelectionRange(start, end);
    // textContent drops the highlighting spans and keeps leading whitespace; the diff cell also holds the line's own newline
    const line = codeLine.textContent.replace(/[\r\n]+$/, '');
    const {text, cursor} = buildSuggestionInsert(textarea.value.slice(0, start), line);
    replaceTextareaSelection(textarea, text);
    textarea.setSelectionRange(start + cursor, start + cursor);
  });
}
