import '../../../fomantic/build/fomantic.js';
import {createElementFromHTML} from '../../utils/dom.ts';
import {hideScopedEmptyDividers, initAriaDropdownPatch} from './dropdown.ts';
import {initFomanticTransition} from './transition.ts';

$.fn.fomanticExt = {};
initFomanticTransition(); // labels of multiple selection dropdowns are shown with a transition
initAriaDropdownPatch();

// values are IDs, as in the protected branch user pickers, so a name taken from the value is wrong
function createMultipleDropdown(value: string) {
  const el = createElementFromHTML<HTMLElement>(`<div class="ui multiple selection dropdown">
  <input type="hidden" value="${value}">
  <div class="text"></div>
  <div class="menu">
    <div class="item" data-value="1">Alpha</div>
    <div class="item" data-value="2">Beta</div>
  </div>
</div>`);
  document.body.append(el);
  $(el).dropdown();
  return el;
}

test('dropdown-item-literal-text', () => {
  // a "choice" workflow_dispatch input can offer the string "false" as an option.
  // jQuery `.data()` would coerce `data-text="false"` to the boolean `false`, which then renders as empty text.
  const $dropdown = $(`<select class="ui dropdown">
    <option value="1">1</option>
    <option value="0">0</option>
    <option value="true">true</option>
    <option value="false">false</option>
  </select>`).dropdown();
  for (const value of ['1', '0', 'true', 'false']) {
    $dropdown.dropdown('set selected', value);
    expect($dropdown.dropdown('get text')).toEqual(value);
    expect($dropdown.dropdown('get value')).toEqual(value);
  }
});

test('hideScopedEmptyDividers-simple', () => {
  const container = createElementFromHTML(`<div>
<div class="divider"></div>
<div class="item">a</div>
<div class="divider"></div>
<div class="divider"></div>
<div class="divider"></div>
<div class="item">b</div>
<div class="divider"></div>
</div>`);
  hideScopedEmptyDividers(container);
  expect(container.innerHTML).toEqual(`
<div class="divider hidden"></div>
<div class="item">a</div>
<div class="divider hidden"></div>
<div class="divider hidden"></div>
<div class="divider"></div>
<div class="item">b</div>
<div class="divider hidden"></div>
`);
});

test('hideScopedEmptyDividers-items-all-filtered', () => {
  const container = createElementFromHTML(`<div>
<div class="any"></div>
<div class="divider"></div>
<div class="item filtered">a</div>
<div class="item filtered">b</div>
<div class="divider"></div>
<div class="any"></div>
</div>`);
  hideScopedEmptyDividers(container);
  expect(container.innerHTML).toEqual(`
<div class="any"></div>
<div class="divider hidden"></div>
<div class="item filtered">a</div>
<div class="item filtered">b</div>
<div class="divider"></div>
<div class="any"></div>
`);
});

test('hideScopedEmptyDividers-hide-last', () => {
  const container = createElementFromHTML(`<div>
<div class="item">a</div>
<div class="divider" data-scope="b"></div>
<div class="item tw-hidden" data-scope="b">b</div>
</div>`);
  hideScopedEmptyDividers(container);
  expect(container.innerHTML).toEqual(`
<div class="item">a</div>
<div class="divider hidden" data-scope="b"></div>
<div class="item tw-hidden" data-scope="b">b</div>
`);
});

test('hideScopedEmptyDividers-scoped-items', () => {
  const container = createElementFromHTML(`<div>
<div class="item" data-scope="">a</div>
<div class="divider" data-scope="b"></div>
<div class="item tw-hidden" data-scope="b">b</div>
<div class="divider" data-scope=""></div>
<div class="item" data-scope="">c</div>
</div>`);
  hideScopedEmptyDividers(container);
  expect(container.innerHTML).toEqual(`
<div class="item" data-scope="">a</div>
<div class="divider hidden" data-scope="b"></div>
<div class="item tw-hidden" data-scope="b">b</div>
<div class="divider hidden" data-scope=""></div>
<div class="item" data-scope="">c</div>
`);
});

// these tests share document.body and the announcer, so they must not run concurrently (see vitest.config.ts)
describe('multiple selection aria', {concurrent: false}, () => {
  beforeAll(() => {
    window.config.i18n.remove_label_str = 'Remove item "%s"';
    window.config.i18n.selected_item_str = 'Selected "%s"';
    window.config.i18n.deselected_item_str = 'Deselected "%s"';
  });
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    document.body.replaceChildren();
  });

  test('AC4: delete icons are named by the label text, not the value', () => {
    const el = createMultipleDropdown('2');
    expect(el.querySelector('.ui.label .delete.icon')!.getAttribute('aria-label')).toBe('Remove item "Beta"');
    $(el).dropdown('set selected', '1');
    const labelIcons = Array.from(el.querySelectorAll('.ui.label .delete.icon'), (icon) => icon.getAttribute('aria-label'));
    expect(labelIcons).toContain('Remove item "Alpha"');
  });

  const ariaSelected = (el: Element) => Array.from(el.querySelectorAll('.menu > .item'), (item) => item.getAttribute('aria-selected'));
  const announced = () => Array.from(document.querySelectorAll('[role="status"] > div'), (el) => el.textContent);

  test('AC1: popup is multiselectable and every option has aria-selected', () => {
    const el = createMultipleDropdown('2');
    vi.advanceTimersByTime(0);
    expect(el.querySelector(':scope > .menu')!.getAttribute('aria-multiselectable')).toBe('true');
    expect(ariaSelected(el)).toEqual(['false', 'true']);
  });

  test('AC2: aria-selected follows selection changes', () => {
    const el = createMultipleDropdown('2');
    $(el).dropdown('set selected', '1');
    $(el).dropdown('remove selected', '2');
    vi.advanceTimersByTime(0);
    expect(ariaSelected(el)).toEqual(['true', 'false']);
  });

  test('AC3: changes are announced by name, the initial selection is not', () => {
    const el = createMultipleDropdown('2');
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual([]);
    $(el).dropdown('set selected', '1');
    $(el).dropdown('remove selected', '2');
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual(['Selected "Alpha"', 'Deselected "Beta"']);
  });

  // the issue sidebar pickers keep their items in a nested scrolling menu
  test('nested scrolling menu items get option roles', () => {
    const el = createElementFromHTML<HTMLElement>(`<div class="ui dropdown">
  <div class="menu">
    <div class="ui icon search input"><input type="text"></div>
    <div class="scrolling menu">
      <a class="item" href="#" data-value="1">bug</a>
    </div>
  </div>
</div>`);
    document.body.append(el);
    $(el).dropdown();
    const item = el.querySelector('.scrolling.menu > .item')!;
    expect(item.getAttribute('role')).toBe('option');
    expect(item.id).not.toBe('');
  });

  // Fomantic blurred anything focused but an input, so picking an issue sidebar item dropped focus to <body>
  test('picking an item keeps the focus on a dropdown without a search input', () => {
    const el = createElementFromHTML<HTMLElement>(`<div class="ui dropdown">
  <div class="menu"><div class="item" data-value="1">bug</div></div>
</div>`);
    document.body.append(el);
    $(el).dropdown({action: 'nothing'});
    el.focus();
    expect(document.activeElement).toBe(el);
    el.querySelector<HTMLElement>('.item')!.click();
    expect(document.activeElement).toBe(el);
  });

  test('single selection dropdowns are not multiselectable', () => {
    const el = createElementFromHTML<HTMLElement>(`<div class="ui selection dropdown">
  <input type="hidden" value="1"><div class="text"></div>
  <div class="menu"><div class="item" data-value="1">Alpha</div></div>
</div>`);
    document.body.append(el);
    $(el).dropdown();
    expect(el.querySelector(':scope > .menu')!.hasAttribute('aria-multiselectable')).toBe(false);
    expect(el.querySelector('.item')!.hasAttribute('aria-selected')).toBe(false);
  });
});
