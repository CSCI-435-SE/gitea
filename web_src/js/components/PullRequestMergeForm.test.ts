import {createApp, nextTick} from 'vue';
import PullRequestMergeForm from './PullRequestMergeForm.vue';

// the props prepareMergeBoxFormProps (routers/web/repo/pull_merge_form.go) sends, reduced to what the buttons read
async function mountMergeForm(props: Record<string, any>) {
  const el = document.createElement('div'); // the mount point in templates/repo/issue/view_content/pull_merge_box.tmpl
  document.body.append(el);
  const app = createApp(PullRequestMergeForm, {mergeFormProps: {
    baseLink: '/user2/repo1/pulls/2',
    textCancel: 'Cancel',
    textAutoMergeButtonWhenSucceed: '(When checks succeed)',
    textAutoMergeWhenSucceed: 'Merge when checks succeed',
    textAutoMergeCancelSchedule: 'Cancel auto merge',
    canMergeNow: true,
    allOverridableChecksOk: true,
    emptyCommit: false,
    defaultMergeStyle: 'merge',
    mergeStyles: [],
    hasPendingPullRequestMerge: false,
    hasPendingPullRequestMergeTip: '',
    ...props,
  }});
  app.mount(el);
  await nextTick();
  return {el, unmount: () => {
    app.unmount();
    el.remove();
  }};
}

test('#144 Decision 3 a scheduled merge keeps its notice and cancel button when no merge style is offered', async () => {
  const {el, unmount} = await mountMergeForm({
    hasPendingPullRequestMerge: true,
    hasPendingPullRequestMergeTip: 'user2 scheduled this pull request to auto merge when all checks succeed',
  });
  try {
    expect(el.querySelector('.merge-button')).toBeNull();
    expect(el.textContent).toContain('user2 scheduled this pull request to auto merge when all checks succeed');
    expect(Array.from(el.querySelectorAll('button'), (b) => b.textContent.trim())).toEqual(['Cancel auto merge']);
  } finally {
    unmount();
  }
});

test('#144 control: an allowed merge style still gets its merge button', async () => {
  const {el, unmount} = await mountMergeForm({
    mergeStyles: [{name: 'merge', allowed: true, textDoMerge: 'Create merge commit', hideAutoMerge: true}],
  });
  try {
    expect(el.querySelector('.merge-button')!.textContent).toContain('Create merge commit');
  } finally {
    unmount();
  }
});
