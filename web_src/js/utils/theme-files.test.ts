// Validates the theme stylesheets in web_src/css/themes, which both the Go backend
// (services/webtheme) and isDarkTheme() in web_src/js/utils.ts read at runtime.
import {readFileSync, readdirSync, existsSync} from 'node:fs';
import {join} from 'node:path';

const themesDir = join(import.meta.dirname, '../../css/themes');
const themeFiles = readdirSync(themesDir).filter((f) => f.endsWith('.css')).sort();

const read = (file: string) => readFileSync(join(themesDir, file), 'utf8');

// Splice in plain `@import "./x.css";` so inherited declarations are visible. Media-scoped
// imports (the `auto` themes) are left alone, since only one of them applies at a time.
function flattenImports(file: string): string {
  return read(file).replaceAll(/@import\s+"\.\/([-\w.]+\.css)"\s*;/g, (_, f) => flattenImports(f));
}

/** Mirrors parseThemeMetaInfoToMap in services/webtheme/webtheme.go: the last block wins. */
function parseMeta(css: string): Record<string, string> | null {
  const blocks = css.matchAll(/\bgitea-theme-meta-info\s*\{([^}]*)\}/g).toArray();
  if (!blocks.length) return null;
  const meta: Record<string, string> = {};
  for (const d of blocks.at(-1)![1].matchAll(/(--[\w-]+)\s*:\s*"([^"]*)"\s*;?/g)) meta[d[1]] = d[2];
  return meta;
}

/** Every `:root` custom property, in source order, with var() chains resolved. */
function rootVars(file: string): Record<string, string> {
  const raw: Record<string, string> = {};
  for (const block of flattenImports(file).matchAll(/:root\s*\{([^}]*)\}/g)) {
    for (const d of block[1].matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) raw[d[1]] = d[2].trim();
  }
  const deref = (v: string, depth = 0): string => {
    const m = /^var\((--[\w-]+)\)$/.exec(v);
    return !m || depth > 10 ? v : deref(raw[m[1]] ?? v, depth + 1);
  };
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(raw)) {
    const r = deref(v);
    if (/^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i.test(r)) out[k] = r;
  }
  return out;
}

function channels(hex: string): [number, number, number, number] {
  let h = hex.slice(1);
  if (h.length === 3) h = [...h].map((c) => c + c).join(''); // #fff appears in the stock themes
  return [
    parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16),
    h.length === 8 ? parseInt(h.slice(6, 8), 16) / 255 : 1,
  ];
}

/** WCAG 2.1 contrast ratio, compositing a translucent foreground over the background first. */
function contrast(fg: string, bg: string): number {
  const [br, bgc, bb] = channels(bg);
  let [r, g, b] = channels(fg);
  const a = channels(fg)[3];
  if (a < 1) { r = r * a + br * (1 - a); g = g * a + bgc * (1 - a); b = b * a + bb * (1 - a) }
  const lin = (c: number) => (c / 255 <= 0.03928 ? c / 255 / 12.92 : (((c / 255) + 0.055) / 1.055) ** 2.4);
  const lum = (x: number, y: number, z: number) => 0.2126 * lin(x) + 0.7152 * lin(y) + 0.0722 * lin(z);
  const l1 = lum(r, g, b), l2 = lum(br, bgc, bb);
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

// [foreground, background, minimum ratio]
const PAIRS: Array<[string, string, number]> = [
  ['--color-text', '--color-body', 4.5],
  ['--color-text', '--color-box-body', 4.5],
  ['--color-text', '--color-box-header', 4.5],
  ['--color-text', '--color-menu', 4.5],
  ['--color-text', '--color-card', 4.5],
  ['--color-text', '--color-button', 4.5],
  ['--color-text', '--color-code-bg', 4.5],
  ['--color-text', '--color-secondary-bg', 4.5],
  ['--color-text-light-1', '--color-body', 4.5],
  ['--color-text-light-2', '--color-body', 4.5],
  ['--color-text-light-3', '--color-body', 4.5],
  ['--color-nav-text', '--color-nav-bg', 4.5],
  ['--color-primary', '--color-body', 4.5],
  ['--color-primary-contrast', '--color-primary', 4.5],
  ['--color-primary-contrast', '--color-primary-hover', 4.5],
  ['--color-diff-added-fg', '--color-diff-added-row-bg', 4.5],
  ['--color-diff-removed-fg', '--color-diff-removed-row-bg', 4.5],
  ['--color-text', '--color-diff-added-row-bg', 4.5],
  ['--color-text', '--color-diff-removed-row-bg', 4.5],
  ['--color-text', '--color-diff-added-word-bg', 4.5],
  ['--color-text', '--color-diff-removed-word-bg', 4.5],
  ['--color-secondary-dark-1', '--color-input-background', 3],
  ['--color-accent', '--color-body', 3],
  ['--color-timeline', '--color-body', 3],
  ...['keyword', 'string', 'comment', 'number', 'type', 'name', 'control', 'operator', 'variable',
    'property', 'attribute', 'preproc', 'regexp', 'bool', 'invalid', 'text-alt', 'entity', 'escape',
    'decorator', 'namespace'].map((k): [string, string, number] => [`--color-syntax-${k}`, '--color-code-bg', 4.5]),
];

function ratios(file: string): Map<string, number> {
  const v = rootVars(file);
  const out = new Map<string, number>();
  for (const [fg, bg] of PAIRS) {
    if (v[fg] && v[bg]) out.set(`${fg} on ${bg}`, contrast(v[fg], v[bg]));
  }
  return out;
}

test('there are theme files to check', () => {
  expect(themeFiles.length).toBeGreaterThan(0);
});

describe.each(themeFiles)('%s', (file) => {
  test('declares a meta block the backend can read', () => {
    const meta = parseMeta(read(file));
    expect(meta, 'no gitea-theme-meta-info block').not.toBeNull();
    expect(meta!['--theme-display-name']).toBeTruthy();
    expect(['light', 'dark', 'auto']).toContain(meta!['--theme-color-scheme']);
    if (meta!['--theme-colorblind-type']) {
      expect(['red-green', 'blue-yellow']).toContain(meta!['--theme-colorblind-type']);
    }
  });

  test('every @import target exists', () => {
    for (const m of read(file).matchAll(/@import\s+"\.\/([-\w.]+\.css)"/g)) {
      expect(existsSync(join(themesDir, m[1])), `${file} imports missing ${m[1]}`).toBe(true);
    }
  });

  test('--is-dark-theme agrees with the declared color scheme', () => {
    const scheme = parseMeta(read(file))!['--theme-color-scheme'];
    if (scheme === 'auto') {
      // An auto theme pairs exactly one light and one dark import under media conditions.
      const imports = read(file).matchAll(/@import\s+"\.\/([-\w.]+\.css)"\s*\(prefers-color-scheme:\s*(\w+)\)/g).toArray();
      expect(imports.map((m) => m[2]).sort()).toEqual(['dark', 'light']);
      for (const m of imports) {
        expect(parseMeta(read(m[1]))!['--theme-color-scheme']).toBe(m[2]);
      }
      return;
    }
    // Later declarations win at equal specificity, so the last one is what the browser uses.
    const decls = flattenImports(file).matchAll(/--is-dark-theme\s*:\s*(true|false)/g).toArray();
    expect(decls.length, '--is-dark-theme is never set').toBeGreaterThan(0);
    expect(decls.at(-1)![1]).toBe(String(scheme === 'dark'));
  });
});

test('every theme is distinguishable in the picker', () => {
  // The picker sorts on (colorblind type, display name) only, so that pair must be unique.
  const seen = new Map<string, string>();
  for (const file of themeFiles) {
    const meta = parseMeta(read(file))!;
    const key = `${meta['--theme-colorblind-type'] ?? ''}|${meta['--theme-display-name']}`;
    expect(seen.has(key), `${file} is indistinguishable from ${seen.get(key)}`).toBe(false);
    seen.set(key, file);
  }
});

// Gitea's own themes sit below several of these thresholds on purpose (subtle borders, a muted
// --color-text-light-3). So the bar for a palette theme is "no worse than the stock theme it is
// built on", plus the absolute threshold wherever stock already clears it.
describe('contrast', () => {
  const baseline = {light: ratios('theme-gitea-light.css'), dark: ratios('theme-gitea-dark.css')};
  const palettes = themeFiles.filter((f) => !f.startsWith('theme-gitea-') &&
    ['light', 'dark'].includes(parseMeta(read(f))!['--theme-color-scheme']));

  test('there are palette themes to check', () => {
    expect(palettes.length).toBeGreaterThan(0);
  });

  test.each(palettes)('%s is no less readable than the gitea theme it extends', (file) => {
    const scheme = parseMeta(read(file))!['--theme-color-scheme'] as 'light' | 'dark';
    const base = baseline[scheme];
    for (const [pair, ratio] of ratios(file)) {
      const stock = base.get(pair);
      if (stock === undefined) continue; // the stock theme leaves this one unset (e.g. `inherit`)
      const [, , min] = PAIRS.find(([fg, bg]) => `${fg} on ${bg}` === pair)!;
      if (stock >= min) {
        expect(ratio, `${pair}: ${ratio.toFixed(2)} is below the ${min}:1 minimum that gitea-${scheme} meets (${stock.toFixed(2)})`).toBeGreaterThanOrEqual(min);
      } else {
        // Stock is already under the bar here, so only guard against a further drop.
        expect(ratio, `${pair}: ${ratio.toFixed(2)} is worse than gitea-${scheme} (${stock.toFixed(2)})`).toBeGreaterThan(stock - 0.25);
      }
    }
  });
});
