import {getIssueColorClass, getIssueIcon} from './issue.ts';
import type {Issue} from '../types.ts';

function makeIssue(fields: Partial<Issue>): Issue {
  return {
    id: 1, number: 1, title: '', body: '', state: 'closed', close_reason: '', created_at: '', html_url: '',
    repository: {full_name: 'user2/repo1', html_url: ''}, labels: [], ...fields,
  };
}

test('closed issues by close reason, as close_reason_icon.tmpl draws them', () => {
  const looks = [
    ['completed', 'octicon-issue-closed', 'tw-text-purple'],
    ['not_planned', 'octicon-skip', 'tw-text-text-light'],
    ['duplicate', 'octicon-duplicate', 'tw-text-text-light'],
    ['other', 'octicon-note', 'tw-text-text-light'],
    ['', 'octicon-issue-closed', 'tw-text-red'], // closed before close reasons, keeps its old look
  ] as const;
  for (const [close_reason, icon, color] of looks) {
    const issue = makeIssue({close_reason});
    expect(getIssueIcon(issue)).toEqual(icon);
    expect(getIssueColorClass(issue)).toEqual(color);
  }
});

test('closed pull requests by close reason', () => {
  const pull = {draft: false, merged: false};
  expect(getIssueIcon(makeIssue({pull_request: pull, close_reason: 'not_planned'}))).toEqual('octicon-skip');
  expect(getIssueColorClass(makeIssue({pull_request: pull, close_reason: 'not_planned'}))).toEqual('tw-text-text-light');
  expect(getIssueIcon(makeIssue({pull_request: pull}))).toEqual('octicon-git-pull-request-closed');
  expect(getIssueColorClass(makeIssue({pull_request: pull}))).toEqual('tw-text-red');
});

test('merged and open items ignore close reasons', () => {
  const merged = makeIssue({pull_request: {draft: false, merged: true}});
  expect(getIssueIcon(merged)).toEqual('octicon-git-merge');
  expect(getIssueColorClass(merged)).toEqual('tw-text-purple');
  const open = makeIssue({state: 'open'});
  expect(getIssueIcon(open)).toEqual('octicon-issue-opened');
  expect(getIssueColorClass(open)).toEqual('tw-text-green');
});
