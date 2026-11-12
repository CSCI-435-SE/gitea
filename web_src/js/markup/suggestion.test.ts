import {initMarkupCodeSuggestion} from './suggestion.ts';

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
