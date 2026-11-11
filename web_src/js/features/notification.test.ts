import {initNotificationCount} from './notification.ts';
import {GET} from '../modules/fetch.ts';
import {localUserSettings} from '../modules/user-settings.ts';

vi.mock('../modules/fetch.ts', () => ({
  GET: vi.fn(),
}));

class FakeNotification {
  static permission: NotificationPermission = 'granted';
  static shown: string[] = [];
  constructor(_title: string, options: NotificationOptions) {
    FakeNotification.shown.push(options.body!);
  }
  addEventListener() {}
}

const pollEvery = 1000;

// /notifications/new answers with these unread counts, one per poll
function serveCounts(...counts: number[]) {
  for (const count of counts) {
    vi.mocked(GET).mockResolvedValueOnce({ok: true, json: () => Promise.resolve({new: count})} as Response);
  }
}

function setVisibility(state: DocumentVisibilityState) {
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue(state);
}

const countText = () => document.querySelector('.notification_count')!.textContent;

// the polling fallback, used when the shared worker or the event stream is unavailable
describe('notification count poller', {concurrent: false}, () => {
  beforeEach(() => {
    vi.useFakeTimers();
    document.documentElement.lang = 'en-US'; // trN resolves plural rules from the document language
    document.body.innerHTML = `
      <span class="notification_count" data-desktop-alert-title="Gitea" data-desktop-alert-1="You have %d new notification" data-desktop-alert-n="You have %d new notifications">0</span>`;
    vi.stubGlobal('localStorage', new Storage()); // Node's own localStorage global hides happy-dom's
    FakeNotification.shown = [];
    vi.stubGlobal('Notification', FakeNotification);
    localUserSettings.setBoolean('notification-desktop-alerts', true);
    // EventSourceUpdateTime stays 0, so initNotificationCount takes the poller path
    Object.assign(window.config.notificationSettings, {MinTimeout: pollEvery, TimeoutStep: pollEvery, MaxTimeout: pollEvery});
  });
  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.mocked(GET).mockReset();
    vi.unstubAllGlobals();
    Object.assign(window.config.notificationSettings, {MinTimeout: 0, TimeoutStep: 0, MaxTimeout: 0});
    document.body.replaceChildren();
  });

  test('AC5 a hidden tab alerts with the number of new notifications since its last poll', async () => {
    setVisibility('hidden');
    serveCounts(2, 3);
    initNotificationCount();
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('2');
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('3');
    expect(FakeNotification.shown).toEqual(['You have 2 new notifications', 'You have 1 new notification']);
  });

  test('AC7 a visible tab does not alert, and a lower count does not alert', async () => {
    setVisibility('visible');
    serveCounts(2);
    initNotificationCount();
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('2');
    setVisibility('hidden');
    serveCounts(1);
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('1');
    expect(FakeNotification.shown).toEqual([]);
  });

  test('polling keeps going when the browser refuses to show the alert', async () => {
    // Android Chrome exposes Notification but its constructor throws "Illegal constructor"
    vi.stubGlobal('Notification', class {
      static permission = 'granted';
      constructor() {
        throw new TypeError('Illegal constructor');
      }
    });
    vi.spyOn(console, 'error').mockImplementation(() => {});
    setVisibility('hidden');
    serveCounts(1, 2);
    initNotificationCount();
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('1');
    await vi.advanceTimersByTimeAsync(pollEvery);
    expect(countText()).toBe('2');
    expect(GET).toHaveBeenCalledTimes(2);
  });
});
