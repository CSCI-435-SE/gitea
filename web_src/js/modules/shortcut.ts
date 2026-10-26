import {registerGlobalInitFunc} from './observer.ts';
import {hideElem, toggleElem} from '../utils/dom.ts';
import {html, htmlRaw} from '../utils/html.ts';
import {localUserSettings} from './user-settings.ts';

const sequenceTimeoutMs = 1500; // the longest wait between the keys of a sequence
const enabledSettingKey = 'keyboard-shortcuts-enabled';

export type Shortcut = {
  keys: string, // keys pressed one after another, separated by spaces, e.g. "g c"
  description: string, // translated
  section?: string, // translated
  element?: HTMLElement, // the shortcut only exists while this element is in the document
  alwaysOn?: boolean, // still runs when the user has turned shortcuts off
  action: () => void,
};

export type ShortcutInfo = Pick<Shortcut, 'keys' | 'description' | 'section' | 'alwaysOn'>;

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
  private shortcuts: Shortcut[] = [];
  private pendingKeys = ''; // the start of a sequence pressed so far
  private pendingTime = 0;
  // kept in memory as well, so the switch works for this page even where storage throws
  private enabled = localUserSettings.getBoolean(enabledSettingKey, true);

  isEnabled(): boolean {
    return this.enabled;
  }

  setEnabled(enabled: boolean) {
    this.enabled = enabled;
    this.pendingKeys = '';
    localUserSettings.setBoolean(enabledSettingKey, enabled);
  }

  register(shortcut: Shortcut) {
    const keys = shortcut.keys.trim().split(/\s+/).join(' ').toLowerCase(); // "g  c" would otherwise never match a press
    // a shortcut whose element has left the page, e.g. a hint replaced by file tree navigation, gives way to the new one
    this.shortcuts = this.shortcuts.filter(isActive);
    const conflict = this.shortcuts.find((s) => s.keys === keys || s.keys.startsWith(`${keys} `) || keys.startsWith(`${s.keys} `));
    if (conflict) throw new Error(`Keyboard shortcut "${keys}" conflicts with "${conflict.keys}"`);
    this.shortcuts.push({...shortcut, keys});
  }

  list(): ShortcutInfo[] {
    return this.shortcuts.filter(isActive).map(({keys, description, section, alwaysOn}) => ({keys, description, section, alwaysOn}));
  }

  handleKeyDown(e: KeyboardEvent) {
    const target = e.target as HTMLElement;
    // the key was already used by a widget such as a menu, or is being typed into a field, or belongs to an open dialog, not the page behind it
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || target.matches('input, textarea, select') || target.isContentEditable || document.querySelector('dialog[open]')) {
      this.pendingKeys = '';
      return;
    }
    if (e.key === 'Shift') return; // not a press of its own, e.g. on the way to "?"
    if (e.repeat) return; // a held key runs its shortcut once, and doesn't start over after a sequence

    if (Date.now() - this.pendingTime > sequenceTimeoutMs) this.pendingKeys = '';
    const key = e.key.toLowerCase();
    // WCAG 2.1.4: a dictated word must not run shortcuts, so with them off only the always-on ones match
    const shortcuts = this.shortcuts.filter((s) => isActive(s) && (this.enabled || s.alwaysOn));
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

export function areShortcutsEnabled(): boolean {
  return globalRegistry.isEnabled();
}

export function setShortcutsEnabled(enabled: boolean) {
  globalRegistry.setEnabled(enabled);
}

// Groups shortcuts by section in the order each section first appears, the default section first.
// A shortcut without a section belongs to the default one.
export function groupShortcuts(shortcuts: ShortcutInfo[], defaultSection: string): Map<string, ShortcutInfo[]> {
  const groups = new Map<string, ShortcutInfo[]>([[defaultSection, []]]);
  for (const shortcut of shortcuts) {
    const section = shortcut.section ?? defaultSection;
    if (!groups.has(section)) groups.set(section, []);
    groups.get(section)!.push(shortcut);
  }
  return groups;
}

function renderShortcutKeys(keys: string): string {
  return keys.split(' ').map((key) => html`<kbd>${key.toUpperCase()}</kbd>`).join('');
}

// the help dialog lists the shortcuts registered when it opens, so it also shows ones added after page load
export function initShortcutHelp(dialog: HTMLDialogElement) {
  const general = dialog.getAttribute('data-shortcut-section')!;
  const toggle = dialog.querySelector<HTMLInputElement>('.shortcut-help-enabled')!;
  toggle.addEventListener('change', () => {
    setShortcutsEnabled(toggle.checked);
    dialog.classList.toggle('shortcuts-off', !toggle.checked);
  });
  registerShortcut({
    keys: dialog.getAttribute('data-shortcut-keys')!,
    description: dialog.getAttribute('data-shortcut-description')!,
    section: general,
    element: dialog,
    alwaysOn: true, // the keyboard-only way back to the switch that turns shortcuts on again
    action: () => {
      const groups = groupShortcuts(getShortcuts(), general);
      dialog.querySelector('.shortcut-help-sections')!.innerHTML = Array.from(groups, ([section, shortcuts]) => html`
        <section>
          <h3>${section}</h3>
          <dl>${htmlRaw(shortcuts.map((s) => html`<div class="shortcut-help-row${s.alwaysOn ? ' always-on' : ''}"><dt>${s.description}</dt><dd>${htmlRaw(renderShortcutKeys(s.keys))}</dd></div>`).join(''))}</dl>
        </section>
      `).join('');
      toggle.checked = areShortcutsEnabled();
      dialog.classList.toggle('shortcuts-off', !toggle.checked);
      const opener = document.activeElement as HTMLElement;
      dialog.addEventListener('close', () => opener.focus(), {once: true}); // browsers differ on where focus goes when a dialog closes
      dialog.showModal();
    },
  });
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
    section: kbd.getAttribute('data-shortcut-section') ?? undefined,
    element: kbd,
    action: () => elem.focus(),
  });
}

// a link declares a shortcut that goes where it leads, described separately because its text may hold more, e.g. a count
export function initShortcutLink(link: HTMLAnchorElement) {
  registerShortcut({
    keys: link.getAttribute('data-shortcut-keys')!,
    description: link.getAttribute('data-shortcut-description')!,
    section: link.getAttribute('data-shortcut-section') ?? undefined,
    element: link,
    action: () => window.location.assign(link.getAttribute('data-shortcut-href') ?? link.href), // e.g. "g c" goes to the default branch, not the browsed one
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
  registerGlobalInitFunc('onShortcutLink', initShortcutLink);
  registerGlobalInitFunc('onShortcutHelp', initShortcutHelp);
  document.addEventListener('keydown', (e: KeyboardEvent) => globalRegistry.handleKeyDown(e));

  // A <kbd> element next to an <input> declares a keyboard shortcut for that input.
  // initShortcutKbd registers it, so pressing the matching key focuses the sibling input.
  // When Escape is pressed inside such an input, the input is cleared and blurred.
  // The <kbd> element is shown/hidden automatically based on input focus and value.
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    // Ctrl/Alt/Cmd+Escape is left alone
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
