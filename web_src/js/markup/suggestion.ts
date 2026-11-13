import {queryElems} from '../utils/dom.ts';

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

// Replaces ```suggestion blocks in a PR code comment with a diff against the commented line.
// The server only sets data-suggestion-original on new-side code comments, so plain comments keep a code block.
export function initMarkupCodeSuggestion(elMarkup: HTMLElement): void {
  const original = elMarkup.getAttribute('data-suggestion-original');
  if (original === null) return;
  const label = elMarkup.getAttribute('data-suggestion-label')!;
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
  });
}
