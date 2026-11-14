import {buildSuggestionInsert, initSuggestChangeButton, suggestionFence, suggestionInsertPos, suggestionInsertRange} from './repo-diff-suggestion.ts';
import type {ComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';

// expected values come from #155's acceptance criteria and CommonMark's fence rule
test('suggestionFence', () => {
  // AC4: fewer than 3 backticks in a row keep a 3-backtick fence
  expect(suggestionFence('return x')).toEqual('```');
  expect(suggestionFence('const s = `a` + ``b``;')).toEqual('```');
  // AC4: a run of N >= 3 gives N+1, the longest run wins
  expect(suggestionFence('```js')).toEqual('````');
  expect(suggestionFence('`````')).toEqual('``````');
  expect(suggestionFence('``` and ````')).toEqual('`````');
});

test('buildSuggestionInsert', () => {
  // AC1, AC3: fence, the line with its leading tab, closing fence; cursor at the end of the line
  expect(buildSuggestionInsert('', '\treturn err')).toEqual({
    text: '```suggestion\n\treturn err\n```\n',
    cursor: '```suggestion\n\treturn err'.length,
  });
  // AC2: text before that does not end a line gets a newline so the block starts on its own line
  expect(buildSuggestionInsert('looks off:', 'x := 1')).toEqual({
    text: '\n```suggestion\nx := 1\n```\n',
    cursor: '\n```suggestion\nx := 1'.length,
  });
  // AC2, AC12: text before that ends a line (such as an earlier block) gets no extra newline
  expect(buildSuggestionInsert('```suggestion\nx := 1\n```\n', 'x := 1').text).toEqual('```suggestion\nx := 1\n```\n');
  // AC4: the line's own fence is held by a longer one
  expect(buildSuggestionInsert('', '```js').text).toEqual('````suggestion\n```js\n````\n');
  // AC5: empty and whitespace-only lines are kept as is
  expect(buildSuggestionInsert('', '').text).toEqual('```suggestion\n\n```\n');
  expect(buildSuggestionInsert('', '  \t').text).toEqual('```suggestion\n  \t\n```\n');
});

test('suggestionInsertPos', () => {
  const block = '```suggestion\nx := 1\n```\n';
  // outside any block the cursor stays where it is (AC2)
  expect(suggestionInsertPos('nit: rename', 4)).toEqual(4);
  expect(suggestionInsertPos(`${block}after`, block.length + 2)).toEqual(block.length + 2);
  expect(suggestionInsertPos(`before\n${block}`, 7)).toEqual(7); // at the start of the opening fence line
  // AC12: a cursor inside a block (in its body, or on either fence line) moves past the closing fence line
  expect(suggestionInsertPos(block, '```suggestion\nx := 1'.length)).toEqual(block.length);
  expect(suggestionInsertPos(block, 3)).toEqual(block.length);
  expect(suggestionInsertPos(`${block}tail`, block.length - 1)).toEqual(block.length);
  expect(suggestionInsertPos('```suggestion\nx := 1\n```', 15)).toEqual('```suggestion\nx := 1\n```'.length); // no trailing newline
  // CommonMark: a shorter fence or one with an info string does not close; an unclosed block runs to the end
  const long = '````suggestion\n```js\n````\nrest';
  expect(suggestionInsertPos(long, 16)).toEqual('````suggestion\n```js\n````\n'.length);
  expect(suggestionInsertPos('```go\na\n```suggestion\nb', 7)).toEqual('```go\na\n```suggestion\nb'.length);
  // CommonMark: a backtick fence whose info string holds a backtick is not a fence
  expect(suggestionInsertPos('``` a`b\ntext', 10)).toEqual(10);
});

// expected values from CommonMark 0.31.2 section 4.5 (fenced code blocks) and #155's AC12
test('suggestionInsertPos fence edge cases', () => {
  // a fence needs at least 3 characters, so a 2-tilde line opens nothing
  expect(suggestionInsertPos('~~\ntext', 5)).toEqual(5);
  // a fence may be indented by at most 3 spaces, so a run after other text is not one
  expect(suggestionInsertPos('x ```\ntext', 8)).toEqual(8);
  // only backtick fences forbid backticks in the info string; a tilde fence may hold them
  const tilde = '~~~ a`b\ntext\n~~~\n';
  expect(suggestionInsertPos(`${tilde}after`, 10)).toEqual(tilde.length);
  // a closing fence must be at least as long as the opening one
  const nested = '````\n```\nx\n````\n';
  expect(suggestionInsertPos(`${nested}rest`, 9)).toEqual(nested.length);
  // a closing fence may be followed by spaces
  const trailing = '```\na\n```  \n';
  expect(suggestionInsertPos(`${trailing}rest`, 4)).toEqual(trailing.length);
  // AC12: an unclosed block opened on the last line, with no newline after it, still runs to the end
  expect(suggestionInsertPos('a\n```', 3)).toEqual(5);
});

test('suggestionInsertRange', () => {
  const block = '```suggestion\nx := 1\n```\n';
  // AC2: a selection outside any block is replaced; a collapsed cursor stays put
  expect(suggestionInsertRange('nit: rename this', 5, 16)).toEqual([5, 16]);
  expect(suggestionInsertRange('nit', 3, 3)).toEqual([3, 3]);
  // a selection covering a whole block replaces it whole, leaving no broken fence
  expect(suggestionInsertRange(`a\n${block}b`, 0, block.length + 3)).toEqual([0, block.length + 3]);
  // AC12: a selection ending inside a block, or starting in one, replaces nothing and inserts after that block
  const text = `intro\n${block}`;
  expect(suggestionInsertRange(text, 2, 6 + '```suggestion\nx'.length)).toEqual([text.length, text.length]);
  expect(suggestionInsertRange(`${block}tail`, 16, block.length + 2)).toEqual([block.length, block.length]);
});

test('initSuggestChangeButton', () => {
  document.body.innerHTML = `
    <div class="combo-markdown-editor">
      <div class="markdown-toolbar-group tw-hidden"><button class="markdown-button-suggest-change">s</button></div>
      <textarea></textarea>
    </div>
    <code class="code-inner"><span>\treturn</span> err\n</code>`;
  const container = document.querySelector<HTMLElement>('.combo-markdown-editor')!;
  const textarea = container.querySelector('textarea')!;
  const button = container.querySelector<HTMLButtonElement>('.markdown-button-suggest-change')!;
  initSuggestChangeButton({container, textarea} as ComboMarkdownEditor, document.querySelector<HTMLElement>('.code-inner')!);

  // the hidden group is shown
  expect(button.parentElement!.classList.contains('tw-hidden')).toBe(false);
  // AC1, AC3: the line's text without markup or its newline, cursor at the end of the line
  button.click();
  const block = '```suggestion\n\treturn err\n```\n';
  expect(textarea.value).toEqual(block);
  const cursor = '```suggestion\n\treturn err'.length;
  expect([textarea.selectionStart, textarea.selectionEnd]).toEqual([cursor, cursor]);
  // AC12: a second click adds a second block after the first
  button.click();
  expect(textarea.value).toEqual(`${block}${block}`);
  expect(textarea.selectionStart).toEqual(block.length + cursor);
});

test('initSuggestChangeButton in a form', () => {
  document.body.innerHTML = `
    <form>
      <div class="combo-markdown-editor">
        <div class="markdown-toolbar-group tw-hidden"><button class="markdown-button-suggest-change">s</button></div>
        <textarea></textarea>
      </div>
    </form>
    <code class="code-inner">x\n</code>`;
  const container = document.querySelector<HTMLElement>('.combo-markdown-editor')!;
  const textarea = container.querySelector('textarea')!;
  const button = container.querySelector<HTMLButtonElement>('.markdown-button-suggest-change')!;
  initSuggestChangeButton({container, textarea} as ComboMarkdownEditor, document.querySelector<HTMLElement>('.code-inner')!);

  // HTML spec: a <button> with no type attribute submits its form, so the click must not
  const click = new MouseEvent('click', {bubbles: true, cancelable: true});
  button.dispatchEvent(click);
  expect(click.defaultPrevented).toBe(true);

  // AC2: inserted mid-text, the block starts on a line of its own and the text after it is kept
  textarea.value = 'ab\n';
  textarea.setSelectionRange(1, 1);
  button.click();
  expect(textarea.value).toEqual('a\n```suggestion\nx\n```\nb\n');
});

test('initSuggestChangeButton without a button', () => {
  document.body.innerHTML = '<div class="combo-markdown-editor"><textarea></textarea></div><code class="code-inner">x</code>';
  const container = document.querySelector<HTMLElement>('.combo-markdown-editor')!;
  const editor = {container, textarea: container.querySelector('textarea')!} as ComboMarkdownEditor;
  expect(() => initSuggestChangeButton(editor, document.querySelector<HTMLElement>('.code-inner')!)).not.toThrow();
});
