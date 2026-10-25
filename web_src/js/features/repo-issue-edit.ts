import {handleReply} from './repo-issue.ts';
import {getComboMarkdownEditor, initComboMarkdownEditor, ComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';
import {POST} from '../modules/fetch.ts';
import {showErrorToast} from '../modules/toast.ts';
import {hideElem, querySingleVisibleElem, showElem} from '../utils/dom.ts';
import {errorMessage} from '../modules/errors.ts';
import {triggerUploadStateChanged} from './comp/EditorUpload.ts';
import {convertHtmlToMarkdown} from '../markup/html2markdown.ts';
import {applyAreYouSure, reinitializeAreYouSure} from '../vendor/jquery.are-you-sure.ts';
import {registerShortcut} from '../modules/shortcut.ts';

async function tryOnEditContent(e: Event) {
  const clickTarget = (e.target as HTMLElement).closest('.edit-content');
  if (!clickTarget) return;

  e.preventDefault();
  const commentContent = clickTarget.closest('.comment-header')!.nextElementSibling!;
  const editContentZone = commentContent.querySelector('.edit-content-zone')!;
  let renderContent = commentContent.querySelector('.render-content')!;
  const rawContent = commentContent.querySelector('.raw-content')!;

  let comboMarkdownEditor : ComboMarkdownEditor;

  const cancelAndReset = (e: Event) => {
    e.preventDefault();
    showElem(renderContent);
    hideElem(editContentZone);
    comboMarkdownEditor.dropzoneReloadFiles();
  };

  const saveAndRefresh = async (e: Event) => {
    e.preventDefault();
    // we are already in a form, do not bubble up to the document otherwise there will be other "form submit handlers"
    // at the moment, the form submit event conflicts with initRepoDiffConversationForm (global '.conversation-holder form' event handler)
    e.stopPropagation();
    renderContent.classList.add('is-loading');
    showElem(renderContent);
    hideElem(editContentZone);
    try {
      const params = new URLSearchParams({
        content: comboMarkdownEditor.value(),
        context: String(editContentZone.getAttribute('data-context')),
        content_version: String(editContentZone.getAttribute('data-content-version')),
      });
      for (const file of comboMarkdownEditor.dropzoneGetFiles() ?? []) {
        params.append('files[]', file);
      }

      const response = await POST(editContentZone.getAttribute('data-update-url')!, {data: params});
      const data = await response.json();
      if (!response.ok) {
        showErrorToast(data?.errorMessage ?? window.config.i18n.error_occurred);
        return;
      }

      reinitializeAreYouSure(editContentZone.querySelector('form')); // the form is no longer dirty
      editContentZone.setAttribute('data-content-version', data.contentVersion);

      // replace the render content with new one, to trigger re-initialization of all features
      const newRenderContent = renderContent.cloneNode(false) as HTMLElement;
      newRenderContent.innerHTML = data.content;
      renderContent.replaceWith(newRenderContent);
      renderContent = newRenderContent;

      rawContent.textContent = comboMarkdownEditor.value();

      if (!commentContent.querySelector('.dropzone-attachments')) {
        if (data.attachments !== '') {
          commentContent.insertAdjacentHTML('beforeend', data.attachments);
        }
      } else if (data.attachments === '') {
        commentContent.querySelector('.dropzone-attachments')!.remove();
      } else {
        commentContent.querySelector('.dropzone-attachments')!.outerHTML = data.attachments;
      }
      comboMarkdownEditor.dropzoneSubmitReload();
    } catch (error) {
      showErrorToast(`Failed to save the content: ${errorMessage(error)}`);
      console.error(error);
    } finally {
      renderContent.classList.remove('is-loading');
    }
  };

  // Show write/preview tab and copy raw content as needed
  showElem(editContentZone);
  hideElem(renderContent);

  comboMarkdownEditor = getComboMarkdownEditor(editContentZone.querySelector('.combo-markdown-editor'))!;
  if (!comboMarkdownEditor) {
    editContentZone.innerHTML = document.querySelector('#issue-comment-editor-template')!.innerHTML;
    const form = editContentZone.querySelector('form')!;
    applyAreYouSure(form);
    const saveButton = querySingleVisibleElem<HTMLButtonElement>(editContentZone, '.ui.primary.button')!;
    const cancelButton = querySingleVisibleElem<HTMLButtonElement>(editContentZone, '.ui.cancel.button')!;
    comboMarkdownEditor = await initComboMarkdownEditor(editContentZone.querySelector('.combo-markdown-editor')!);
    const syncUiState = () => saveButton.disabled = comboMarkdownEditor.isUploading();
    comboMarkdownEditor.container.addEventListener(ComboMarkdownEditor.EventUploadStateChanged, syncUiState);
    cancelButton.addEventListener('click', cancelAndReset);
    form.addEventListener('submit', saveAndRefresh);
  }
  // when the content has changed on server side, there is no sync, and this page doesn't have the latest content,
  // the editor still shows the old content, server will reject end user's submit by "data-content-version" check
  comboMarkdownEditor.value(rawContent.textContent);
  comboMarkdownEditor.switchTabToEditor();
  comboMarkdownEditor.focus();
  triggerUploadStateChanged(comboMarkdownEditor.container);
}

function extractSelectedMarkdown(container: HTMLElement) {
  const selection = window.getSelection();
  if (!selection?.rangeCount) return '';
  const range = selection.getRangeAt(0);
  if (!container.contains(range.commonAncestorContainer)) return '';

  // todo: if commonAncestorContainer parent has "[data-markdown-original-content]" attribute, use the parent's markdown content
  // otherwise, use the selected HTML content and respect all "[data-markdown-original-content]/[data-markdown-generated-content]" attributes
  const contents = selection.getRangeAt(0).cloneContents();
  const el = document.createElement('div');
  el.append(contents);
  return convertHtmlToMarkdown(el);
}

// the rendered markup of the one comment, or issue description, that holds the whole selection
export function selectedCommentMarkup(selection: Selection | null): HTMLElement | null {
  if (!selection?.rangeCount || selection.isCollapsed) return null;
  const ancestor = selection.getRangeAt(0).commonAncestorContainer;
  const markup = (ancestor instanceof Element ? ancestor : ancestor.parentElement)?.closest<HTMLElement>('.render-content.markup');
  // only a comment with raw content can be quoted, which leaves out the reply box's own preview
  if (!markup?.parentElement!.querySelector(':scope > .raw-content')) return null;
  return markup;
}

function focusReplyEditor(editor: ComboMarkdownEditor) {
  if (editor.tabEditor && !editor.tabEditor.classList.contains('active')) editor.switchTabToEditor(); // the Preview tab hides the cursor and any quote
  editor.focus();
  editor.moveCursorToEnd();
}

// shared by the "Quote reply" menu item and the "r" shortcut
function appendQuoteReply(editor: ComboMarkdownEditor, contentToQuote: string) {
  const quotedContent = `${contentToQuote.replace(/^/mg, '> ')}\n\n`;
  if (editor.value()) {
    editor.value(`${editor.value()}\n\n${quotedContent}`);
  } else {
    editor.value(quotedContent);
  }
  focusReplyEditor(editor);
}

async function tryOnQuoteReply(e: Event) {
  const clickTarget = (e.target as HTMLElement).closest('.quote-reply');
  if (!clickTarget) return;

  e.preventDefault();
  const contentToQuoteId = clickTarget.getAttribute('data-target');
  const targetRawToQuote = document.querySelector<HTMLElement>(`#${contentToQuoteId}.raw-content`)!;
  const targetMarkupToQuote = targetRawToQuote.parentElement!.querySelector<HTMLElement>('.render-content.markup')!;
  let contentToQuote = extractSelectedMarkdown(targetMarkupToQuote);
  if (!contentToQuote) contentToQuote = targetRawToQuote.textContent;

  let editor;
  if (clickTarget.classList.contains('quote-reply-diff')) {
    const replyBtn = clickTarget.closest('.comment-code-cloud')!.querySelector<HTMLElement>('button.comment-form-reply')!;
    editor = await handleReply(replyBtn);
  } else {
    // for normal issue/comment page
    editor = getComboMarkdownEditor(document.querySelector('#comment-form .combo-markdown-editor'))!;
  }
  appendQuoteReply(editor, contentToQuote);
}

// "r" exists only where the server rendered the reply form, so it needs no permission checks of its own
export function initRepoIssueQuoteReplyShortcut(form: HTMLElement) {
  const editorContainer = form.querySelector<HTMLElement>('.combo-markdown-editor')!;
  registerShortcut({
    keys: form.getAttribute('data-shortcut-keys')!,
    description: form.getAttribute('data-shortcut-description')!,
    section: form.getAttribute('data-shortcut-section')!,
    element: editorContainer,
    action: () => {
      const editor = getComboMarkdownEditor(editorContainer);
      if (!editor) return; // still loading
      const markup = selectedCommentMarkup(window.getSelection());
      if (markup) {
        appendQuoteReply(editor, extractSelectedMarkdown(markup));
      } else {
        focusReplyEditor(editor);
      }
    },
  });
}

export function initRepoIssueCommentEdit() {
  document.addEventListener('click', (e) => {
    tryOnEditContent(e); // Edit issue or comment content
    tryOnQuoteReply(e); // Quote reply to the comment editor
  });
}
