import './aria-menu.ts';

// tests touch document.activeElement, so they must stay synchronous to be safe under concurrent sequencing
function renderMenu(labels: string[] = ['Apple', 'Banana', 'Cherry', 'Blueberry']) {
  const root = document.createElement('div');
  root.innerHTML = `
    <aria-menu>
      <button type="button">Fruit</button>
      <div class="menu">
        ${labels.map((label) => `<div class="item">${label}</div>`).join('')}
        <div class="divider"></div>
        <a class="item" href="#">Settings</a>
      </div>
    </aria-menu>
    <button type="button" class="after">after</button>
  `;
  document.body.append(root);
  const menu = root.querySelector<HTMLElement>('aria-menu')!;
  const trigger = menu.querySelector<HTMLButtonElement>(':scope > button')!;
  const popup = menu.querySelector<HTMLElement>('.menu')!;
  const items = Array.from(popup.querySelectorAll<HTMLElement>('.item'));
  return {root, menu, trigger, popup, items, cleanup: () => root.remove()};
}

function press(el: Element, key: string, opts: KeyboardEventInit = {}): KeyboardEvent {
  const e = new KeyboardEvent('keydown', {key, bubbles: true, cancelable: true, ...opts});
  el.dispatchEvent(e);
  return e;
}

function isOpen(trigger: HTMLElement, popup: HTMLElement) {
  return trigger.getAttribute('aria-expanded') === 'true' && !popup.classList.contains('tw-hidden');
}

describe('AC5 ARIA attributes', () => {
  test('trigger and popup carry the menu button roles', () => {
    const {trigger, popup, items, cleanup} = renderMenu();
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu');
    expect(trigger.getAttribute('aria-expanded')).toBe('false');
    expect(popup.id).not.toBe('');
    expect(trigger.getAttribute('aria-controls')).toBe(popup.id);
    expect(popup.getAttribute('role')).toBe('menu');
    for (const item of items) {
      expect(item.getAttribute('role')).toBe('menuitem');
      expect(item.getAttribute('tabindex')).toBe('-1');
    }
    expect(popup.querySelector('.divider')!.getAttribute('role')).toBe('separator');
    expect(popup.classList.contains('tw-hidden')).toBe(true);
    cleanup();
  });

  test('aria-expanded follows the open state', () => {
    const {trigger, popup, cleanup} = renderMenu();
    trigger.click();
    expect(isOpen(trigger, popup)).toBe(true);
    trigger.click();
    expect(isOpen(trigger, popup)).toBe(false);
    cleanup();
  });

  test('the popup is named by its trigger unless it already has a name', () => {
    const {trigger, popup, cleanup} = renderMenu();
    expect(trigger.id).not.toBe('');
    expect(popup.getAttribute('aria-labelledby')).toBe(trigger.id);
    cleanup();

    const root = document.createElement('div');
    root.innerHTML = `<aria-menu><button type="button">x</button><div class="menu" aria-label="Own name"><div class="item">y</div></div></aria-menu>`;
    document.body.append(root);
    expect(root.querySelector('.menu')!.hasAttribute('aria-labelledby')).toBe(false);
    root.remove();
  });

  test('a non-button trigger becomes a focusable button', () => {
    const root = document.createElement('div');
    root.innerHTML = `<aria-menu><span>more</span><div class="menu"><div class="item">x</div></div></aria-menu>`;
    document.body.append(root);
    const trigger = root.querySelector('span')!;
    expect(trigger.getAttribute('role')).toBe('button');
    expect(trigger.getAttribute('tabindex')).toBe('0');
    root.remove();
  });
});

describe('AC1 opening', () => {
  for (const key of ['Enter', ' ', 'ArrowDown']) {
    test(`"${key}" on the trigger opens and focuses the first item`, () => {
      const {trigger, popup, items, cleanup} = renderMenu();
      trigger.focus();
      expect(press(trigger, key).defaultPrevented).toBe(true);
      expect(isOpen(trigger, popup)).toBe(true);
      expect(document.activeElement).toBe(items[0]);
      cleanup();
    });
  }

  test('ArrowUp on the trigger opens and focuses the last item', () => {
    const {trigger, popup, items, cleanup} = renderMenu();
    trigger.focus();
    press(trigger, 'ArrowUp');
    expect(isOpen(trigger, popup)).toBe(true);
    expect(document.activeElement).toBe(items.at(-1));
    cleanup();
  });
});

describe('AC2 arrow navigation', () => {
  test('ArrowDown and ArrowUp wrap around', () => {
    const {trigger, items, cleanup} = renderMenu();
    press(trigger, 'ArrowUp');
    press(items.at(-1)!, 'ArrowDown');
    expect(document.activeElement).toBe(items[0]);
    press(items[0], 'ArrowUp');
    expect(document.activeElement).toBe(items.at(-1));
    press(items.at(-1)!, 'ArrowUp');
    expect(document.activeElement).toBe(items.at(-2));
    cleanup();
  });

  test('hidden items are skipped when opening and navigating', () => {
    const {trigger, items, cleanup} = renderMenu();
    items[0].classList.add('tw-hidden');
    items[2].classList.add('tw-hidden');
    press(trigger, 'Enter');
    expect(document.activeElement).toBe(items[1]);
    press(items[1], 'ArrowDown');
    expect(document.activeElement).toBe(items[3]);
    press(items[3], 'End');
    press(items.at(-1)!, 'ArrowDown');
    expect(document.activeElement).toBe(items[1]);
    cleanup();
  });

  test('Home and End go to the first and last item', () => {
    const {trigger, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    press(items[0], 'End');
    expect(document.activeElement).toBe(items.at(-1));
    press(items.at(-1)!, 'Home');
    expect(document.activeElement).toBe(items[0]);
    cleanup();
  });
});

describe('AC3 type-ahead', () => {
  test('a character moves to the next item starting with it', () => {
    const {trigger, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    press(items[0], 'b');
    expect(document.activeElement).toBe(items[1]); // Banana
    press(items[1], 'B');
    expect(document.activeElement).toBe(items[3]); // Blueberry, case-insensitive
    press(items[3], 'b');
    expect(document.activeElement).toBe(items[1]); // wraps back to Banana
    cleanup();
  });

  test('a character no item starts with leaves focus alone', () => {
    const {trigger, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    press(items[0], 'c');
    press(items[2], 'z');
    expect(document.activeElement).toBe(items[2]);
    cleanup();
  });

  test('modifier shortcuts are not treated as type-ahead', () => {
    const {trigger, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    expect(press(items[0], 'c', {ctrlKey: true}).defaultPrevented).toBe(false);
    expect(document.activeElement).toBe(items[0]);
    cleanup();
  });
});

describe('AC4 closing', () => {
  test('Escape closes and returns focus to the trigger', () => {
    const {trigger, popup, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    press(items[1], 'Escape');
    expect(isOpen(trigger, popup)).toBe(false);
    expect(document.activeElement).toBe(trigger);
    cleanup();
  });

  test('Tab closes and lets the browser move on from the trigger', () => {
    const {trigger, popup, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    const e = press(items[1], 'Tab');
    expect(e.defaultPrevented).toBe(false); // the default action then moves focus past the trigger
    expect(isOpen(trigger, popup)).toBe(false);
    expect(document.activeElement).toBe(trigger);
    cleanup();
  });

  test('clicking the trigger while open returns focus to it', () => {
    const {trigger, popup, items, cleanup} = renderMenu();
    press(trigger, 'Enter');
    expect(document.activeElement).toBe(items[0]);
    trigger.click(); // Safari does not focus a clicked button, so focus would otherwise stay on a hidden item
    expect(isOpen(trigger, popup)).toBe(false);
    expect(document.activeElement).toBe(trigger);
    cleanup();
  });

  test('removing an open menu resets its state', () => {
    const {root, menu, trigger, popup, cleanup} = renderMenu();
    press(trigger, 'Enter');
    menu.remove();
    expect(isOpen(trigger, popup)).toBe(false);
    root.append(menu);
    trigger.click();
    expect(isOpen(trigger, popup)).toBe(true);
    cleanup();
  });

  test('a click outside closes without stealing focus', () => {
    const {root, trigger, popup, cleanup} = renderMenu();
    press(trigger, 'Enter');
    const after = root.querySelector<HTMLElement>('.after')!;
    after.focus();
    after.dispatchEvent(new MouseEvent('pointerdown', {bubbles: true}));
    expect(isOpen(trigger, popup)).toBe(false);
    expect(document.activeElement).toBe(after);
    cleanup();
  });
});

describe('AC6 activation', () => {
  for (const [name, activate] of [
    ['Enter', (item: HTMLElement) => press(item, 'Enter')],
    ['Space', (item: HTMLElement) => press(item, ' ')],
    ['click', (item: HTMLElement) => item.click()],
  ] as const) {
    test(`${name} runs the item's click handler and closes`, () => {
      const {trigger, popup, items, cleanup} = renderMenu();
      const handler = vi.fn();
      items[2].addEventListener('click', handler);
      press(trigger, 'Enter');
      activate(items[2]);
      expect(handler).toHaveBeenCalledTimes(1);
      expect(isOpen(trigger, popup)).toBe(false);
      expect(document.activeElement).toBe(trigger);
      cleanup();
    });
  }

  test('focus moved by the handler is kept', () => {
    const {root, trigger, items, cleanup} = renderMenu();
    const after = root.querySelector<HTMLElement>('.after')!;
    items[0].addEventListener('click', () => after.focus());
    press(trigger, 'Enter');
    press(items[0], 'Enter');
    expect(document.activeElement).toBe(after);
    cleanup();
  });
});
