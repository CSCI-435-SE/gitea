import {initMarkupCodeSuggestion} from './suggestion.ts';
import {POST} from '../modules/fetch.ts';
import {showErrorToast} from '../modules/toast.ts';

vi.mock('../modules/fetch.ts', () => ({
  POST: vi.fn(),
}));

vi.mock('../modules/toast.ts', () => ({
  showErrorToast: vi.fn(),
}));

function makeMarkup(attrs: Record<string, string>, body: string): HTMLElement {
  const el = document.createElement('div');
  el.className = 'render-content markup';
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  el.innerHTML = body;
  document.createElement('div').append(el); // as in the comment templates, the body always has a parent (#157)
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

test('a multi-line comment removes every line of its range', () => {
  const range = {'data-suggestion-original': 'l18\n\tl19\nl20\nl21\nl22', 'data-suggestion-label': 'Suggested change'};
  // #158 AC5: five removed lines in order, then the three block lines added
  const el = makeMarkup(range, codeBlock('a\nb\nc\n'));
  initMarkupCodeSuggestion(el);
  expect(rows(el)).toEqual([
    {kind: 'removed', text: 'l18'}, {kind: 'removed', text: '\tl19'}, {kind: 'removed', text: 'l20'},
    {kind: 'removed', text: 'l21'}, {kind: 'removed', text: 'l22'},
    {kind: 'added', text: 'a'}, {kind: 'added', text: 'b'}, {kind: 'added', text: 'c'},
  ]);
  // #158 AC6: an empty block removes the whole range and adds nothing
  const empty = makeMarkup(range, codeBlock(''));
  initMarkupCodeSuggestion(empty);
  expect(rows(empty).map((r) => r.kind)).toEqual(['removed', 'removed', 'removed', 'removed', 'removed']);
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

  test('pressing Enter again while the commit is posting sends it once', async () => {
    const {promise, resolve: finish} = Promise.withResolvers<Response>();
    vi.mocked(POST).mockReturnValue(promise);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const {apply, message} = setupApply();
    apply!.click();
    const enter = () => message().dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true}));
    enter();
    enter();
    finish({ok: true} as Response);
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    expect(POST).toHaveBeenCalledTimes(1);
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

  // #159 moved the POST into postApply, shared with the batch bar; these pin what #156 already did
  test('AC1 a commit that succeeds shows no error and keeps the button disabled while the page reloads', async () => {
    vi.mocked(POST).mockResolvedValue({ok: true} as Response);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const {apply, button, visible} = setupApply();
    apply!.click();
    button('Commit suggestion').click();
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    await Promise.resolve();
    expect(visible('.markup-suggestion-apply-error')).toBe(false);
    expect(button('Commit suggestion').disabled).toBe(true);
  });

  test('a refusal whose JSON has no message shows the generic error', async () => {
    window.config.i18n.error_occurred = 'An error occurred.';
    vi.mocked(POST).mockResolvedValue({ok: false, json: () => Promise.resolve({})} as Response);
    const {el, apply, button, visible} = setupApply();
    apply!.click();
    button('Commit suggestion').click();
    await vi.waitFor(() => expect(visible('.markup-suggestion-apply-error')).toBe(true));
    expect(el.querySelector('.markup-suggestion-apply-error')!.textContent).toEqual('An error occurred.');
  });
});

// what the server adds next to Apply's attributes (#159)
function batchAttrs(id: number, path: string, line: number, message = `Apply suggestion from @ana on ${path}`) {
  return {
    ...applyAttrs,
    'data-suggestion-apply-url': `/user2/repo/pulls/1/files/reviews/comments/${id}/apply`,
    'data-suggestion-apply-message': message,
    'data-suggestion-id': String(id),
    'data-suggestion-path': path,
    'data-suggestion-line': String(line),
    'data-suggestion-batch-url': '/user2/repo/pulls/1/files/reviews/comments/apply',
    'data-suggestion-batch-max': '50',
    'data-suggestion-batch-add-label': 'Add to batch',
    'data-suggestion-batch-remove-label': 'Remove from batch',
    'data-suggestion-batch-selected-label': 'suggestions selected',
    'data-suggestion-batch-apply-label': 'Apply suggestions',
    'data-suggestion-batch-too-large': 'A batch can hold at most 50 suggestions.',
    'data-suggestion-batch-overlap': 'This suggestion overlaps another suggestion in this batch.',
  };
}

describe('applying several suggestions in one commit', {concurrent: false}, () => {
  const toggles: HTMLButtonElement[] = [];
  // a comment on the page, as the batch only keeps suggestions whose toggle is still there
  function suggestion(id: number, path: string, line: number, message?: string): HTMLButtonElement {
    const el = makeMarkup(batchAttrs(id, path, line, message), codeBlock('x\n'));
    document.body.append(el);
    initMarkupCodeSuggestion(el);
    const toggle = el.querySelector<HTMLButtonElement>('.markup-suggestion-batch')!;
    toggles.push(toggle);
    return toggle;
  }
  const comment = (toggle: HTMLElement) => toggle.closest<HTMLElement>('.render-content')!;
  const bar = () => document.querySelector('.markup-suggestion-batch-bar');
  const barMessage = () => bar()!.querySelector('input')!.value;
  const applyButton = () => Array.from(bar()!.querySelectorAll('button')).find((b) => b.textContent === 'Apply suggestions')!;

  afterEach(() => {
    for (const toggle of toggles) {
      // the batch is module state; a toggle no longer on the page belongs to a comment the test removed
      if (toggle.isConnected && toggle.textContent === 'Remove from batch') toggle.click();
    }
    for (const toggle of toggles) toggle.closest('.render-content')?.remove();
    toggles.length = 0;
    vi.mocked(POST).mockReset();
    vi.mocked(showErrorToast).mockReset();
    vi.restoreAllMocks();
  });

  test('AC13 Add to batch becomes Remove from batch; AC12 the bar shows the count and goes when the batch is empty', () => {
    const first = suggestion(1, 'main.go', 2);
    const second = suggestion(2, 'main.go', 5);
    expect(first.textContent).toEqual('Add to batch');
    expect(bar()).toBeNull();
    first.click();
    second.click();
    expect(first.textContent).toEqual('Remove from batch');
    expect(bar()!.textContent).toContain('2 suggestions selected');
    first.click();
    expect(first.textContent).toEqual('Add to batch');
    expect(bar()!.textContent).toContain('1 suggestions selected');
    second.click();
    expect(bar()).toBeNull();
  });

  test('AC1 AC11 the message is the suggestion\'s own for one, "Apply N suggestions from code review" for more, and an edit is kept', () => {
    const first = suggestion(1, 'main.go', 2, 'Apply suggestion from @ana on main.go');
    const second = suggestion(2, 'util.go', 2);
    first.click();
    expect(barMessage()).toEqual('Apply suggestion from @ana on main.go');
    second.click();
    expect(barMessage()).toEqual('Apply 2 suggestions from code review');
    const input = bar()!.querySelector('input')!;
    input.value = 'Tidy up';
    input.dispatchEvent(new Event('input'));
    second.click();
    expect(barMessage()).toEqual('Tidy up');
  });

  test('AC2 a suggestion on a line already in the batch is refused; other lines and files are not', () => {
    const first = suggestion(1, 'main.go', 3);
    const sameLine = suggestion(2, 'main.go', 3);
    const nextLine = suggestion(3, 'main.go', 4);
    const otherFile = suggestion(4, 'util.go', 3);
    first.click();
    sameLine.click();
    expect(showErrorToast).toHaveBeenCalledWith('This suggestion overlaps another suggestion in this batch.');
    expect(sameLine.textContent).toEqual('Add to batch');
    nextLine.click();
    otherFile.click();
    expect(showErrorToast).toHaveBeenCalledTimes(1);
    expect(bar()!.textContent).toContain('3 suggestions selected');
  });

  test('a comment re-rendered after an edit keeps showing that it is in the batch', () => {
    const first = suggestion(1, 'main.go', 2);
    first.click();
    comment(first).remove();
    const again = suggestion(1, 'main.go', 2); // #154 AC8 re-renders an edited comment
    expect(again.textContent).toEqual('Remove from batch');
    again.click();
    expect(bar()).toBeNull();
  });

  // #196 review: a suggestion that leaves the page must not stay in the batch, or every Apply would fail on it
  test('a deleted comment, or one edited to have no suggestion, leaves the batch before it is posted', async () => {
    vi.mocked(POST).mockResolvedValue({ok: true} as Response);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const kept = suggestion(1, 'main.go', 2);
    const deleted = suggestion(2, 'main.go', 5);
    const edited = suggestion(3, 'util.go', 4);
    for (const toggle of [kept, deleted, edited]) toggle.click();
    expect(bar()!.textContent).toContain('3 suggestions selected');

    comment(deleted).remove();
    comment(edited).innerHTML = '<p>never mind</p>'; // the edited comment re-renders without a suggestion
    applyButton().click();
    expect(POST).not.toHaveBeenCalled(); // the bar first shows what is left
    expect(bar()!.textContent).toContain('1 suggestions selected');
    expect(barMessage()).toEqual('Apply suggestion from @ana on main.go');

    applyButton().click();
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    expect((vi.mocked(POST).mock.calls[0][1]!.data as FormData).getAll('comment_ids')).toEqual(['1']);
  });

  test('when every suggestion in the batch has left the page, the bar goes', () => {
    const gone = suggestion(1, 'main.go', 2);
    gone.click();
    comment(gone).remove();
    applyButton().click();
    expect(POST).not.toHaveBeenCalled();
    expect(bar()).toBeNull();
  });

  test('AC2 AC9 a suggestion above one already in the batch, in the same file, is accepted', () => {
    suggestion(1, 'main.go', 5).click();
    const above = suggestion(2, 'main.go', 2);
    above.click();
    expect(showErrorToast).not.toHaveBeenCalled();
    expect(above.textContent).toEqual('Remove from batch');
  });

  test('the bar\'s message field is labelled like Apply\'s', () => {
    suggestion(1, 'main.go', 2).click();
    expect(bar()!.querySelector('input')!.getAttribute('aria-label')).toEqual('Commit message');
  });

  test('Apply posts once while posting, shows a refusal, hides it on retry, and shows nothing once it succeeds', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    suggestion(7, 'main.go', 2).click();
    suggestion(9, 'util.go', 4).click();
    const apply = Array.from(bar()!.querySelectorAll('button')).find((b) => b.textContent === 'Apply suggestions')!;
    const error = bar()!.querySelector('.markup-suggestion-batch-error')!;
    const shown = () => !error.classList.contains('tw-hidden');

    const first = Promise.withResolvers<Response>();
    vi.mocked(POST).mockReturnValueOnce(first.promise);
    apply.click();
    apply.click(); // a second click while the first is posting
    expect(POST).toHaveBeenCalledTimes(1);
    first.resolve({ok: false, json: () => Promise.resolve({errorMessage: 'refused'})} as Response);
    await vi.waitFor(() => expect(shown()).toBe(true));
    expect(error.textContent).toEqual('refused');

    const second = Promise.withResolvers<Response>();
    vi.mocked(POST).mockReturnValueOnce(second.promise);
    apply.click();
    expect(shown()).toBe(false); // the old refusal goes while retrying
    second.resolve({ok: true} as Response);
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    await Promise.resolve();
    expect(shown()).toBe(false);
    expect(apply.disabled).toBe(true);
  });

  test('AC14 the 51st suggestion is refused', () => {
    for (let i = 1; i <= 50; i++) suggestion(i, 'main.go', i).click();
    const extra = suggestion(51, 'main.go', 51);
    extra.click();
    expect(showErrorToast).toHaveBeenCalledWith('A batch can hold at most 50 suggestions.');
    expect(extra.textContent).toEqual('Add to batch');
    expect(bar()!.textContent).toContain('50 suggestions selected');
  });

  test('AC1 Apply posts every suggestion, the head and the message, then reloads the page', async () => {
    vi.mocked(POST).mockResolvedValue({ok: true} as Response);
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    suggestion(7, 'main.go', 2).click();
    suggestion(9, 'util.go', 4).click();
    Array.from(bar()!.querySelectorAll('button')).find((b) => b.textContent === 'Apply suggestions')!.click();
    await vi.waitFor(() => expect(reload).toHaveBeenCalled());
    const [url, opts] = vi.mocked(POST).mock.calls[0];
    expect(url).toEqual('/user2/repo/pulls/1/files/reviews/comments/apply');
    const data = opts!.data as FormData;
    expect(data.getAll('comment_ids')).toEqual(['7', '9']);
    expect(data.get('head_commit_id')).toEqual('abc123');
    expect(data.get('message')).toEqual('Apply 2 suggestions from code review');
  });

  test('AC3 a refusal shows the server message in the bar', async () => {
    const outdated = 'This suggestion is outdated: the code changed since it was written. (util.go:4)';
    vi.mocked(POST).mockResolvedValue({ok: false, json: () => Promise.resolve({errorMessage: outdated})} as Response);
    suggestion(7, 'main.go', 2).click();
    suggestion(9, 'util.go', 4).click();
    const apply = Array.from(bar()!.querySelectorAll('button')).find((b) => b.textContent === 'Apply suggestions')!;
    apply.click();
    await vi.waitFor(() => expect(bar()!.querySelector('.markup-suggestion-batch-error')!.textContent).toEqual(outdated));
    expect(apply.disabled).toBe(false);
  });
});

// what the server renders beside the comment body once its suggestion was applied (#157)
const appliedMarker = '<template class="markup-suggestion-applied"><span class="markup-suggestion-applied-text">Applied by <a href="/ben">ben</a> in <a href="/user2/repo/commit/a1b2c3d4e5f6"><code>a1b2c3d4e5</code></a></span></template>';

function withAppliedMarker(el: HTMLElement): HTMLElement {
  el.parentElement!.insertAdjacentHTML('beforeend', appliedMarker);
  return el;
}

describe('an applied suggestion', () => {
  test('#157 AC1 shows who applied it and the commit in the header, with no Apply', () => {
    const el = withAppliedMarker(makeMarkup(applyAttrs, codeBlock('\treturn nil\n')));
    initMarkupCodeSuggestion(el);
    const marker = el.querySelector('.markup-suggestion-header .markup-suggestion-applied-text')!;
    expect(marker.textContent).toEqual('Applied by ben in a1b2c3d4e5');
    expect(marker.querySelector('a[href="/user2/repo/commit/a1b2c3d4e5f6"]')).not.toBeNull();
    expect(el.querySelector('.markup-suggestion-apply')).toBeNull();
  });

  test('#157 AC1 the header keeps the label on the left and the marker on the right, as for Apply', () => {
    const el = withAppliedMarker(makeMarkup(attrs, codeBlock('\treturn nil\n')));
    initMarkupCodeSuggestion(el);
    const header = el.querySelector('.markup-suggestion-header')!;
    // the layout #156 gives the header for Apply, which the marker takes over (#157 Design Decision)
    expect([...header.classList]).toEqual(expect.arrayContaining(['tw-flex', 'tw-items-center', 'tw-justify-between']));
    expect(header.firstChild!.textContent).toEqual('Suggested change');
    expect(header.lastElementChild!.classList.contains('markup-suggestion-applied-text')).toBe(true);
  });

  test('#157 AC9 the marker comes back when an edit re-renders the comment', () => {
    const el = withAppliedMarker(makeMarkup(attrs, codeBlock('\treturn nil\n')));
    initMarkupCodeSuggestion(el);
    // like repo-issue-edit.ts: a shallow clone keeps the attributes, the new content replaces the old
    const edited = el.cloneNode(false) as HTMLElement;
    edited.innerHTML = codeBlock('\treturn nil // edited\n');
    el.replaceWith(edited);
    initMarkupCodeSuggestion(edited);
    expect(edited.querySelectorAll('.markup-suggestion-applied-text')).toHaveLength(1);
  });

  test('#157 AC9 an edit that adds a second suggestion keeps the marker on the first', () => {
    const el = withAppliedMarker(makeMarkup(attrs, codeBlock('a\n') + codeBlock('b\n')));
    initMarkupCodeSuggestion(el);
    const diffs = el.querySelectorAll('.markup-suggestion');
    expect(diffs[0].querySelector('.markup-suggestion-applied-text')).not.toBeNull();
    expect(diffs[1].querySelector('.markup-suggestion-applied-text')).toBeNull();
  });

  test('a suggestion that was not applied has no marker', () => {
    const el = makeMarkup(attrs, codeBlock('\treturn nil\n'));
    initMarkupCodeSuggestion(el);
    expect(el.querySelector('.markup-suggestion-applied-text')).toBeNull();
  });
});
