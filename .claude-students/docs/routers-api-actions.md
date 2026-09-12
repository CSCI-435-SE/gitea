---
scope: routers/api/actions
verified-at: c0092050a4
---

# routers/api/actions — the runner protocol and artifact endpoints

**Read when:** changing how runners register, collect tasks or report results, or touching artifact
upload/download.
**Not here:** what creates the tasks -> `services-actions.md`; the rows -> `models-actions.md`; the user-facing REST API -> `routers-api-v1.md`.

## Responsibilities

The machine-facing endpoints mounted at `/api/actions` (see `NormalRoutes` in `routers/init.go`).
These are consumed by `act_runner`, not by browsers, and they are a separate API from `/api/v1`
with their own auth.

## Key files

| Path | What it holds |
| --- | --- |
| `routers/api/actions/runner/runner.go` | the runner service: `Register`, `Declare`, `FetchTask`, `UpdateTask`, `UpdateLog` |
| `models/actions/tasks_version.go` | `GetTasksVersionByScope`, `IncreaseTaskVersion` — the polling handshake |
| `routers/api/actions/runner/interceptor.go` | runner authentication |
| `routers/api/actions/actions.go` | route wiring |
| `routers/api/actions/artifacts.go`, `artifacts_chunks.go`, `artifacts_utils.go` | the v1 artifact protocol |
| `routers/api/actions/artifactsv4.go` | the v4 artifact protocol |
| `routers/api/actions/artifact.proto`, `artifact.pb.go` | the generated protobuf types |
| `routers/api/actions/job_summary.go` | job summary upload |
| `routers/api/actions/ping/` | the runner health check |

## Conventions & invariants

- **This is a Connect/gRPC-style protocol, not REST.** The runner service is served through
  `NewRunnerServiceHandler`, and its message types are generated. Swagger conventions from
  `routers-api-v1.md` do not apply here.
- The runner's lifecycle is `Register` (once, with a registration token) → `Declare` (labels and
  version) → `FetchTask` (polled repeatedly; see the handshake below) → `UpdateTask` / `UpdateLog`
  (progress and results).
- **Status codes on the wire are the numeric `Status` values** from `models/actions/status.go`,
  which is why the first five must keep matching `runnerv1.Result_*` (`models-actions.md`).
- Authentication is the runner token via the interceptor — not a user session and not an API token,
  so none of the `/api/v1` guards (`reqToken`, `tokenRequiresScopes`) apply.
- **Two artifact protocol versions coexist** (`artifacts.go` and `artifactsv4.go`) because runner
  versions differ. A change to artifact handling usually has to be made in both, or deliberately
  gated by version.
- **`FetchTask` is a version handshake, not a long poll.** The runner sends the `TasksVersion` it
  last saw; the server compares it with `GetTasksVersionByScope` and only searches for work when
  they differ, then returns immediately either way. That comparison is what keeps continuous
  polling cheap — code that makes the versions always differ turns every runner into a busy loop
  against the database.
- When task picking is throttled (`MAX_CONCURRENT_TASK_PICKS`), the handler deliberately does
  **not** advance the runner's version, so the runner retries on its next poll instead of waiting
  for the next bump. Preserve that behaviour when touching the pick path.

## Recipes

**Add a field the runner reports.** Update the `.proto`, regenerate `artifact.pb.go`, handle it in
`UpdateTask`, and persist it on `ActionTask` with a migration (`models-migrations.md`).

**Debug a runner that never picks up jobs.** Check registration (`Register`/`Declare` succeeded and
labels match the workflow's `runs-on`), then whether any job is actually in a runnable state
(`services-actions.md`), then whether the task version is being bumped at all, then the
interceptor's auth.

**Trace a task result.** `UpdateTask` writes the status and step results; `UpdateLog` streams
output into `models/dbfs`.

## Gotchas

- Generated files (`artifact.pb.go`) must not be hand-edited; change the `.proto`.
- These routes are mounted separately from `/api/v1` in `routers/init.go` — adding a route to
  `routers/api/v1/api.go` will not expose it to runners.
- Because runners poll continuously, anything added to `FetchTask` runs at high frequency across
  every runner on the instance.

## Related

- `services-actions.md` — job dispatch and readiness
- `models-actions.md` — `ActionTask`, `Status`, artifacts
- `routers-api-v1.md` — the separate, user-facing API
