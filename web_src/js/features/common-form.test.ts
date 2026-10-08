import {initGlobalEnterQuickSubmit} from './common-form.ts';
import {createElementFromHTML} from '../utils/dom.ts';

// these tests share document.body and a document listener, so they must not run concurrently (see vitest.config.ts)
describe('initGlobalEnterQuickSubmit', {concurrent: false}, () => {
  beforeAll(() => initGlobalEnterQuickSubmit());
  afterEach(() => document.body.replaceChildren());

  // the same layout as the repo topic editor: a bare ".ui.form" with a primary button, no <form>
  function createBareForm() {
    const form = createElementFromHTML<HTMLElement>(`<div class="ui form"><input type="text"><button class="ui primary button" style="display: block">Save</button></div>`);
    document.body.append(form);
    const onSave = vi.fn();
    form.querySelector('button')!.addEventListener('click', onSave);
    return {input: form.querySelector('input')!, onSave};
  }
  const pressEnter = (el: Element) => el.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true}));

  test('Enter in a bare input clicks the primary button', () => {
    const {input, onSave} = createBareForm();
    pressEnter(input);
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  test('Enter already handled by a widget does not also submit', () => {
    const {input, onSave} = createBareForm();
    input.addEventListener('keydown', (e) => e.preventDefault()); // as a Fomantic dropdown does when it picks an item
    pressEnter(input);
    expect(onSave).not.toHaveBeenCalled();
  });
});
