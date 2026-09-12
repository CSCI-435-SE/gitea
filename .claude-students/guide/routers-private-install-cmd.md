---
source: docs/routers-private-install-cmd.md
source-hash: f71d94534eb56351
verified-at: c0092050a4
---

<!-- Derived from docs/routers-private-install-cmd.md. Do not edit by hand: fix the reference doc
     and regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Git hooks, the installer and the command line, in plain English

**In one sentence:** when you `git push`, git runs a hook that runs the Gitea binary that makes an
HTTP call back to the Gitea server — and that chain explains a lot of otherwise strange code.
**Come here when:** you are working on git hooks, what happens on the server side of a push, the
first-run installer, or a command-line subcommand.

## What this is

Three things that share a theme: they are all the parts of Gitea that are not a web page.

`routers/private` is an **internal** API — HTTP endpoints only Gitea itself calls, never reachable
from outside. `routers/install` is the first-run setup, served instead of the normal site until
setup completes. `cmd` is every command-line subcommand: starting the server, handling SSH, running
hooks, the admin tools, the self-check.

## Why it exists

**Git hooks are separate programs, and that is the whole story here.**

When you push, git runs scripts in the repository. Those scripts are not part of the running Gitea
server — they are separate processes, started by git, with no access to the server's memory, caches
or database connections.

But they need to make decisions that require all of that. Is this branch protected? Is the pusher
allowed? Does this commit close an issue?

So the hook does the only thing it can: it runs the Gitea binary, which makes an **HTTP call to the
running server**, which has everything. That is what `routers/private` serves.

Once you know that, several things stop being strange. Why push logic lives in services rather than
in the hook — the hook has no database. Why there is an internal API with its own authentication —
it is an internal call that must never be exposed. Why latency in the push path matters so much —
the user is watching `git push` sit there.

## Words you'll meet

- **git hook** — a script git runs at a particular point. Gitea installs its own into every
  repository.
- **pre-receive** — the hook that runs *before* a push is accepted, and can reject it.
- **post-receive** — the hook that runs *after*, and cannot.
- **internal API** — endpoints Gitea calls itself, not exposed to users.
- **internal token** — the shared secret protecting those endpoints.
- **subcommand** — `gitea web`, `gitea serv`, `gitea doctor`.
- **doctor** — the built-in self-check and repair tool.

## What's in these files

| Where | What it is for |
| --- | --- |
| `routers/private/hook_pre_receive.go` | Deciding whether to accept a push. |
| `routers/private/hook_post_receive.go` | Reacting to one that was accepted. |
| `routers/private/hook_proc_receive.go` | Creating pull requests by pushing to a special ref. |
| `routers/private/serv.go`, `key.go` | Deciding what an SSH user may do. |
| `routers/private/internal.go` | The internal authentication. |
| `routers/private/manager.go` | Runtime management used by the admin CLI. |
| `cmd/web.go` | Starting the server — the normal entry point. |
| `cmd/serv.go` | Run by SSH for every git operation over SSH. |
| `cmd/hook.go` | The binary the installed hooks call. |
| `cmd/doctor.go` | The self-check commands. |

## The rules, and why

**The hook calls the binary, which calls the server.** That indirection is why push logic belongs in
services rather than in the hook — the hook is a thin messenger with no state.

**The internal API uses an internal token, not user authentication**, and must never be mounted
publicly. Handlers there take the private context type (`services-context.md`).

**Pre-receive is where you *prevent* something.** Branch protection, size limits — anything that
should stop a push goes there. Post-receive runs after the push has already been accepted and
**cannot reject it**. Putting a check in the wrong hook produces code that looks right and enforces
nothing.

**The push path is inside the user's `git push`**, so its latency is visible to them
(`services-repository.md`).

**`cmd` is the leftmost layer.** It may import anything, and nothing imports it
(`architecture.md`).

**A new subcommand is a file registered with the others**, usually with a test beside it.

## How to actually do it

**Reject a push under a new condition.** The check goes in the pre-receive handler — but put the
*rule itself* in a service, so the web UI can apply the same rule. Otherwise pushing is forbidden
and the equivalent web action is not, and the two slowly disagree.

**Add a command-line subcommand.** A new file following a neighbour's structure, registered with the
others, with a test.

**Add a doctor check.** The checks live in the doctor service; the subcommand only exposes them.

## Traps, and what they look like

**Your check does not prevent anything.** You put it in post-receive. That hook runs after the push
is accepted and has no power to refuse.

**You try to read a cache or reuse a connection in the hook.** The hook is a separate process. It
shares nothing with the server — everything goes over the internal API.

**Pushing breaks after an upgrade.** The hook scripts written into each repository on disk have to
match what the server expects. They are regenerated (`services-repository.md`), but a mismatch after
an upgrade is a real failure mode — which is why changing the internal API is a compatibility
concern *within a single installation*, not just across versions.

**Your route is not reachable before setup.** The installer routes are mounted *instead of* the
normal ones until setup finishes.

**Pushing gets slow for everyone.** Something expensive was added to the push path, which runs while
the user waits.

## Where to go next

- `docs/routers-private-install-cmd.md` (in this folder) — the reference page this was written from
- `services-repository.md` — what happens after a push is accepted
- `services-context.md` — the private context type
- `architecture.md` — why `cmd` sits where it does
