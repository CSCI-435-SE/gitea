import {renderSimilarIssues} from './repo-issue-similar.ts';
import type {Issue} from '../types.ts';

function makePanel(): HTMLElement {
  const panel = document.createElement('div');
  panel.setAttribute('data-locale-heading', 'Similar issues');
  panel.hidden = true;
  return panel;
}

// fills in the fields renderSimilarIssues/getIssueIcon/getIssueColorClass never look at, so each
// test only has to spell out what it actually cares about
function makeIssue(overrides: Partial<Issue>): Issue {
  return {
    id: 1,
    number: 1,
    title: 'title',
    body: '',
    state: 'open',
    created_at: '',
    html_url: '',
    repository: {full_name: '', html_url: ''},
    labels: [],
    ...overrides,
  };
}

describe('renderSimilarIssues', () => {
  test('renders a row per issue and reveals the panel', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [
      makeIssue({number: 7, title: 'Login page crashes', state: 'open', html_url: '/o/r/issues/7'}),
      makeIssue({number: 9, title: 'Login is broken', state: 'closed', html_url: '/o/r/issues/9'}),
      makeIssue({number: 11, title: 'Fix login', state: 'closed', html_url: '/o/r/pulls/11', pull_request: {draft: false, merged: true}}),
    ]);

    expect(panel.hidden).toBeFalsy();
    expect(panel.textContent).toContain('Similar issues');
    const links = panel.querySelectorAll('a');
    expect(links).toHaveLength(3);
    expect(links[0].getAttribute('href')).toEqual('/o/r/issues/7');
    expect(links[0].textContent).toContain('Login page crashes');
    expect(links[0].getAttribute('target')).toEqual('_blank');

    // open issue, closed issue, and merged pull request each get their own icon color;
    // this is the exact case getIssueIcon/getIssueColorClass exist to get right and a hand-rolled
    // open/closed check would get wrong (a merged PR is not just "a closed issue")
    expect(links[0].querySelector('svg')!.getAttribute('class')).toContain('tw-text-green');
    expect(links[1].querySelector('svg')!.getAttribute('class')).toContain('tw-text-red');
    expect(links[2].querySelector('svg')!.getAttribute('class')).toContain('tw-text-purple');
  });

  test('hides the panel when there are no results', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [makeIssue({number: 7, title: 'x', state: 'open', html_url: '/o/r/issues/7'})]);
    expect(panel.hidden).toBeFalsy();

    renderSimilarIssues(panel, []);
    expect(panel.hidden).toBeTruthy();
    expect(panel.querySelectorAll('a')).toHaveLength(0);
  });

  test('escapes titles rather than injecting them as markup', () => {
    const panel = makePanel();
    renderSimilarIssues(panel, [
      makeIssue({number: 1, title: '<img src=x onerror=alert(1)>', state: 'open', html_url: '/o/r/issues/1'}),
    ]);
    expect(panel.querySelector('img')).toBeNull();
    expect(panel.textContent).toContain('<img src=x onerror=alert(1)>');
  });
});
