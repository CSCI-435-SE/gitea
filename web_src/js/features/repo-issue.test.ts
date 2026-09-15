import {initRepoIssueTitleEdit} from './repo-issue.ts';
import {initInputCharCounter} from './common-page.ts';
import {createElementFromHTML} from '../utils/dom.ts';

function makeTitleEditor(oldTitle: string): HTMLElement {
  const root = createElementFromHTML<HTMLElement>(`
    <div>
      <div id="issue-title-display">
        <button id="issue-title-edit-show"></button>
      </div>
      <form class="tw-hidden" id="issue-title-editor">
        <div data-locale-chars-left-1="%d character left" data-locale-chars-left-n="%d characters left">
          <input name="title" value="${oldTitle}" data-old-title="${oldTitle}" maxlength="255">
        </div>
        <button type="button" class="ui cancel button"></button>
        <button type="submit" class="ui primary button"></button>
      </form>
    </div>
  `);
  document.body.append(root);
  initInputCharCounter(root.querySelector('#issue-title-editor > div')!);
  initRepoIssueTitleEdit();
  return root;
}

describe('initRepoIssueTitleEdit', () => {
  beforeEach(() => {
    document.documentElement.lang = 'en-US'; // trN resolves plural rules from the document language
    document.body.replaceChildren();
  });

  test('the char counter follows the title restored after an emptied field is cancelled', () => {
    const root = makeTitleEditor('hello');
    const input = root.querySelector('input')!;
    const counter = root.querySelector('.input-char-counter')!;

    root.querySelector<HTMLElement>('#issue-title-edit-show')!.click();
    input.value = '';
    input.dispatchEvent(new Event('input'));
    expect(counter.textContent).toBe('255 characters left');

    root.querySelector<HTMLElement>('.ui.cancel.button')!.click();
    root.querySelector<HTMLElement>('#issue-title-edit-show')!.click();
    expect(input.value).toBe('hello');
    expect(counter.textContent).toBe('250 characters left');
  });
});
