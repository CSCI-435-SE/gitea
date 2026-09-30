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

// the markup an open issue's form gets; a closed one has no hidden reason and no menu
const reasonMenuHtml = `
  <div class="ui dropdown icon button">
    <div class="menu">
      <div class="item js-aria-clickable selected" data-value="completed" data-status="Close as completed" data-status-and-comment="Close as completed with comment">Completed</div>
      <div class="item js-aria-clickable" data-value="not_planned" data-status="Close as not planned" data-status-and-comment="Close as not planned with comment">Not planned</div>
      <div class="item js-aria-clickable" data-value="duplicate" data-status="Close as duplicate" data-status-and-comment="Close as duplicate with comment">Duplicate</div>
    </div>
  </div>`;

function createForm(textareaValue = '', {isClosed = false} = {}) {
  const elForm = createElementFromHTML(`
    <form>
      <div class="combo-markdown-editor"><textarea class="markdown-text-editor">${textareaValue}</textarea></div>
      ${isClosed ? '' : '<input type="hidden" name="close_reason" value="completed">'}
      <div class="ui buttons">
        <button id="status-button" ${isClosed ?
          'data-status="Reopen Issue" data-status-and-comment="Reopen with Comment"' :
          'data-status="Close as completed" data-status-and-comment="Close as completed with comment"'}>
          <span class="status-button-text"></span>
        </button>
        ${isClosed ? '' : reasonMenuHtml}
      </div>
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
