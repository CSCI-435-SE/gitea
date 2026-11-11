import {buildNotificationFaviconSvg, buildStatusFaviconSvg, resetActionFavicon, syncActionRunFavicon, syncNotificationFaviconDot} from './favicon-status.ts';

test('buildStatusFaviconSvg uses action status icons', () => {
  const success = buildStatusFaviconSvg('success');
  expect(success).toContain('viewBox="0 0 640 640"');
  expect(success).toContain('fill:#609926');
  expect(success).toContain('data-actions-status-name="success"');

  const running = buildStatusFaviconSvg('running');
  expect(running).toContain('data-actions-status-name="running"');

  const failure = buildStatusFaviconSvg('failure');
  expect(failure).toContain('data-actions-status-name="failure"');
});

test('syncActionRunFavicon updates favicon links', () => {
  document.head.innerHTML = `
    <link rel="icon" href="/assets/img/favicon.svg" type="image/svg+xml">
    <link rel="alternate icon" href="/assets/img/favicon.png" type="image/png">
  `;
  const links = Array.from(document.querySelectorAll<HTMLLinkElement>('link[rel~="icon"]'));
  syncActionRunFavicon('running');
  for (const link of links) {
    expect(link.href).toMatch(/^data:image\/svg\+xml,/);
    expect(decodeURIComponent(link.href)).toContain('data-actions-status-name="running"');
  }
  resetActionFavicon();
  expect(links[0].href).toContain('favicon.svg');
});

// the module remembers the page's links once, as on a real page, so keep the links an earlier test created
function setupFaviconLinks() {
  if (!document.querySelector('link[rel~="icon"]')) {
    document.head.innerHTML = `
      <link rel="icon" href="/assets/img/favicon.svg" type="image/svg+xml">
      <link rel="alternate icon" href="/assets/img/favicon.png" type="image/png">
    `;
  }
  return Array.from(document.querySelectorAll<HTMLLinkElement>('link[rel~="icon"]'));
}

const showsDot = (link: HTMLLinkElement) => decodeURIComponent(link.href).includes('data-notification-dot');
const showsRunStatus = (link: HTMLLinkElement) => decodeURIComponent(link.href).includes('data-actions-status-name=');

describe('notification dot on the tab icon', {concurrent: false}, () => {
  afterEach(() => {
    resetActionFavicon();
    syncNotificationFaviconDot(false);
  });

  test('the dot uses the primary color, like the navbar unread count', () => {
    document.documentElement.style.setProperty('--color-primary', '#123456');
    expect(buildNotificationFaviconSvg()).toContain('fill="#123456"');
    document.documentElement.style.removeProperty('--color-primary');
  });

  test('AC1 unread notifications add a dot and reading them all restores the default icon', () => {
    const links = setupFaviconLinks();
    syncNotificationFaviconDot(true);
    for (const link of links) expect(showsDot(link)).toBe(true);
    syncNotificationFaviconDot(false);
    expect(links[0].href).toContain('favicon.svg');
    expect(links[1].href).toContain('favicon.png');
  });

  test('AC2 an Actions run status wins over the dot, and the dot returns when the run page is left', () => {
    const links = setupFaviconLinks();
    syncNotificationFaviconDot(true);
    syncActionRunFavicon('running');
    expect(showsRunStatus(links[0])).toBe(true);
    expect(showsDot(links[0])).toBe(false);
    syncNotificationFaviconDot(true); // a count update while on the run page must not take the icon back
    expect(showsRunStatus(links[0])).toBe(true);
    resetActionFavicon();
    expect(showsDot(links[0])).toBe(true);
  });

  test('AC2 leaving a run page with nothing unread restores the default icon', () => {
    const links = setupFaviconLinks();
    syncActionRunFavicon('success');
    resetActionFavicon();
    expect(links[0].href).toContain('favicon.svg');
  });
});
