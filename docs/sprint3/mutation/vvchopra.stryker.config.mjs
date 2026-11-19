// vvchopra's StrykerJS config for #157, following ab-wm's pattern (docs/sprint3/mutation/ab-wm.stryker.config.mjs). Run from the repository root:
//   pnpm exec stryker run docs/sprint3/mutation/vvchopra.stryker.config.mjs --mutate <file>[,<file>...]
import base from '../../../stryker.config.mjs';

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  ...base,
  jsonReporter: {fileName: 'reports/mutation/vvchopra/mutation.json'},
  htmlReporter: {fileName: 'reports/mutation/vvchopra/mutation.html'},
};
