// One polite live region for the whole page, appended to <body> on first use: the issue sidebar
// replaces its own DOM after saving, which would destroy a region placed inside it.
let liveRegion: HTMLElement | null = null;
let pendingMessages: Array<string> = [];
let flushTimer: ReturnType<typeof setTimeout> | undefined;

function getLiveRegion(): HTMLElement {
  if (liveRegion?.isConnected) return liveRegion;
  // messages queued for a region that has since been removed are stale
  clearTimeout(flushTimer);
  pendingMessages = [];
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
