import type {Issue} from '../types.ts';
import type {SvgName} from '../svg.ts';

// the getIssueIcon/getIssueColorClass logic should be kept the same as "templates/shared/issueicon.tmpl",
// and closeReasonLooks the same as the "list" look in "templates/shared/close_reason_icon.tmpl"
const closeReasonLooks: Record<Exclude<Issue['close_reason'], ''>, {icon: SvgName, color: string}> = {
  completed: {icon: 'octicon-issue-closed', color: 'tw-text-purple'},
  not_planned: {icon: 'octicon-skip', color: 'tw-text-text-light'},
  duplicate: {icon: 'octicon-duplicate', color: 'tw-text-text-light'},
  other: {icon: 'octicon-note', color: 'tw-text-text-light'},
};

export function getIssueIcon(issue: Issue): SvgName {
  if (issue.pull_request) {
    if (issue.state === 'open') {
      if (issue.pull_request.draft) {
        return 'octicon-git-pull-request-draft'; // WIP PR
      }
      return 'octicon-git-pull-request'; // Open PR
    } else if (issue.pull_request.merged) {
      return 'octicon-git-merge'; // Merged PR
    }
    return issue.close_reason ? closeReasonLooks[issue.close_reason].icon : 'octicon-git-pull-request-closed'; // Closed PR
  }

  if (issue.state === 'open') {
    return 'octicon-issue-opened'; // Open Issue
  }
  return issue.close_reason ? closeReasonLooks[issue.close_reason].icon : 'octicon-issue-closed'; // Closed Issue
}

export function getIssueColorClass(issue: Issue) {
  if (issue.pull_request) {
    if (issue.state === 'open') {
      if (issue.pull_request.draft) {
        return 'tw-text-text-light'; // WIP PR
      }
      return 'tw-text-green'; // Open PR
    } else if (issue.pull_request.merged) {
      return 'tw-text-purple'; // Merged PR
    }
    return issue.close_reason ? closeReasonLooks[issue.close_reason].color : 'tw-text-red'; // Closed PR
  }

  if (issue.state === 'open') {
    return 'tw-text-green'; // Open Issue
  }
  return issue.close_reason ? closeReasonLooks[issue.close_reason].color : 'tw-text-red'; // Closed Issue
}
