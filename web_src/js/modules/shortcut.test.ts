import {ShortcutRegistry, getShortcuts, groupShortcuts, initGlobalShortcut, initShortcutHelp, initShortcutKbd, initShortcutLink} from './shortcut.ts';

function setup() {
  const registry = new ShortcutRegistry();
  const root = document.createElement('div');
  root.innerHTML = `
    <button type="button">button</button>
    <input><textarea></textarea><select></select><div contenteditable="true"></div>
  `;
  root.addEventListener('keydown', (e) => registry.handleKeyDown(e));
  document.body.append(root);
  const button = root.querySelector('button')!;
  const add = (keys: string, opts: {element?: HTMLElement} = {}) => {
    const action = vi.fn();
    registry.register({keys, description: `run ${keys}`, action, ...opts});
    return action;
  };
  return {registry, root, button, add, cleanup: () => root.remove()};
}

function press(el: Element, key: string, opts: KeyboardEventInit = {}): KeyboardEvent {
  const e = new KeyboardEvent('keydown', {key, bubbles: true, cancelable: true, ...opts});
  el.dispatchEvent(e);
  return e;
}

// these tests share document.body and fake timers, so they must not run concurrently (see vitest.config.ts)
describe('ShortcutRegistry', {concurrent: false}, () => {
  let ctx: ReturnType<typeof setup>;
  beforeEach(() => {
    vi.useFakeTimers();
    ctx = setup();
  });
  afterEach(() => {
    ctx.cleanup();
    vi.useRealTimers();
  });

  test('AC3 a single key runs its action once and nothing else', () => {
    const action = ctx.add('?');
    expect(press(ctx.button, '?', {shiftKey: true}).defaultPrevented).toBe(true);
    expect(action).toHaveBeenCalledTimes(1);
    expect(press(ctx.button, 'x').defaultPrevented).toBe(false);
  });

  test('AC3 keys match regardless of case', () => {
    const action = ctx.add('S');
    press(ctx.button, 'S', {shiftKey: true});
    press(ctx.button, 's');
    expect(action).toHaveBeenCalledTimes(2);
  });

  test('AC4 pressing Shift in the middle of a sequence does not drop it', () => {
    const sequence = ctx.add('g c');
    press(ctx.button, 'g');
    press(ctx.button, 'Shift', {shiftKey: true});
    press(ctx.button, 'C', {shiftKey: true});
    expect(sequence).toHaveBeenCalledTimes(1);
  });

  test('AC4 a sequence runs once and its keys run no single-key shortcut', () => {
    const sequence = ctx.add('g c');
    const single = ctx.add('c');
    expect(press(ctx.button, 'g').defaultPrevented).toBe(true);
    vi.advanceTimersByTime(1500);
    expect(press(ctx.button, 'c').defaultPrevented).toBe(true);
    expect(sequence).toHaveBeenCalledTimes(1);
    expect(single).not.toHaveBeenCalled();
  });

  test('AC3 AC4 holding a key down runs its shortcut once', () => {
    const help = ctx.add('?');
    const sequence = ctx.add('g c');
    const single = ctx.add('c');
    press(ctx.button, '?', {shiftKey: true});
    press(ctx.button, '?', {shiftKey: true, repeat: true});
    press(ctx.button, 'g');
    press(ctx.button, 'c');
    press(ctx.button, 'c', {repeat: true});
    expect(help).toHaveBeenCalledTimes(1);
    expect(sequence).toHaveBeenCalledTimes(1);
    expect(single).not.toHaveBeenCalled();
  });

  test('AC5 waiting longer than the timeout drops the first key', () => {
    const sequence = ctx.add('g c');
    const single = ctx.add('c');
    press(ctx.button, 'g');
    vi.advanceTimersByTime(1501);
    press(ctx.button, 'c');
    expect(sequence).not.toHaveBeenCalled();
    expect(single).toHaveBeenCalledTimes(1);
  });

  test('AC5 a key that finishes no sequence drops the first key and acts as a fresh press', () => {
    const sequence = ctx.add('g c');
    const single = ctx.add('x');
    press(ctx.button, 'g');
    press(ctx.button, 'x');
    press(ctx.button, 'c');
    expect(sequence).not.toHaveBeenCalled();
    expect(single).toHaveBeenCalledTimes(1);
    expect(press(ctx.button, 'z').defaultPrevented).toBe(false);
  });

  test('AC6 keys typed into a field run no shortcut', () => {
    const action = ctx.add('s');
    for (const field of ctx.root.querySelectorAll('input, textarea, select, [contenteditable]')) {
      expect(press(field, 's').defaultPrevented).toBe(false);
    }
    expect(action).not.toHaveBeenCalled();
  });

  test('AC6 typing into a field drops a started sequence', () => {
    const sequence = ctx.add('g c');
    press(ctx.button, 'g');
    press(ctx.root.querySelector('input')!, 'a');
    press(ctx.button, 'c');
    expect(sequence).not.toHaveBeenCalled();
  });

  test('AC7 a key a widget already used runs no shortcut and drops a started sequence', () => {
    const sequence = ctx.add('g c');
    const single = ctx.add('x');
    ctx.button.addEventListener('keydown', (e) => e.preventDefault(), {once: true});
    press(ctx.button, 'x');
    expect(single).not.toHaveBeenCalled();

    press(ctx.button, 'g');
    ctx.button.addEventListener('keydown', (e) => e.preventDefault(), {once: true});
    press(ctx.button, 'b');
    press(ctx.button, 'c');
    expect(sequence).not.toHaveBeenCalled();
  });

  test('keys pressed with Ctrl, Alt or Cmd run no shortcut', () => {
    const action = ctx.add('s');
    for (const modifier of ['ctrlKey', 'altKey', 'metaKey']) {
      expect(press(ctx.button, 's', {[modifier]: true}).defaultPrevented).toBe(false);
    }
    expect(action).not.toHaveBeenCalled();
  });

  test('AC8 adding the same keys, or keys that start one another, throws', () => {
    ctx.add('s');
    ctx.add('g c');
    expect(() => ctx.add('S')).toThrow('conflicts with "s"');
    expect(() => ctx.add('g')).toThrow('conflicts with "g c"');
    expect(() => ctx.add('g c x')).toThrow('conflicts with "g c"');
    expect(() => ctx.add('c')).not.toThrow();
  });

  test('AC4 AC8 extra spaces in the keys are ignored', () => {
    const sequence = ctx.add(' g  c ');
    expect(() => ctx.add('g c')).toThrow('conflicts with "g c"');
    press(ctx.button, 'g');
    press(ctx.button, 'c');
    expect(sequence).toHaveBeenCalledTimes(1);
  });

  test('AC8 a shortcut whose element left the page stops working and gives way to a new one', () => {
    const oldElem = document.createElement('kbd');
    const newElem = document.createElement('kbd');
    ctx.root.append(oldElem, newElem);
    const oldAction = ctx.add('t', {element: oldElem});
    expect(() => ctx.add('t', {element: newElem})).toThrow('conflicts with "t"');

    oldElem.remove();
    expect(press(ctx.button, 't').defaultPrevented).toBe(false); // before anything new registers
    expect(oldAction).not.toHaveBeenCalled();
    const newAction = ctx.add('t', {element: newElem});
    press(ctx.button, 't');
    expect(oldAction).not.toHaveBeenCalled();
    expect(newAction).toHaveBeenCalledTimes(1);
  });

  test('AC9 listing gives keys, description and section of the shortcuts on the page', () => {
    const elem = document.createElement('kbd');
    ctx.root.append(elem);
    ctx.registry.register({keys: 'g c', description: 'Code', section: 'Repository', action: () => {}});
    ctx.add('t', {element: elem});
    expect(ctx.registry.list()).toEqual([
      {keys: 'g c', description: 'Code', section: 'Repository'},
      {keys: 't', description: 'run t', section: undefined},
    ]);

    elem.remove();
    expect(ctx.registry.list().map((s) => s.keys)).toEqual(['g c']);
  });
});

describe('groupShortcuts', () => {
  test('#57 AC5 groups by section in order of first appearance, the default section first', () => {
    const groups = groupShortcuts([
      {keys: 's', description: 'Search code…', section: 'Repository'},
      {keys: 'x', description: 'No section'},
      {keys: 'r', description: 'Quote reply', section: 'Issue'},
      {keys: 't', description: 'Go to file', section: 'Repository'},
      {keys: '?', description: 'Show keyboard shortcuts', section: 'General'},
    ], 'General');
    expect(Array.from(groups.keys())).toEqual(['General', 'Repository', 'Issue']);
    expect(groups.get('General')!.map((s) => s.keys)).toEqual(['x', '?']);
    expect(groups.get('Repository')!.map((s) => s.keys)).toEqual(['s', 't']);
  });
});

// the markup of templates/repo/home_sidebar_top.tmpl and RepoFileSearch.vue
function renderHint(key: string, placeholder: string, section?: string) {
  const wrapper = document.createElement('div');
  wrapper.className = 'ui input global-shortcut-wrapper';
  wrapper.innerHTML = `<input placeholder="${placeholder}"><kbd data-shortcut-keys="${key}">${key.toUpperCase()}</kbd>`;
  document.body.append(wrapper);
  const input = wrapper.querySelector('input')!;
  const kbd = wrapper.querySelector('kbd')!;
  if (section) kbd.setAttribute('data-shortcut-section', section);
  initShortcutKbd(kbd);
  return {wrapper, input, kbd};
}

// the markup of templates/base/footer.tmpl
function renderHelpDialog() {
  const dialog = document.createElement('dialog');
  dialog.setAttribute('data-shortcut-keys', '?');
  dialog.setAttribute('data-shortcut-description', 'Show keyboard shortcuts');
  dialog.setAttribute('data-shortcut-section', 'General');
  dialog.innerHTML = `<form method="dialog"><h2>Keyboard shortcuts</h2><button aria-label="Close">x</button></form><div class="shortcut-help-sections"></div>`;
  document.body.append(dialog);
  initShortcutHelp(dialog);
  return dialog;
}

// a tab link, whose text also holds an open count
function renderLink(keys: string, href: string, description: string, section?: string, shortcutHref?: string) {
  const link = document.createElement('a');
  link.href = href;
  link.setAttribute('data-shortcut-keys', keys);
  link.setAttribute('data-shortcut-description', description);
  if (section) link.setAttribute('data-shortcut-section', section);
  if (shortcutHref) link.setAttribute('data-shortcut-href', shortcutHref);
  link.innerHTML = `${description} <span class="ui small label">3</span>`;
  document.body.append(link);
  initShortcutLink(link);
  return link;
}

// these tests share document.body and its focus, so they must not run concurrently (see vitest.config.ts)
describe('shortcut hints and links', {concurrent: false}, () => {
  beforeAll(() => {
    initGlobalShortcut();
  });
  afterEach(() => {
    document.body.replaceChildren();
    vi.restoreAllMocks();
  });

  test('AC1 pressing a hint key focuses its box and hides the hint', () => {
    const search = renderHint('s', 'Search code…');
    const findFile = renderHint('t', 'Go to file');
    expect(press(document.body, 's').defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(search.input);
    expect(search.kbd.classList.contains('tw-hidden')).toBe(true);

    search.input.blur();
    press(document.body, 't');
    expect(document.activeElement).toBe(findFile.input);
    expect(findFile.kbd.classList.contains('tw-hidden')).toBe(true);
  });

  test('AC2 Escape in a box with a hint clears it and moves focus away', () => {
    const {input, kbd} = renderHint('s', 'Search code…');
    input.focus();
    input.value = 'needle';
    press(input, 'Escape');
    expect(input.value).toBe('');
    expect(document.activeElement).not.toBe(input);
    expect(kbd.classList.contains('tw-hidden')).toBe(false);
  });

  test('AC8 a hint that loads again keeps its key working', () => {
    const before = renderHint('t', 'Go to file');
    before.wrapper.remove(); // as moving through the file tree replaces the page content
    const after = renderHint('t', 'Go to file');
    press(document.body, 't');
    expect(document.activeElement).toBe(after.input);
  });

  test('AC8 a second hint for the same key on the page throws', () => {
    renderHint('s', 'Search code…');
    expect(() => renderHint('s', 'Search issues…')).toThrow('conflicts with "s"');
  });

  test('AC9 a hint is listed with its box\'s placeholder and its section', () => {
    renderHint('s', 'Search code…', 'Repository');
    renderHint('t', 'Go to file', 'Repository');
    renderHint('x', 'No section');
    expect(getShortcuts()).toEqual([
      {keys: 's', description: 'Search code…', section: 'Repository'},
      {keys: 't', description: 'Go to file', section: 'Repository'},
      {keys: 'x', description: 'No section', section: undefined},
    ]);
  });

  test('#58 AC1 a link\'s shortcut goes where the link leads', () => {
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    renderLink('g i', 'https://gitea.example/owner/repo/issues', 'Issues');
    press(document.body, 'g');
    press(document.body, 'i');
    expect(assign).toHaveBeenCalledExactlyOnceWith('https://gitea.example/owner/repo/issues');
  });

  test('#58 AC2 a link\'s shortcut can go to an address of its own instead', () => {
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    renderLink('g c', 'https://gitea.example/owner/repo/src/branch/feature', 'Code', 'Repository', 'https://gitea.example/owner/repo');
    press(document.body, 'g');
    press(document.body, 'c');
    expect(assign).toHaveBeenCalledExactlyOnceWith('https://gitea.example/owner/repo');
  });

  test('#58 AC5 a link moved to another part of the page keeps its shortcut', () => {
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    const link = renderLink('g p', 'https://gitea.example/owner/repo/pulls', 'Pull requests');
    const menu = document.createElement('div'); // as a narrow tab bar moves tabs into its "more" menu
    menu.style.display = 'none'; // closed
    document.body.append(menu);
    menu.append(link);
    press(document.body, 'g');
    press(document.body, 'p');
    expect(assign).toHaveBeenCalledExactlyOnceWith('https://gitea.example/owner/repo/pulls');
  });

  test('#58 AC8 a link is listed with its own description, not its text, and its section', () => {
    renderLink('g i', 'https://gitea.example/owner/repo/issues', 'Issues', 'Repository');
    expect(getShortcuts()).toEqual([{keys: 'g i', description: 'Issues', section: 'Repository'}]);
  });

  test('#57 AC1 AC5 ? opens a dialog that lists the shortcuts on the page by section', () => {
    renderHint('s', 'Search <code>', 'Repository');
    const dialog = renderHelpDialog();
    const button = document.createElement('button');
    document.body.append(button);
    button.focus();
    expect(press(button, '?', {shiftKey: true}).defaultPrevented).toBe(true);
    expect(dialog.open).toBe(true);
    const sections = Array.from(dialog.querySelectorAll('section'), (section) => ({
      heading: section.querySelector('h3')!.textContent,
      rows: Array.from(section.querySelectorAll('.shortcut-help-row'), (row) => [row.querySelector('dd')!.textContent, row.querySelector('dt')!.textContent]),
    }));
    expect(sections).toEqual([
      {heading: 'General', rows: [['?', 'Show keyboard shortcuts']]},
      {heading: 'Repository', rows: [['S', 'Search <code>']]}, // the description is text, not markup
    ]);

    dialog.close();
    expect(document.activeElement).toBe(button);
  });

  test('#57 AC7 AC8 while the dialog is open, no shortcut runs, ? included', () => {
    const {input} = renderHint('s', 'Search code…');
    const dialog = renderHelpDialog();
    press(document.body, '?', {shiftKey: true});
    const rows = dialog.querySelectorAll('.shortcut-help-row').length;
    expect(press(dialog, 's').defaultPrevented).toBe(false);
    expect(document.activeElement).not.toBe(input);
    expect(press(document.body, '?', {shiftKey: true}).defaultPrevented).toBe(false);
    expect(dialog.open).toBe(true);
    expect(dialog.querySelectorAll('.shortcut-help-row')).toHaveLength(rows);
    dialog.close();
  });
});
