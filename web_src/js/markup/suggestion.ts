import {queryElems, hideElem, showElem} from '../utils/dom.ts';
import {POST} from '../modules/fetch.ts';

function makeLine(kind: 'removed' | 'added', text: string): HTMLElement {
  const row = document.createElement('div');
  row.className = `markup-suggestion-line ${kind}`;
  const marker = document.createElement('span');
  marker.className = 'markup-suggestion-marker';
  marker.textContent = kind === 'removed' ? '-' : '+';
  const code = document.createElement('code');
  code.textContent = text; // never innerHTML: suggestion text is user input
  row.append(marker, code);
  return row;
}

function makeButton(text: string, className: string): HTMLButtonElement {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = className;
  button.textContent = text;
  return button;
}

// Adds Apply to the suggestion's header (#156). The server only sets data-suggestion-apply-url when this viewer
// may push to the head branch and the comment is not outdated; it checks everything again on submit.
function addApplyButton(elMarkup: HTMLElement, diff: HTMLElement): void {
  const url = elMarkup.getAttribute('data-suggestion-apply-url')!;
  const header = diff.querySelector('.markup-suggestion-header')!;
  header.classList.add('tw-flex', 'tw-items-center', 'tw-justify-between');
  const applyButton = makeButton(elMarkup.getAttribute('data-suggestion-apply-label')!, 'ui mini basic button markup-suggestion-apply');
  header.append(applyButton);

  const form = document.createElement('div');
  form.className = 'markup-suggestion-apply-form tw-flex tw-items-center tw-gap-2 tw-p-2 tw-hidden';
  const inputWrapper = document.createElement('div');
  inputWrapper.className = 'ui small input tw-flex-1';
  const message = document.createElement('input');
  message.type = 'text';
  message.value = elMarkup.getAttribute('data-suggestion-apply-message')!; // editable, AC1
  message.setAttribute('aria-label', elMarkup.getAttribute('data-suggestion-message-label')!);
  inputWrapper.append(message);
  const commitButton = makeButton(elMarkup.getAttribute('data-suggestion-commit-label')!, 'ui small primary button');
  const cancelButton = makeButton(elMarkup.getAttribute('data-suggestion-cancel-label')!, 'ui small basic button');
  form.append(inputWrapper, commitButton, cancelButton);
  const error = document.createElement('div');
  error.className = 'markup-suggestion-apply-error tw-text-red tw-px-2 tw-pb-2 tw-hidden';
  diff.append(form, error);

  applyButton.addEventListener('click', () => {
    hideElem(applyButton);
    showElem(form);
    message.focus();
  });
  cancelButton.addEventListener('click', () => {
    hideElem([form, error]);
    showElem(applyButton);
  });
  const submit = async () => {
    commitButton.disabled = true;
    hideElem(error);
    const data = new FormData();
    data.append('head_commit_id', elMarkup.getAttribute('data-suggestion-apply-head')!);
    data.append('message', message.value);
    let errorMessage = window.config.i18n.error_occurred; // e.g. a 403, which has no message of ours
    try {
      const response = await POST(url, {data});
      if (response.ok) {
        window.location.reload(); // shows the new commit and the updated diff
        return;
      }
      const json = await response.json();
      if (json.errorMessage) errorMessage = json.errorMessage;
    } catch {} // a network error or a body that isn't JSON keeps the generic message
    // the outdated, moved-branch and protected-branch messages (AC3, AC4, AC6) stay next to the suggestion
    error.textContent = errorMessage;
    showElem(error);
    commitButton.disabled = false;
  };
  commitButton.addEventListener('click', submit);
  message.addEventListener('keydown', async (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.isComposing) {
      e.preventDefault();
      await submit();
    }
  });
}

// Replaces ```suggestion blocks in a PR code comment with a diff against the commented line.
// The server only sets data-suggestion-original on new-side code comments, so plain comments keep a code block.
export function initMarkupCodeSuggestion(elMarkup: HTMLElement): void {
  const original = elMarkup.getAttribute('data-suggestion-original');
  if (original === null) return;
  const label = elMarkup.getAttribute('data-suggestion-label')!;
  const diffs: HTMLElement[] = [];
  queryElems(elMarkup, 'code.language-suggestion', (elCode) => {
    const block = elCode.closest('.code-block-container') ?? elCode.closest('pre')!;
    // an empty fence renders "" (delete the line); otherwise every line ends with "\n", so one blank line is "\n"
    const raw = elCode.textContent;
    const diff = document.createElement('div');
    diff.className = 'markup-suggestion';
    const header = document.createElement('div');
    header.className = 'markup-suggestion-header';
    header.textContent = label;
    diff.append(header, makeLine('removed', original));
    if (raw !== '') {
      for (const line of raw.replace(/\r?\n$/, '').split(/\r?\n/)) diff.append(makeLine('added', line));
    }
    block.replaceWith(diff);
    diffs.push(diff);
  });
  // a comment with several suggestions has no Apply (#156 AC7; applying several at once is S6)
  if (diffs.length === 1 && elMarkup.hasAttribute('data-suggestion-apply-url')) addApplyButton(elMarkup, diffs[0]);
}
