# Student Guide — Running Gitea Locally

Welcome! This is a course fork of [Gitea](https://about.gitea.com/), a self-hosted Git service
written in **Go** (backend) and **Vue/TypeScript** (frontend). You'll be improving the kind of tool
you use every day — issues, pull requests, code review, CI/Actions.

This guide gets you from a fresh clone to a running Gitea on your machine, backed by **SQLite** (no
separate database server needed). It should take ~15–30 minutes, most of which is the first backend
compile.

---

## 1. Prerequisites

Install these before you start. Versions below are the minimums this fork is pinned to.

| Tool | Version | Notes |
| --- | --- | --- |
| **Go** | **1.26.4+** | Backend. `go.mod` requires `go 1.26.4`. |
| **Node.js** | **22.18.0+** | Frontend build only (not needed at runtime). |
| **pnpm** | **11.0.0+** | Frontend package manager. Easiest via `corepack enable`. |
| **Git** | 2.x | Required both to build **and at runtime** (Gitea shells out to git). |

> 💡 **New to Go?** Gitea's backend is written in Go. [A Tour of Go](https://go.dev/tour/) is the official interactive tutorial (~2 hours) and covers everything you need for typical backend tasks. Pay attention to goroutines and interfaces — both appear frequently in this codebase.

> 📖 **Developer docs:** [`docs/development.md`](docs/development.md) in this repo covers the development workflow, how to run tests, database migration conventions, and the overall package structure. Read it before picking a task.

**You do NOT need a C compiler (gcc).** This version of Gitea uses a pure-Go SQLite driver
(`modernc.org/sqlite`) and builds with `CGO_ENABLED=0`, so there's no MSYS2/MinGW/Xcode-tools step.
This is the main reason setup is painless on Windows.

**Optional:** `make` (convenience only — every command below also has a no-`make` form), and Docker
(if you'd rather run the official container instead of building).

### Installing the toolchain
- **Windows:** `winget install GoLang.Go` and `winget install OpenJS.NodeJS`, then `corepack enable`.
- **macOS:** `brew install go node` then `corepack enable`.
- **Linux:** use your package manager (or the official Go tarball) for Go + Node, then `corepack enable`.

Verify:
```bash
go version      # go1.26.x
node --version  # v22.18+ (or newer)
pnpm --version  # 11.x
git --version
```

---

## 2. Get the code

```bash
git clone https://github.com/musta55/gitea.git
cd gitea
```

---

## 3. Build

Gitea is built in two halves: the frontend assets (with pnpm/vite) and the backend binary (with Go).

### Option A — with `make` (recommended if you have it)
```bash
make build          # builds frontend + backend into ./gitea (or gitea.exe on Windows)
```

### Option B — without `make` (works everywhere)
```bash
# 1) frontend assets -> public/assets/
pnpm install
pnpm exec vite build

# 2) backend binary  (pure-Go SQLite, no gcc)
#    Linux/macOS:
CGO_ENABLED=0 go build -o gitea .
#    Windows (PowerShell):
$env:CGO_ENABLED=0; go build -buildmode=exe -o gitea.exe .
```

> The first backend compile takes a few minutes and produces a ~100+ MB binary. Later builds are fast.

---

## 4. Run (SQLite, zero config)

Just start the web server from the repo root and let the built-in installer set everything up:

```bash
# Linux/macOS:
./gitea web
# Windows:
./gitea.exe web
```

Then open **http://localhost:3000** and you'll see the **Install** page:

1. **Database Type:** choose **SQLite3** (the default path is fine).
2. Leave the rest at defaults for local dev.
3. Expand **Administrator Account Settings** and create your admin username / password / email
   (do this now so you don't have to register separately).
4. Click **Install Gitea**.

That's it — you now have a running Gitea with a SQLite database at `data/gitea.db`. Log in with the
admin account you just created.

> **Port already in use?** If something else owns `:3000`, start Gitea on another port:
> `./gitea web --port 3030` and open http://localhost:3030 instead.

### On macOS
The same `./gitea web` command works — a few Mac-specific notes:

- **Firewall prompt:** the first launch may ask *"Do you want the application `gitea` to accept
  incoming network connections?"* — click **Allow** (it's the binary you just built).
- **`permission denied`:** make the binary executable first: `chmod +x gitea`.
- **Gatekeeper "cannot be opened" warning:** only affects binaries downloaded from a browser, not
  ones you built yourself. If you ever run a downloaded release binary instead, clear the
  quarantine flag: `xattr -d com.apple.quarantine ./gitea`.
- **Apple Silicon (M1–M4):** fully supported — Go produces a native arm64 binary, nothing extra to do.

### Live-reload while developing (optional)
If you have `make`:
```bash
make watch          # rebuilds frontend + backend on file changes (uses air)
```

---

## 5. Try it out

Create a repository (with "Initialize repository"), open an issue, push a branch, and open a pull
request so you can see the code-review UI. These are the areas most course tasks touch:

- **Code review UI** — the diff viewer and PR review flow.
- **Issues / Kanban** — issue tracking and project boards.
- **CI / Actions** — the built-in Actions runner and workflow UI.

---

## 6. Running the tests

Gitea's test suite is validated on **Linux CI**. Run it on Linux, macOS, WSL2, or the provided
`.devcontainer` — **not native Windows**, where a handful of unit tests fail for platform reasons
(symlink semantics, temp-file cleanup) rather than real bugs.

```bash
# unit tests
CGO_ENABLED=0 go test ./...
# or, with make:
make test-backend         # unit tests
make test-integration     # integration tests (defaults to SQLite via GITEA_TEST_DATABASE)
```

Frontend tests:
```bash
pnpm exec vitest run
```

---

## 7. Common gotchas

- **`git` not found at runtime:** Gitea runs git commands for every repo operation. Make sure `git`
  is on your `PATH`, not just in your IDE.
- **`go build` is slow the first time:** normal — it compiles ~3000 packages once, then caches.
- **Don't commit build artifacts:** `gitea`/`gitea.exe`, `data/`, `custom/`, and `public/assets/*`
  are gitignored. Keep them out of your commits.
- **Windows line endings:** the repo expects LF. Your editor/`.gitattributes` should keep it that way.

---

---

## Contributing workflow

All team members have write access to this repository, so the team uses a **branch-based** workflow — not forks. Here is the background and the commands.

**Why not forks?** Forking is the standard model for contributing to open-source projects where you _don't_ have write access: you fork to your own GitHub account, clone your fork, and open a PR from your fork back to the original. You will encounter this when contributing to the upstream project. But for your course team — where everyone has write access to the shared repo — it just adds confusion: two clones on your machine, two remotes to keep in sync, merge conflicts that are harder to reason about.

**Branch-based workflow** is what most professional teams use internally. You clone the shared repo once, create a short-lived branch for each issue, push the branch back to the same repo, and open a PR from that branch into `main`. One clone, one remote, full PR workflow.

### For each issue you work on

```bash
# One-time setup: clone the team repo (skip if already done)
git clone https://github.com/CSCI-435-SE/gitea.git
cd gitea

# Before starting each issue: make sure you are on a fresh main
git checkout main
git pull origin main

# Create a branch named for the issue
git checkout -b feat/issue-17-dark-mode      # new feature
git checkout -b fix/issue-42-toast-dismiss   # bug fix

# ... make your changes, run tests ...

# Stage and commit
git add <the files you changed>
git commit -m "feat: add dark mode toggle (#17)"

# Push the branch to the team repo
git push origin feat/issue-17-dark-mode
```

After pushing, GitHub shows a **"Compare & pull request"** banner on the repository page. Click it to open a PR from your branch into `main`. Fill in the description (what changed and why), reference the issue (`Closes #17`), and request a review from a teammate.

**Branch naming:**

| Prefix | Use for |
|---|---|
| `feat/issue-<N>-short-description` | new features |
| `fix/issue-<N>-short-description` | bug fixes |
| `chore/short-description` | docs, config, dependency updates |

> ⚠️ **`main` is protected — direct pushes are blocked.** All changes go through a reviewed PR. If you accidentally commit to `main` locally, move your changes to a branch before pushing:
>
> ```bash
> git checkout -b fix/issue-42-my-fix   # create branch from your current state
> git checkout main
> git reset --hard origin/main          # revert local main to match remote
> ```

**After your PR is merged**, delete the branch to keep the repo tidy:

```bash
git checkout main
git pull origin main
git branch -d feat/issue-17-dark-mode
```

## 8. Project documentation & policies (required reading)

📚 **Official documentation:** <https://docs.gitea.com/> — usage, administration, and development
guides.

Gitea has its own established contribution processes. They are **not restated here** — you are
responsible for finding, reading, and following them from the sources below:

| You must take care of | Where to find it |
| --- | --- |
| How to use the tool | <https://docs.gitea.com/> |
| Code review process | [CONTRIBUTING.md — Code review](CONTRIBUTING.md#code-review) |
| Bug / issue resolution process | [CONTRIBUTING.md — How to report issues](CONTRIBUTING.md#how-to-report-issues) |
| Pull request conventions & PR policies | [CONTRIBUTING.md — Pull request format](CONTRIBUTING.md#pull-request-format) + [AGENTS.md](AGENTS.md) (commit/PR conventions) |
| AI policies | [CONTRIBUTING.md — AI Contribution Policy](CONTRIBUTING.md#ai-contribution-policy) + [AGENTS.md](AGENTS.md) |
