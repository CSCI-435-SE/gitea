import {announce, announceSelectionChange} from './aria-announce.ts';

const liveRegions = () => document.querySelectorAll('[role="status"]');

// these tests share document.body and the module's live region, so they must not run concurrently (see vitest.config.ts)
describe('announce', {concurrent: false}, () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test('announce-uses-one-region-on-body', () => {
    announce('first');
    vi.advanceTimersByTime(100);
    announce('second');
    vi.advanceTimersByTime(100);
    expect(liveRegions()).toHaveLength(1);
    expect(liveRegions()[0].parentElement).toBe(document.body);
    expect(liveRegions()[0].textContent).toBe('second');
  });

  test('announce-batches-messages-in-the-same-window', () => {
    announce('a deselected');
    announce('b selected');
    expect(liveRegions()[0].textContent).toBe('');
    vi.advanceTimersByTime(100);
    expect(Array.from(liveRegions()[0].children, (el) => el.textContent)).toEqual(['a deselected', 'b selected']);
  });

  test('announce-recreates-a-removed-region', () => {
    document.body.replaceChildren();
    announce('again');
    vi.advanceTimersByTime(100);
    expect(liveRegions()).toHaveLength(1);
    expect(liveRegions()[0].textContent).toBe('again');
  });

  test('announceSelectionChange-uses-localised-strings', () => {
    window.config.i18n.selected_item_str = 'Selected "%s"';
    window.config.i18n.deselected_item_str = 'Deselected "%s"';
    announceSelectionChange('bug', true);
    announceSelectionChange('docs', false);
    vi.advanceTimersByTime(100);
    expect(Array.from(liveRegions()[0].children, (el) => el.textContent)).toEqual(['Selected "bug"', 'Deselected "docs"']);
  });

  test('announce-drops-messages-queued-for-a-removed-region', () => {
    announce('stale');
    document.body.replaceChildren();
    announce('fresh');
    vi.advanceTimersByTime(100);
    expect(Array.from(liveRegions()[0].children, (el) => el.textContent)).toEqual(['fresh']);
  });
});
