// ab-wm's StrykerJS config: the shared settings, tied to no issue. Run from the repository root:
//   pnpm exec stryker run docs/sprint3/mutation/ab-wm.stryker.config.mjs --mutate <file>[,<file>...]
import base from '../../../stryker.config.mjs';

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  ...base,
  jsonReporter: {fileName: 'reports/mutation/ab-wm/mutation.json'},
  htmlReporter: {fileName: 'reports/mutation/ab-wm/mutation.html'},
};
