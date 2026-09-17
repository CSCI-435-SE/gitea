import {createApp} from 'vue';

export async function initRepoIssuesChart() {
  const el = document.querySelector('#repo-issues-chart');
  if (!el) return;

  const {default: RepoIssuesChart} = await import('../components/RepoIssuesChart.vue');
  try {
    const View = createApp(RepoIssuesChart, {
      locale: {
        loadingTitle: el.getAttribute('data-locale-loading-title'),
        loadingTitleFailed: el.getAttribute('data-locale-loading-title-failed'),
        loadingInfo: el.getAttribute('data-locale-loading-info'),
        title: el.getAttribute('data-locale-title'),
        open: el.getAttribute('data-locale-open'),
        opened: el.getAttribute('data-locale-opened'),
        closed: el.getAttribute('data-locale-closed'),
        empty: el.getAttribute('data-locale-empty'),
        rangeMonth: el.getAttribute('data-locale-range-month'),
        rangeQuarter: el.getAttribute('data-locale-range-quarter'),
        rangeYear: el.getAttribute('data-locale-range-year'),
        rangeAll: el.getAttribute('data-locale-range-all'),
      },
    });
    View.mount(el);
  } catch (err) {
    console.error('RepoIssuesChart failed to load', err);
    el.textContent = el.getAttribute('data-locale-component-failed-to-load');
  }
}
