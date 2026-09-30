import {ComboMarkdownEditor, getComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';

// The close/reopen button in an issue or pull request's comment form: its text says whether the typed
// comment is posted with it, and it is disabled while files upload. The editor's events bubble up to the form.
export function initRepoIssueStatusButton(elButtons: HTMLElement) {
  const statusButton = elButtons.querySelector<HTMLButtonElement>('#status-button')!;
  const elForm = elButtons.closest('form')!;
  const elEditor = elForm.querySelector<HTMLElement>('.combo-markdown-editor')!;
  const syncStatusButton = (content: string, isUploading: boolean) => {
    const statusText = statusButton.getAttribute(content.trim() ? 'data-status-and-comment' : 'data-status');
    statusButton.querySelector<HTMLElement>('.status-button-text')!.textContent = statusText;
    statusButton.disabled = isUploading;
  };
  const syncFromEditor = () => {
    const editor = getComboMarkdownEditor(elEditor)!; // these events only come from an initialized editor
    syncStatusButton(editor.value(), editor.isUploading());
  };
  elForm.addEventListener(ComboMarkdownEditor.EventEditorContentChanged, syncFromEditor);
  elForm.addEventListener(ComboMarkdownEditor.EventUploadStateChanged, syncFromEditor);
  // the editor attaches itself before its async init finishes, so start from the textarea, which holds any comment the browser restored
  syncStatusButton(elEditor.querySelector<HTMLTextAreaElement>('textarea.markdown-text-editor')!.value, false);
}
