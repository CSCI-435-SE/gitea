import {buildIssueInfoUrl} from './ref-issue.ts';

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
