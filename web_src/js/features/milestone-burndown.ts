import {createApp} from 'vue';

export async function initRepoMilestoneBurndown() {
  const el = document.querySelector('#milestone-burndown-chart');
  if (!el) return;

  try {
    // inside the try, so a chunk that fails to load still gets the localised fallback
    const {default: RepoMilestoneBurndown} = await import('../components/RepoMilestoneBurndown.vue');
    const View = createApp(RepoMilestoneBurndown, {
      canReadIssues: el.hasAttribute('data-can-read-issues'),
      canReadPulls: el.hasAttribute('data-can-read-pulls'),
      locale: {
        loadingTitle: el.getAttribute('data-locale-loading-title'),
        loadingTitleFailed: el.getAttribute('data-locale-loading-title-failed'),
        loadingInfo: el.getAttribute('data-locale-loading-info'),
        title: el.getAttribute('data-locale-title'),
        remaining: el.getAttribute('data-locale-remaining'),
        scope: el.getAttribute('data-locale-scope'),
        ideal: el.getAttribute('data-locale-ideal'),
        projection: el.getAttribute('data-locale-projection'),
        scopeAdded: el.getAttribute('data-locale-scope-added'),
        scopeRemoved: el.getAttribute('data-locale-scope-removed'),
        includePulls: el.getAttribute('data-locale-include-pulls'),
        changesOn: el.getAttribute('data-locale-changes-on'),
        changeClosed: el.getAttribute('data-locale-change-closed'),
        changeReopened: el.getAttribute('data-locale-change-reopened'),
        changesMore: el.getAttribute('data-locale-changes-more'),
        empty: el.getAttribute('data-locale-empty'),
        noDeadline: el.getAttribute('data-locale-no-deadline'),
        statusDone: el.getAttribute('data-locale-status-done'),
        statusClosed: el.getAttribute('data-locale-status-closed'),
        statusNotBurning: el.getAttribute('data-locale-status-not-burning'),
        statusInsufficient: el.getAttribute('data-locale-status-insufficient'),
        projected: el.getAttribute('data-locale-projected'),
        projectedOnTime: el.getAttribute('data-locale-projected-on-time'),
        projectedOnTimeOne: el.getAttribute('data-locale-projected-on-time-one'),
        projectedOnDueDate: el.getAttribute('data-locale-projected-on-due-date'),
        projectedLate: el.getAttribute('data-locale-projected-late'),
        projectedLateOne: el.getAttribute('data-locale-projected-late-one'),
      },
    });
    View.mount(el);
  } catch (err) {
    console.error('RepoMilestoneBurndown failed to load', err);
    el.textContent = el.getAttribute('data-locale-component-failed-to-load');
  }
}
