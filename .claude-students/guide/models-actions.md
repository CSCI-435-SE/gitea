---
source: docs/models-actions.md
source-hash: f800aaac10f3ed2a
verified-at: c0092050a4
---

<!-- Derived from docs/models-actions.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# CI runs in the database, in plain English

**In one sentence:** four nested levels of record describing a CI run, where the numbers on the
status values are part of a network protocol and must never be renumbered.
**Come here when:** you are querying or changing Actions data — runs, jobs, tasks, runners,
artifacts or schedules.

## What this is

Gitea Actions is Gitea's built-in CI: you commit a workflow file, and when something happens a
*runner* picks up the work and reports back.

These tables record all of it, plus one unusual extra: a small filesystem stored in the database,
used to hold task logs.

## Why it exists

**Why four levels rather than two?** Because "run this workflow" and "this job executed on that
machine" are genuinely different things, and re-runs are what force them apart.

A **run** is one triggering event: someone pushed, so the workflow ran.

An **attempt** exists because a run can be re-run. Attempt 1 failed, you press the button, attempt 2
starts. The run is still the same run.

A **job** is one block declared in the workflow file — "build", "test". A run has several, some
waiting on others.

A **task** is one actual execution of a job on a runner: this machine, these logs, this exit code.

Collapse any of those and something breaks. Without attempts there is nowhere to put a second try.
Without separating job from task you cannot say "the same job, run twice, on different machines".

**Why logs in the database?** Because a runner streams output while the job runs, and it must be
readable immediately from any Gitea instance, including ones sharing storage awkwardly. So there is
a small filesystem implemented over database rows.

## Words you'll meet

- **workflow** — the YAML file in the repository describing what should happen.
- **run** — one execution of a workflow, caused by one event.
- **attempt** — one try at a run. A re-run adds another.
- **job** — one block in the workflow file.
- **task** — one execution of a job on a runner.
- **runner** — a machine registered with Gitea that executes tasks.
- **artifact** — a file a job produced and uploaded.
- **optimistic lock** — a version number on a row, so two concurrent writers cannot silently
  overwrite each other.
- **wire protocol** — the agreed format Gitea and the runner use to talk. Both sides must agree
  exactly.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/actions/run.go` | One workflow run. |
| `models/actions/run_attempt.go` | One attempt at that run. |
| `models/actions/run_job.go` | One job within an attempt. |
| `models/actions/task.go` | One execution of a job on a runner. |
| `models/actions/status.go` | The status values shared by all of the above. |
| `models/actions/runner.go`, `runner_token.go` | Registered runners and their registration tokens. |
| `models/actions/artifact.go` | Uploaded artifacts. |
| `models/actions/schedule.go`, `schedule_spec.go` | Scheduled workflows. |
| `models/actions/variable.go` | Variables available to workflows. |
| `models/actions/scoped_workflow.go` | Workflows whose file comes from another repository. |
| `models/dbfs/dbfs.go` | The database-backed file store holding task logs. |

## The rules, and why

**A re-run clones jobs; it does not reuse them.** This is the most important thing on this page.

Pressing "re-run" inserts a **new attempt** and then **copies every job** into new rows belonging to
that attempt. Even jobs you are not re-running get copied — as pass-through rows that keep their old
status and point back at the original task, so the UI can still show their results.

The consequence: **"the jobs of this run" spans attempts.** A query that filters only by run and
ignores the attempt gets every job from every attempt, so counts double after the first re-run. The
run records which attempt is current, and a setting caps how many attempts are allowed.

This is the kind of bug that passes every test — because your test data has one attempt — and then
produces visibly wrong numbers the first time anyone re-runs anything.

**The run number in the URL is per-repository**, exactly like issue numbers, with the same
consequences (`models-issues.md`).

**Run status is protected by an optimistic lock.** A version field is on the row because runners
update status concurrently — several jobs finishing at once, on different machines. Write status
through the helpers that respect the version. A plain update silently loses whichever write arrives
second.

**The status numbers are part of the network protocol. Never renumber them.** The first five —
unknown, success, failure, cancelled, skipped — are numbered to match exactly what the runner
protocol sends. The remaining three (waiting, running, blocked) are Gitea's own.

Renumbering does not break the build. It breaks *communication with every existing runner*, silently
and in production, because the runner sends `2` meaning failure and Gitea now reads `2` as something
else.

**Fork pull requests are treated as untrusted.** Two fields exist for this: whether the run came
from a fork, and whether it needs approval before running. Code deciding whether a run may execute
must consider both — a workflow from a fork is code a stranger wrote, running on your infrastructure
with your secrets nearby.

**The workflow file is not always from the repository being built.** Two fields record where the
file content actually came from, for workflows sourced from elsewhere.

**Task logs are not files on disk.** They are in the database-backed store.

## How to actually do it

**Add a field to a run.** Add it to the struct, write the migration, and then check whether the
runner protocol needs to carry it too (`routers-api-actions.md`).

**Query a run's jobs.** Use the list types rather than looping per row — they batch their loading
the same way issue lists do. And filter by attempt unless you genuinely want the history.

**Add a status value.** Almost certainly a mistake. If you truly need an intermediate state, append
it after the last one. Never insert.

## Traps, and what they look like

**Counts are correct until someone re-runs something, then double.** A query ignoring which attempt
it belongs to.

**Your status update is lost with no error.** You wrote the row directly instead of going through
the helpers that respect the version, and another writer won the race.

**Runners start behaving bizarrely after a change that compiled fine.** Something renumbered the
status values.

**You use "job" and "task" interchangeably.** The UI blurs them; the database does not. Reading the
wrong table gives plausible-looking answers that are wrong specifically for re-runs.

**Attempt-related code crashes on an old run.** Runs created before the attempt model exists with no
attempt recorded and are treated as attempt 1. Code reading attempts has to tolerate that legacy
shape.

**You change the database-backed file store thinking it is general infrastructure.** It exists
almost entirely for Actions logs; changes affect log retention and cleanup.

**You look for a schedule's cron expression and the row does not have one.** Schedules are two rows:
the schedule, and a spec holding the parsed expression and next run time.

## Where to go next

- `docs/models-actions.md` (in this folder) — the reference page this was written from
- `services-actions.md` — what creates and advances these rows
- `routers-api-actions.md` — how runners report back
- `models-db.md` — optimistic locking and transactions
