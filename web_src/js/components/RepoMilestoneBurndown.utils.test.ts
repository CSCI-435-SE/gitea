import {burndownAxisMax, burndownSummary, formatLocale, idealLine, projectionLine, type MilestoneBurndown} from './RepoMilestoneBurndown.utils.ts';

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
    points: [{date: '2026-01-05', remaining: 3, scope: 3}, {date: '2026-01-06', remaining: 2, scope: 3}],
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
