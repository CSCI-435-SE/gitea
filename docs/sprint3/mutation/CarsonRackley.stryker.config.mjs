// CarsonRackley's StrykerJS config for #118, copied from ab-wm's (docs/sprint3/mutation/ab-wm.stryker.config.mjs). Run from the repository root:
//   pnpm exec stryker run docs/sprint3/mutation/CarsonRackley.stryker.config.mjs --mutate <file>[,<file>...]
import base from '../../../stryker.config.mjs';

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  ...base,
  jsonReporter: {fileName: 'reports/mutation/CarsonRackley/mutation.json'},
  htmlReporter: {fileName: 'reports/mutation/CarsonRackley/mutation.html'},
};
