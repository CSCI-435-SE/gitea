// mirrors MilestoneBurndown in services/issue/milestone_burndown.go
export type BurndownStatus = 'empty' | 'done' | 'closed' | 'projected' | 'not_burning' | 'insufficient_data';

export type BurndownPoint = {
  date: string, // "YYYY-MM-DD" in the instance's zone, used as is so every viewer sees the same day
  remaining: number,
  scope: number,
};

export type MilestoneBurndown = {
  points: BurndownPoint[],
  deadline: string,
  ideal: {from: string, fromValue: number, to: string} | null,
  status: BurndownStatus,
  projected: string,
  daysLate: number | null,
  completedOn: string,
};

export type BurndownSummaryLocale = {
  statusDone: string,
  statusClosed: string,
  statusNotBurning: string,
  statusInsufficient: string,
  projected: string,
  projectedOnTime: string,
  projectedOnDueDate: string,
  projectedLate: string,
};

export type LinePoint = {x: string, y: number};

// fills %s and %d in order, the way the server-side locale strings are written
export function formatLocale(template: string, ...args: Array<string | number>): string {
  let next = 0;
  return template.replace(/%[sd]/g, () => String(args[next++]));
}

// the one-line verdict above the chart; it carries the late or early state in words, not only in colour
export function burndownSummary(data: MilestoneBurndown, locale: BurndownSummaryLocale): string {
  switch (data.status) {
    case 'done':
      return formatLocale(locale.statusDone, data.completedOn);
    case 'closed':
      return locale.statusClosed;
    case 'not_burning':
      return locale.statusNotBurning;
    case 'insufficient_data':
      return locale.statusInsufficient;
    case 'projected':
      if (data.daysLate === null) return formatLocale(locale.projected, data.projected);
      if (data.daysLate > 0) return formatLocale(locale.projectedLate, data.projected, data.daysLate);
      if (data.daysLate === 0) return formatLocale(locale.projectedOnDueDate, data.projected);
      return formatLocale(locale.projectedOnTime, data.projected, -data.daysLate);
    default:
      return '';
  }
}

export function idealLine(data: MilestoneBurndown): LinePoint[] {
  if (!data.ideal) return [];
  return [{x: data.ideal.from, y: data.ideal.fromValue}, {x: data.ideal.to, y: 0}];
}

// from today's remaining work down to zero on the projected day
export function projectionLine(data: MilestoneBurndown): LinePoint[] {
  if (data.status !== 'projected' || data.points.length === 0) return [];
  const today = data.points[data.points.length - 1];
  return [{x: today.date, y: today.remaining}, {x: data.projected, y: 0}];
}

// the axis must reach the due date and the projected finish, or their lines are cut off
export function burndownAxisMax(data: MilestoneBurndown): string {
  let latest = data.points[data.points.length - 1].date;
  for (const date of [data.deadline, data.projected]) {
    if (date > latest) latest = date; // ISO dates sort as strings, and an empty one never wins
  }
  return latest;
}
