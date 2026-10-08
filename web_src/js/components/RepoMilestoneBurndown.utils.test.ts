import {
  burndownAxisMax,
  burndownDataUrl,
  burndownSummary,
  changeLink,
  changeTooltipLines,
  changesOn,
  formatLocale,
  idealLine,
  includePullsFromSearch,
  latestChangeDate,
  projectionLine,
  scopeMarkers,
  searchWithIncludePulls,
  type BurndownChange,
  type MilestoneBurndown,
} from './RepoMilestoneBurndown.utils.ts';

const locale = {
  statusDone: 'Completed on %s',
  statusClosed: 'closed',
  statusNotBurning: 'not burning',
  statusInsufficient: 'insufficient',
  projected: 'Projected %s',
  projectedOnTime: 'Projected %s, %d early',
  projectedOnDueDate: 'Projected %s, on the day',
  projectedLate: 'Projected %s, %d late',
};

function burndown(fields: Partial<MilestoneBurndown>): MilestoneBurndown {
  return {
    points: [
      {date: '2026-01-05', remaining: 3, scope: 3, added: 0, removed: 0},
      {date: '2026-01-06', remaining: 2, scope: 3, added: 0, removed: 0},
    ],
    deadline: '',
    ideal: null,
    status: 'projected',
    projected: '',
    daysLate: null,
    completedOn: '',
    ...fields,
  };
}

test('formatLocale', () => {
  expect(formatLocale('%s then %d', 'a', 2)).toEqual('a then 2');
  expect(formatLocale('no placeholders')).toEqual('no placeholders');
});

test('burndownSummary', () => {
  expect(burndownSummary(burndown({status: 'done', completedOn: '2026-01-06'}), locale)).toEqual('Completed on 2026-01-06');
  expect(burndownSummary(burndown({status: 'closed'}), locale)).toEqual('closed');
  expect(burndownSummary(burndown({status: 'not_burning'}), locale)).toEqual('not burning');
  expect(burndownSummary(burndown({status: 'insufficient_data'}), locale)).toEqual('insufficient');
  expect(burndownSummary(burndown({status: 'empty'}), locale)).toEqual('');

  expect(burndownSummary(burndown({projected: '2026-01-09'}), locale)).toEqual('Projected 2026-01-09');
  expect(burndownSummary(burndown({projected: '2026-01-09', daysLate: 3}), locale)).toEqual('Projected 2026-01-09, 3 late');
  expect(burndownSummary(burndown({projected: '2026-01-09', daysLate: 0}), locale)).toEqual('Projected 2026-01-09, on the day');
  // early is sent as negative days late and read back as a positive count
  expect(burndownSummary(burndown({projected: '2026-01-09', daysLate: -4}), locale)).toEqual('Projected 2026-01-09, 4 early');
});

test('idealLine', () => {
  expect(idealLine(burndown({}))).toEqual([]);
  expect(idealLine(burndown({ideal: {from: '2026-01-05', fromValue: 3, to: '2026-01-10'}}))).toEqual([
    {x: '2026-01-05', y: 3},
    {x: '2026-01-10', y: 0},
  ]);
});

test('projectionLine', () => {
  expect(projectionLine(burndown({status: 'not_burning'}))).toEqual([]);
  expect(projectionLine(burndown({projected: '2026-01-08'}))).toEqual([
    {x: '2026-01-06', y: 2},
    {x: '2026-01-08', y: 0},
  ]);
});

test('burndownAxisMax', () => {
  expect(burndownAxisMax(burndown({}))).toEqual('2026-01-06');
  expect(burndownAxisMax(burndown({deadline: '2026-01-10'}))).toEqual('2026-01-10');
  expect(burndownAxisMax(burndown({deadline: '2026-01-10', projected: '2026-01-14'}))).toEqual('2026-01-14');
  // a due date already passed does not pull the axis back
  expect(burndownAxisMax(burndown({deadline: '2026-01-01'}))).toEqual('2026-01-06');
});

test('scopeMarkers', () => {
  expect(scopeMarkers(burndown({}), 'added')).toEqual([]);
  const changed = burndown({points: [
    {date: '2026-01-05', remaining: 3, scope: 3, added: 0, removed: 0},
    {date: '2026-01-06', remaining: 4, scope: 4, added: 2, removed: 1},
    {date: '2026-01-07', remaining: 3, scope: 3, added: 0, removed: 1},
  ]});
  // markers sit on the scope line and carry the count, which the tooltip reads instead of y
  expect(scopeMarkers(changed, 'added')).toEqual([{x: '2026-01-06', y: 4, count: 2}]);
  expect(scopeMarkers(changed, 'removed')).toEqual([
    {x: '2026-01-06', y: 4, count: 1},
    {x: '2026-01-07', y: 3, count: 1},
  ]);
});

test('include pull requests in the URL', () => {
  expect(includePullsFromSearch('')).toBe(false);
  expect(includePullsFromSearch('?include_pulls=1')).toBe(true);
  expect(includePullsFromSearch('?include_pulls=0')).toBe(false);
  // the issue list's own filters on the same page survive the toggle both ways
  expect(searchWithIncludePulls('?state=closed&q=bug', true)).toEqual('?state=closed&q=bug&include_pulls=1');
  expect(searchWithIncludePulls('?state=closed&include_pulls=1', false)).toEqual('?state=closed');
  expect(searchWithIncludePulls('?include_pulls=1', false)).toEqual('');
  expect(searchWithIncludePulls('', true)).toEqual('?include_pulls=1');

  expect(burndownDataUrl('/o/r/milestone/3/burndown', false)).toEqual('/o/r/milestone/3/burndown');
  expect(burndownDataUrl('/o/r/milestone/3/burndown', true)).toEqual('/o/r/milestone/3/burndown?include_pulls=1');
});

test('changes', () => {
  const issue: BurndownChange = {index: 7, title: 'Fix it', isPull: false, kind: 'closed'};
  const pull: BurndownChange = {index: 8, title: 'Merge it', isPull: true, kind: 'closed'};
  expect(changeLink('/o/r', issue)).toEqual('/o/r/issues/7');
  expect(changeLink('/o/r', pull)).toEqual('/o/r/pulls/8');

  const data = burndown({points: [
    {date: '2026-01-05', remaining: 2, scope: 2, added: 0, removed: 0, changes: [issue]},
    {date: '2026-01-06', remaining: 1, scope: 2, added: 0, removed: 0, changes: [pull]},
    {date: '2026-01-07', remaining: 1, scope: 2, added: 0, removed: 0},
  ]});
  expect(changesOn(data, '2026-01-06')).toEqual([pull]);
  expect(changesOn(data, '2026-01-07')).toEqual([]); // a quiet day has no list in the JSON
  expect(changesOn(data, '2026-02-01')).toEqual([]);
  expect(latestChangeDate(data)).toEqual('2026-01-06');
  expect(latestChangeDate(burndown({}))).toEqual('');
});

test('changeTooltipLines', () => {
  const labels = {closed: 'Closed', reopened: 'Reopened', added: 'Added to milestone', removed: 'Removed from milestone'};
  const change = (index: number, title: string, kind: BurndownChange['kind']): BurndownChange => ({index, title, isPull: false, kind});
  expect(changeTooltipLines([], labels, '%d more')).toEqual([]);
  expect(changeTooltipLines([change(3, 'Fix login', 'closed'), change(9, 'SSO', 'added')], labels, '%d more')).toEqual([
    'Closed: #3 Fix login',
    'Added to milestone: #9 SSO',
  ]);
  // a long title is cut so the tooltip stays inside the chart
  const [long] = changeTooltipLines([change(1, 'x'.repeat(80), 'reopened')], labels, '%d more');
  expect(long).toEqual(`Reopened: #1 ${'x'.repeat(49)}…`);
  // a busy day shows five and counts the rest, which the list under the chart shows in full
  const busy = Array.from({length: 8}, (_, i) => change(i + 1, `item ${i + 1}`, 'closed'));
  const lines = changeTooltipLines(busy, labels, '%d more');
  expect(lines).toHaveLength(6);
  expect(lines[4]).toEqual('Closed: #5 item 5');
  expect(lines[5]).toEqual('3 more');
});
