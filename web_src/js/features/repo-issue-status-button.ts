import {ComboMarkdownEditor, getComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';

// The close/reopen button in an issue or pull request's comment form: its text says whether the typed
// comment is posted with it, and it is disabled while files upload. The editor's events bubble up to the form.
// While the item is open, the menu beside it picks the close reason: the button then takes that reason's texts.
export function initRepoIssueStatusButton(elButtons: HTMLElement) {
  const statusButton = elButtons.querySelector<HTMLButtonElement>('#status-button')!;
  const elForm = elButtons.closest('form')!;
  const elEditor = elForm.querySelector<HTMLElement>('.combo-markdown-editor')!;
  // the editor attaches itself before its async init finishes, so start from the textarea, which holds any comment the browser restored
  let content = elEditor.querySelector<HTMLTextAreaElement>('textarea.markdown-text-editor')!.value;
  let isUploading = false;
  const syncStatusButton = () => {
    const statusText = statusButton.getAttribute(content.trim() ? 'data-status-and-comment' : 'data-status');
    statusButton.querySelector<HTMLElement>('.status-button-text')!.textContent = statusText;
    statusButton.disabled = isUploading;
  };
  const syncFromEditor = () => {
    const editor = getComboMarkdownEditor(elEditor)!; // these events only come from an initialized editor
    content = editor.value();
    isUploading = editor.isUploading();
    syncStatusButton();
  };
  elForm.addEventListener(ComboMarkdownEditor.EventEditorContentChanged, syncFromEditor);
  elForm.addEventListener(ComboMarkdownEditor.EventUploadStateChanged, syncFromEditor);
  syncStatusButton();

  const elReasonMenu = elButtons.querySelector('.ui.dropdown > .menu');
  if (!elReasonMenu) return; // a closed item only offers reopening
  const reasonInput = elForm.querySelector<HTMLInputElement>('input[name="close_reason"]')!;
  // a plain click listener rather than the dropdown's onChange; the template marks the items js-aria-clickable, so Enter clicks them too
  elReasonMenu.addEventListener('click', (e) => {
    const elItem = (e.target as Element).closest<HTMLElement>('.item[data-value]');
    if (!elItem) return;
    reasonInput.value = elItem.getAttribute('data-value')!;
    statusButton.setAttribute('data-status', elItem.getAttribute('data-status')!);
    statusButton.setAttribute('data-status-and-comment', elItem.getAttribute('data-status-and-comment')!);
    syncStatusButton();
  });
}
