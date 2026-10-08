# Multi-select Dropdown ARIA State and Announcements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Screen reader users hear whether each option of a multi-select dropdown is selected, and
hear "Selected …" / "Deselected …" when they change it, in both Fomantic multiple-selection
dropdowns and the issue sidebar's multi-select pickers (issue #63).

**Architecture:** A new `web_src/js/modules/aria-announce.ts` owns one polite live region on
`<body>`. `web_src/js/modules/fomantic/dropdown.ts` (the existing Fomantic a11y patch) gains
`aria-multiselectable`/`aria-selected` for `.multiple` dropdowns, wraps Fomantic's `onAdd` /
`onRemove` to announce, names delete icons by visible text, and gives roles to items in a nested
`.scrolling.menu`. `web_src/js/features/repo-issue-sidebar-combolist.ts` keeps `aria-selected` in
step with its own `.checked` class and announces from `onChange`.

**Tech Stack:** TypeScript, jQuery + hard-forked Fomantic UI 2.8.7, Vitest (jsdom), Playwright.

**Spec:** GitHub issue CSCI-435-SE/gitea#63, plus the design decisions by @vvchopra drafted in
`docs/sprint1/issue-63-design-decisions.md` (Decisions 2-5).

---

## Background the engineer needs

- `.claude-students/docs/frontend-js.md` and `.claude-students/docs/i18n.md`.
- `web_src/js/modules/fomantic/aria.md` explains why the patch exists.
- **Two different multi-select mechanisms.** Fomantic `ui multiple … dropdown` (repo topics,
  protected-branch/tag user and team pickers) tracks selection in its own state and marks chosen
  menu items with the `active` class; it calls `settings.onAdd(value, text, $item)` and
  `settings.onRemove(value, text, $item)` but **not on the initial load**
  (`web_src/fomantic/build/components/dropdown.js:2970`, `:3127`), which is why announcing from
  there never reads out pre-selected values. The issue sidebar (`templates/repo/issue/sidebar/*_list.tmpl`)
  is a plain `ui dropdown` whose items live in a nested `.scrolling.menu`; selection is the
  `.checked` class toggled by `IssueSidebarComboList.onItemClick`.
- `ariaDropdownFn` calls Fomantic first and only then patches, so labels and items rendered during
  Fomantic's own initialisation already exist when `attachStaticElements` runs.
- Vitest runs with `globals: true` (`test`, `expect`, `vi` need no import). `window.config.i18n`
  is `{}` in tests (`web_src/js/vitest.setup.ts`), so every test touching i18n sets its strings.
- No `make` on this machine. Commands: `pnpm exec vitest run <path>`,
  `pnpm exec eslint --color --max-warnings=0 web_src/js tools *.ts tests/e2e`, `pnpm exec vue-tsc`.
- **The human commits.** Each "Commit" step means: stop, show the diff summary and hand over the
  command; do not run `git commit`.

## File map

| File | Change |
| --- | --- |
| `web_src/js/modules/aria-announce.ts` | **create**: shared live region, `announce`, `announceSelectionChange` |
| `web_src/js/modules/aria-announce.test.ts` | **create** |
| `options/locale/locale_en-US.json` | add `selected_item_str`, `deselected_item_str` next to `remove_label_str` |
| `templates/base/head_script.tmpl` | expose the two keys on `window.config.i18n` |
| `web_src/js/modules/fomantic/dropdown.ts` | AC1-AC4 for Fomantic multiples; nested item roles |
| `web_src/js/modules/fomantic/dropdown.test.ts` | apply the patch; new tests |
| `web_src/js/features/repo-issue-sidebar-combolist.ts` | `aria-multiselectable`, `aria-selected`, announcements |
| `web_src/js/features/repo-issue-sidebar-combolist.test.ts` | new tests |
| `web_src/js/modules/fomantic/aria.md` | AC6 |
| `tests/e2e/multiselect-aria.test.ts` | **create**: AC5 keyboard flows |

---

### Task 1: Shared announcer and its strings

**Files:**
- Create: `web_src/js/modules/aria-announce.ts`
- Create: `web_src/js/modules/aria-announce.test.ts`
- Modify: `options/locale/locale_en-US.json:91`
- Modify: `templates/base/head_script.tmpl:23`

- [ ] **Step 1: Write the failing test** — `web_src/js/modules/aria-announce.test.ts`

```ts
import {announce, announceSelectionChange} from './aria-announce.ts';

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const liveRegions = () => document.querySelectorAll('[role="status"]');

test('announce-uses-one-region-on-body', () => {
  announce('first');
  vi.advanceTimersByTime(100);
  announce('second');
  vi.advanceTimersByTime(100);
  expect(liveRegions()).toHaveLength(1);
  expect(liveRegions()[0].parentElement).toBe(document.body);
  expect(liveRegions()[0].textContent).toBe('second');
});

test('announce-batches-messages-in-the-same-window', () => {
  announce('a deselected');
  announce('b selected');
  expect(liveRegions()[0].textContent).toBe('');
  vi.advanceTimersByTime(100);
  expect(Array.from(liveRegions()[0].children, (el) => el.textContent)).toEqual(['a deselected', 'b selected']);
});

test('announce-recreates-a-removed-region', () => {
  document.body.innerHTML = '';
  announce('again');
  vi.advanceTimersByTime(100);
  expect(liveRegions()).toHaveLength(1);
  expect(liveRegions()[0].textContent).toBe('again');
});

test('announceSelectionChange-uses-localised-strings', () => {
  window.config.i18n.selected_item_str = 'Selected "%s"';
  window.config.i18n.deselected_item_str = 'Deselected "%s"';
  announceSelectionChange('bug', true);
  announceSelectionChange('docs', false);
  vi.advanceTimersByTime(100);
  expect(Array.from(liveRegions()[0].children, (el) => el.textContent)).toEqual(['Selected "bug"', 'Deselected "docs"']);
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `pnpm exec vitest run web_src/js/modules/aria-announce.test.ts`
Expected: FAIL — cannot resolve `./aria-announce.ts`.

- [ ] **Step 3: Implement** — `web_src/js/modules/aria-announce.ts`

```ts
// One polite live region for the whole page, appended to <body> on first use: the issue sidebar
// replaces its own DOM after saving, which would destroy a region placed inside it.
let liveRegion: HTMLElement | null = null;
let pendingMessages: Array<string> = [];
let flushTimer: ReturnType<typeof setTimeout> | undefined;

function getLiveRegion(): HTMLElement {
  if (liveRegion?.isConnected) return liveRegion;
  liveRegion = document.createElement('div');
  liveRegion.className = 'tw-sr-only';
  liveRegion.setAttribute('role', 'status'); // implies aria-live="polite"
  document.body.append(liveRegion);
  return liveRegion;
}

// Messages within the same short window are read together (a scoped label replacing another is two
// changes). The region is emptied first so that repeating the previous message is announced again.
export function announce(message: string) {
  const region = getLiveRegion();
  region.textContent = '';
  pendingMessages.push(message);
  clearTimeout(flushTimer);
  flushTimer = setTimeout(() => {
    region.replaceChildren(...pendingMessages.map((msg) => {
      const el = document.createElement('div');
      el.textContent = msg;
      return el;
    }));
    pendingMessages = [];
  }, 100);
}

export function announceSelectionChange(itemName: string, selected: boolean) {
  const i18n = window.config.i18n;
  announce((selected ? i18n.selected_item_str : i18n.deselected_item_str).replace('%s', itemName));
}
```

- [ ] **Step 4: Add the strings.** In `options/locale/locale_en-US.json`, directly after the
`"remove_label_str"` line:

```json
  "selected_item_str": "Selected \"%s\"",
  "deselected_item_str": "Deselected \"%s\"",
```

In `templates/base/head_script.tmpl`, directly after the `remove_label_str:` line (tabs, matching
the file):

```
			selected_item_str: {{ctx.Locale.Tr "selected_item_str"}},
			deselected_item_str: {{ctx.Locale.Tr "deselected_item_str"}},
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `pnpm exec vitest run web_src/js/modules/aria-announce.test.ts`
Expected: 4 passed.

- [ ] **Step 6: Commit** (hand to the human)

```bash
git add web_src/js/modules/aria-announce.ts web_src/js/modules/aria-announce.test.ts options/locale/locale_en-US.json templates/base/head_script.tmpl
git commit -m "feat(a11y): add a shared live region for screen reader announcements (#63)" -m "One polite status region on <body>, created on first use, so it survives the issue sidebar replacing its DOM. Messages in the same 100ms window are read together." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 2: Name delete icons by visible text (AC4, Decision 5)

**Files:**
- Modify: `web_src/js/modules/fomantic/dropdown.ts:56-69` (`updateSelectionLabel`)
- Modify: `web_src/js/modules/fomantic/dropdown.test.ts`

- [ ] **Step 1: Apply the patch in the test file and add a fixture.** Change the import at the top
of `dropdown.test.ts` and add the setup below the imports:

```ts
import '../../../fomantic/build/fomantic.js';
import {createElementFromHTML} from '../../utils/dom.ts';
import {hideScopedEmptyDividers, initAriaDropdownPatch} from './dropdown.ts';

$.fn.fomanticExt = {};
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
```

- [ ] **Step 2: Write the failing test** (append to `dropdown.test.ts`)

```ts
describe('multiple selection aria', () => {
  beforeAll(() => {
    window.config.i18n.remove_label_str = 'Remove item "%s"';
    window.config.i18n.selected_item_str = 'Selected "%s"';
    window.config.i18n.deselected_item_str = 'Deselected "%s"';
  });
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = '';
  });

  test('AC4: delete icons are named by the label text, not the value', () => {
    const el = createMultipleDropdown('2');
    expect(el.querySelector('.ui.label .delete.icon')!.getAttribute('aria-label')).toBe('Remove item "Beta"');
    $(el).dropdown('set selected', '1');
    const labelIcons = Array.from(el.querySelectorAll('.ui.label .delete.icon'), (icon) => icon.getAttribute('aria-label'));
    expect(labelIcons).toContain('Remove item "Alpha"');
  });
});
```

- [ ] **Step 3: Run it to verify it fails**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts`
Expected: FAIL — `expected 'Remove item "2"' to be 'Remove item "Beta"'`. The existing 5 tests
must still pass with the patch applied; if `dropdown-item-literal-text` breaks, stop and report.

- [ ] **Step 4: Implement.** In `updateSelectionLabel`, replace the `aria-label` line:

```ts
  const deleteIcon = label.querySelector('.delete.icon');
  if (deleteIcon) {
    // the value can be an internal ID (the protected branch user and team pickers), so name it by what is shown
    const labelName = label.textContent.trim() || label.getAttribute('data-value')!;
    deleteIcon.setAttribute('aria-hidden', 'false');
    deleteIcon.setAttribute('aria-label', window.config.i18n.remove_label_str.replace('%s', labelName));
    deleteIcon.setAttribute('role', 'button');
  }
```

- [ ] **Step 5: Run to verify it passes**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts`
Expected: all passed.

- [ ] **Step 6: Commit** (hand to the human)

```bash
git add web_src/js/modules/fomantic/dropdown.ts web_src/js/modules/fomantic/dropdown.test.ts
git commit -m "fix(a11y): name selection label delete icons by their visible text (#63)" -m "The name was built from data-value, which is a user or team ID in the protected branch pickers, so screen readers heard e.g. Remove item \"12\"." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 3: Selection state and announcements for Fomantic multiples (AC1-AC3)

**Files:**
- Modify: `web_src/js/modules/fomantic/dropdown.ts`
- Modify: `web_src/js/modules/fomantic/dropdown.test.ts`

- [ ] **Step 1: Write the failing tests** (inside the `multiple selection aria` describe)

```ts
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
```

- [ ] **Step 2: Run to verify they fail**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts`
Expected: AC1, AC2 and AC3 FAIL (no `aria-multiselectable`, `aria-selected` is `null`, nothing
announced); the single-selection test passes.

- [ ] **Step 3: Implement.** In `dropdown.ts`:

Add the import:

```ts
import {announceSelectionChange} from '../aria-announce.ts';
```

Add below `updateSelectionLabel`:

```ts
// Fomantic marks the chosen items of a multiple selection dropdown "active"
function refreshAriaSelected(dropdown: HTMLElement) {
  if (!dropdown.classList.contains('multiple')) return;
  for (const item of dropdown.querySelectorAll(':scope > .menu > .item')) {
    item.setAttribute('aria-selected', item.classList.contains('active') ? 'true' : 'false');
  }
}
```

In `delegateDropdownModule`, inside the `dropdownTemplates.menu` wrapper, after the
`deferredRefreshAriaActiveItem()` call (AJAX items arrive after the static pass):

```ts
    setTimeout(() => refreshAriaSelected($dropdown[0]), 0);
```

In `delegateDropdownModule`, after the `onLabelCreate` block:

```ts
  // Fomantic doesn't call onAdd/onRemove on the initial load, so pre-selected items stay silent
  for (const [callbackName, selected] of [['onAdd', true], ['onRemove', false]] as const) {
    const callbackOld = dropdownCall('setting', callbackName);
    dropdownCall('setting', callbackName, function(this: any, value: string, text: string, $item: any) {
      const ret = callbackOld.call(this, value, text, $item);
      if ($dropdown[0].classList.contains('multiple')) {
        // a value typed by the user and removed by its label has no menu item
        announceSelectionChange($item?.[0]?.textContent.trim() || value, selected);
        setTimeout(() => refreshAriaSelected($dropdown[0]), 0); // Fomantic updates the "active" classes after the callback
      }
      return ret;
    });
  }
```

In `attachStaticElements`, directly after `menu.setAttribute('role', …)`:

```ts
  if (dropdown.classList.contains('multiple')) menu.setAttribute('aria-multiselectable', 'true');
  refreshAriaSelected(dropdown);
```

- [ ] **Step 4: Run to verify they pass**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts`
Expected: all passed. If AC1 fails only on the `['false', 'true']` assertion, Fomantic marks the
initial items after the patch runs: also call `refreshAriaSelected(dropdown)` at the end of
`refreshAriaActiveItem` in `attachDomEvents`, and re-run.

- [ ] **Step 5: Commit** (hand to the human)

```bash
git add web_src/js/modules/fomantic/dropdown.ts web_src/js/modules/fomantic/dropdown.test.ts
git commit -m "feat(a11y): expose and announce the selection state of multiple selection dropdowns (#63)" -m "The listbox carries aria-multiselectable and every option aria-selected, refreshed after each add or remove. Additions and removals are announced through the shared live region; Fomantic skips these callbacks on the initial load, so existing selections are not read out." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 4: Roles for items in a nested `.scrolling.menu`

The issue sidebar's options sit in `.menu > .scrolling.menu > .item`, which the `> .item` pass
never reaches: they have no `role`, no `id`, and are never the `aria-activedescendant`, so arrowing
through them is silent no matter what `aria-selected` says.

**Files:**
- Modify: `web_src/js/modules/fomantic/dropdown.ts` (`attachStaticElements`, `refreshAriaActiveItem`)
- Modify: `web_src/js/modules/fomantic/dropdown.test.ts`

- [ ] **Step 1: Write the failing test** (top level of `dropdown.test.ts`)

```ts
test('nested-scrolling-menu-items-get-option-roles', () => {
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
  el.remove();
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts -t nested`
Expected: FAIL — `role` is `null`.

- [ ] **Step 3: Implement.** In `attachStaticElements`:

```ts
  $(menu).find('> .item, > .scrolling.menu > .item').each((_, item) => updateMenuItem(dropdown, item));
```

In `refreshAriaActiveItem`:

```ts
    const active = $(menu).find('> .item.active, > .item.selected, > .scrolling.menu > .item.selected')[0];
```

- [ ] **Step 4: Run the whole file**

Run: `pnpm exec vitest run web_src/js/modules/fomantic/dropdown.test.ts`
Expected: all passed.

- [ ] **Step 5: Commit** (hand to the human)

```bash
git add web_src/js/modules/fomantic/dropdown.ts web_src/js/modules/fomantic/dropdown.test.ts
git commit -m "fix(a11y): give dropdown items in a nested scrolling menu their aria role (#63)" -m "The issue sidebar pickers keep their items in .menu > .scrolling.menu, which the patch never reached, so the items had no role or id and were never the active descendant." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 5: Issue sidebar pickers (AC1-AC3, AC5; Decisions 2 and 4)

**Files:**
- Modify: `web_src/js/features/repo-issue-sidebar-combolist.ts`
- Modify: `web_src/js/features/repo-issue-sidebar-combolist.test.ts`

- [ ] **Step 1: Write the failing tests.** Change the imports at the top of the test file:

```ts
import '../../fomantic/build/fomantic.js';
import {IssueSidebarComboList, syncIssueMainContentTimelineItems} from './repo-issue-sidebar-combolist.ts';
import {createElementFromHTML} from '../utils/dom.ts';
```

Append:

```ts
describe('IssueSidebarComboList aria', () => {
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
  const item = (container: Element, value: string) => container.querySelector<HTMLElement>(`.item[data-value="${value}"]`)!;
  const ariaSelected = (container: Element) => Array.from(container.querySelectorAll('.scrolling.menu > .item:not(.clear-selection)'), (el) => el.getAttribute('aria-selected'));
  const announced = () => Array.from(document.querySelectorAll('[role="status"] > div'), (el) => el.textContent);

  beforeAll(() => {
    window.config.i18n.selected_item_str = 'Selected "%s"';
    window.config.i18n.deselected_item_str = 'Deselected "%s"';
  });
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = '';
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
```

- [ ] **Step 2: Run to verify they fail**

Run: `pnpm exec vitest run web_src/js/features/repo-issue-sidebar-combolist.test.ts`
Expected: the new aria tests FAIL (`aria-multiselectable`/`aria-selected` are `null`, nothing
announced); `AC5` and the two existing tests pass.

- [ ] **Step 3: Implement.** In `repo-issue-sidebar-combolist.ts`, add the import:

```ts
import {announceSelectionChange} from '../modules/aria-announce.ts';
```

Add these methods to `IssueSidebarComboList`, after `collectCheckedValues`:

```ts
  // keep each item's aria-selected in step with its "checked" class, and return the items that changed
  syncAriaSelected(): Array<HTMLElement> {
    const changed: Array<HTMLElement> = [];
    for (const el of this.elDropdown.querySelectorAll<HTMLElement>('.menu > .item:not(.clear-selection)')) {
      const selected = el.classList.contains('checked') ? 'true' : 'false';
      if (el.getAttribute('aria-selected') === selected) continue;
      if (el.hasAttribute('aria-selected')) changed.push(el); // the first sync on init is not a change
      el.setAttribute('aria-selected', selected);
    }
    return changed;
  }

  getItemName(elItem: HTMLElement): string {
    const el = elItem.cloneNode(true) as HTMLElement;
    queryElems(el, '.item-check-mark, .item-secondary-info', (el) => el.remove());
    return el.textContent.trim().replace(/\s+/g, ' ');
  }
```

At the start of `onChange`:

```ts
  async onChange() {
    const changedItems = this.syncAriaSelected();
    if (this.selectionMode === 'multiple') {
      for (const el of changedItems) announceSelectionChange(this.getItemName(el), el.classList.contains('checked'));
    }
    if (this.selectionMode === 'single') {
```

In `init`, after `this.initialValues = this.collectCheckedValues();`:

```ts
    if (this.selectionMode === 'multiple') this.elDropdown.querySelector(':scope > .menu')!.setAttribute('aria-multiselectable', 'true');
    this.syncAriaSelected();
```

- [ ] **Step 4: Run to verify they pass**

Run: `pnpm exec vitest run web_src/js/features/repo-issue-sidebar-combolist.test.ts`
Expected: all passed.

- [ ] **Step 5: Commit** (hand to the human)

```bash
git add web_src/js/features/repo-issue-sidebar-combolist.ts web_src/js/features/repo-issue-sidebar-combolist.test.ts
git commit -m "feat(issues): expose and announce selection state in the issue sidebar pickers (#63)" -m "Labels, assignees, reviewers and projects keep aria-selected in step with their checked class, and multiple-mode pickers announce each toggle, including a scoped label replaced by another and a cleared selection." -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 6: Update `aria.md` (AC6)

**Files:**
- Modify: `web_src/js/modules/fomantic/aria.md` (the line "Multiple selection dropdown is not well-supported yet, it needs more work.")

- [ ] **Step 1: Confirm the "still not working" item.** On the devtest page or in a quick check,
confirm that Left/Right arrow navigation between selection labels in a Fomantic multiple dropdown
does not move `aria-activedescendant` (`refreshAriaActiveItem` only looks at menu items). Keep the
bullet below only if that holds; otherwise replace it with what you actually found.

- [ ] **Step 2: Replace that line with:**

```markdown
Multiple selection dropdowns (`ui multiple ... dropdown`) are partially supported:

* the listbox has `aria-multiselectable="true"` and every option has `aria-selected`,
  refreshed after each add or remove (Fomantic marks the chosen items `active`)
* adding or removing an item is announced through the shared live region in
  `web_src/js/modules/aria-announce.ts`; Fomantic doesn't call `onAdd`/`onRemove` on the
  initial load, so existing selections are not read out
* a selection label's delete icon is named after the label's visible text

Still not working: moving between selection labels with Left/Right is not announced, because
`aria-activedescendant` only ever points at menu items.

The issue sidebar's label, assignee, reviewer and project pickers are not Fomantic multiple
dropdowns: selection there is the `checked` class managed by
`web_src/js/features/repo-issue-sidebar-combolist.ts`, which keeps `aria-selected` and the
announcements in step itself. Their items sit in a nested `.scrolling.menu`, which the patch
also covers.

The Fomantic part of this is temporary by design: it goes away when these dropdowns move off
Fomantic (see #61), while the sidebar part stays unless the sidebar itself is rewritten.
```

- [ ] **Step 3: Commit** (hand to the human)

```bash
git add web_src/js/modules/fomantic/aria.md
git commit -m "docs(a11y): describe what multiple selection dropdowns now support (#63)" -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 7: Keyboard e2e for labels and topics (AC5)

**Files:**
- Create: `tests/e2e/multiselect-aria.test.ts`

- [ ] **Step 1: Write the test**

```ts
import {env} from 'node:process';
import {test, expect} from '@playwright/test';
import {login, apiCreateRepo, apiCreateIssue, apiDeleteRepo, apiHeaders, baseUrl, randomString} from './utils.ts';

test('issue labels can be selected and deselected by keyboard', async ({page}) => {
  const repoName = `e2e-label-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName, autoInit: false})]);
  try {
    await Promise.all([
      ...['bug', 'docs'].map((name) => page.request.post(`${baseUrl()}/api/v1/repos/${user}/${repoName}/labels`, {
        headers: apiHeaders(), data: {name, color: '#ee0701'},
      })),
      apiCreateIssue(page.request, {owner: user, repo: repoName, title: 'Label keyboard test'}),
    ]);
    await page.goto(`/${user}/${repoName}/issues/1`);
    const dropdown = page.locator('.issue-sidebar-combo:has(input[name="label_ids"]) > .ui.dropdown');
    const bug = dropdown.locator('.scrolling.menu > .item', {hasText: 'bug'});
    const liveRegion = page.locator('body > [role="status"]');

    await dropdown.focus(); // the patch opens the menu on focus
    await expect(bug).toHaveAttribute('role', 'option');
    await expect(bug).toHaveAttribute('aria-selected', 'false');
    for (let i = 0; i < 5 && !(await bug.evaluate((el) => el.classList.contains('selected'))); i++) {
      await page.keyboard.press('ArrowDown');
    }
    await page.keyboard.press('Enter');
    await expect(bug).toHaveAttribute('aria-selected', 'true');
    await expect(liveRegion).toHaveText('Selected "bug"');

    await page.keyboard.press('Enter');
    await expect(bug).toHaveAttribute('aria-selected', 'false');
    await expect(liveRegion).toHaveText('Deselected "bug"');

    await page.keyboard.press('Enter');
    await page.keyboard.press('Escape'); // closing the menu saves the selection
    await expect(page.locator('.issue-sidebar-combo:has(input[name="label_ids"]) .ui.list')).toContainText('bug');
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
});

test('repo topics can be added and removed by keyboard', async ({page}) => {
  const repoName = `e2e-topic-aria-${randomString(8)}`;
  const user = env.GITEA_TEST_E2E_USER;
  await Promise.all([login(page), apiCreateRepo(page.request, {name: repoName})]);
  try {
    await page.goto(`/${user}/${repoName}`);
    await page.locator('#manage_topic').click(); // focuses the topic search input
    const liveRegion = page.locator('body > [role="status"]');
    for (const topic of ['alpha', 'beta']) {
      await page.keyboard.type(topic);
      await expect(page.locator('#topic_edit .menu > .item', {hasText: topic}).first()).toBeVisible();
      await page.keyboard.press('Enter');
      await expect(liveRegion).toHaveText(`Selected "${topic}"`);
    }
    await expect(page.locator('#topic_edit .ui.label .delete.icon').first()).toHaveAttribute('aria-label', /alpha/);
    await page.keyboard.press('Backspace'); // first press selects the last label, the second removes it
    await page.keyboard.press('Backspace');
    await expect(liveRegion).toHaveText('Deselected "beta"');
    await page.locator('#save_topic').click();
    await expect(page.locator('#repo-topics')).toContainText('alpha');
    await expect(page.locator('#repo-topics')).not.toContainText('beta');
  } finally {
    await apiDeleteRepo(page.request, user, repoName);
  }
});
```

- [ ] **Step 2: Build and run** (see memory `gitea-e2e-needs-bindata-build`; this takes minutes)

```bash
pnpm exec vite build
go generate -tags bindata ./modules/public/... ./modules/options/... ./modules/templates/... ./modules/migration/...
go build -tags bindata -o gitea-e2e.exe
EXECUTABLE=gitea-e2e.exe PLAYWRIGHT_MODE=local bash ./tools/test-e2e.sh run --project=chromium tests/e2e/multiselect-aria.test.ts
```

Expected: 2 passed. If a keyboard step doesn't behave as written (Fomantic's Backspace or
focus-to-open behaviour), fix the **test's** keystrokes to match real behaviour, never the
assertions about `aria-selected`, the live region or the saved values, and note the change in the
PR.

- [ ] **Step 3: Commit** (hand to the human)

```bash
git add tests/e2e/multiselect-aria.test.ts
git commit -m "test(a11y): cover keyboard labelling and topic editing end to end (#63)" -m "Assisted-by: Claude Code:claude-opus-5-5"
```

---

### Task 8: Verify and hand off

- [ ] **Step 1: Full frontend checks**

```bash
pnpm exec vitest run
pnpm exec eslint --color --max-warnings=0 web_src/js tools *.ts tests/e2e
pnpm exec vue-tsc
node tools/lint-templates-svg.ts
./.claude-students/check.sh
```

Expected: all pass. Also check no trailing whitespace in edited files: `git diff --check main`.

- [ ] **Step 2: Paste the spec additions.** Remind the human to paste
`docs/sprint1/issue-63-design-decisions.md` into issue #63 (and apply its AC edits), then delete
that file; it is not committed.

- [ ] **Step 3: Draft the PR** in the course format (`.claude-students/COURSE-WORKFLOW.md`):
`Closes #63`, what changed, test strategy (unit tests per AC + the e2e file), design line pointing
at Decisions 2-5, and the AI assistance section per `.claude-students/AI-ASSISTANCE.md` §2 with no
log link. Title: `feat(a11y): announce and expose multi-select dropdown state (#63)`.

- [ ] **Step 4: Logging reminders.** `specstory sync`, file the log under
`ai-logs/sprint1/vvchopra/` (copy and rename only), and draft the AI Assistance issue comment for
#63.

---

## Addendum: changes made while executing

- **Test setup.** `vitest.config.ts` runs tests concurrently (`sequence.concurrent: true`) with
  `isolate: false`, so the DOM-sharing groups use `describe(name, {concurrent: false}, ...)`.
  `dropdown.test.ts` also calls `initFomanticTransition()`, because Fomantic shows labels with a
  transition. The announcer drops messages queued for a region that has since been removed.
- **Task 4 test** sits inside the sequential group rather than at top level.
- **Task 7** found two focus bugs that already exist on `main`. Both are fixed under Decision 6
  (`docs/sprint1/issue-63-design-decisions.md`):
  - a `GITEA-PATCH` at the IE11 blur in `web_src/fomantic/build/components/dropdown.js`,
    tested in `dropdown.test.ts`;
  - `initGlobalEnterQuickSubmit` skips an Enter that has already been handled, tested in the new
    `web_src/js/features/common-form.test.ts`.

  In the topic editor, one Backspace removes the last label, so the e2e presses it once.
