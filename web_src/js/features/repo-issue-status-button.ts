import {ComboMarkdownEditor, getComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';
import {getIssueColorClass, getIssueIcon} from './issue.ts';
import {createTippy} from '../modules/tippy.ts';
import {svg} from '../svg.ts';
import {parseIssueHref} from '../utils.ts';
import {createElementFromAttrs, createElementFromHTML, toggleElem} from '../utils/dom.ts';
import {matchIssue} from '../utils/match.ts';
import {debounce} from 'perfect-debounce';
import type {Instance} from 'tippy.js';
import type {Issue} from '../types.ts';

type ReasonPopup = {reason: string, field: HTMLInputElement, elPopup: HTMLElement, elInput: HTMLInputElement, isUsable: () => boolean, markUnusable: () => void, tippy: Instance};

// Shows the issue with the typed number beside the duplicate box, found by the same search as the comment box's # list,
// which only returns what the viewer can read. A number is accepted once that issue is found, so a close that would be
// refused (the issue itself, or no such issue) is never sent with its comment; if the search fails, the server decides.
function initDuplicatePreview(elPreview: HTMLElement, elInput: HTMLInputElement, onResize: () => void, onAccepted: (number: string) => void) {
  const {ownerName, repoName, indexString} = parseIssueHref(window.location.href);
  const elField = elPreview.closest('.field')!;
  let wantedNumber = ''; // an answer that arrives for an older number is dropped
  const show = (...children: Array<Node | string>) => {
    elField.classList.remove('error');
    elPreview.replaceChildren(...children);
    toggleElem(elPreview, children.length > 0);
    onResize(); // the popup grows or shrinks with the preview
  };
  // a number that can't be used: Gitea's red "error" field around the box, and the reason in red beside it
  const showError = (text: string) => {
    show(createElementFromAttrs('span', {class: 'tw-text-red'}, text));
    elField.classList.add('error');
  };
  const lookUp = debounce(async (number: string) => {
    let issues: Issue[];
    try {
      issues = await matchIssue(ownerName, repoName, indexString, number);
    } catch {
      if (number !== wantedNumber) return;
      show(); // no preview then, and the server checks the number on close
      onAccepted(number);
      return;
    }
    if (number !== wantedNumber) return;
    const issue = issues.find((i) => String(i.number) === number);
    if (issue) {
      show(
        createElementFromHTML(svg(getIssueIcon(issue), 16, [getIssueColorClass(issue)])),
        `#${issue.number}`,
        createElementFromAttrs('span', {class: 'gt-ellipsis'}, issue.title), // a text node, so a title can't inject markup
      );
      onAccepted(number);
    } else {
      showError(elPreview.getAttribute('data-locale-not-found')!.replace('%s', number));
    }
  }, 300); // the same delay as the # list
  // text, or a number like 0, -3 or 1.5, gets the same error as a number with no issue;
  // a number box reads letters as '', so for those there is nothing to name and only the box is marked
  const showUnusable = () => {
    if (elInput.value) {
      showError(elPreview.getAttribute('data-locale-not-found')!.replace('%s', elInput.value));
    } else {
      show();
      elField.classList.add('error');
    }
  };
  const update = () => {
    const isEmpty = !elInput.value && !elInput.validity.badInput; // text that isn't a number reads as '' too, but is "bad input"
    const number = /^[1-9]\d*$/.test(elInput.value) ? elInput.value : '';
    wantedNumber = number;
    if (isEmpty) {
      show();
    } else if (!number) {
      showUnusable();
    } else if (number === indexString) {
      showError(elPreview.getAttribute('data-locale-self')!);
    } else {
      lookUp(number);
    }
  };
  return {update, showUnusable};
}

// The close/reopen button in an issue or pull request's comment form: its text says whether the typed
// comment is posted with it, and it is disabled while files upload. The editor's events bubble up to the form.
// While the item is open, the menu beside it picks the close reason: the button then takes that reason's texts,
// and a reason that needs more (a duplicate's number, an "other" description) opens a small popup under the buttons.
export function initRepoIssueStatusButton(elButtons: HTMLElement) {
  const statusButton = elButtons.querySelector<HTMLButtonElement>('#status-button')!;
  const elForm = elButtons.closest('form')!;
  const elEditor = elForm.querySelector<HTMLElement>('.combo-markdown-editor')!;
  // hidden fields filled from the popups; none on a closed item
  const reasonFields = elForm.querySelectorAll<HTMLInputElement>('input[type="hidden"][data-close-reason]');
  // the editor attaches itself before its async init finishes, so start from the textarea, which holds any comment the browser restored;
  // Chromium may restore one later (going Back), which initSingleCommentEditor announces once the editor is ready
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
  let elPickedItem = elReasonMenu.querySelector<HTMLElement>('.item.selected')!;
  // Firefox restores these hidden fields on a reload as the last pick left them, while the menu and button are drawn anew
  reasonInput.value = elPickedItem.getAttribute('data-value')!;
  for (const field of reasonFields) {
    field.value = '';
    field.disabled = true;
  }
  const popups: ReasonPopup[] = [];

  // the button reads the picked item's texts, or a popup's own texts once it is filled in ("Close as duplicate of #12")
  const applyStatusTexts = () => {
    let [text, textWithComment] = [elPickedItem.getAttribute('data-status')!, elPickedItem.getAttribute('data-status-and-comment')!];
    const popup = popups.find((p) => p.reason === reasonInput.value);
    if (popup?.elPopup.hasAttribute('data-locale-status') && popup.field.value) {
      text = popup.elPopup.getAttribute('data-locale-status')!.replace('%s', popup.field.value);
      textWithComment = popup.elPopup.getAttribute('data-locale-status-and-comment')!.replace('%s', popup.field.value);
    }
    statusButton.setAttribute('data-status', text);
    statusButton.setAttribute('data-status-and-comment', textWithComment);
    syncStatusButton();
  };

  for (const elPopup of elForm.querySelectorAll<HTMLElement>('[data-close-reason-popup]')) {
    const reason = elPopup.getAttribute('data-close-reason-popup')!;
    const field = Array.from(reasonFields).find((el) => el.getAttribute('data-close-reason') === reason)!;
    const elField = elPopup.querySelector<HTMLElement>('.field')!;
    const elInput = elPopup.querySelector<HTMLInputElement>('input')!;
    const isUsable = () => elInput.type === 'number' ? /^[1-9]\d*$/.test(elInput.value) : Boolean(elInput.value.trim());
    const tippy = createTippy(elPopup, {
      content: elPopup,
      trigger: 'manual',
      placement: 'bottom-end',
      interactive: true,
      hideOnClick: true,
      getReferenceClientRect: () => elButtons.getBoundingClientRect(),
    });
    const elPreview = elPopup.querySelector<HTMLElement>('[data-close-duplicate-preview]');
    const preview = elPreview ? initDuplicatePreview(elPreview, elInput, () => tippy.popperInstance?.update(), (number) => {
      field.value = number;
      applyStatusTexts();
    }) : null;
    // the popup lives outside the form while shown, so its input fills the form's hidden field, only with a usable value
    elInput.addEventListener('input', () => {
      elField.classList.remove('error'); // typing answers the error; the preview marks a number it can't use again
      field.value = preview || !isUsable() ? '' : elInput.value; // a duplicate's number counts once the preview has accepted it
      preview?.update();
      applyStatusTexts();
    });
    // a duplicate says what it needs; the description box's own placeholder already asks for it
    const markUnusable = preview ? preview.showUnusable : () => elField.classList.add('error');
    elInput.addEventListener('keydown', (e) => {
      if (e.isComposing) return; // the input method's own Enter (confirm) or Escape (cancel)
      if (e.key === 'Enter' || e.key === 'Escape') {
        e.preventDefault(); // Enter only confirms the value; closing stays a deliberate click on the button
        tippy.hide();
        statusButton.focus();
      }
    });
    popups.push({reason, field, elPopup, elInput, isUsable, markUnusable, tippy});
  }

  // a close whose popup has no usable value yet sends nothing, not even the comment, and reopens that popup instead,
  // the way a browser points at a required field on submit
  statusButton.addEventListener('click', (e) => {
    const popup = popups.find((p) => p.reason === reasonInput.value);
    if (!popup || popup.field.value) return;
    e.preventDefault();
    if (!popup.isUsable()) popup.markUnusable(); // otherwise it is being looked up, or its preview says why not
    popup.tippy.show();
    setTimeout(() => popup.elInput.focus(), 0);
  });

  // a plain click listener rather than the dropdown's onChange; the template marks the items js-aria-clickable, so Enter clicks them too
  elReasonMenu.addEventListener('click', (e) => {
    const elItem = (e.target as Element).closest<HTMLElement>('.item[data-value]');
    if (!elItem) return;
    elPickedItem = elItem;
    reasonInput.value = elItem.getAttribute('data-value')!;
    for (const popup of popups) {
      const isNeeded = popup.reason === reasonInput.value;
      popup.field.disabled = !isNeeded; // a disabled field is not sent; its popup keeps what was typed
      if (isNeeded) {
        popup.tippy.show();
        setTimeout(() => popup.elInput.focus(), 0); // after the dropdown's own click handling, which keeps the focus on itself
      } else {
        popup.tippy.hide();
      }
    }
    applyStatusTexts();
  });
}
