import {localUserSettings} from '../modules/user-settings.ts';
import {registerGlobalInitFunc} from '../modules/observer.ts';
import {trN} from '../modules/i18n.ts';
import {toggleElem} from '../utils/dom.ts';

const {appSubUrl} = window.config;
// stored per browser, because the permission it depends on is per browser too
const desktopAlertsSettingKey = 'notification-desktop-alerts';

function notificationApiAvailable(): boolean {
  return 'Notification' in window;
}

export function desktopAlertsEnabled(): boolean {
  // a permission revoked after opting in reads as off
  return notificationApiAvailable() && Notification.permission === 'granted' && localUserSettings.getBoolean(desktopAlertsSettingKey);
}

export function showDesktopAlert(newCount: number) {
  if (newCount <= 0 || !desktopAlertsEnabled()) return;
  const textEl = document.querySelector('.notification_count')!;
  // only a count, so no private issue or repo names reach a lock screen
  const body = trN(newCount, textEl.getAttribute('data-desktop-alert-1')!, textEl.getAttribute('data-desktop-alert-n')!);
  const icon = document.querySelector<HTMLLinkElement>('link[rel~="icon"]')?.href;
  // the shared tag makes the browser replace an earlier alert instead of stacking one per update or tab
  const alert = new Notification(textEl.getAttribute('data-desktop-alert-title')!, {body, icon, tag: `gitea-notifications:${appSubUrl}`});
  alert.addEventListener('click', () => {
    window.focus();
    window.location.assign(`${appSubUrl}/notifications`);
    alert.close();
  });
}

export function initNotificationDesktopToggle(el: HTMLElement) {
  if (!notificationApiAvailable()) return; // the wrapper stays hidden
  const button = el.querySelector<HTMLButtonElement>('.notification-desktop-toggle')!;
  const blocked = el.querySelector('.notification-desktop-blocked')!;

  const render = (showBlocked: boolean) => {
    const enabled = desktopAlertsEnabled();
    toggleElem(el.querySelector('.notification-desktop-enable')!, !enabled);
    toggleElem(el.querySelector('.notification-desktop-disable')!, enabled);
    toggleElem(blocked, showBlocked);
    button.setAttribute('aria-pressed', String(enabled));
  };

  button.addEventListener('click', async () => {
    if (desktopAlertsEnabled()) {
      localUserSettings.setBoolean(desktopAlertsSettingKey, false);
      render(false);
      return;
    }
    // browsers only allow the permission prompt from a user click, so this is the one place it is asked
    const permission = Notification.permission === 'default' ? await Notification.requestPermission() : Notification.permission;
    localUserSettings.setBoolean(desktopAlertsSettingKey, permission === 'granted');
    render(permission === 'denied');
  });

  render(Notification.permission === 'denied');
  toggleElem(el, true);
}

export function initNotificationDesktop() {
  registerGlobalInitFunc('initNotificationDesktopToggle', initNotificationDesktopToggle);
}
