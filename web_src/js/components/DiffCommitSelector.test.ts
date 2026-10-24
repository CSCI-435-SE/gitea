import {createApp} from 'vue';
import DiffCommitSelector from './DiffCommitSelector.vue';
import {getShortcuts} from '../modules/shortcut.ts';

test('#59 AC6 the commit picker lists c with its label and section', () => {
  const el = document.createElement('div'); // the mount point in templates/repo/diff/box.tmpl
  el.id = 'diff-commit-select';
  el.setAttribute('data-filter_changes_by_commit', 'Filter by commit');
  el.setAttribute('data-shortcut-section', 'Pull Requests');
  document.body.append(el);
  const app = createApp(DiffCommitSelector);
  app.mount(el);
  try {
    expect(getShortcuts()).toContainEqual({keys: 'c', description: 'Filter by commit', section: 'Pull Requests'});
  } finally {
    app.unmount(); // its shortcut leaves with its element, so other tests can use c
    el.remove();
  }
});
