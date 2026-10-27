import {initRepoIssueQuoteReplyShortcut, selectedCommentMarkup} from './repo-issue-edit.ts';
import {getShortcuts, initGlobalShortcut} from '../modules/shortcut.ts';
import {createElementFromHTML} from '../utils/dom.ts';

// the real editor needs a whole page, so a stand-in is attached to the editor element instead
vi.mock('./comp/ComboMarkdownEditor.ts', () => ({
  getComboMarkdownEditor: (el: any) => el._giteaComboMarkdownEditor,
}));

initGlobalShortcut(); // listens for the key presses

const timeline = `<div>
  <div class="comment" id="first"><div class="render-content markup"><p>first line</p><p>second line</p></div><div class="raw-content tw-hidden">raw</div></div>
  <div class="comment" id="second"><div class="render-content markup"><p>other comment</p></div><div class="raw-content tw-hidden">raw</div></div>
  <div class="comment" id="dismissed"><div class="render-content markup"><p>dismissed review reason</p></div></div>
  <p id="outside">outside every comment</p>
</div>`;

function select(start: Node, startOffset: number, end: Node, endOffset: number) {
  const range = document.createRange();
  range.setStart(start, startOffset);
  range.setEnd(end, endOffset);
  const selection = window.getSelection()!;
  selection.removeAllRanges();
  selection.addRange(range);
  return selection;
}

function text(root: HTMLElement, selector: string) {
  return root.querySelector(selector)!.firstChild!;
}

test('selectedCommentMarkup', () => {
  const root = createElementFromHTML<HTMLElement>(timeline);
  document.body.append(root);
  const first = root.querySelector('#first .markup');

  expect(selectedCommentMarkup(select(text(root, '#first p'), 0, text(root, '#first p'), 5))).toBe(first);
  expect(selectedCommentMarkup(select(text(root, '#first p'), 0, text(root, '#first p:last-child'), 6))).toBe(first); // across paragraphs
  expect(selectedCommentMarkup(select(text(root, '#first p'), 2, text(root, '#first p'), 2))).toBeNull(); // collapsed
  expect(selectedCommentMarkup(select(text(root, '#first p'), 0, text(root, '#second p'), 5))).toBeNull(); // two comments
  expect(selectedCommentMarkup(select(text(root, '#outside'), 0, text(root, '#outside'), 7))).toBeNull();
  expect(selectedCommentMarkup(select(text(root, '#dismissed p'), 0, text(root, '#dismissed p'), 9))).toBeNull(); // no raw content to quote
  window.getSelection()!.removeAllRanges();
  expect(selectedCommentMarkup(window.getSelection())).toBeNull();
  root.remove();
});

test('r quotes the selection after the existing text, or only moves to the reply box', () => {
  const root = createElementFromHTML<HTMLElement>(timeline);
  const form = createElementFromHTML<HTMLElement>(`<form id="comment-form" data-shortcut-keys="r" data-shortcut-description="Quote Reply" data-shortcut-section="Conversation">
    <div class="combo-markdown-editor"></div>
  </form>`);
  document.body.append(root, form);
  let value = 'earlier text';
  const editor = {
    textarea: undefined as HTMLTextAreaElement | undefined, // still loading
    tabEditor: createElementFromHTML('<a class="item"></a>'), // the Preview tab is active
    value: (v?: string) => v === undefined ? value : (value = v),
    switchTabToEditor: vi.fn(() => editor.tabEditor.classList.add('active')),
    focus: vi.fn(),
    moveCursorToEnd: vi.fn(),
  };
  Object.assign(form.querySelector('.combo-markdown-editor')!, {_giteaComboMarkdownEditor: editor});
  initRepoIssueQuoteReplyShortcut(form);
  expect(getShortcuts()).toContainEqual({keys: 'r', description: 'Quote Reply', section: 'Conversation'});

  select(text(root, '#first p'), 0, text(root, '#first p:last-child'), 6);
  document.body.dispatchEvent(new KeyboardEvent('keydown', {key: 'r', bubbles: true}));
  expect(value).toBe('earlier text'); // ignored until the editor has loaded
  expect(editor.focus).not.toHaveBeenCalled();

  editor.textarea = document.createElement('textarea');
  document.body.dispatchEvent(new KeyboardEvent('keydown', {key: 'r', bubbles: true}));
  expect(value).toMatch(/^earlier text\n\n> first line\n(> \n)?> second\n/);
  const quoted = value;
  expect(editor.switchTabToEditor).toHaveBeenCalledTimes(1);
  expect(editor.moveCursorToEnd).toHaveBeenCalledTimes(1);

  window.getSelection()!.removeAllRanges();
  document.body.dispatchEvent(new KeyboardEvent('keydown', {key: 'r', bubbles: true}));
  expect(value).toBe(quoted);
  expect(editor.switchTabToEditor).toHaveBeenCalledTimes(1); // already on the Write tab
  expect(editor.moveCursorToEnd).toHaveBeenCalledTimes(2);

  root.remove();
  form.remove();
  expect(getShortcuts()).not.toContainEqual(expect.objectContaining({keys: 'r'}));
});
