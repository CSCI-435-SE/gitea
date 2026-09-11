---
scope: services/actions, modules/actions
verified-at: c0092050a4
---

# services/actions — triggering workflows and dispatching jobs

**Read when:** changing when a workflow runs, how jobs become runnable, approvals, re-runs,
concurrency or commit statuses from Actions.
**Not here:** the rows -> `models-actions.md`; the runner's HTTP API -> `routers-api-actions.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `services/actions` | the trigger path, job readiness, approvals, re-runs, concurrency, artifacts, cleanup, commit statuses |
| `modules/actions` | parsing workflow files: `workflows.go`, `jobparser/`, `workflowpattern/`, `github.go` (the GitHub-compatible context), `log.go`, `task_state.go` |

## Key files

| Path | What it holds |
| --- | --- |
| `services/actions/notifier.go`, `notifier_helper.go` | the trigger path — `notifyInput`, `Notify`, `handleWorkflows`, `skipWorkflows` |
| `services/actions/init.go` | registers the notifier and starts the queues |
| `services/actions/job_emitter.go` | `EmitJobsIfReadyByRun`, `EmitJobsIfReadyByJobs`, the `jobEmitterQueue` |
| `services/actions/concurrency.go` | concurrency-group blocking |
| `services/actions/approve.go` | fork-PR approval |
| `services/actions/rerun.go` | re-running a run or a single job |
| `services/actions/schedule_tasks.go` | turning cron schedules into runs |
| `services/actions/commit_status.go` | publishing job results as commit statuses |
| `services/actions/artifacts.go`, `cleanup.go` | artifact handling and retention |
| `services/actions/workflow.go`, `reusable_workflow.go` | workflow resolution, including reusable workflows |
| `modules/actions/workflows.go`, `jobparser/` | parsing `.gitea/workflows/*.yaml` into jobs |

## Conventions & invariants

- **Actions is a notifier, not a caller.** `init.go` does
  `notify_service.RegisterNotifier(NewNotifier())`, so workflows are triggered by the same
  `notify_service.*` events that drive webhooks and mail (`services-issue.md`). Nothing calls
  "run the workflows" directly — you make an event happen and the notifier decides.
- The trigger path builds a `notifyInput` fluently (`newNotifyInput(...).WithDoer(...).WithRef(...)
  .WithPayload(...)`) and ends in `Notify(ctx)`. Add a trigger by adding the notifier method and
  the input, not by reaching into `handleWorkflows`.
- **Jobs do not start themselves.** `EmitJobsIfReadyByRun` pushes onto `jobEmitterQueue`, and the
  handler re-evaluates which jobs have satisfied dependencies (`needs`) and concurrency limits.
  Anything that could unblock a job must enqueue, not flip a status.
- A run from a forked PR (`IsForkPullRequest`) may require approval before any job executes —
  never bypass `approve.go` when adding a new execution path.
- Workflow files are parsed in `modules/actions`, which is dependency-light on purpose
  (`architecture.md`); the parsed job payload is stored on the job row.
- `modules/actions/github.go` builds the GitHub-compatible expression context. Compatibility with
  GitHub Actions is the design goal, so a new context field should match GitHub's name.

## Recipes

**Trigger workflows on a new event.** Add the method to the Actions notifier, build a
`notifyInput` with the right `webhook_module.HookEventType` and payload, and confirm the `on:`
keyword is recognised in `modules/actions/workflows.go`.

**Change when a job becomes runnable.** `job_emitter.go` — specifically the readiness check the
queue handler performs. Enqueue from wherever the unblocking change happened.

**Debug "my workflow did not run".** In order: was the notify event fired; did `skipWorkflows`
filter it (`[skip ci]` and friends); did the workflow file parse; is the run waiting on approval;
is it blocked by concurrency; is a runner registered with matching labels.

## Gotchas

- `services/actions/notifier.go` implements the notifier interface, so a method there runs on
  *every* matching event repo-wide — keep it cheap and let the queue do the work.
- Concurrency blocking and `needs` dependencies both produce `StatusBlocked`, so a stuck job needs
  both checked.
- `token_permission_design.md` sits in this package and documents the token permission model —
  read it before touching `permission_parser.go`.

## Related

- `models-actions.md` — the rows this package writes
- `routers-api-actions.md` — how runners collect and report tasks
- `services-repository.md` — the push that starts most runs
