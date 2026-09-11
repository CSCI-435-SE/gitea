---
source: docs/services-actions.md
source-hash: 8d5d4dd96b9c96d3
verified-at: c0092050a4
---

<!-- Derived from docs/services-actions.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# What makes CI actually run, in plain English

**In one sentence:** nothing in Gitea ever says "run the workflows" — Actions *listens* for events
and decides for itself, and jobs become runnable through a queue rather than by anyone starting them.
**Come here when:** you are changing when a workflow triggers, how jobs become runnable, approvals,
re-runs, or why a workflow did not run.

## What this is

Two packages. `services/actions` decides when workflows should run and manages everything after
that: readiness, approvals, re-runs, concurrency limits, artifacts, cleanup, and publishing results
as commit statuses.

`modules/actions` is the part that reads the workflow file and turns YAML into jobs. It is
deliberately dependency-light, which is why it sits in `modules/` (`architecture.md`).

## Why it exists

The design has two surprising decisions, and both are worth understanding before you change
anything.

**Actions subscribes; it does not get called.** You might expect the push code to call "start the CI
now". It does not. Actions registers itself as a *listener* for events, alongside webhooks and email
— all three hear about a push the same way.

That matters because CI is triggered by many things: pushes, pull requests opening, issues being
commented on, releases, schedules. If every one of those places had to remember to call Actions,
some would forget. Instead, each place announces what happened, and Actions decides whether any
workflow cares.

**Jobs do not start themselves.** A workflow's jobs often depend on each other — "test" waits for
"build". So a job becoming runnable is not an event, it is a *conclusion* you reach by re-examining
the whole run: are its dependencies done, is it within the concurrency limits?

So when something changes that might unblock a job, the code does not try to work out which job that
was. It puts the run on a queue, and a worker re-evaluates everything. Enqueueing is cheap; being
clever is how you get jobs that never start.

## Words you'll meet

- **trigger** — the event causing a workflow to run.
- **notifier** — a subscriber to Gitea's internal events.
- **emit** — declaring a job ready to be picked up.
- **`needs`** — a workflow keyword making one job wait for another.
- **concurrency group** — a limit on how many runs of a thing happen at once.
- **approval** — the manual "yes, run this" for untrusted code.
- **reusable workflow** — a workflow called by another workflow.
- **expression context** — the `${{ ... }}` values available inside a workflow file.
- **label** — a tag on a runner saying what it can do; `runs-on` matches against it.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/actions/notifier.go`, `notifier_helper.go` | The trigger path: what happens when an event arrives. |
| `services/actions/init.go` | Registers the notifier and starts the queues. |
| `services/actions/job_emitter.go` | The queue that re-evaluates which jobs are now runnable. |
| `services/actions/concurrency.go` | Concurrency limits. |
| `services/actions/approve.go` | Approving a run from an untrusted source. |
| `services/actions/rerun.go` | Re-running a run or a single job. |
| `services/actions/schedule_tasks.go` | Turning scheduled workflows into runs. |
| `services/actions/commit_status.go` | Publishing results as the ticks and crosses on commits. |
| `services/actions/artifacts.go`, `cleanup.go` | Artifacts and how long they are kept. |
| `services/actions/workflow.go`, `reusable_workflow.go` | Working out which workflow file applies. |
| `modules/actions/workflows.go`, `jobparser/` | Parsing the workflow YAML into jobs. |

## The rules, and why

**Actions is a notifier, not something you call.** Registration happens at startup. To make a new
event trigger workflows, you add a method to the Actions notifier — you do not add a call to Actions
from wherever the event happens.

**The trigger path is built up in steps and ends in a notify call.** An input object is assembled —
who did it, which ref, what payload — and then handed off. Add a trigger by adding the notifier
method and building that input, not by reaching into the middle of the handling code.

**Anything that might unblock a job enqueues.** Do not flip a job's status yourself. Put the run on
the queue and let the handler re-evaluate. The queue ignores duplicates, so enqueueing more often
than necessary is free — and far safer than enqueueing too rarely.

**A run from a forked pull request may need approval first.** This is a security boundary, not a
formality: a workflow from a fork is code written by a stranger, and it is about to run on your
infrastructure. Never add an execution path that bypasses the approval check.

**Workflow parsing lives in `modules/actions`**, and the parsed result is stored on the job row.

**GitHub compatibility is the goal.** The expression context is built to match GitHub Actions, so
workflows written for GitHub mostly work. A new context field should use GitHub's name for it, not a
Gitea-specific one.

## How to actually do it

**Make a new event trigger workflows.** Add the method to the Actions notifier, build the input with
the right event type and payload, and check that the `on:` keyword a user would write is recognised
by the workflow parser. Both halves are needed — the event firing, and the parser knowing the
keyword.

**Change when a job becomes runnable.** The readiness logic is in the queue handler in
`job_emitter.go`. Then make sure whatever change could affect readiness enqueues the run.

**Work out why a workflow did not run.** Go in this order, because each step depends on the one
before:

1. Was the event actually fired?
2. Was it filtered out — `[skip ci]` in the commit message and similar?
3. Did the workflow file parse?
4. Is the run waiting for approval?
5. Is it blocked by a concurrency limit?
6. Is there a runner registered whose labels match the workflow's `runs-on`?

Most "my workflow did not run" reports are step 3 or step 6.

## Traps, and what they look like

**A job sits at "blocked" forever.** Two different things produce that state — an unsatisfied
dependency, and a concurrency limit. Check both; people usually check only the first.

**Everything on the instance gets slower after your change.** The notifier's methods run on *every*
matching event across every repository. Anything expensive there is paid site-wide. Keep it cheap
and hand real work to the queue.

**Your new trigger works when you test it and not in general.** You added the notifier method but
the parser does not recognise the `on:` keyword, or the reverse.

**A job never becomes runnable.** Something changed a status directly instead of enqueueing, so
nothing re-evaluated readiness.

**You change token permissions without reading the design note.** There is a document in this
package describing the token permission model. Read it before touching the permission parser — it is
the kind of code where a small change quietly widens what a workflow can do.

## Where to go next

- `docs/services-actions.md` (in this folder) — the reference page this was written from
- `models-actions.md` — the rows this package creates and advances
- `routers-api-actions.md` — how runners pick up and report work
- `services-repository.md` — the push that starts most runs
