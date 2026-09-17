import {registerGlobalInitFunc} from '../modules/observer.ts';
import {GET} from '../modules/fetch.ts';
import {svg} from '../svg.ts';
import {errorName} from '../modules/errors.ts';
import {getIssueColorClass, getIssueIcon} from './issue.ts';
import type {Issue} from '../types.ts';

const debounceMs = 300;
const minTitleLength = 3;

function renderIssueRow(issue: Issue): HTMLAnchorElement {
  const link = document.createElement('a');
  link.classList.add('issue-similar-item', 'flex-text-block');
  link.href = issue.html_url;
  link.target = '_blank';
  link.rel = 'noopener';

  // reuse the shared icon/color mapping (also used by TextExpander.ts) so merged/draft/closed
  // pull requests get their real icon instead of a plain open/closed issue circle
  const icon = document.createElement('span');
  icon.innerHTML = svg(getIssueIcon(issue), 16, [getIssueColorClass(issue)]);
  link.append(icon);

  const index = document.createElement('span');
  index.classList.add('tw-text-text-light-3');
  index.textContent = `#${issue.number}`;
  link.append(index);

  // textContent, not innerHTML: issue titles are user input.
  const title = document.createElement('span');
  title.classList.add('gt-ellipsis');
  title.textContent = issue.title;
  link.append(title);

  return link;
}

// Exported for tests: rendering is the part worth asserting on, separately from the fetching.
//
// The heading and list nodes are created once and reused across calls: assistive tech only
// announces mutations inside a live region it already knows about, not a brand new subtree that
// happens to carry aria-live on its root, so the list element (and its aria-live attribute) must
// survive from render to render with only its children replaced.
export function renderSimilarIssues(panel: HTMLElement, issues: Issue[]): void {
  let heading = panel.querySelector<HTMLElement>('.issue-similar-heading');
  if (!heading) {
    heading = document.createElement('div');
    heading.classList.add('issue-similar-heading');
    panel.append(heading);
  }
  heading.textContent = panel.getAttribute('data-locale-heading')!;

  let list = panel.querySelector<HTMLElement>('.issue-similar-list');
  if (!list) {
    list = document.createElement('div');
    list.classList.add('issue-similar-list');
    list.setAttribute('aria-live', 'polite');
    panel.append(list);
  }

  if (!issues.length) {
    list.replaceChildren();
    panel.hidden = true;
    return;
  }

  list.replaceChildren(...issues.map(renderIssueRow));
  panel.hidden = false;
}

function initSimilarIssuesPanel(panel: HTMLElement): void {
  const titleInput = document.querySelector<HTMLInputElement>('#issue_title');
  if (!titleInput) return;

  const searchURL = panel.getAttribute('data-search-url')!;
  const isPull = panel.getAttribute('data-is-pull')!;

  let timer: number | undefined;
  let abortController: AbortController | undefined;

  const search = async () => {
    const title = titleInput.value.trim();
    if (title.length < minTitleLength) {
      renderSimilarIssues(panel, []);
      return;
    }

    // A newer keystroke invalidates the request in flight, so a slow earlier response can never
    // repaint over a newer one.
    abortController?.abort();
    abortController = new AbortController();

    try {
      const params = new URLSearchParams({q: title, is_pull: isPull});
      const response = await GET(`${searchURL}?${params}`, {signal: abortController.signal});
      if (!response.ok) {
        renderSimilarIssues(panel, []);
        return;
      }
      renderSimilarIssues(panel, await response.json());
    } catch (err) {
      // An abort from the deliberate abort-on-newer-keystroke above is expected and stays silent;
      // anything else (bad JSON, network failure) is a real bug and must not vanish silently.
      if (errorName(err) !== 'AbortError') console.error('Failed to load similar issues:', err);
      renderSimilarIssues(panel, []);
    }
  };

  titleInput.addEventListener('input', () => {
    window.clearTimeout(timer);
    timer = window.setTimeout(search, debounceMs);
  });

  // a title can already be filled in at load time (the `?title=` deep link, or an issue
  // template's default title) — that is exactly the duplicate-prone case this feature targets,
  // so it must not wait for the user to first edit the field. `search()` itself no-ops without
  // firing a request when the field is empty or below minTitleLength, so no separate guard is needed here.
  search();
}

export function initRepoIssueSimilar(): void {
  registerGlobalInitFunc('initRepoIssueSimilarSuggestions', initSimilarIssuesPanel);
}
