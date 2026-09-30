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

function createForm(textareaValue = '') {
  const elForm = createElementFromHTML(`
    <form>
      <div class="combo-markdown-editor"><textarea class="markdown-text-editor">${textareaValue}</textarea></div>
      <div class="ui buttons">
        <button id="status-button" data-status="Close as completed" data-status-and-comment="Close as completed with comment">
          <span class="status-button-text"></span>
        </button>
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
