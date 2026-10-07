// mirrors MilestoneBurndown in services/issue/milestone_burndown.go
export type BurndownStatus = 'empty' | 'done' | 'closed' | 'projected' | 'not_burning' | 'insufficient_data';

export type BurndownPoint = {
  date: string, // "YYYY-MM-DD" in the instance's zone, used as is so every viewer sees the same day
  remaining: number,
  scope: number,
  added: number, // items that joined that day; the first day of work is never marked
  removed: number,
  changes?: BurndownChange[], // left out on days nothing changed
};

export type BurndownChange = {
  index: number,
  title: string,
  isPull: boolean,
  kind: 'closed' | 'reopened' | 'added' | 'removed',
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

export type MarkerPoint = LinePoint & {count: number};

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

// a marker on the scope line for each day the milestone gained or lost items, carrying how many
export function scopeMarkers(data: MilestoneBurndown, kind: 'added' | 'removed'): MarkerPoint[] {
  return data.points.filter((p) => p[kind] > 0).map((p) => ({x: p.date, y: p.scope, count: p[kind]}));
}

const includePullsParam = 'include_pulls';

export function includePullsFromSearch(search: string): boolean {
  return new URLSearchParams(search).get(includePullsParam) === '1';
}

// the page's query with the toggle set or cleared, keeping the issue list's own filters intact
export function searchWithIncludePulls(search: string, includePulls: boolean): string {
  const params = new URLSearchParams(search);
  if (includePulls) {
    params.set(includePullsParam, '1');
  } else {
    params.delete(includePullsParam);
  }
  const query = params.toString();
  return query ? `?${query}` : '';
}

export function burndownDataUrl(link: string, includePulls: boolean): string {
  return includePulls ? `${link}?${includePullsParam}=1` : link;
}

export function changeLink(repoLink: string, change: BurndownChange): string {
  return `${repoLink}/${change.isPull ? 'pulls' : 'issues'}/${change.index}`;
}

export function changesOn(data: MilestoneBurndown, date: string): BurndownChange[] {
  return data.points.find((p) => p.date === date)?.changes ?? [];
}

// the list starts on the latest day anything changed, so it is useful before anyone hovers
export function latestChangeDate(data: MilestoneBurndown): string {
  return data.points.findLast((p) => p.changes !== undefined)?.date ?? '';
}
