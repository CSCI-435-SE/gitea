// Shared StrykerJS settings for mutation testing the frontend. It names no files: personal configs
// import it and the files to mutate are passed with --mutate, see .claude-students/docs/mutation-testing.md.
import {readFileSync} from 'node:fs';

// Stryker reads ignorePatterns with .gitignore semantics, so the sandbox only gets what git would track
const gitignored = readFileSync(new URL('.gitignore', import.meta.url), 'utf8')
  .split('\n').filter((line) => line && !line.startsWith('#'));

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  plugins: ['@stryker-mutator/vitest-runner'], // the default "@stryker-mutator/*" search misses it in pnpm's store
  ignorePatterns: gitignored, // local folders can also break the copy, e.g. the symlinked directory in .venv
  testRunner: 'vitest',
  vitest: {configFile: 'stryker.vitest.config.ts'}, // related mode (on by default) runs only tests that reach the mutated files
  coverageAnalysis: 'perTest',
  reporters: ['clear-text', 'progress', 'html', 'json'], // json is the report D6 commits
};
