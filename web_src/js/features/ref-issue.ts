import {parseIssueHref} from '../utils.ts';
import {GET} from '../modules/fetch.ts';
import {createApp} from 'vue';
import {createTippy, getAttachedTippyInstance} from '../modules/tippy.ts';
import {addDelegatedEventListener} from '../utils/dom.ts';
import type {Issue} from '../types.ts';

type IssueInfo = {
  convertedIssue: Issue,
  renderedLabels: string,
};

// null marks a lookup that failed, so unreadable or missing issues are not refetched on every hover
export const issueInfoCache = new Map<string, IssueInfo | null>();

// distinguishes a cached failure from a fresh one, so the caller can log the first and stay silent on repeats
class CachedIssueInfoError extends Error {}

// builds the canonical info endpoint from a link's parts, because the link may point at a
// sub-path such as /pulls/1/files where appending /info would 404
export function buildIssueInfoUrl(href: string): string | null {
  const {ownerName, repoName, pathType, indexString} = parseIssueHref(href);
  if (!ownerName) return null;
  return `${window.config.appSubUrl}/${ownerName}/${repoName}/${pathType}/${indexString}/info`;
}

// decides whether a link deserves a hover preview; kept pure so every surface is unit-testable
export function shouldAttachIssuePopup(link: HTMLAnchorElement, currentPath: string): boolean {
  const target = parseIssueHref(link.getAttribute('href')!);
  if (!target.ownerName) return false;
  // another forge's issue URL parses the same way, but /info here would answer for a different issue
  if (link.origin !== window.location.origin) return false;
  // parseIssueHref's regex is unanchored, so a browsing path such as
  // /owner/repo/src/branch/main/issues/12 parses as a reference to issue 12 as well
  const canonicalPath = `${window.config.appSubUrl}/${target.ownerName}/${target.repoName}/${target.pathType}/${target.indexString}`;
  if (link.pathname !== canonicalPath && !link.pathname.startsWith(`${canonicalPath}/`)) return false;
  if (link.classList.contains('ref-external-issue')) return false;
  if (link.closest('[data-issue-popup="off"]')) return false;
  if (getAttachedTippyInstance(link)) return false;

  // previewing the page you are already reading is useless; path type is ignored because a pull
  // request is reachable at both /issues/{index} and /pulls/{index}
  const current = parseIssueHref(currentPath);
  // owner/repo routing is case-insensitive in Gitea, but links preserve the stored case
  return !(current.ownerName?.toLowerCase() === target.ownerName.toLowerCase() &&
    current.repoName?.toLowerCase() === target.repoName.toLowerCase() &&
    current.indexString === target.indexString);
}

export async function getIssueInfo(url: string): Promise<IssueInfo> {
  if (issueInfoCache.has(url)) {
    const cached = issueInfoCache.get(url);
    if (!cached) throw new CachedIssueInfoError('issue info previously failed to load');
    return cached;
  }
  let resp: Response;
  try {
    resp = await GET(url);
  } catch (err) {
    issueInfoCache.set(url, null);
    throw err;
  }
  if (!resp.ok) {
    issueInfoCache.set(url, null);
    throw new Error(resp.statusText || 'Unknown network error');
  }
  const data = await resp.json();
  issueInfoCache.set(url, data);
  return data;
}

async function showRefIssuePopup(link: HTMLAnchorElement) {
  const infoUrl = buildIssueInfoUrl(link.getAttribute('href')!);
  if (!infoUrl) return;
  const [data, {default: ContextPopup}] = await Promise.all([
    getIssueInfo(infoUrl),
    import('../components/ContextPopup.vue'),
  ]);
  const el = document.createElement('div');
  const app = createApp(ContextPopup, {
    issue: data.convertedIssue,
    renderedLabels: data.renderedLabels,
  });
  app.mount(el);
  // suppress ancestor title like from .commit-summary to prevent double tooltip
  link.title = '';
  createTippy(link, {
    theme: 'default',
    content: el,
    trigger: 'mouseenter focus',
    placement: 'top-start',
    interactive: true,
    role: 'dialog',
    interactiveBorder: 5,
    onDestroy: () => app.unmount(),
  }).show();
}

export function initRefIssueContextPopup() {
  const selector = 'a[href]:not([data-ref-issue-popup])';
  addDelegatedEventListener<HTMLAnchorElement, MouseEvent>(document, 'mouseover', selector, (link) => {
    if (!shouldAttachIssuePopup(link, window.location.pathname)) return;
    link.setAttribute('data-ref-issue-popup', '');

    // delay so a mouse passing over the link doesn't fire a fetch
    let timer: ReturnType<typeof setTimeout>;
    const cancel = () => {
      clearTimeout(timer);
      link.removeAttribute('data-ref-issue-popup');
      link.removeEventListener('mouseleave', cancel);
    };
    timer = setTimeout(async () => {
      link.removeEventListener('mouseleave', cancel);
      try {
        await showRefIssuePopup(link);
      } catch (err) {
        // a cached failure already logged on its first occurrence; repeating it on every hover would drown out real signal
        if (!(err instanceof CachedIssueInfoError)) console.error('Failed to load issue info:', err);
        link.removeAttribute('data-ref-issue-popup');
      }
    }, 300);
    link.addEventListener('mouseleave', cancel);
  });
}
