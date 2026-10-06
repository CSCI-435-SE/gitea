import {createApp} from 'vue';

export async function initRepoMilestoneBurndown() {
  const el = document.querySelector('#milestone-burndown-chart');
  if (!el) return;

  const {default: RepoMilestoneBurndown} = await import('../components/RepoMilestoneBurndown.vue');
  try {
    const View = createApp(RepoMilestoneBurndown, {
      locale: {
        loadingTitle: el.getAttribute('data-locale-loading-title'),
        loadingTitleFailed: el.getAttribute('data-locale-loading-title-failed'),
        loadingInfo: el.getAttribute('data-locale-loading-info'),
        title: el.getAttribute('data-locale-title'),
        remaining: el.getAttribute('data-locale-remaining'),
        scope: el.getAttribute('data-locale-scope'),
        ideal: el.getAttribute('data-locale-ideal'),
        projection: el.getAttribute('data-locale-projection'),
        empty: el.getAttribute('data-locale-empty'),
        noDeadline: el.getAttribute('data-locale-no-deadline'),
        statusDone: el.getAttribute('data-locale-status-done'),
        statusClosed: el.getAttribute('data-locale-status-closed'),
        statusNotBurning: el.getAttribute('data-locale-status-not-burning'),
        statusInsufficient: el.getAttribute('data-locale-status-insufficient'),
        projected: el.getAttribute('data-locale-projected'),
        projectedOnTime: el.getAttribute('data-locale-projected-on-time'),
        projectedOnDueDate: el.getAttribute('data-locale-projected-on-due-date'),
        projectedLate: el.getAttribute('data-locale-projected-late'),
      },
    });
    View.mount(el);
  } catch (err) {
    console.error('RepoMilestoneBurndown failed to load', err);
    el.textContent = el.getAttribute('data-locale-component-failed-to-load');
  }
}
