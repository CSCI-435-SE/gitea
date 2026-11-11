import {NotificationAlertTracker} from './notification-alert.ts';

test('AC5 a count increase while every tab is hidden alerts once with the number of new notifications', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: false, count: 0});
  expect(tracker.onCount(1)).toEqual({tab: 'a', newCount: 1});
});

test('AC6 several hidden tabs get one alert, sent to the tab used last', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: false, count: 2});
  tracker.setTabState('b', {visible: true, count: 2});
  tracker.setTabState('b', {visible: false, count: 2});
  tracker.setTabState('c', {visible: false, count: 2});
  expect(tracker.onCount(3)).toEqual({tab: 'b', newCount: 1});
});

test('AC7 no alert while any Gitea tab is visible', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: false, count: 0});
  tracker.setTabState('b', {visible: true, count: 0});
  expect(tracker.onCount(1)).toBeNull();
});

test('AC7 no alert when the count stays the same or goes down', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: false, count: 3});
  expect(tracker.onCount(3)).toBeNull();
  expect(tracker.onCount(1)).toBeNull();
});

test('the alert counts notifications that arrived since Gitea was last in view', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: true, count: 1});
  tracker.setTabState('a', {visible: false, count: 1});
  expect(tracker.onCount(2)).toEqual({tab: 'a', newCount: 1});
  expect(tracker.onCount(3)).toEqual({tab: 'a', newCount: 2}); // replaces the first alert, so it covers both
  tracker.setTabState('a', {visible: true, count: 3});
  tracker.setTabState('a', {visible: false, count: 3});
  expect(tracker.onCount(4)).toEqual({tab: 'a', newCount: 1});
});

test('a closed tab is never picked to show the alert', () => {
  const tracker = new NotificationAlertTracker<string>();
  tracker.setTabState('a', {visible: true, count: 0});
  tracker.setTabState('b', {visible: false, count: 0});
  tracker.removeTab('a');
  expect(tracker.onCount(1)).toEqual({tab: 'b', newCount: 1});
  tracker.removeTab('b');
  expect(tracker.onCount(2)).toBeNull();
});
