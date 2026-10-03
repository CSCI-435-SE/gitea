import {POST} from '../modules/fetch.ts';
import {toggleElem} from '../utils/dom.ts';
import {confirmModal} from './comp/ConfirmModal.ts';
import {showErrorToast} from '../modules/toast.ts';
import {errorMessage} from '../modules/errors.ts';
import {registerGlobalInitFunc} from '../modules/observer.ts';

// session storage keeps a selection while paging through one view and drops it with the browser tab
const storageKey = 'gitea:notification-selection';

// all: every notification in the view, which the server works out itself, so ids only tracks the visible page
export type NotificationSelection = {view: string, ids: string[], all: boolean};

function emptySelection(view: string): NotificationSelection {
  return {view, ids: [], all: false};
}

// loadSelection returns the view's stored selection; one stored for another view is dropped, so a new tab or filter starts empty
export function loadSelection(view: string): NotificationSelection {
  try {
    const stored = JSON.parse(sessionStorage.getItem(storageKey) ?? 'null');
    if (stored?.view === view && Array.isArray(stored.ids)) {
      return {view, ids: stored.ids.map(String), all: stored.all === true};
    }
    sessionStorage.removeItem(storageKey);
  } catch (e) {
    console.error('Unable to read the notification selection', e);
  }
  return emptySelection(view);
}

export function saveSelection(sel: NotificationSelection) {
  try {
    if (sel.ids.length || sel.all) {
      sessionStorage.setItem(storageKey, JSON.stringify(sel));
    } else {
      sessionStorage.removeItem(storageKey);
    }
  } catch (e) {
    console.error('Unable to save the notification selection', e);
  }
}

// setSelected adds or removes ids; any change by hand leaves "all in view" mode
export function setSelected(sel: NotificationSelection, ids: string[], selected: boolean): NotificationSelection {
  const picked = new Set(sel.ids);
  for (const id of ids) {
    if (selected) {
      picked.add(id);
    } else {
      picked.delete(id);
    }
  }
  return {view: sel.view, ids: Array.from(picked), all: false};
}

export function selectedCount(sel: NotificationSelection, viewCount: number): number {
  return sel.all ? viewCount : sel.ids.length;
}

// initNotificationBulkActions runs for every #notification_div, including the ones swapped in by live refreshes
export function initNotificationBulkActions(root: HTMLElement) {
  const bar = root.querySelector('.notification-bulk-bar');
  if (!bar) return; // empty view
  const view = root.getAttribute('data-view-key')!;
  const viewCount = Number(root.getAttribute('data-view-count'));
  const selectPage = bar.querySelector<HTMLInputElement>('.notification-checkbox-all')!;
  const checkboxes = Array.from(root.querySelectorAll<HTMLInputElement>('.notification-checkbox'));
  const pageIds = checkboxes.map((el) => el.getAttribute('data-notification-id')!);
  let sel = loadSelection(view);

  const update = (next: NotificationSelection) => {
    sel = next;
    saveSelection(sel);
    const picked = new Set(sel.ids);
    for (const el of checkboxes) el.checked = sel.all || picked.has(el.getAttribute('data-notification-id')!);
    const checkedOnPage = checkboxes.filter((el) => el.checked).length;
    selectPage.checked = checkedOnPage === checkboxes.length;
    selectPage.indeterminate = checkedOnPage > 0 && !selectPage.checked;
    const count = selectedCount(sel, viewCount);
    bar.querySelector('.notification-selected-count')!.textContent = String(count);
    toggleElem(bar.querySelector('.notification-bulk-actions')!, count > 0);
    toggleElem(bar.querySelector('.notification-select-all-view')!, selectPage.checked && !sel.all && viewCount > checkboxes.length);
  };

  for (const el of checkboxes) {
    el.addEventListener('change', () => {
      const base = sel.all ? {...sel, ids: pageIds} : sel; // leaving "all" keeps the rest of the page ticked
      update(setSelected(base, [el.getAttribute('data-notification-id')!], el.checked));
    });
  }
  selectPage.addEventListener('change', () => {
    update(!selectPage.checked && sel.all ? emptySelection(view) : setSelected(sel, pageIds, selectPage.checked));
  });
  bar.querySelector('.notification-select-all-view')!.addEventListener('click', (e) => {
    e.preventDefault();
    update({view, ids: pageIds, all: true});
  });

  for (const btn of bar.querySelectorAll<HTMLButtonElement>('.notification-bulk-action')) {
    btn.addEventListener('click', async () => {
      const confirmText = btn.getAttribute('data-confirm');
      if (confirmText && !await confirmModal({content: confirmText, confirmButtonColor: 'red'})) return;
      const data = new URLSearchParams({action: btn.getAttribute('data-action')!});
      if (sel.all) {
        data.set('all', 'true');
      } else {
        data.set('notification_ids', sel.ids.join(','));
      }
      try {
        const resp = await POST(root.getAttribute('data-bulk-url')!, {data});
        const json = await resp.json();
        if (!resp.ok) {
          showErrorToast(json.errorMessage || `Error ${resp.status}`);
          return;
        }
        saveSelection(emptySelection(view)); // a finished action clears the selection
        window.location.assign(json.redirect);
      } catch (err) {
        showErrorToast(errorMessage(err));
      }
    });
  }

  update(sel);
}

export function initNotificationBulk() {
  registerGlobalInitFunc('initNotificationBulkActions', initNotificationBulkActions);
}
