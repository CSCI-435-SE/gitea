---
source: docs/routers-api-actions.md
source-hash: 16fcbaf5bfc2082f
verified-at: c0092050a4
---

<!-- Derived from docs/routers-api-actions.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# How runners talk to Gitea, in plain English

**In one sentence:** a separate machine-facing API where runners register, ask for work and report
results — nothing like the JSON API, with its own authentication and its own rules.
**Come here when:** you are changing how runners register, collect tasks or report back, or touching
artifact upload and download.

## What this is

The endpoints a runner talks to. These are consumed by the runner software, never by a browser, and
they are a **completely separate API** from `/api/v1` — different location, different
authentication, different conventions.

## Why it exists

A runner is a program on another machine, possibly behind a firewall, possibly on someone's laptop.
Gitea cannot connect *to* it. So the relationship is inverted: **the runner keeps asking Gitea
whether there is anything to do.**

That creates a scaling problem. If an instance has fifty runners each asking every few seconds,
that is a constant stream of "anything for me?" requests, mostly answered "no". Doing real work on
every one of those would be a self-inflicted denial of service.

The solution is a **version handshake**. Gitea keeps a number that changes whenever the set of
available work changes. The runner remembers the last number it saw and sends it back. If the
numbers match, nothing has changed since last time and Gitea answers immediately without looking at
anything. Only when they differ does it actually search for work.

So the common case — a quiet instance with idle runners — costs almost nothing. Understanding that
is what stops you accidentally breaking it.

## Words you'll meet

- **runner** — the program that executes CI jobs, running elsewhere.
- **poll** — asking repeatedly, because the server cannot call you.
- **task version** — the number that changes when available work changes.
- **handshake** — comparing that number instead of doing a real search.
- **registration token** — the secret a runner uses to enrol itself.
- **interceptor** — middleware for this protocol, checking that token.
- **protobuf** — a compact message format where the message types are generated from a schema file.
- **generated file** — code produced by a tool, which you must not edit by hand.
- **throttle** — deliberately refusing to do work right now to protect the server.

## What's in these files

| Where | What it is for |
| --- | --- |
| `routers/api/actions/runner/runner.go` | The runner service: registering, declaring capabilities, fetching a task, reporting progress and logs. |
| `routers/api/actions/runner/interceptor.go` | Runner authentication. |
| `routers/api/actions/actions.go` | Wiring the routes up. |
| `routers/api/actions/artifacts.go`, `artifacts_chunks.go`, `artifacts_utils.go` | The older artifact protocol. |
| `routers/api/actions/artifactsv4.go` | The newer one. |
| `routers/api/actions/artifact.proto`, `artifact.pb.go` | The message schema, and the code generated from it. |
| `models/actions/tasks_version.go` | The version number the handshake compares. |
| `routers/api/actions/job_summary.go` | Uploading a job summary. |
| `routers/api/actions/ping/` | A health check. |

## The rules, and why

**This is not REST, and the API conventions do not apply here.** The message types are generated
from a schema, and the swagger rules from `routers-api-v1.md` are irrelevant. Do not try to apply
them.

**The runner's lifecycle is fixed:** register once with a registration token, declare what it can do
(labels and version), then repeatedly ask for a task, and report progress and logs while running
one.

**Status values on the wire are the numbers from the shared status enum.** This is the other half of
the "never renumber" rule in `models-actions.md` — this protocol is what depends on those numbers.

**Authentication is the runner token, checked by the interceptor.** Not a user session, not an API
token. None of the guards used on `/api/v1` apply, which means you cannot reason about access here
by analogy with the rest of Gitea.

**Two artifact protocol versions exist side by side**, because runners in the wild are different
versions and Gitea has to serve them all. A change to artifact handling usually has to be made in
both — or deliberately limited to one, on purpose, with that decision written down.

**Fetching a task is a version handshake, not a blocking wait.** The runner sends the version it
last saw; the server compares and only searches for work when they differ, returning immediately
either way.

**Code that makes the versions always differ turns every runner into a busy loop against the
database.** That is the failure mode to watch for when touching this path, and it will not show up
in testing with one runner and one repository.

**When task picking is throttled, the version is deliberately not advanced.** That is not an
oversight. Leaving the version unchanged means the runner sees "still different" and retries on its
next poll, rather than believing it is up to date and waiting for the next real change. Preserve
that.

## How to actually do it

**Add a field the runner reports.** Four steps, in order: update the schema file, regenerate the
generated Go file, handle the field where results are received, and persist it with a migration.

**Work out why a runner never picks up jobs.** In order:

1. Did registration and declaration succeed, and do the runner's labels match what the workflow's
   `runs-on` asks for?
2. Is any job actually in a runnable state (`services-actions.md`)?
3. Is the task version being bumped at all — if it never changes, the runner is correctly told
   there is nothing new, forever.
4. Is the interceptor rejecting the token?

**Follow a task's results.** Status and step results arrive through the update endpoint; log output
is streamed separately into the database-backed store.

## Traps, and what they look like

**Every runner starts hammering the database.** Something made the version comparison always
differ, so the cheap path is never taken. With one runner in testing this is invisible; with fifty
in production it is an outage.

**You edit the generated Go file and your change disappears.** It is generated from the schema.
Change the schema and regenerate.

**Your new route is invisible to runners.** These routes are mounted separately from the JSON API.
Adding something to the `/api/v1` route table does not expose it here.

**A small addition to the fetch path has an outsized cost.** That endpoint runs constantly across
every runner on the instance. Anything added there is multiplied by fifty machines times however
often they poll.

**Artifacts work for one runner and not another.** The two runners are using different artifact
protocol versions, and your change only went into one of them.

## Where to go next

- `docs/routers-api-actions.md` (in this folder) — the reference page this was written from
- `services-actions.md` — what decides a job is ready to be picked up
- `models-actions.md` — the task rows, and the status numbers this protocol depends on
- `routers-api-v1.md` — the separate, user-facing API
