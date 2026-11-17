// Decides, once for all tabs sharing the event stream, whether an unread count change deserves a desktop alert.
// No DOM access: it runs inside the shared worker.

export type NotificationTabState = {visible: boolean, count: number};
export type NotificationAlert<Tab> = {tab: Tab, newCount: number};

export class NotificationAlertTracker<Tab> {
  tabs = new Map<Tab, boolean>(); // tab -> visible
  lastVisibleTab: Tab | null = null;
  lastCount: number | null = null;
  baseline = 0; // the count when the user last had Gitea in view, so "new" means new since then

  anyVisible(): boolean {
    for (const visible of this.tabs.values()) {
      if (visible) return true;
    }
    return false;
  }

  setTabState(tab: Tab, {visible, count}: NotificationTabState) {
    this.tabs.set(tab, visible);
    if (this.lastCount === null) {
      this.lastCount = count;
      this.baseline = count;
    }
    if (visible) {
      this.lastVisibleTab = tab;
      this.baseline = this.lastCount;
    }
  }

  removeTab(tab: Tab) {
    this.tabs.delete(tab);
    if (this.lastVisibleTab === tab) this.lastVisibleTab = null;
  }

  onCount(count: number): NotificationAlert<Tab> | null {
    const lastCount = this.lastCount;
    this.lastCount = count;
    if (this.tabs.size === 0) return null;
    if (this.anyVisible()) {
      this.baseline = count;
      return null;
    }
    if (count < this.baseline) this.baseline = count;
    if (lastCount === null || count <= lastCount) return null;
    const tab = this.lastVisibleTab ?? this.tabs.keys().next().value!;
    return {tab, newCount: count - this.baseline};
  }
}
