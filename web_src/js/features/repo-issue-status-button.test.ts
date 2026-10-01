import {initRepoIssueStatusButton} from './repo-issue-status-button.ts';
import {createElementFromHTML} from '../utils/dom.ts';
import {EventEditorContentChanged} from './comp/EditorMarkdown.ts';
import {EventUploadStateChanged} from './comp/EditorUpload.ts';

// the real editor needs a whole page, so a stand-in is attached to the editor element instead
vi.mock('./comp/ComboMarkdownEditor.ts', async () => {
  const {EventEditorContentChanged} = await import('./comp/EditorMarkdown.ts');
  const {EventUploadStateChanged} = await import('./comp/EditorUpload.ts');
  return {
    ComboMarkdownEditor: {EventEditorContentChanged, EventUploadStateChanged},
    getComboMarkdownEditor: (el: any) => el._giteaComboMarkdownEditor,
  };
});
// a fixed stand-in for the # list's search: the answers depend only on the query, since the tests run concurrently
vi.mock('../utils/match.ts', () => ({
  matchIssue: async (_owner: string, _repo: string, _index: string, query: string) => {
    if (query === '404') throw new Error('network down');
    if (query === '405') { // fails late, after the box has changed
      await new Promise((resolve) => setTimeout(resolve, 50));
      throw new Error('network down');
    }
    if (query === '7') await new Promise((resolve) => setTimeout(resolve, 50)); // answers late, after the box has changed
    const issues = [
      {number: 12, title: 'Login fails', state: 'open'},
      {number: 3, title: '<b>not bold</b>', state: 'closed'},
      {number: 7, title: 'Slow answer', state: 'open'},
    ];
    return issues.filter((i) => String(i.number).startsWith(query));
  },
}));
// the real debounce without its wait: as on the page, a lookup starts only once the one before it has answered
vi.mock('perfect-debounce', async (importOriginal) => {
  const real = await importOriginal<Record<string, any>>();
  return {debounce: (fn: any) => real.debounce(fn, 0)};
});
// the page the preview believes it is on, without changing the location that other test files share
vi.mock('../utils.ts', async (importOriginal) => ({
  ...await importOriginal<Record<string, unknown>>(),
  parseIssueHref: () => ({ownerName: 'user2', repoName: 'repo1', pathType: 'issues', indexString: '8'}),
}));

// a stand-in popup that only records whether it is shown, keyed by the element it was made for
const popups = new Map<Element, {isShown: boolean}>();
vi.mock('../modules/tippy.ts', () => ({
  createTippy: (target: Element) => {
    const popup = {isShown: false};
    popups.set(target, popup);
    return {show: () => { popup.isShown = true }, hide: () => { popup.isShown = false }};
  },
}));

// the markup an open issue's form gets; a closed one has no hidden reason and no menu
const reasonMenuHtml = `
  <div class="ui dropdown icon button">
    <div class="menu">
      <div class="item js-aria-clickable selected" data-value="completed" data-status="Close as completed" data-status-and-comment="Close as completed with comment">Completed</div>
      <div class="item js-aria-clickable" data-value="not_planned" data-status="Close as not planned" data-status-and-comment="Close as not planned with comment">Not planned</div>
      <div class="item js-aria-clickable" data-value="duplicate" data-status="Close as duplicate" data-status-and-comment="Close as duplicate with comment">Duplicate</div>
      <div class="item js-aria-clickable" data-value="other" data-status="Close with other reason" data-status-and-comment="Close with other reason and comment">Other</div>
    </div>
  </div>`;
const reasonFieldsHtml = `
  <input type="hidden" name="close_duplicate_index" data-close-reason="duplicate" disabled>
  <input type="hidden" name="close_reason_text" data-close-reason="other" disabled>`;
const reasonPopupsHtml = `
  <div class="tippy-target" data-close-reason-popup="duplicate"
    data-locale-status="Close as duplicate of #%s" data-locale-status-and-comment="Close as duplicate of #%s with comment">
    <div class="field flex-text-block">
      <input type="number" min="1" step="1">
      <span class="tw-hidden" data-close-duplicate-preview data-locale-not-found="No #%s found in this repository" data-locale-self="Can't be a duplicate of itself"></span>
    </div>
  </div>
  <div class="tippy-target" data-close-reason-popup="other"><div class="field"><input type="text" maxlength="255"></div></div>`;

function createForm(textareaValue = '', {isClosed = false} = {}) {
  const elForm = createElementFromHTML(`
    <form>
      <div class="combo-markdown-editor"><textarea class="markdown-text-editor">${textareaValue}</textarea></div>
      ${isClosed ? '' : '<input type="hidden" name="close_reason" value="completed">'}
      ${isClosed ? '' : reasonFieldsHtml}
      <div class="ui buttons">
        <button id="status-button" ${isClosed ?
          'data-status="Reopen Issue" data-status-and-comment="Reopen with Comment"' :
          'data-status="Close as completed" data-status-and-comment="Close as completed with comment"'}>
          <span class="status-button-text"></span>
        </button>
        ${isClosed ? '' : reasonMenuHtml}
      </div>
      ${isClosed ? '' : reasonPopupsHtml}
    </form>
  `);
  const elEditor = elForm.querySelector<HTMLElement>('.combo-markdown-editor')!;
  const statusButton = elForm.querySelector<HTMLButtonElement>('#status-button')!;
  // like the real editor, it is attached before its init finishes, and reading it before then throws
  const editor = {
    ready: false, content: textareaValue, uploading: false,
    value() { if (!this.ready) throw new Error('editor not initialized'); return this.content },
    isUploading() { if (!this.ready) throw new Error('editor not initialized'); return this.uploading },
  };
  (elEditor as any)._giteaComboMarkdownEditor = editor;
  return {
    statusButton,
    buttonText: () => statusButton.querySelector('.status-button-text')!.textContent,
    sentReason: () => elForm.querySelector<HTMLInputElement>('input[name="close_reason"]')!.value,
    pick: (label: string) => Array.from(elForm.querySelectorAll<HTMLElement>('.menu .item')).find((el) => el.textContent === label)!.click(),
    clickMenuGap: () => elForm.querySelector<HTMLElement>('.menu')!.click(),
    popupShown: (reason: string) => popups.get(elForm.querySelector(`[data-close-reason-popup="${CSS.escape(reason)}"]`)!)!.isShown,
    sentField: (reason: string) => { // what the form sends for a reason's popup, or null when it sends nothing
      const el = elForm.querySelector<HTMLInputElement>(`input[type="hidden"][data-close-reason="${CSS.escape(reason)}"]`)!;
      return el.disabled ? null : el.value;
    },
    typeInPopup: (reason: string, value: string) => {
      const el = elForm.querySelector<HTMLInputElement>(`[data-close-reason-popup="${CSS.escape(reason)}"] input`)!;
      el.value = value;
      el.dispatchEvent(new Event('input', {bubbles: true}));
    },
    typeTextInNumberBox: () => { // a browser lets text like "e" into a number box, then reports it as '' with "bad input"
      const el = elForm.querySelector<HTMLInputElement>('[data-close-reason-popup="duplicate"] input')!;
      el.value = '';
      Object.defineProperty(el, 'validity', {value: {badInput: true}, configurable: true});
      el.dispatchEvent(new Event('input', {bubbles: true}));
      delete (el as any).validity; // back to the real validity for the next thing typed
    },
    preview: () => {
      const el = elForm.querySelector<HTMLElement>('[data-close-duplicate-preview]')!;
      // the pieces sit side by side in the page, so read them as separate words
      return el.classList.contains('tw-hidden') ? null : Array.from(el.childNodes, (node) => node.textContent!.trim()).filter(Boolean).join(' ');
    },
    previewHtml: () => elForm.querySelector<HTMLElement>('[data-close-duplicate-preview]')!.innerHTML,
    popupFieldHasError: (reason: string) => elForm.querySelector(`[data-close-reason-popup="${CSS.escape(reason)}"] .field`)!.classList.contains('error'),
    isCloseBlocked: () => { // clicks the close button; true when that click would not send the form
      const event = new MouseEvent('click', {bubbles: true, cancelable: true});
      statusButton.dispatchEvent(event);
      return event.defaultPrevented;
    },
    pressInPopup: (reason: string, key: string, {isComposing = false} = {}) => {
      const event = new KeyboardEvent('keydown', {key, isComposing, bubbles: true, cancelable: true});
      elForm.querySelector(`[data-close-reason-popup="${CSS.escape(reason)}"] input`)!.dispatchEvent(event);
      return event;
    },
    restoreFields: (values: Record<string, string>) => { // what a browser may put back into the form on a reload
      for (const [name, value] of Object.entries(values)) {
        const el = elForm.querySelector<HTMLInputElement>(`input[name="${CSS.escape(name)}"]`)!;
        el.value = value;
        el.disabled = false;
      }
    },
    attach: () => { // focus only moves within the page
      document.body.append(elForm);
      return () => elForm.remove();
    },
    popupInput: (reason: string) => elForm.querySelector<HTMLInputElement>(`[data-close-reason-popup="${CSS.escape(reason)}"] input`)!,
    init: () => initRepoIssueStatusButton(elForm.querySelector('.ui.buttons')!),
    finishEditorInit: () => { editor.ready = true },
    typeComment: (content: string) => {
      editor.content = content;
      elEditor.dispatchEvent(new CustomEvent(EventEditorContentChanged, {bubbles: true}));
    },
    setUploading: (uploading: boolean) => {
      editor.uploading = uploading;
      elEditor.dispatchEvent(new CustomEvent(EventUploadStateChanged, {bubbles: true}));
    },
  };
}

test('text follows whether a comment is typed', () => {
  const form = createForm();
  form.init();
  expect(form.buttonText()).toBe('Close as completed');

  form.finishEditorInit();
  form.typeComment('looks done');
  expect(form.buttonText()).toBe('Close as completed with comment');
  form.typeComment('  ');
  expect(form.buttonText()).toBe('Close as completed');
});

test('disabled while files upload', () => {
  const form = createForm();
  form.init();
  form.finishEditorInit();
  form.setUploading(true);
  expect(form.statusButton.disabled).toBe(true);
  form.setUploading(false);
  expect(form.statusButton.disabled).toBe(false);
});

test('a comment restored before the editor is ready counts', () => {
  const form = createForm('restored draft');
  form.init(); // the editor is attached but not initialized yet
  expect(form.buttonText()).toBe('Close as completed with comment');
});

test('picking a reason changes the button and what is sent', () => {
  const form = createForm();
  form.init();
  form.pick('Not planned');
  expect(form.buttonText()).toBe('Close as not planned');
  expect(form.sentReason()).toBe('not_planned');

  form.clickMenuGap(); // a click in the menu that misses every item changes nothing
  expect(form.sentReason()).toBe('not_planned');
});

test('the picked reason still follows the comment, before and after picking', () => {
  const form = createForm();
  form.init();
  form.finishEditorInit();
  form.typeComment('closing this');
  form.pick('Duplicate');
  expect(form.buttonText()).toBe('Close as duplicate with comment');
  form.typeComment('');
  expect(form.buttonText()).toBe('Close as duplicate');
  form.pick('Completed');
  expect(form.buttonText()).toBe('Close as completed');
  expect(form.sentReason()).toBe('completed');
});

test('a closed item has no reason menu, and its button still follows the comment', () => {
  const form = createForm('', {isClosed: true});
  form.init();
  form.finishEditorInit();
  expect(form.buttonText()).toBe('Reopen Issue');
  form.typeComment('reopening');
  expect(form.buttonText()).toBe('Reopen with Comment');
});

test('a reason that needs more opens its popup, and only that reason\'s value is sent', async () => {
  const form = createForm();
  form.init();
  expect([form.popupShown('duplicate'), form.popupShown('other')]).toEqual([false, false]);
  expect([form.sentField('duplicate'), form.sentField('other')]).toEqual([null, null]);

  form.pick('Duplicate');
  expect([form.popupShown('duplicate'), form.popupShown('other')]).toEqual([true, false]);
  expect([form.sentField('duplicate'), form.sentField('other')]).toEqual(['', null]);

  form.typeInPopup('duplicate', '12');
  await vi.waitFor(() => expect(form.sentField('duplicate')).toBe('12')); // once the preview has found #12
  form.pick('Other');
  expect([form.popupShown('duplicate'), form.popupShown('other')]).toEqual([false, true]);
  expect([form.sentField('duplicate'), form.sentField('other')]).toEqual([null, '']); // the number is kept, but not sent

  form.pick('Duplicate');
  expect(form.sentField('duplicate')).toBe('12');
  form.pick('Not planned');
  expect([form.popupShown('duplicate'), form.popupShown('other')]).toEqual([false, false]);
  expect([form.sentField('duplicate'), form.sentField('other')]).toEqual([null, null]);
});

test('the duplicate\'s number goes into the button, and closing waits for a whole number', async () => {
  const form = createForm();
  form.init();
  form.finishEditorInit();
  form.pick('Duplicate');
  expect([form.buttonText(), form.statusButton.disabled]).toEqual(['Close as duplicate', false]); // clickable, it explains itself
  expect([form.isCloseBlocked(), form.popupFieldHasError('duplicate')]).toEqual([true, true]);

  form.typeInPopup('duplicate', '12');
  expect(form.popupFieldHasError('duplicate')).toBe(false); // typing answers the error
  await vi.waitFor(() => expect(form.sentField('duplicate')).toBe('12'));
  expect([form.buttonText(), form.isCloseBlocked()]).toEqual(['Close as duplicate of #12', false]);
  form.typeComment('same as #12');
  expect(form.buttonText()).toBe('Close as duplicate of #12 with comment');

  for (const notWhole of ['1.5', '0', '-3', '']) {
    form.typeInPopup('duplicate', notWhole);
    expect([form.buttonText(), form.sentField('duplicate')]).toEqual(['Close as duplicate with comment', '']);
    expect([form.isCloseBlocked(), form.popupFieldHasError('duplicate')]).toEqual([true, true]);
    expect(form.preview()).toBe(notWhole ? `No #${notWhole} found in this repository` : null); // an empty box is only marked
  }
});

test('text in the number box, or a number that is not whole, is marked like a number with no issue', () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '0');
  expect([form.preview(), form.popupFieldHasError('duplicate')]).toEqual(['No #0 found in this repository', true]);
  form.typeTextInNumberBox(); // letters read as '', so there is nothing to name
  expect([form.preview(), form.popupFieldHasError('duplicate')]).toEqual([null, true]);
  form.typeInPopup('duplicate', ''); // an emptied box is not an error until closing is tried
  expect([form.preview(), form.popupFieldHasError('duplicate')]).toEqual([null, false]);
  expect(form.isCloseBlocked()).toBe(true);
  expect([form.preview(), form.popupFieldHasError('duplicate')]).toEqual([null, true]);
});

test('a close that is blocked reopens the popup instead of sending anything', () => {
  const form = createForm();
  form.init();
  expect(form.isCloseBlocked()).toBe(false); // a reason without a popup closes straight away
  form.pick('Duplicate');
  form.pressInPopup('duplicate', 'Escape'); // the popup is closed and nothing was typed
  expect(form.popupShown('duplicate')).toBe(false);
  expect(form.isCloseBlocked()).toBe(true);
  expect(form.popupShown('duplicate')).toBe(true);
});

test('the other popup\'s text is sent as typed, and spaces alone don\'t count', () => {
  const form = createForm();
  form.init();
  form.pick('Other');
  form.typeInPopup('other', ' '.repeat(3));
  expect(form.sentField('other')).toBe('');
  expect([form.isCloseBlocked(), form.popupFieldHasError('other')]).toEqual([true, true]);
  form.typeInPopup('other', ' superseded ');
  expect([form.buttonText(), form.sentField('other'), form.popupFieldHasError('other')]).toEqual(['Close with other reason', ' superseded ', false]);
  expect(form.isCloseBlocked()).toBe(false);
});

test('Enter or Escape in a popup only closes the popup', () => {
  const form = createForm();
  form.init();
  for (const key of ['Enter', 'Escape']) {
    form.pick('Duplicate');
    expect(form.pressInPopup('duplicate', key).defaultPrevented).toBe(true); // Enter does not submit the form
    expect(form.popupShown('duplicate')).toBe(false);
  }
});

test('Enter or Escape for an input method stays in the popup', () => {
  const form = createForm();
  form.init();
  form.pick('Other');
  for (const key of ['Enter', 'Escape']) { // confirming or cancelling a conversion, as when typing Japanese or Chinese
    expect(form.pressInPopup('other', key, {isComposing: true}).defaultPrevented).toBe(false);
    expect(form.popupShown('other')).toBe(true);
  }
});

test('a reload starts again from the reason the page shows, whatever the browser restored', () => {
  const form = createForm();
  form.restoreFields({close_reason: 'other', close_reason_text: 'from before the reload'}); // as Firefox does
  form.init();
  expect(form.buttonText()).toBe('Close as completed');
  expect(form.sentReason()).toBe('completed');
  expect(form.sentField('other')).toBeNull();
  expect(form.sentField('duplicate')).toBeNull();
});

test('the duplicate popup previews the issue with the typed number', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '12');
  await vi.waitFor(() => expect(form.preview()).toBe('#12 Login fails'));
  expect(form.previewHtml()).toContain('<svg'); // the issue's state icon

  form.typeInPopup('duplicate', '3');
  await vi.waitFor(() => expect(form.preview()).toBe('#3 <b>not bold</b>'));
  expect(form.previewHtml()).not.toContain('<b>'); // the title is text, never markup

  form.typeInPopup('duplicate', '99');
  await vi.waitFor(() => expect(form.preview()).toBe('No #99 found in this repository'));
  form.typeInPopup('duplicate', '8');
  expect(form.preview()).toBe('Can\'t be a duplicate of itself'); // the page's own number, without a search
  form.typeInPopup('duplicate', '');
  expect(form.preview()).toBeNull();
});

test('a late answer for a number no longer in the box is dropped', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '7'); // answers late
  await new Promise((resolve) => setTimeout(resolve, 10)); // its lookup is under way
  form.typeInPopup('duplicate', '8'); // the item itself, which needs no lookup
  await new Promise((resolve) => setTimeout(resolve, 80)); // #7's answer has come in by now
  expect([form.preview(), form.sentField('duplicate')]).toEqual(["Can't be a duplicate of itself", '']);
});

test('a late failed search for a number no longer in the box is dropped', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '405'); // its search fails late
  await new Promise((resolve) => setTimeout(resolve, 10));
  form.typeInPopup('duplicate', '');
  await new Promise((resolve) => setTimeout(resolve, 80));
  expect([form.preview(), form.sentField('duplicate')]).toEqual([null, '']);
});

test('a number with leading zeros is the same number', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '012');
  await vi.waitFor(() => expect(form.preview()).toBe('#12 Login fails'));
  expect([form.sentField('duplicate'), form.buttonText()]).toEqual(['12', 'Close as duplicate of #12']);
});

test('the focus moves into a popup when it opens, and back to the button when it is done', async () => {
  const form = createForm();
  const detach = form.attach();
  form.init();
  const nextTick = () => new Promise((resolve) => setTimeout(resolve, 0));
  form.pick('Other');
  await nextTick(); // after the dropdown's own click handling, which keeps the focus
  expect(document.activeElement).toBe(form.popupInput('other'));
  form.pressInPopup('other', 'Escape');
  expect(document.activeElement).toBe(form.statusButton);
  form.isCloseBlocked(); // nothing typed yet, so the popup opens again
  await nextTick();
  expect(document.activeElement).toBe(form.popupInput('other'));
  detach();
});

test('a failed search leaves no preview, and closing still works', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '12');
  await vi.waitFor(() => expect(form.preview()).toBe('#12 Login fails'));
  form.typeInPopup('duplicate', '404');
  await vi.waitFor(() => expect(form.preview()).toBeNull()); // not "not found": the search itself failed
  expect([form.isCloseBlocked(), form.sentField('duplicate')]).toEqual([false, '404']); // so the server decides
});

test('closing as a duplicate waits until the preview has found the issue', async () => {
  const form = createForm();
  form.init();
  form.pick('Duplicate');
  form.typeInPopup('duplicate', '7'); // its search answers late
  expect([form.isCloseBlocked(), form.sentField('duplicate')]).toEqual([true, '']); // still looking
  expect(form.popupFieldHasError('duplicate')).toBe(false); // a whole number being looked up is not an error
  await vi.waitFor(() => expect(form.sentField('duplicate')).toBe('7'));
  expect(form.isCloseBlocked()).toBe(false);

  for (const [number, message] of [['8', 'Can\'t be a duplicate of itself'], ['99', 'No #99 found in this repository']]) {
    form.typeInPopup('duplicate', number);
    await vi.waitFor(() => expect(form.preview()).toBe(message));
    expect([form.buttonText(), form.sentField('duplicate')]).toEqual(['Close as duplicate', '']);
    expect(form.popupFieldHasError('duplicate')).toBe(true); // Gitea's red field around the box
    expect(form.previewHtml()).toContain('tw-text-red');
    expect(form.isCloseBlocked()).toBe(true);
  }
  form.typeInPopup('duplicate', '12');
  await vi.waitFor(() => expect(form.preview()).toBe('#12 Login fails'));
  expect([form.popupFieldHasError('duplicate'), form.isCloseBlocked()]).toEqual([false, false]);
});
