import {mergeConfig} from 'vitest/config';
import baseConfig from './vitest.config.ts';

// Stryker runs all related test files in one worker; with isolate off, module state leaks between them
export default mergeConfig(baseConfig, {test: {isolate: true}});
