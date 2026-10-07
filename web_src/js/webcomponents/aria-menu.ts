import {generateElemId, hideElem, showElem} from '../utils/dom.ts';

// A menu button following https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/ with no Fomantic.
// Markup: <aria-menu><button type="button">label</button><div class="menu"><a class="item">...</a></div></aria-menu>
// Items get real focus (roving tabindex) instead of aria-activedescendant, see web_src/js/modules/fomantic/aria.md
window.customElements.define('aria-menu', class extends HTMLElement {
  trigger!: HTMLElement;
  popup!: HTMLElement;
  inited = false;

  connectedCallback() {
    // in <head>-loaded components, the children are not parsed yet while the page is still loading
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', this.init, {once: true});
    } else {
      this.init();
    }
  }

  disconnectedCallback() {
    document.removeEventListener('DOMContentLoaded', this.init);
    if (this.inited) this.close(false); // also removes the outside click listener
  }

  init = () => {
    if (this.inited) return;
    const popup = this.querySelector<HTMLElement>(':scope > .menu');
    const trigger = Array.from(this.children).find((el) => el !== popup) as HTMLElement | undefined;
    if (!popup || !trigger) return;
    this.inited = true;
    this.popup = popup;
    this.trigger = trigger;

    if (trigger.tagName !== 'BUTTON') {
      trigger.setAttribute('role', 'button');
      trigger.tabIndex = 0;
    }
    if (!popup.id) popup.id = generateElemId('aria-menu-popup-');
    trigger.setAttribute('aria-haspopup', 'menu');
    trigger.setAttribute('aria-expanded', 'false');
    trigger.setAttribute('aria-controls', popup.id);
    popup.setAttribute('role', 'menu');
    if (!popup.hasAttribute('aria-label') && !popup.hasAttribute('aria-labelledby')) {
      if (!trigger.id) trigger.id = generateElemId('aria-menu-trigger-');
      popup.setAttribute('aria-labelledby', trigger.id); // keeps the menu's name once focus is on an item
    }
    hideElem(popup);
    this.updateItemRoles();

    trigger.addEventListener('click', () => this.isOpen() ? this.close(true) : this.open('first'));
    trigger.addEventListener('keydown', this.onTriggerKeyDown);
    trigger.addEventListener('keyup', this.preventSpaceClick);
    popup.addEventListener('keydown', this.onPopupKeyDown);
    popup.addEventListener('keyup', this.preventSpaceClick);
    popup.addEventListener('click', (e) => {
      // bubble phase, so the item's own click handlers have already run
      if ((e.target as Element).closest('[role="menuitem"]')) this.close(true);
    });
  };

  // items may be added after init, e.g. by a feature script, so refresh them on every open
  updateItemRoles() {
    for (const child of this.popup.children) {
      if (child.classList.contains('item')) {
        child.setAttribute('role', 'menuitem');
        child.setAttribute('tabindex', '-1');
      } else if (child.classList.contains('divider')) {
        child.setAttribute('role', 'separator');
      } else {
        child.setAttribute('role', 'none');
      }
    }
  }

  items(): HTMLElement[] {
    // hidden items can't take focus, so leaving them in would stall opening and arrow keys
    return Array.from(this.popup.querySelectorAll<HTMLElement>(':scope > [role="menuitem"]:not(.tw-hidden)'));
  }

  isOpen() {
    return this.trigger.getAttribute('aria-expanded') === 'true';
  }

  open(focus: 'first' | 'last') {
    this.updateItemRoles();
    showElem(this.popup);
    this.trigger.setAttribute('aria-expanded', 'true');
    document.addEventListener('pointerdown', this.onPointerDownOutside, {capture: true});
    const items = this.items();
    (focus === 'first' ? items[0] : items.at(-1))?.focus();
  }

  close(restoreFocus: boolean) {
    // only restore when focus is still in the popup, so focus moved elsewhere by an item handler is kept
    const focusInPopup = this.popup.contains(document.activeElement);
    hideElem(this.popup);
    this.trigger.setAttribute('aria-expanded', 'false');
    document.removeEventListener('pointerdown', this.onPointerDownOutside, {capture: true});
    if (restoreFocus && focusInPopup) this.trigger.focus();
  }

  onPointerDownOutside = (e: Event) => {
    if (!this.contains(e.target as Node)) this.close(false);
  };

  // a native button fires "click" on Space keyup, which would toggle the menu shut again or double-activate an item
  preventSpaceClick = (e: KeyboardEvent) => {
    if (e.key === ' ') e.preventDefault();
  };

  onTriggerKeyDown = (e: KeyboardEvent) => {
    if (e.isComposing) return;
    if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
      this.open('first');
    } else if (e.key === 'ArrowUp') {
      this.open('last');
    } else {
      return;
    }
    e.preventDefault();
  };

  onPopupKeyDown = (e: KeyboardEvent) => {
    if (e.isComposing) return;
    const items = this.items();
    const current = items.indexOf(e.target as HTMLElement);
    if (e.key === 'Tab') {
      // focus the trigger and let the default action move on from there
      this.close(true);
      return;
    }
    if (e.key === 'Escape') {
      e.stopPropagation(); // don't also close a surrounding modal
      this.close(true);
    } else if (e.key === 'ArrowDown') {
      items[(current + 1) % items.length].focus();
    } else if (e.key === 'ArrowUp') {
      items.at(current <= 0 ? -1 : current - 1)!.focus();
    } else if (e.key === 'Home') {
      items[0].focus();
    } else if (e.key === 'End') {
      items.at(-1)!.focus();
    } else if (e.key === 'Enter' || e.key === ' ') {
      if (current < 0) return;
      items[current].click();
    } else if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      this.focusByFirstChar(items, current, e.key);
    } else {
      return;
    }
    e.preventDefault();
  };

  focusByFirstChar(items: HTMLElement[], current: number, char: string) {
    const lower = char.toLowerCase();
    for (let i = 1; i <= items.length; i++) {
      const item = items[(current + i) % items.length];
      if (item.textContent.trim().toLowerCase().startsWith(lower)) {
        item.focus();
        return;
      }
    }
  }
});
