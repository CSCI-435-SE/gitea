import {sanitizeRepoName, substituteRepoOpenWithUrl, updateIssuesMeta} from './repo-common.ts';

test('substituteRepoOpenWithUrl', () => {
  // For example: "x-github-client://openRepo/https://github.com/go-gitea/gitea"
  expect(substituteRepoOpenWithUrl('proto://a/{url}', 'https://gitea')).toEqual('proto://a/https://gitea');
  expect(substituteRepoOpenWithUrl('proto://a?link={url}', 'https://gitea')).toEqual('proto://a?link=https%3A%2F%2Fgitea');
});

test('sanitizeRepoName', () => {
  expect(sanitizeRepoName(' a b ')).toEqual('a-b');
  expect(sanitizeRepoName('a-b_c.git ')).toEqual('a-b_c');
  expect(sanitizeRepoName('/x.git/')).toEqual('-x.git-');
  expect(sanitizeRepoName('.profile')).toEqual('.profile');
  expect(sanitizeRepoName('.profile.')).toEqual('.profile');
  expect(sanitizeRepoName('.pro..file')).toEqual('.pro.file');

  expect(sanitizeRepoName('foo.rss.atom.git.wiki')).toEqual('foo');

  expect(sanitizeRepoName('.')).toEqual('');
  expect(sanitizeRepoName('..')).toEqual('');
  expect(sanitizeRepoName('-')).toEqual('');
});

test('updateIssuesMeta', async () => {
  // the browser's fetch rather than a mock of fetch.ts, which another test file may already have loaded for real (isolate: false)
  const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, {status: 200}));
  const sent = (url: string) => Object.fromEntries(fetchSpy.mock.calls.find(([u]) => u === url)![1]!.body as URLSearchParams);
  try {
    await updateIssuesMeta('/user2/repo1/issues/status', 'close', '1,2', '', 'not_planned');
    expect(sent('/user2/repo1/issues/status')).toEqual({action: 'close', issue_ids: '1,2', id: '', close_reason: 'not_planned'});
    // every other action sends no reason at all, as before
    await updateIssuesMeta('/user2/repo1/issues/labels', 'toggle', '1,2', '5');
    expect(sent('/user2/repo1/issues/labels')).toEqual({action: 'toggle', issue_ids: '1,2', id: '5'});
  } finally {
    fetchSpy.mockRestore(); // a failed assertion must not leave fetch mocked for the files after this one
  }
});
