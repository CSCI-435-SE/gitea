import '../../fomantic/build/fomantic.js';
import {IssueSidebarComboList, syncIssueMainContentTimelineItems} from './repo-issue-sidebar-combolist.ts';
import {createElementFromHTML} from '../utils/dom.ts';

describe('syncIssueMainContentTimelineItems', () => {
  test('InsertNew', () => {
    const oldContent = createElementFromHTML(`
    <div>
        <div class="timeline-item">First</div>
        <div class="timeline-item" id="timeline-comments-end"></div>
    </div>
  `);
    const newContent = createElementFromHTML(`
    <div>
        <div class="timeline-item" id="a">New</div>
    </div>
  `);
    syncIssueMainContentTimelineItems(oldContent, newContent);
    expect(oldContent.innerHTML.replace(/>\s+</g, '><').trim()).toBe(
      `<div class="timeline-item">First</div>` +
      `<div class="timeline-item" id="a">New</div>` +
      `<div class="timeline-item" id="timeline-comments-end"></div>`,
    );
  });

  test('Sync', () => {
    const oldContent = createElementFromHTML(`
    <div>
      <div class="timeline-item">First</div>
      <div class="timeline-item" id="it-1">Item 1</div>
      <div class="timeline-item event" id="it-2">Item 2</div>
      <div class="timeline-item" id="it-3">Item 3</div>
      <div class="timeline-item event" id="it-4">Item 4</div>
      <div class="timeline-item" id="timeline-comments-end"></div>
      <div class="timeline-item">Other</div>
    </div>
  `);
    const newContent = createElementFromHTML(`
    <div>
      <div class="timeline-item" id="it-1">New 1</div>
      <div class="timeline-item event" id="it-2">New 2</div>
      <div class="timeline-item" id="it-x">New X</div>
    </div>
  `);
    syncIssueMainContentTimelineItems(oldContent, newContent);

    // Item 1 won't be replaced because it's not an event
    // Item 2 will be replaced with New 2
    // Item 3 will be kept because it's not in new content
    // Item 4 will be removed because it's not in new content, and it's an event
    // New X will be inserted at the end of timeline items (before timeline-comments-end)
    expect(oldContent.innerHTML.replace(/>\s+</g, '><').trim()).toBe(
      `<div class="timeline-item">First</div>` +
      `<div class="timeline-item" id="it-1">Item 1</div>` +
      `<div class="timeline-item event" id="it-2">New 2</div>` +
      `<div class="timeline-item" id="it-3">Item 3</div>` +
      `<div class="timeline-item" id="it-x">New X</div>` +
      `<div class="timeline-item" id="timeline-comments-end"></div>` +
      `<div class="timeline-item">Other</div>`,
    );
  });
});

// these tests share document.body and the announcer, so they must not run concurrently (see vitest.config.ts)
describe('IssueSidebarComboList aria', {concurrent: false}, () => {
  // the same layout as templates/repo/issue/sidebar/label_list.tmpl, items 3 and 4 share a scope
  function createLabelCombo(value: string, mode = 'multiple') {
    const container = createElementFromHTML<HTMLElement>(`<div class="issue-sidebar-combo" data-selection-mode="${mode}" data-update-algo="diff">
  <input class="combo-value" type="hidden" value="${value}">
  <div class="ui dropdown">
    <a class="fixed-text">Labels</a>
    <div class="menu">
      <div class="ui icon search input"><input type="text"></div>
      <div class="scrolling menu">
        <a class="item clear-selection" href="#">Clear labels</a>
        <a class="item" href="#" data-value="1"><span class="item-check-mark"></span><span class="ui label">bug</span><div class="item-secondary-info"><small>Something is broken</small></div></a>
        <a class="item" href="#" data-value="2"><span class="item-check-mark"></span><span class="ui label">docs</span></a>
        <a class="item" href="#" data-scope="priority" data-value="3"><span class="item-check-mark"></span><span class="ui label">priority/high</span></a>
        <a class="item" href="#" data-scope="priority" data-value="4"><span class="item-check-mark"></span><span class="ui label">priority/low</span></a>
      </div>
    </div>
  </div>
</div>`);
    document.body.append(container);
    new IssueSidebarComboList(container).init();
    return container;
  }
  const item = (container: Element, value: string) => container.querySelector<HTMLElement>(`.item[data-value="${CSS.escape(value)}"]`)!;
  const ariaSelected = (container: Element) => Array.from(container.querySelectorAll('.scrolling.menu > .item:not(.clear-selection)'), (el) => el.getAttribute('aria-selected'));
  const announced = () => Array.from(document.querySelectorAll('[role="status"] > div'), (el) => el.textContent);

  beforeAll(() => {
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

  test('AC1: multiselectable popup and initial aria-selected, not announced', () => {
    const container = createLabelCombo('2,3');
    expect(container.querySelector('.ui.dropdown > .menu')!.getAttribute('aria-multiselectable')).toBe('true');
    expect(ariaSelected(container)).toEqual(['false', 'true', 'true', 'false']);
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual([]);
  });

  test('AC2/AC3: toggling updates aria-selected and announces the item name', () => {
    const container = createLabelCombo('');
    item(container, '1').click();
    expect(ariaSelected(container)).toEqual(['true', 'false', 'false', 'false']);
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual(['Selected "bug"']);
    item(container, '1').click();
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual(['Deselected "bug"']);
  });

  test('AC3: a scoped label announces the one it replaced', () => {
    const container = createLabelCombo('3');
    item(container, '4').click();
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual(['Deselected "priority/high"', 'Selected "priority/low"']);
  });

  test('AC3: clearing announces every removed item', () => {
    const container = createLabelCombo('1,2');
    container.querySelector<HTMLElement>('.clear-selection')!.click();
    vi.advanceTimersByTime(100);
    expect(announced()).toEqual(['Deselected "bug"', 'Deselected "docs"']);
  });

  test('AC5: the form value is the same as before', () => {
    const container = createLabelCombo('1');
    item(container, '2').click();
    item(container, '4').click();
    item(container, '1').click();
    expect(container.querySelector<HTMLInputElement>('.combo-value')!.value).toBe('2,4');
  });

  test('single selection pickers are not multiselectable and do not announce', () => {
    const container = createLabelCombo('', 'single');
    item(container, '1').click();
    vi.advanceTimersByTime(100);
    expect(container.querySelector('.ui.dropdown > .menu')!.hasAttribute('aria-multiselectable')).toBe(false);
    expect(item(container, '1').getAttribute('aria-selected')).toBe('true');
    expect(announced()).toEqual([]);
  });
});
