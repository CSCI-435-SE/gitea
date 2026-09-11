---
scope: models/actions, models/dbfs
verified-at: c0092050a4
---

# models/actions — CI runs, jobs, tasks, runners

**Read when:** querying or changing Actions rows: runs, jobs, tasks, runners, artifacts, schedules
or variables.
**Not here:** triggering and dispatch -> `services-actions.md`; the runner HTTP protocol -> `routers-api-actions.md`.

## Responsibilities

The persistence for Gitea Actions, plus `models/dbfs` — a database-backed virtual filesystem used
to store task logs.

## Key files

| Path | What it holds |
| --- | --- |
| `models/actions/run.go` | `ActionRun` — one workflow run |
| `models/actions/run_job.go` | `ActionRunJob` — one job inside a run |
| `models/actions/task.go` | `ActionTask` — one attempt at executing a job on a runner |
| `models/actions/run_attempt.go` | `ActionRunAttempt` — one attempt of a run |
| `models/actions/status.go` | the `Status` enum shared by runs, jobs and tasks |
| `models/actions/runner.go`, `runner_token.go` | registered runners and their registration tokens |
| `models/actions/artifact.go` | uploaded artifacts |
| `models/actions/schedule.go`, `schedule_spec.go` | cron-scheduled workflows |
| `models/actions/variable.go` | Actions variables |
| `models/actions/scoped_workflow.go` | workflows sourced from another repo |
| `models/dbfs/dbfs.go` | the DB-backed file store for logs |

## Conventions & invariants

- **Four levels:** `ActionRun` → `ActionRunAttempt` → `ActionRunJob` → `ActionTask`. A job is the
  unit declared in the workflow file; a task is one execution of that job on a runner.
- **A re-run clones jobs, it does not reuse them.** `RerunWorkflowRunJobs` inserts a new
  `ActionRunAttempt` and clones *every* job into new `ActionRunJob` rows carrying that attempt's
  `RunAttemptID`; jobs not being re-run are cloned as pass-through with their status preserved and
  `SourceTaskID` pointing at the original task. So "the jobs of a run" spans attempts — a query
  that ignores `RunAttemptID` double-counts after any re-run. `run.LatestAttemptID` identifies the
  current one, and `setting.Actions.MaxRerunAttempts` caps the total.
- `ActionRun.Index` is the per-repository run number shown in the UI (`unique(repo_index)` with
  `RepoID`), exactly like issue numbers — see `models-issues.md`.
- **`ActionRun.Status` is guarded by an optimistic lock**: the `Version` field is tagged
  `xorm:"version default 0"` because runners update status concurrently. Write status through the
  helpers that respect it; a blind `Update` will silently lose races.
- The `Status` values are deliberately numbered to match the runner protocol:
  `StatusUnknown`/`Success`/`Failure`/`Cancelled`/`Skipped` (0-4) map onto `runnerv1.Result_*`.
  `StatusWaiting`, `StatusRunning` and `StatusBlocked` (5-7) are Gitea-only. **Never renumber
  them** — the wire protocol depends on the first five.
- `IsForkPullRequest` and `NeedApproval` exist because a PR from a fork runs untrusted code.
  Anything that decides whether a run may execute must consider both.
- `WorkflowRepoID` / `WorkflowCommitSHA` record where the workflow *file content* came from, which
  is not necessarily the repo being built (`IsScopedRun`).
- Task logs are not files on disk — they live in `models/dbfs`.

## Recipes

**Add a field to a run.** Add it to `ActionRun`, write the migration
(`models-migrations.md`), and check whether the runner protocol in `routers-api-actions.md` needs
to carry it.

**Query a run's jobs.** Go through the list types (`run_job_list.go`, `task_list.go`) rather than
looping per row; they batch like `IssueList` does.

**Add a status.** Almost certainly wrong — see the numbering invariant above. A new *intermediate*
state must be appended after `StatusBlocked`, never inserted.

## Gotchas

- "Job" and "task" are not interchangeable here even though the UI blurs them. Reading the wrong
  table, or ignoring `RunAttemptID`, gives plausible-looking but wrong results for re-runs.
- A run that predates the attempt model has `LatestAttemptID == 0` and is treated as attempt 1;
  `services/actions/rerun.go` backfills it. Code reading attempts must tolerate that legacy shape.
- `models/dbfs` looks like infrastructure but exists almost entirely for Actions logs; changing it
  affects log retention and cleanup.
- Schedules have both a `schedule` and a `schedule_spec` row; the spec holds the parsed cron
  expression and the next run time.

## Related

- `services-actions.md` — what creates and advances these rows
- `routers-api-actions.md` — the runner protocol that updates tasks
- `models-db.md` — optimistic locking and transactions
