import {desktopAlertsEnabled, initNotificationDesktopToggle, showDesktopAlert} from './notification-desktop.ts';
import {createElementFromHTML} from '../utils/dom.ts';

class FakeNotification extends EventTarget {
  static permission: NotificationPermission = 'default';
  static requestPermission = vi.fn<() => Promise<NotificationPermission>>();
  static shown: FakeNotification[] = [];
  title: string;
  options: NotificationOptions;
  close = vi.fn();
  constructor(title: string, options: NotificationOptions) {
    super();
    this.title = title;
    this.options = options;
    FakeNotification.shown.push(this);
  }
}

// the navbar count every signed-in page renders, and the toggle notification_div.tmpl renders
function setupPage() {
  document.body.innerHTML = `
    <span class="notification_count" data-desktop-alert-title="Gitea" data-desktop-alert-1="You have %d new notification" data-desktop-alert-n="You have %d new notifications">0</span>`;
  const toggle = createElementFromHTML<HTMLElement>(`
    <div class="tw-hidden">
      <button type="button" class="notification-desktop-toggle">
        <span class="notification-desktop-enable">Enable desktop notifications</span>
        <span class="notification-desktop-disable tw-hidden">Disable desktop notifications</span>
      </button>
      <span class="notification-desktop-blocked tw-hidden">Desktop notifications are blocked by your browser.</span>
    </div>`);
  document.body.append(toggle);
  initNotificationDesktopToggle(toggle);
  const visible = (selector: string) => !toggle.querySelector(selector)!.classList.contains('tw-hidden');
  const button = toggle.querySelector<HTMLButtonElement>('.notification-desktop-toggle')!;
  return {toggle, button, visible};
}

// these tests share document.body and the stubbed globals, so they must not run concurrently
describe('desktop notifications', {concurrent: false}, () => {
  beforeEach(() => {
    document.documentElement.lang = 'en-US'; // trN resolves plural rules from the document language
    vi.stubGlobal('localStorage', new Storage()); // Node's own localStorage global hides happy-dom's
    FakeNotification.permission = 'default';
    FakeNotification.requestPermission = vi.fn();
    FakeNotification.shown = [];
    vi.stubGlobal('Notification', FakeNotification);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    document.body.replaceChildren();
  });

  test('AC3 never enabled: no alert and no permission prompt', () => {
    const {toggle, visible} = setupPage();
    showDesktopAlert(1);
    expect(FakeNotification.shown).toHaveLength(0);
    expect(FakeNotification.requestPermission).not.toHaveBeenCalled();
    expect(toggle.classList.contains('tw-hidden')).toBe(false);
    expect(visible('.notification-desktop-enable')).toBe(true);
  });

  test('AC3 permission granted for something else does not turn alerts on by itself', () => {
    FakeNotification.permission = 'granted';
    setupPage();
    showDesktopAlert(1);
    expect(FakeNotification.shown).toHaveLength(0);
  });

  test('AC4 denying the browser prompt says it is blocked and leaves the setting off', async () => {
    FakeNotification.requestPermission.mockResolvedValue('denied');
    const {button, visible} = setupPage();
    button.click();
    await vi.waitFor(() => expect(visible('.notification-desktop-blocked')).toBe(true));
    expect(FakeNotification.requestPermission).toHaveBeenCalledTimes(1);
    expect(desktopAlertsEnabled()).toBe(false);
    expect(visible('.notification-desktop-enable')).toBe(true);
  });

  test('allowing the browser prompt turns alerts on', async () => {
    FakeNotification.requestPermission.mockImplementation(() => {
      FakeNotification.permission = 'granted';
      return Promise.resolve('granted');
    });
    const {button, visible} = setupPage();
    button.click();
    await vi.waitFor(() => expect(visible('.notification-desktop-disable')).toBe(true));
    expect(desktopAlertsEnabled()).toBe(true);
    expect(visible('.notification-desktop-blocked')).toBe(false);
  });

  test('AC5 the alert says how many new notifications there are and nothing else', () => {
    FakeNotification.permission = 'granted';
    const {button} = setupPage();
    button.click(); // permission is already granted, so this enables without a prompt
    showDesktopAlert(1);
    showDesktopAlert(3);
    expect(FakeNotification.requestPermission).not.toHaveBeenCalled();
    expect(FakeNotification.shown.map((n) => n.options.body)).toEqual(['You have 1 new notification', 'You have 3 new notifications']);
    expect(FakeNotification.shown[0].title).toBe('Gitea');
  });

  test('AC8 clicking the alert focuses Gitea and opens the notifications page', () => {
    FakeNotification.permission = 'granted';
    const {button} = setupPage();
    button.click();
    const focus = vi.spyOn(window, 'focus').mockImplementation(() => {});
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    showDesktopAlert(1);
    FakeNotification.shown[0].dispatchEvent(new Event('click'));
    expect(focus).toHaveBeenCalled();
    expect(assign).toHaveBeenCalledWith('/notifications');
    expect(FakeNotification.shown[0].close).toHaveBeenCalled();
  });

  test('a browser that refuses to construct the alert does not make showDesktopAlert throw', () => {
    FakeNotification.permission = 'granted';
    const {button} = setupPage();
    button.click();
    // Android Chrome exposes Notification but its constructor throws "Illegal constructor"
    class ThrowingNotification extends FakeNotification {
      constructor(title: string, options: NotificationOptions) {
        super(title, options);
        throw new TypeError('Illegal constructor');
      }
    }
    vi.stubGlobal('Notification', ThrowingNotification);
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => showDesktopAlert(1)).not.toThrow();
    expect(consoleError).toHaveBeenCalled();
  });

  test('AC9 disabling stops further alerts', async () => {
    FakeNotification.permission = 'granted';
    const {button, visible} = setupPage();
    button.click();
    expect(visible('.notification-desktop-disable')).toBe(true);
    button.click();
    await vi.waitFor(() => expect(visible('.notification-desktop-enable')).toBe(true));
    showDesktopAlert(1);
    expect(FakeNotification.shown).toHaveLength(0);
  });
});
