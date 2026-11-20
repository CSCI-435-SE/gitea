import {queryElems, hideElem, showElem} from '../utils/dom.ts';
import {POST} from '../modules/fetch.ts';
import {showErrorToast} from '../modules/toast.ts';

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

// Posts an Apply and reloads the page to show the new commit; if it is refused, returns the message to show
async function postApply(url: string, data: FormData): Promise<string> {
  let errorMessage = window.config.i18n.error_occurred; // e.g. a 403, which has no message of ours
  try {
    const response = await POST(url, {data});
    if (response.ok) {
      window.location.reload(); // shows the new commit and the updated diff
      return '';
    }
    const json = await response.json();
    if (json.errorMessage) errorMessage = json.errorMessage;
  } catch {} // a network error or a body that isn't JSON keeps the generic message
  return errorMessage;
}

// Adds Apply to the suggestion's header (#156) and returns the header's button group. The server only sets
// data-suggestion-apply-url when this viewer may push to the head branch and the comment is not outdated; it
// checks everything again on submit.
function addApplyButton(elMarkup: HTMLElement, diff: HTMLElement): HTMLElement {
  const url = elMarkup.getAttribute('data-suggestion-apply-url')!;
  const header = diff.querySelector('.markup-suggestion-header')!;
  header.classList.add('tw-flex', 'tw-items-center', 'tw-justify-between');
  const buttons = document.createElement('div');
  buttons.className = 'tw-flex tw-gap-1';
  const applyButton = makeButton(elMarkup.getAttribute('data-suggestion-apply-label')!, 'ui mini basic button markup-suggestion-apply');
  buttons.append(applyButton);
  header.append(buttons);

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
  error.className = 'markup-suggestion-apply-error tw-text-red tw-whitespace-pre-line tw-px-2 tw-pb-2 tw-hidden'; // a hook's message has lines
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
    if (commitButton.disabled) return; // a second Enter while the first commit is still posting
    commitButton.disabled = true;
    hideElem(error);
    const data = new FormData();
    data.append('head_commit_id', elMarkup.getAttribute('data-suggestion-apply-head')!);
    data.append('message', message.value);
    const errorMessage = await postApply(url, data);
    if (!errorMessage) return;
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
  return buttons;
}

// The suggestions picked to apply together (#159). The batch lives only in this page, so a reload empties it (AC4).
type BatchItem = {id: string, path: string, start: number, end: number, message: string, toggle: HTMLElement};
const batch = new Map<string, BatchItem>();
let batchBar: {el: HTMLElement, count: HTMLElement, message: HTMLInputElement, edited: boolean} | null = null;

// a suggestion whose comment was deleted, or edited so it no longer offers one, has lost its toggle from the page and
// can't be applied, so it leaves the batch; true if any did
function dropRemovedFromBatch(): boolean {
  let dropped = false;
  for (const [id, item] of batch) {
    if (!item.toggle.isConnected) {
      batch.delete(id);
      dropped = true;
    }
  }
  return dropped;
}

// not batch.values().some(): Iterator helpers need Safari 18.4 / Firefox 131, newer than the es2020 build target
function overlapsBatch(item: BatchItem): boolean {
  for (const other of batch.values()) {
    if (other.path === item.path && other.start <= item.end && item.start <= other.end) return true;
  }
  return false;
}

// The default message matches ApplySuggestions' on the server, and is the suggestion's own for one (AC1, AC11)
function batchMessage(): string {
  if (batch.size === 1) return batch.values().next().value!.message;
  return `Apply ${batch.size} suggestions from code review`;
}

// Shows the bar while the batch has suggestions and removes it when it is empty (AC12)
function updateBatchBar(elMarkup: HTMLElement): void {
  if (batch.size === 0) {
    batchBar?.el.remove();
    batchBar = null;
    return;
  }
  if (!batchBar) {
    const attr = (name: string) => elMarkup.getAttribute(`data-suggestion-${name}`)!;
    const el = document.createElement('div');
    el.className = 'markup-suggestion-batch-bar ui segment tw-fixed tw-bottom-0 tw-inset-x-0 tw-z-50 tw-m-0 tw-flex tw-flex-wrap tw-items-center tw-gap-2';
    const count = document.createElement('span');
    const inputWrapper = document.createElement('div');
    inputWrapper.className = 'ui small input tw-flex-1';
    const message = document.createElement('input');
    message.type = 'text';
    message.setAttribute('aria-label', attr('message-label'));
    inputWrapper.append(message);
    const applyButton = makeButton(attr('batch-apply-label'), 'ui small primary button');
    const error = document.createElement('div');
    error.className = 'markup-suggestion-batch-error tw-w-full tw-text-red tw-whitespace-pre-line tw-hidden';
    el.append(count, inputWrapper, applyButton, error);
    document.body.append(el);
    const bar = {el, count, message, edited: false};
    batchBar = bar;
    message.addEventListener('input', () => { bar.edited = true });
    applyButton.addEventListener('click', async () => {
      if (dropRemovedFromBatch()) { // show what is left rather than posting a different batch than the bar showed
        updateBatchBar(elMarkup);
        return;
      }
      applyButton.disabled = true;
      hideElem(error);
      const data = new FormData();
      for (const id of batch.keys()) data.append('comment_ids', id);
      data.append('head_commit_id', attr('apply-head'));
      data.append('message', message.value);
      const errorMessage = await postApply(attr('batch-url'), data);
      if (!errorMessage) return;
      error.textContent = errorMessage; // e.g. the outdated suggestion, named (AC3)
      showElem(error);
      applyButton.disabled = false;
    });
  }
  batchBar.count.textContent = `${batch.size} ${elMarkup.getAttribute('data-suggestion-batch-selected-label')}`;
  if (!batchBar.edited) batchBar.message.value = batchMessage();
}

// Adds Add to batch / Remove from batch next to Apply (#159)
function addBatchToggle(elMarkup: HTMLElement, buttons: HTMLElement): void {
  const attr = (name: string) => elMarkup.getAttribute(`data-suggestion-${name}`)!;
  const end = Number(attr('line'));
  const id = attr('id');
  // a comment re-rendered after an edit (#154 AC8) may already be in the batch
  const toggle = makeButton(attr(batch.has(id) ? 'batch-remove-label' : 'batch-add-label'), 'ui mini basic button markup-suggestion-batch');
  const item: BatchItem = {
    id,
    path: attr('path'),
    start: end - attr('original').split('\n').length + 1, // a range's original holds one line per "\n"
    end,
    message: attr('apply-message'),
    toggle,
  };
  if (batch.has(id)) batch.set(id, item); // the re-rendered comment's toggle and text replace the old ones
  buttons.append(toggle);
  toggle.addEventListener('click', () => {
    if (batch.has(item.id)) {
      batch.delete(item.id);
    } else if (batch.size >= Number(attr('batch-max'))) {
      showErrorToast(attr('batch-too-large')); // AC14
      return;
    } else if (overlapsBatch(item)) {
      showErrorToast(attr('batch-overlap')); // AC2
      return;
    } else {
      batch.set(item.id, item);
    }
    toggle.textContent = attr(batch.has(item.id) ? 'batch-remove-label' : 'batch-add-label'); // so it can't be added twice (AC13)
    updateBatchBar(elMarkup);
  });
}

// Shows "Applied by @user in <commit>" in the suggestion's header (#157). The server renders it in a <template>
// beside the comment body rather than in it, so it survives an edit, which replaces the body's children (AC9).
function addAppliedMarker(elMarkup: HTMLElement, diff: HTMLElement): boolean {
  const marker = elMarkup.parentElement!.querySelector<HTMLTemplateElement>(':scope > template.markup-suggestion-applied');
  if (!marker) return false;
  const header = diff.querySelector('.markup-suggestion-header')!;
  header.classList.add('tw-flex', 'tw-items-center', 'tw-justify-between');
  header.append(marker.content.cloneNode(true));
  return true;
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
    diff.append(header);
    // a multi-line comment's original holds one line per "\n"; split on "\n" only so a CRLF line keeps its "\r"
    for (const line of original.split('\n')) diff.append(makeLine('removed', line));
    if (raw !== '') {
      for (const line of raw.replace(/\r?\n$/, '').split(/\r?\n/)) diff.append(makeLine('added', line));
    }
    block.replaceWith(diff);
    diffs.push(diff);
  });
  if (diffs.length && addAppliedMarker(elMarkup, diffs[0])) return; // an applied suggestion can't be applied again
  // a comment with several suggestions has no Apply (#156 AC7), and so can't join a batch either
  if (diffs.length === 1 && elMarkup.hasAttribute('data-suggestion-apply-url')) {
    addBatchToggle(elMarkup, addApplyButton(elMarkup, diffs[0]));
  }
}
