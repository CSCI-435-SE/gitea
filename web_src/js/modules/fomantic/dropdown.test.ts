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
    expect(ariaSelected(el)).toEqual(['true', 'false']); // in the same interaction, not a task later
    vi.advanceTimersByTime(0);
    expect(ariaSelected(el)).toEqual(['true', 'false']); // and the deferred refresh agrees
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

// these tests switch the run mode and spy on console.warn, so they must not run concurrently
describe('icon-only trigger names', {concurrent: false}, () => {
  const iconTrigger = '<svg class="svg octicon-kebab-horizontal"></svg><div class="menu"><div class="item">Edit</div></div>';

  function initDropdown(html: string) {
    const wrapper = createElementFromHTML<HTMLElement>(`<div>${html}</div>`);
    document.body.append(wrapper);
    const el = wrapper.querySelector<HTMLElement>('.ui.dropdown')!;
    $(el).dropdown();
    return el;
  }
  // count only this element's warnings, other dropdowns in the file are unnamed too
  const warningsAbout = (el: Element) => vi.mocked(console.warn).mock.calls.filter((args) => args.includes(el)).length;

  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
  });
  afterEach(() => {
    vi.restoreAllMocks();
    window.config.runModeIsProd = true;
    document.body.replaceChildren();
  });

  test('AC1: an icon-only trigger keeps the name given in its template', () => {
    const el = initDropdown(`<div class="ui dropdown" aria-label="More Operations">${iconTrigger}</div>`);
    expect(el.getAttribute('aria-label')).toBe('More Operations');
  });

  test('AC4: an existing aria-label is not overwritten by the tooltip', () => {
    const el = initDropdown(`<div class="ui dropdown" aria-label="Theme" data-tooltip-content="Other">${iconTrigger}</div>`);
    expect(el.getAttribute('aria-label')).toBe('Theme');
  });

  test('AC4: a tooltip still becomes the aria-label when there is none', () => {
    const el = initDropdown(`<div class="ui dropdown" data-tooltip-content="More Operations">${iconTrigger}</div>`);
    expect(el.getAttribute('aria-label')).toBe('More Operations');
  });

  test('AC5: an unnamed icon-only trigger is reported once in development', () => {
    window.config.runModeIsProd = false;
    const el = initDropdown(`<div class="ui dropdown">${iconTrigger}</div>`);
    expect(console.warn).toHaveBeenCalledWith(expect.stringContaining('no accessible name'), el);
    $(el).dropdown(); // re-initialising the module does not repeat it
    expect(warningsAbout(el)).toBe(1);
  });

  test('AC5: nothing is reported in production', () => {
    const el = initDropdown(`<div class="ui dropdown">${iconTrigger}</div>`);
    expect(warningsAbout(el)).toBe(0);
  });

  test.each([
    ['an aria-label', `<div class="ui dropdown" aria-label="More Operations">${iconTrigger}</div>`],
    ['a tooltip', `<div class="ui dropdown" data-tooltip-content="More Operations">${iconTrigger}</div>`],
    ['visible text', `<div class="ui dropdown"><span class="text">Sort</span>${iconTrigger}</div>`],
    ['a linked label', `<label for="icon-trigger-search">Owner</label>
<div class="ui search selection dropdown"><input class="search" id="icon-trigger-search"><div class="text"></div>${iconTrigger}</div>`],
  ])('AC5: a trigger named by %s is not reported', (_, html) => {
    window.config.runModeIsProd = false;
    const el = initDropdown(html);
    expect(warningsAbout(el)).toBe(0);
  });
});
