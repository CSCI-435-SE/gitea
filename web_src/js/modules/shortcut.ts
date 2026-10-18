import {registerGlobalInitFunc} from './observer.ts';
import {hideElem, toggleElem} from '../utils/dom.ts';

const sequenceTimeoutMs = 1500; // the longest wait between the keys of a sequence

export type Shortcut = {
  keys: string, // keys pressed one after another, separated by spaces, e.g. "g c"
  description: string, // translated
  section?: string, // translated
  element?: HTMLElement, // the shortcut only exists while this element is in the document
  action: () => void,
};

export type ShortcutInfo = Pick<Shortcut, 'keys' | 'description' | 'section'>;

function isActive(shortcut: Shortcut): boolean {
  return !shortcut.element || shortcut.element.isConnected;
}

// returns the shortcut for exactly these keys, 'prefix' if they start a longer sequence, or null
function matchKeys(shortcuts: Shortcut[], keys: string): Shortcut | 'prefix' | null {
  for (const shortcut of shortcuts) {
    if (shortcut.keys === keys) return shortcut;
    if (shortcut.keys.startsWith(`${keys} `)) return 'prefix';
  }
  return null;
}

export class ShortcutRegistry {
  shortcuts: Shortcut[] = [];
  pendingKeys = ''; // the start of a sequence pressed so far
  pendingTime = 0;

  register(shortcut: Shortcut) {
    const keys = shortcut.keys.toLowerCase();
    // a shortcut whose element has left the page, e.g. a hint replaced by file tree navigation, gives way to the new one
    this.shortcuts = this.shortcuts.filter(isActive);
    const conflict = this.shortcuts.find((s) => s.keys === keys || s.keys.startsWith(`${keys} `) || keys.startsWith(`${s.keys} `));
    if (conflict) throw new Error(`Keyboard shortcut "${keys}" conflicts with "${conflict.keys}"`);
    this.shortcuts.push({...shortcut, keys});
  }

  list(): ShortcutInfo[] {
    return this.shortcuts.filter(isActive).map(({keys, description, section}) => ({keys, description, section}));
  }

  handleKeyDown(e: KeyboardEvent) {
    const target = e.target as HTMLElement;
    // the key was already used by a widget such as a menu, or is being typed into a field
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || target.matches('input, textarea, select') || target.isContentEditable) {
      this.pendingKeys = '';
      return;
    }
    if (e.key === 'Shift') return; // not a press of its own, e.g. on the way to "?"

    if (Date.now() - this.pendingTime > sequenceTimeoutMs) this.pendingKeys = '';
    const key = e.key.toLowerCase();
    const shortcuts = this.shortcuts.filter(isActive);
    let keys = this.pendingKeys ? `${this.pendingKeys} ${key}` : key;
    let match = matchKeys(shortcuts, keys);
    if (!match && keys !== key) {
      // the key does not continue the sequence, so it acts as a fresh press
      keys = key;
      match = matchKeys(shortcuts, keys);
    }
    this.pendingKeys = '';
    if (!match) return;

    e.preventDefault();
    if (match === 'prefix') {
      this.pendingKeys = keys;
      this.pendingTime = Date.now();
    } else {
      match.action();
    }
  }
}

const globalRegistry = new ShortcutRegistry();

// Adds a keyboard shortcut to the page. Throws if the keys clash with an existing shortcut.
export function registerShortcut(shortcut: Shortcut) {
  globalRegistry.register(shortcut);
}

// The shortcuts available on the current page, e.g. for a help dialog
export function getShortcuts(): ShortcutInfo[] {
  return globalRegistry.list();
}

export function initShortcutKbd(kbd: HTMLElement) {
  // Handle initial state: hide the kbd hint if the associated input already has a value
  // (e.g., from browser autofill or back/forward navigation cache)
  const elem = elemFromKbd(kbd)!;
  if (elem.value) hideElem(kbd);
  kbd.setAttribute('aria-hidden', 'true');
  kbd.setAttribute('aria-keyshortcuts', kbd.getAttribute('data-shortcut-keys')!);
  registerShortcut({
    keys: kbd.getAttribute('data-shortcut-keys')!,
    description: elem.placeholder,
    element: kbd,
    action: () => elem.focus(),
  });
}

function shortcutWrapper(el: HTMLElement): HTMLElement | null {
  const parent = el.parentElement;
  return parent?.matches('.global-shortcut-wrapper') ? parent : null;
}

function elemFromKbd(kbd: HTMLElement): HTMLInputElement | HTMLTextAreaElement | null {
  return shortcutWrapper(kbd)?.querySelector<HTMLInputElement>('input, textarea') || null;
}

function kbdFromElem(input: HTMLElement): HTMLElement | null {
  return shortcutWrapper(input)?.querySelector<HTMLElement>('kbd') || null;
}

export function initGlobalShortcut() {
  registerGlobalInitFunc('onGlobalShortcut', initShortcutKbd);
  document.addEventListener('keydown', (e: KeyboardEvent) => globalRegistry.handleKeyDown(e));

  // A <kbd> element next to an <input> declares a keyboard shortcut for that input.
  // When the matching key is pressed, the sibling input is focused.
  // When Escape is pressed inside such an input, the input is cleared and blurred.
  // The <kbd> element is shown/hidden automatically based on input focus and value.
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    // Modifier keys are not supported yet
    if (e.ctrlKey || e.metaKey || e.altKey) return;

    const target = e.target as HTMLElement;

    // Handle Escape: clear and blur inputs that have an associated keyboard shortcut
    if (e.key === 'Escape') {
      const kbd = kbdFromElem(target);
      if (kbd) {
        (target as HTMLInputElement).value = '';
        (target as HTMLInputElement).blur();
      }
    }
  });

  // Toggle kbd shortcut hint visibility on input focus/blur
  document.addEventListener('focusin', (e) => {
    const kbd = kbdFromElem(e.target as HTMLElement);
    if (!kbd) return;
    hideElem(kbd);
  });

  document.addEventListener('focusout', (e) => {
    const kbd = kbdFromElem(e.target as HTMLElement);
    if (!kbd) return;
    const hasContent = Boolean((e.target as HTMLInputElement).value);
    toggleElem(kbd, !hasContent);
  });
}
