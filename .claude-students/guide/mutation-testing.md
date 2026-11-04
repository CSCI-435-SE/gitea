---
source: docs/mutation-testing.md
source-hash: 1c5de4b1216ba70c
verified-at: b70fe3924a
---

<!-- Derived from docs/mutation-testing.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Mutation testing, in plain English

**In one sentence:** a mutation tool breaks your code on purpose, one small change at a time, and
checks whether your tests notice; this repo has one tool set up for the frontend and one for Go.
**Come here when:** you want to run a mutation tool on files you changed, you are writing your own
config for it, or a run stops before it has mutated anything.

## What this is

Two tools. **StrykerJS** handles the TypeScript under `web_src/js`, and runs the same Vitest tests
you already run. **go-mutesting** handles Go packages, and runs their `go test` tests.

Stryker is installed with the other frontend packages (it is listed in `package.json`), and its
shared settings are in `stryker.config.mjs` at the repository root. go-mutesting is a separate
program you install yourself; nothing about it is recorded in `go.mod`.

There was a third option, Gremlins. It is not used, because its own README warns that very big Go
modules can take hours, and all of Gitea is one module.

## Why it exists

A passing test suite tells you the tests and the code agree. It does not tell you whether the tests
would notice if the code were wrong. Mutation testing checks exactly that: flip a `>` to `>=`,
delete a line, swap `true` for `false`, and rerun the tests. If nothing fails, a real bug of that
shape would also slip through.

Gitea is large and was not built with these tools in mind, so running them takes a few settings
that are not obvious. Those settings live in the shared config so each person does not rediscover
them.

## Words you'll meet

- **mutant** — a copy of the code with one small deliberate change.
- **killed** — at least one test failed on the mutant: the tests noticed. In go-mutesting's output
  this is confusingly printed as `PASS` (the mutation test passed).
- **survived** — every test still passed: the tests did not notice the change.
- **timeout** — the mutant made the tests run too long; it counts as killed.
- **mutation score** — the share of mutants that were killed.
- **sandbox** — a throwaway copy of the project Stryker works in, so your real files are never
  changed.
- **symlink** — a file-system shortcut pointing at another file or folder.
- **pnpm store** — where pnpm actually keeps installed packages; the folders in `node_modules` are
  mostly links into it.
- **worker** — one of the background processes Vitest runs test files in.
- **test isolation** — giving each test file a fresh copy of every module, so nothing one file sets
  up leaks into the next.
- **order-dependent test** — a test that passes or fails depending on which other tests ran before
  it in the same worker.
- **related mode** — Vitest running only the test files that import the files you name.

## What's in these files

| Where | What it is for |
| --- | --- |
| `stryker.config.mjs` | Everyone's shared Stryker settings. It says *how* to run, never *which files* to mutate. |
| `stryker.vitest.config.ts` | The normal Vitest settings with test isolation switched on, for Stryker's runs only. |
| `vitest.config.ts` | The Vitest settings everyone and CI normally use. Isolation is off here. |
| `.gitignore` | Keeps Stryker's sandbox, its reports and go-mutesting's `report.json` out of commits. |

## The rules, and why

**The shared config never names files to mutate.** You say which files on the command line each
time. That way one config works for any piece of work, and a personal config only needs to change
small things such as where reports go.

**Stryker copies only what git tracks.** Before running, Stryker copies the project into a sandbox
folder. Left alone it would copy everything, including your local leftovers: a built binary, a
database, a Python environment. One of those, the `.venv` folder, contains a symlink to a folder
that makes the copy crash. So the shared config hands Stryker every line of `.gitignore` to skip.
`node_modules` is never copied; the sandbox links to it.

**The Vitest plugin is named directly.** Stryker normally finds its plugins by looking in the folder
beside its own package. pnpm keeps that folder in its store under `node_modules/.pnpm`, and the
Vitest runner is not in it. Without the explicit name, Stryker cannot find the plugin and warns
that the `vitest` option is unknown.

**Only the relevant tests run.** Related mode picks the test files that import the mutated files,
and per-test coverage means each mutant only reruns the tests that actually reach it. This is what
keeps a run down to about a minute instead of the whole suite per mutant.

**Stryker runs with test isolation on.** Stryker puts all the related test files into one worker.
The normal Vitest settings turn isolation off, so anything one file leaves behind — a key listener a
module registers when it is imported, for example — is still there when the next file runs. A test
that only passes when it has a worker to itself then fails, and Stryker refuses to start. The
Stryker-only Vitest settings switch isolation back on. Your normal test runs and CI are unaffected.

**go-mutesting edits your real files.** Unlike Stryker, it has no sandbox. For each mutant it
replaces the real source file, runs that package's tests, and puts the original back. It only runs
the package's own `_test.go` tests, never the integration tests under `tests/`.

**go-mutesting always writes `report.json` where you run it.** Its config file has a setting that
claims to switch this on or off, but the program ignores it. That file would otherwise end up in a
commit, so `.gitignore` lists it.

## How to actually do it

**Run Stryker on some files.** From the repository root:

```sh
pnpm exec stryker run <config> --mutate web_src/js/a.ts,web_src/js/b.ts
```

`<config>` is `stryker.config.mjs` or your own config. A summary table prints at the end, and
`mutation.json` and `mutation.html` are written to `reports/mutation/` unless your config moves
them. A module of about 250 lines with about 40 related tests takes around a minute.

**Write your own Stryker config.** Make a `.mjs` file that imports the shared one and changes only
what you need, such as where the JSON report goes:

```js
import base from '<relative path>/stryker.config.mjs';
export default {...base, jsonReporter: {fileName: 'reports/mutation/<name>/mutation.json'}};
```

Paths inside it are worked out from the folder you run Stryker in, so always run from the
repository root.

**Run go-mutesting.**

1. Install it once:

   ```sh
   go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@master
   ```

   `@master` rather than a version number, because the tool's module path is missing the `/v2`
   that Go needs to install its v2 releases by number. Building it needs the Go version its own
   `go.mod` asks for.
2. Run it on the files you care about, using the Go version from Gitea's `go.mod` so the tests
   behave as they do in CI, and save what it prints:

   ```sh
   GOTOOLCHAIN=go<version> go-mutesting <file.go> > <out>.txt
   ```

   The last line of the output gives the score.
3. Run `git status`. No source file should show as changed. Move `report.json` to wherever you are
   keeping it.

## Traps, and what they look like

**Stryker stops with "There were failed tests in the initial test run".** Before mutating anything,
Stryker runs the related tests on your unchanged code, and one failed. Rerun those tests the way
Stryker groups them: `pnpm exec vitest related <file> --run --no-file-parallelism`. If the test
fails there but passes in a normal run, it is an order-dependent test.

**Stryker crashes with `EISDIR ... copyfile`.** It tried to copy a symlinked folder into its
sandbox. Make sure that folder is in `.gitignore`, or add it to `ignorePatterns`.

**Two runs on the same code give slightly different scores.** Whether a mutant hits the timeout can
change from run to run, and a timeout counts as killed. A difference of a mutant or two is normal.

**A Go file has strange changes you did not make.** A go-mutesting run was interrupted while a
mutant was in place. `git diff` shows it and `git restore <file>` puts it back.

**go-mutesting takes forever.** Every mutant recompiles the package and reruns all of its tests.
Packages with database fixtures, such as the ones under `models/`, are slow. Time one
`go test ./<pkg>/` first and multiply by the number of mutants before starting a big run.

**Plenty of Go mutants survive in code you know is tested.** If the test that covers it lives in
`tests/integration`, go-mutesting never runs it.

**Your personal `.mjs` config is never linted.** `make lint-js` (see `Makefile`, `ESLINT_FILES`)
only lints root `*.ts` files and a few folders, nothing under `docs/`.

## Where to go next

- `docs/mutation-testing.md` (in this folder) — the reference page this was written from
- `testing.md` — the kinds of test the mutation tools run, and how each one is run normally
- `build-and-tooling.md` — the lint and build targets
