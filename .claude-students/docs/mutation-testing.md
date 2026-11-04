---
scope: stryker.config.mjs, stryker.vitest.config.ts
verified-at: b70fe3924a
---

# mutation testing — StrykerJS for the frontend, go-mutesting for Go

**Read when:** running a mutation tool on changed files, writing a personal config, or a run that
stops before mutating anything.
**Not here:** how each test tier runs -> `testing.md`; the course deliverable and its file names ->
`COURSE-WORKFLOW.md`.

## Responsibilities

| Side | Tool | Where it comes from |
| --- | --- | --- |
| `web_src/js` (Vitest) | StrykerJS with its Vitest runner | `package.json` devDependencies, `stryker.config.mjs` |
| Go packages | go-mutesting | `go install`, nothing in `go.mod` |

Gremlins is not used: its README warns that very big Go modules can take hours, and Gitea is one
module (`go.mod`).

## Key files

| Path | What it holds |
| --- | --- |
| `stryker.config.mjs` | Shared Stryker options: the Vitest plugin, ignore patterns read from `.gitignore`, runner, reporters. Names no files |
| `stryker.vitest.config.ts` | `vitest.config.ts` merged with `isolate: true`, used only by Stryker |
| `vitest.config.ts` | The normal Vitest config, with `isolate: false` |
| `.gitignore` | `/.stryker-tmp` (Stryker's sandbox), `/reports/mutation` (its reports), `/report.json` (go-mutesting's) |

## Conventions & invariants

- `stryker.config.mjs` sets no `mutate`. Personal configs import it, spread it, and change only
  report paths or similar; the files to mutate are passed on the command line with `--mutate`.
- Stryker copies the project into `.stryker-tmp/` before running. `stryker.config.mjs` feeds every
  `.gitignore` line to `ignorePatterns`, which Stryker matches with `.gitignore` semantics (`!`
  included), so the copy holds only what git tracks. `node_modules` is symlinked, not copied.
- `plugins` names `@stryker-mutator/vitest-runner` directly. Stryker's default `@stryker-mutator/*`
  search looks beside its own package, which pnpm keeps under `node_modules/.pnpm` without the
  runner, so the `vitest` option is then reported as unknown.
- The Vitest runner's `related` mode (on by default) runs only test files that import the mutated
  files, and `coverageAnalysis: 'perTest'` reruns only the tests covering each mutant.
- Stryker runs all related test files in one worker. With `vitest.config.ts`'s `isolate: false`,
  module state (such as a listener registered at import) leaks from one file into the next, so
  `stryker.vitest.config.ts` turns isolation on. Normal runs and CI keep `vitest.config.ts`.
- go-mutesting's built-in test step replaces the real source file with each mutant, runs that
  package's tests, and restores it. Only the package's own `_test.go` tests run, not `tests/`.
- go-mutesting always writes `report.json` to the directory it runs in; the `json_output` setting
  in its YAML config is not read. In its output, `PASS` means the mutant was killed.

## Recipes

**Run Stryker on some files.** From the repository root:
`pnpm exec stryker run <config> --mutate web_src/js/a.ts,web_src/js/b.ts`. The summary table is
printed at the end; `mutation.json` and `mutation.html` land in `reports/mutation/` unless the
config moves them. A run on one ~250-line module with ~40 related tests takes about a minute.

**Write a personal Stryker config.** A `.mjs` file that imports the shared one:

```js
import base from '<relative path>/stryker.config.mjs';
export default {...base, jsonReporter: {fileName: 'reports/mutation/<name>/mutation.json'}};
```

Paths inside it resolve from the directory Stryker runs in, so run it from the repository root.

**Run go-mutesting.**

1. Install: `go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@master`. Its module
   path has no `/v2`, so its v2 tags cannot be installed by version, and building it needs the Go
   version its own `go.mod` asks for.
2. Run on files, with the Go version from `go.mod` so the tests match CI:
   `GOTOOLCHAIN=go<version> go-mutesting <file.go>... > <out>.txt`. The last line holds the score.
3. Check `git status`: no source file may be left changed. Move `report.json` where it belongs.

## Gotchas

- Stryker's "There were failed tests in the initial test run" means a related test fails on
  unchanged code. Rerun it with `pnpm exec vitest related <file> --run --no-file-parallelism` to see
  whether it only fails when files share a worker (an order-dependent test) or fails anyway.
- A Stryker `EISDIR ... copyfile` error comes from copying a symlinked directory; any such folder
  needs to be in `.gitignore` or `ignorePatterns`.
- Mutants that hit Stryker's timeout count as killed, and whether one times out can change between
  runs, so two runs of the same code can differ by a mutant or two.
- An interrupted go-mutesting run can leave a mutated source file behind. `git diff` shows it;
  `git restore <file>` undoes it.
- Each go-mutesting mutant recompiles and reruns the whole package's tests, so packages with
  database fixtures (`models/...`) are slow: time one `go test ./<pkg>/` first and multiply.
- Behaviour covered only by `tests/integration` shows up as surviving go-mutesting mutants.
- `make lint-js` lints the root `*.ts` files, not `.mjs` ones, and nothing under `docs/`
  (`Makefile`, `ESLINT_FILES`).

## Related

- `testing.md` — the test tiers the mutation tools run
- `build-and-tooling.md` — the lint and build targets
