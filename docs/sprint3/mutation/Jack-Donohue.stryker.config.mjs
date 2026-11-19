// Jack-Donohue's StrykerJS config for #158 and #159, following ab-wm's and vvchopra's pattern
// (docs/sprint3/mutation/ab-wm.stryker.config.mjs). Run from the repository root:
//   pnpm exec stryker run docs/sprint3/mutation/Jack-Donohue.stryker.config.mjs --mutate <file>[:<lines>][,...]
import base from '../../../stryker.config.mjs';

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  ...base,
  jsonReporter: {fileName: 'reports/mutation/Jack-Donohue/mutation.json'},
  htmlReporter: {fileName: 'reports/mutation/Jack-Donohue/mutation.html'},
};
