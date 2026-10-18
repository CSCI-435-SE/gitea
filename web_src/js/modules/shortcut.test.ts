import {ShortcutRegistry, initGlobalShortcut, initShortcutKbd} from './shortcut.ts';

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

  test('AC3 keys match regardless of case, and Shift alone is not a press', () => {
    const action = ctx.add('S');
    press(ctx.button, 'Shift', {shiftKey: true});
    press(ctx.button, 'S', {shiftKey: true});
    press(ctx.button, 's');
    expect(action).toHaveBeenCalledTimes(2);
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

  test('AC8 a shortcut whose element left the page gives way to a new one', () => {
    const oldElem = document.createElement('kbd');
    const newElem = document.createElement('kbd');
    ctx.root.append(oldElem, newElem);
    const oldAction = ctx.add('t', {element: oldElem});
    expect(() => ctx.add('t', {element: newElem})).toThrow('conflicts with "t"');

    oldElem.remove();
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

// the markup of templates/repo/home_sidebar_top.tmpl and RepoFileSearch.vue
function renderHint(key: string, placeholder: string) {
  const wrapper = document.createElement('div');
  wrapper.className = 'ui input global-shortcut-wrapper';
  wrapper.innerHTML = `<input placeholder="${placeholder}"><kbd data-shortcut-keys="${key}">${key.toUpperCase()}</kbd>`;
  document.body.append(wrapper);
  const input = wrapper.querySelector('input')!;
  const kbd = wrapper.querySelector('kbd')!;
  initShortcutKbd(kbd);
  return {wrapper, input, kbd};
}

// these tests share document.body and its focus, so they must not run concurrently (see vitest.config.ts)
describe('shortcut hints', {concurrent: false}, () => {
  beforeAll(() => {
    initGlobalShortcut();
  });
  afterEach(() => {
    document.body.replaceChildren();
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
});
