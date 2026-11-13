import {initMarkupCodeSuggestion} from './suggestion.ts';
import {POST} from '../modules/fetch.ts';

vi.mock('../modules/fetch.ts', () => ({
  POST: vi.fn(),
}));

function makeMarkup(attrs: Record<string, string>, body: string): HTMLElement {
  const el = document.createElement('div');
  el.className = 'render-content markup';
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  el.innerHTML = body;
  return el;
}

function codeBlock(text: string, lang = 'suggestion'): string {
  return `<div class="code-block-container code-overflow-scroll"><pre class="code-block"><code class="chroma language-${lang} display">${text}</code></pre></div>`;
}

const attrs = {'data-suggestion-original': '\treturn err', 'data-suggestion-label': 'Suggested change'};

function rows(el: Element) {
  return Array.from(el.querySelectorAll('.markup-suggestion-line'), (row) => ({
    kind: row.classList.contains('removed') ? 'removed' : 'added',
    text: row.querySelector('code')!.textContent,
  }));
}

test('renders the original line as removed and each block line as added', () => {
  const el = makeMarkup(attrs, `<p>try this</p>${codeBlock('\treturn fmt.Errorf("x: %w", err)\n\treturn nil\n')}`);
  initMarkupCodeSuggestion(el);
  expect(el.querySelector('.code-block-container')).toBeNull();
  const diff = el.querySelector('.markup-suggestion')!;
  expect(diff.querySelector('.markup-suggestion-header')!.textContent).toEqual('Suggested change');
  expect(rows(diff)).toEqual([
    {kind: 'removed', text: '\treturn err'},
    {kind: 'added', text: '\treturn fmt.Errorf("x: %w", err)'},
    {kind: 'added', text: '\treturn nil'},
  ]);
  expect(el.querySelector('p')!.textContent).toEqual('try this');
});

test('empty block only removes the line', () => {
  const el = makeMarkup(attrs, codeBlock(''));
  initMarkupCodeSuggestion(el);
  expect(rows(el)).toEqual([{kind: 'removed', text: '\treturn err'}]);
});

test('one blank line replaces the line with a blank line', () => {
  const el = makeMarkup(attrs, codeBlock('\n')); // what the renderer emits for a fence holding one blank line
  initMarkupCodeSuggestion(el);
  expect(rows(el)).toEqual([{kind: 'removed', text: '\treturn err'}, {kind: 'added', text: ''}]);
});

test('capitalised Suggestion is an ordinary code block', () => {
  const el = makeMarkup(attrs, codeBlock('x\n', 'Suggestion'));
  initMarkupCodeSuggestion(el);
  expect(el.querySelector('.markup-suggestion')).toBeNull();
});

test('each block gets its own diff against the same line', () => {
  const el = makeMarkup(attrs, codeBlock('a\n') + codeBlock('b\n'));
  initMarkupCodeSuggestion(el);
  const diffs = el.querySelectorAll('.markup-suggestion');
  expect(diffs).toHaveLength(2);
  expect(rows(diffs[1])).toEqual([{kind: 'removed', text: '\treturn err'}, {kind: 'added', text: 'b'}]);
});

test('HTML in a suggestion stays text', () => {
  const el = makeMarkup(attrs, codeBlock('&lt;img src=x onerror=alert(1)&gt;\n'));
  initMarkupCodeSuggestion(el);
  expect(el.querySelector('img')).toBeNull();
  expect(rows(el)[1].text).toEqual('<img src=x onerror=alert(1)>');
});

test('without the attribute nothing changes', () => {
  const el = makeMarkup({}, codeBlock('x\n'));
  const before = el.innerHTML;
  initMarkupCodeSuggestion(el);
  expect(el.innerHTML).toEqual(before);
});

test('other languages are left alone', () => {
  const el = makeMarkup(attrs, codeBlock('x\n', 'go'));
  initMarkupCodeSuggestion(el);
  expect(el.querySelector('.markup-suggestion')).toBeNull();
});

// what the server adds for a viewer who may push to the head branch (#156)
const applyAttrs = {
  ...attrs,
  'data-suggestion-apply-url': '/user2/repo/pulls/1/files/reviews/comments/7/apply',
  'data-suggestion-apply-head': 'abc123',
  'data-suggestion-apply-message': 'Apply suggestion from @ana on main.go',
  'data-suggestion-apply-label': 'Apply suggestion',
  'data-suggestion-message-label': 'Commit message',
  'data-suggestion-commit-label': 'Commit suggestion',
  'data-suggestion-cancel-label': 'Cancel',
};

function setupApply(attributes: Record<string, string> = applyAttrs) {
  const el = makeMarkup(attributes, codeBlock('\treturn nil\n'));
  initMarkupCodeSuggestion(el);
  const visible = (selector: string) => {
    const found = el.querySelector(selector);
    return found !== null && !found.classList.contains('tw-hidden');
  };
  const apply = el.querySelector<HTMLButtonElement>('.markup-suggestion-apply');
  const message = () => el.querySelector<HTMLInputElement>('.markup-suggestion-apply-form input')!;
  const button = (text: string) => Array.from(el.querySelectorAll('button')).find((b) => b.textContent === text)!;
  return {el, apply, message, button, visible};
}

describe('applying a suggestion', {concurrent: false}, () => {
  afterEach(() => {
    vi.mocked(POST).mockReset();
    vi.restoreAllMocks();
  });

  test('AC1 Apply opens a form with the default message, which can be edited', () => {
    const {el, apply, message, visible} = setupApply();
    expect(el.querySelector('.markup-suggestion-header')!.contains(apply)).toBe(true);
    expect(apply!.textContent).toEqual('Apply suggestion');
    expect(visible('.markup-suggestion-apply-form')).toBe(false);
    apply!.click();
    expect(visible('.markup-suggestion-apply-form')).toBe(true);
    expect(visible('.markup-suggestion-apply')).toBe(false);
    expect(message().value).toEqual('Apply suggestion from @ana on main.go');
    expect(message().getAttribute('aria-label')).toEqual('Commit message');
  });

  test('AC2 without the server saying so there is no Apply', () => {
    expect(setupApply(attrs).apply).toBeNull();
  });

  test('AC7 a comment with several suggestions has no Apply', () => {
    const el = makeMarkup(applyAttrs, codeBlock('a\n') + codeBlock('b\n'));
    initMarkupCodeSuggestion(el);
    expect(el.querySelectorAll('.markup-suggestion')).toHaveLength(2);
    expect(el.querySelector('.markup-suggestion-apply')).toBeNull();
  });

  test('AC1 committing posts the head commit and the edited message, then reloads the page', async () => {
    vi.mocked(POST).mockResolvedValue({ok: true} as Response);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const {apply, message, button} = setupApply();
    apply!.click();
    message().value = 'Use nil';
    button('Commit suggestion').click();
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    expect(POST).toHaveBeenCalledTimes(1);
    const [url, opts] = vi.mocked(POST).mock.calls[0];
    expect(url).toEqual('/user2/repo/pulls/1/files/reviews/comments/7/apply');
    const data = opts!.data as FormData;
    expect(data.get('head_commit_id')).toEqual('abc123');
    expect(data.get('message')).toEqual('Use nil');
  });

  test('Enter in the message commits', async () => {
    vi.mocked(POST).mockResolvedValue({ok: true} as Response);
    vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const {apply, message} = setupApply();
    apply!.click();
    message().dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true}));
    await vi.waitFor(() => expect(POST).toHaveBeenCalledTimes(1));
  });

  test('AC3 a refusal shows the server message next to the suggestion and keeps the form open', async () => {
    const outdated = 'This suggestion is outdated: the code changed since it was written.';
    vi.mocked(POST).mockResolvedValue({ok: false, json: () => Promise.resolve({errorMessage: outdated})} as Response);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const {el, apply, button, visible} = setupApply();
    apply!.click();
    button('Commit suggestion').click();
    await vi.waitFor(() => expect(visible('.markup-suggestion-apply-error')).toBe(true));
    expect(el.querySelector('.markup-suggestion-apply-error')!.textContent).toEqual(outdated);
    expect(visible('.markup-suggestion-apply-form')).toBe(true);
    expect(button('Commit suggestion').disabled).toBe(false); // the user can reload or try again
    expect(reload).not.toHaveBeenCalled();
  });

  test('a refusal without a message, like a 403, shows the generic error', async () => {
    window.config.i18n.error_occurred = 'An error occurred.';
    vi.mocked(POST).mockResolvedValue({ok: false, json: () => Promise.reject(new SyntaxError('not JSON'))} as unknown as Response);
    const {el, apply, button, visible} = setupApply();
    apply!.click();
    button('Commit suggestion').click();
    await vi.waitFor(() => expect(visible('.markup-suggestion-apply-error')).toBe(true));
    expect(el.querySelector('.markup-suggestion-apply-error')!.textContent).toEqual('An error occurred.');
  });

  test('Cancel closes the form and brings Apply back', () => {
    const {apply, button, visible} = setupApply();
    apply!.click();
    button('Cancel').click();
    expect(visible('.markup-suggestion-apply-form')).toBe(false);
    expect(visible('.markup-suggestion-apply')).toBe(true);
    expect(POST).not.toHaveBeenCalled();
  });
});
