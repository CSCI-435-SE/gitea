// controlled-opposition's StrykerJS config, reusing ab-wm's personal-config pattern. Run from the repository root:
//   pnpm exec stryker run docs/sprint3/mutation/controlled-opposition.stryker.config.mjs --mutate <file>[,<file>...]
import base from '../../../stryker.config.mjs';

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  ...base,
  jsonReporter: {fileName: 'reports/mutation/controlled-opposition/mutation.json'},
  htmlReporter: {fileName: 'reports/mutation/controlled-opposition/mutation.html'},
};
