import {toggleElem} from '../utils/dom.ts';
import {registerGlobalInitFunc} from '../modules/observer.ts';

// module state outlives the #notification_div that live refreshes swap in, so a collapsed section stays collapsed;
// it is gone after leaving or paging, which the spec leaves out of scope
const collapsedRepoIDs = new Set<string>();

function setCollapsed(group: HTMLElement, collapsed: boolean) {
  group.querySelector('.notification-group-toggle')!.setAttribute('aria-expanded', String(!collapsed));
  toggleElem(group.querySelector('.notification-group-rows')!, !collapsed);
}

// initNotificationGroups runs for every grouped #notification_table, including the ones swapped in by live refreshes
export function initNotificationGroups(table: HTMLElement) {
  for (const group of table.querySelectorAll<HTMLElement>('.notification-group')) {
    const repoID = group.getAttribute('data-repo-id')!;
    setCollapsed(group, collapsedRepoIDs.has(repoID));
    group.querySelector('.notification-group-header')!.addEventListener('click', (e) => {
      if ((e.target as Element).closest('a')) return; // the repository name is a link
      const collapsed = !collapsedRepoIDs.has(repoID);
      if (collapsed) {
        collapsedRepoIDs.add(repoID);
      } else {
        collapsedRepoIDs.delete(repoID);
      }
      setCollapsed(group, collapsed);
    });
  }
}

export function initNotificationGroup() {
  registerGlobalInitFunc('initNotificationGroups', initNotificationGroups);
}
