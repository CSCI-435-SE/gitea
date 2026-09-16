import {buildIssueInfoUrl, shouldAttachIssuePopup, getIssueInfo, issueInfoCache, initRefIssueContextPopup} from './ref-issue.ts';
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

test('shouldAttachIssuePopup ignores repository paths that merely contain an issue segment', () => {
  // parseIssueHref's regex is unanchored, so a browsing path under a directory named "issues"
  // parses as a reference; without a path check these fire a doomed /info request per hover
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/src/branch/main/tests/issues/1234.go">f</a>'), '/owner/repo')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/src/branch/main/issues/12">f</a>'), '/owner/repo')).toBe(false);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/wiki/issues/5">w</a>'), '/owner/repo')).toBe(false);
});

test('shouldAttachIssuePopup attaches to sub-paths of a real issue', () => {
  // the diff and attachment tabs are still the issue, so they keep their preview
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/pulls/1/files">files</a>'), '/other/repo/issues/9')).toBe(true);
  expect(shouldAttachIssuePopup(makeLink('<a href="/owner/repo/issues/3/attachments">a</a>'), '/other/repo/issues/9')).toBe(true);
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

// concurrent: false: these share the GET mock and its call counter, which vitest's default
// concurrent scheduling (see vitest.config.ts) would let interleave across tests
describe('getIssueInfo caching', {concurrent: false}, () => {
  beforeEach(() => {
    issueInfoCache.clear();
    vi.mocked(GET).mockReset();
  });

  test('caches successful responses', async () => {
    vi.mocked(GET).mockResolvedValue({
      ok: true,
      json: async () => ({convertedIssue: {number: 1}, renderedLabels: ''}),
    } as unknown as Response);

    const first = await getIssueInfo('/owner/repo/issues/1/info');
    const second = await getIssueInfo('/owner/repo/issues/1/info');
    expect(first).toBe(second);
    expect(vi.mocked(GET).mock.calls.length).toEqual(1);
  });

  test('caches failures so repeat hovers do not refetch', async () => {
    vi.mocked(GET).mockResolvedValue({ok: false, statusText: 'Not Found'} as unknown as Response);

    await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
    await expect(getIssueInfo('/owner/repo/issues/404/info')).rejects.toThrow();
    expect(vi.mocked(GET).mock.calls.length).toEqual(1);
  });

  test('caches network errors', async () => {
    vi.mocked(GET).mockRejectedValue(new Error('network down'));

    await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
    await expect(getIssueInfo('/owner/repo/issues/500/info')).rejects.toThrow();
    expect(vi.mocked(GET).mock.calls.length).toEqual(1);
  });

  // exercises the real hover handler (not just getIssueInfo) because the silence requirement is
  // about what initRefIssueContextPopup's catch logs, not about the cache itself; the link has no
  // ref-issue class or container, proving the handler now attaches via shouldAttachIssuePopup alone
  test('a failed hover logs once; a repeat hover on the same link stays silent', async () => {
    vi.mocked(GET).mockResolvedValue({ok: false, statusText: 'Not Found'} as unknown as Response);
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const link = makeLink('<a href="/owner/repo/issues/999">#999</a>');
    document.body.append(link);
    try {
      initRefIssueContextPopup();

      link.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
      await new Promise((resolve) => setTimeout(resolve, 350)); // past the 300ms hover delay
      expect(errorSpy).toHaveBeenCalledTimes(1);

      // the link is re-hoverable once the first attempt's catch clears data-ref-issue-popup;
      // this second lookup hits the cached failure and must not log again
      link.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
      await new Promise((resolve) => setTimeout(resolve, 350));
      expect(errorSpy).toHaveBeenCalledTimes(1);
    } finally {
      errorSpy.mockRestore();
      link.remove();
    }
  });
});
