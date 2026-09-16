import {buildIssueInfoUrl, shouldAttachIssuePopup, getIssueInfo, issueInfoCache} from './ref-issue.ts';
import {GET} from '../modules/fetch.ts';
import type {Instance} from 'tippy.js';

vi.mock('../modules/fetch.ts', () => ({
  GET: vi.fn(),
}));

test('buildIssueInfoUrl', () => {
  expect(buildIssueInfoUrl('/owner/repo/issues/1')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('/owner/repo/pulls/2')).toEqual('/owner/repo/pulls/2/info');

  // non-canonical links must resolve to the canonical info endpoint, not to a sub-path
  expect(buildIssueInfoUrl('/owner/repo/pulls/1/files')).toEqual('/owner/repo/pulls/1/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/3/attachments')).toEqual('/owner/repo/issues/3/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/1?query=x')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('/owner/repo/issues/1#issuecomment-4')).toEqual('/owner/repo/issues/1/info');
  expect(buildIssueInfoUrl('https://example.com/owner/repo/issues/1')).toEqual('/owner/repo/issues/1/info');

  expect(buildIssueInfoUrl('/owner/repo/issues')).toEqual(null);
  expect(buildIssueInfoUrl('/owner/repo')).toEqual(null);
  expect(buildIssueInfoUrl('')).toEqual(null);
});

test('buildIssueInfoUrl with appSubUrl', () => {
  const oldSubUrl = window.config.appSubUrl;
  window.config.appSubUrl = '/sub';
  try {
    // parseIssueHref strips the sub-path, so the rebuilt URL must re-add it
    expect(buildIssueInfoUrl('/sub/owner/repo/issues/1')).toEqual('/sub/owner/repo/issues/1/info');
  } finally {
    window.config.appSubUrl = oldSubUrl;
  }
});

function makeLink(html: string): HTMLAnchorElement {
  const container = document.createElement('div');
  container.innerHTML = html;
  return container.querySelector('a')!;
}

test('shouldAttachIssuePopup attaches to issue-shaped links', () => {
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/1">#1</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/pulls/2">#2</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a class="ref-issue" href="/owner/repo/issues/1">#1</a>'), '/other/repo/issues/9')).toBe(true);
});

test('shouldAttachIssuePopup ignores links that are not issues', () => {
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo">repo</a>'), '/other/repo/issues/9')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues">issues</a>'), '/other/repo/issues/9')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/explore/repos">explore</a>'), '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup ignores external issue references', () => {
  // these point at Jira/Redmine, where no /info endpoint exists
  const link = makeLink('<a class="ref-external-issue" href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup ignores links to other hosts', () => {
  // parseIssueHref ignores scheme and host, so another forge's URL parses as an issue
  // reference; previewing it would show this instance's unrelated issue of the same path
  const link = makeLink('<a href="https://github.com/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/other/repo/issues/9')).toBe(false);

  // an absolute URL to this instance is still a real reference
  const sameHost = makeLink(`<a href="${window.location.origin}/owner/repo/issues/1">#1</a>`);
  expect(shouldAttachIssuePopup(sameHost, '/other/repo/issues/9')).toBe(true);
});

test('shouldAttachIssuePopup honours the opt-out attribute', () => {
  const onLink = makeLink('<a data-issue-popup="off" href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(onLink, '/other/repo/issues/9')).toBe(false);

  const container = document.createElement('div');
  container.innerHTML = '<div data-issue-popup="off"><a href="/owner/repo/issues/1">#1</a></div>';
  expect(shouldAttachIssuePopup(container.querySelector('a')!, '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup ignores links to the current page', () => {
  const link = makeLink('<a href="/owner/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/owner/repo/issues/1')).toBe(false);

  // a PR is the same object under both path types, so /pulls/1 on the /issues/1 page is still self
  const pullLink = makeLink('<a href="/owner/repo/pulls/1">#1</a>');
  expect(shouldAttachIssuePopup(pullLink, '/owner/repo/issues/1')).toBe(false);

  // a different index in the same repo is a real reference
  const otherLink = makeLink('<a href="/owner/repo/issues/2">#2</a>');
  expect(shouldAttachIssuePopup(otherLink, '/owner/repo/issues/1')).toBe(true);

  // the same index in a different repo is a real reference
  const otherRepoLink = makeLink('<a href="/other/repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(otherRepoLink, '/owner/repo/issues/1')).toBe(true);
});

test('shouldAttachIssuePopup treats owner/repo self-reference case-insensitively', () => {
  // Gitea routes owner/repo case-insensitively but links preserve stored case
  const link = makeLink('<a href="/Owner/Repo/issues/1">#1</a>');
  expect(shouldAttachIssuePopup(link, '/owner/repo/issues/1')).toBe(false);

  // the most common real-world case: hovering an issue link from a non-issue page
  // (repo home, commit list, ...) must not be mistaken for a self-reference
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/1">#1</a>'), '/owner/repo')).toBe(true);
});

test('shouldAttachIssuePopup ignores links that already have a tippy instance attached', () => {
  const link = makeLink('<a href="/owner/repo/issues/1">#1</a>');
  link._tippy = {} as Instance;
  expect(shouldAttachIssuePopup(link, '/other/repo/issues/9')).toBe(false);
});

test('shouldAttachIssuePopup attaches to links with a hash or query suffix', () => {
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/1#issuecomment-5">#1</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/1?tab=files">#1</a>'), '/other/repo/issues/9')).toBe(true);
});

test('getIssueInfo caches successful responses', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockResolvedValue({
    ok: true,
    json: async () => ({convertedIssue: {number: 1}, renderedLabels: ''}),
  } as unknown as Response);

  const first = await getIssueInfo('/owner/repo/issues/1/info');
  const second = await getIssueInfo('/owner/repo/issues/1/info');
  expect(first).toBe(second);
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});

test('getIssueInfo caches failures so repeat hovers do not refetch', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockReset();
  vi.mocked(GET).mockResolvedValue({ok: false, statusText: 'Not Found'} as unknown as Response);

  await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
  await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});

test('getIssueInfo caches network errors', async () => {
  issueInfoCache.clear();
  vi.mocked(GET).mockReset();
  vi.mocked(GET).mockRejectedValue(new Error('network down'));

  await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
  await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
  expect(vi.mocked(GET).mock.calls.length).toEqual(1);
});
