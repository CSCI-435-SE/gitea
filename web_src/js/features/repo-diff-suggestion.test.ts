import {buildSuggestionInsert, suggestionFence, suggestionInsertPos} from './repo-diff-suggestion.ts';

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
