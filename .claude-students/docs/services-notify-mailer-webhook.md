---
scope: services/notify, services/webhook, services/mailer, services/uinotification, services/feed, services/cron, services/task
verified-at: c0092050a4
---

# services/notify and its consumers — the event fan-out

**Read when:** adding a new event, a webhook payload, an email, or a scheduled job.
**Not here:** where events are fired from -> `services-issue.md`, `services-repository.md`.

## Responsibilities

`services/notify` is a **publish/subscribe hub**. Service code calls `notify_service.SomeEvent(...)`
and every registered notifier receives it.

| Notifier | Effect |
| --- | --- |
| `services/webhook` | delivers HTTP webhooks, including the per-platform formats |
| `services/mailer` | sends email |
| `services/uinotification` | writes the in-app notification rows |
| `services/feed` | writes the activity feed |
| `services/actions` | triggers workflows (`services-actions.md`) |

`services/cron` runs scheduled maintenance; `services/task` runs one-off background jobs
(migrations, imports).

## Key files

| Path | What it holds |
| --- | --- |
| `services/notify/notifier.go` | the `Notifier` interface — every event Gitea can publish |
| `services/notify/notify.go` | `RegisterNotifier` and the dispatch functions |
| `services/notify/null.go` | `NullNotifier`, the embeddable no-op base |
| `services/webhook/notifier.go`, `deliver.go` | webhook subscription and delivery |
| `services/webhook/slack.go`, `discord.go`, `matrix.go`, `dingtalk.go`, `feishu.go`, `msteams.go`, `packagist.go` | per-platform payload formatting |
| `services/webhook/payloader.go`, `general.go` | the shared payload machinery |
| `services/cron/tasks.go`, `tasks_basic.go`, `tasks_extended.go`, `tasks_actions.go` | the registered scheduled tasks |

## Conventions & invariants

- **Adding an event means changing the `Notifier` interface**, which every notifier must satisfy.
  That is why `NullNotifier` exists: notifiers embed it and override only what they care about, so
  a new method does not break them all.
- Notifiers are registered at startup with `notify_service.RegisterNotifier`. Registration order
  is not a priority — do not rely on one notifier running before another.
- **Notify calls happen after the transaction commits** (`services-issue.md`). This package is the
  reason that rule matters: one call can send mail, deliver webhooks and start CI.
- A notifier method runs for *every* matching event on the instance, so it must be cheap and
  delegate real work to a queue (`modules-infra.md`). Webhook delivery is queued, not inline.
- Webhook payloads are `modules/structs` types (`modules-structs.md`) and are a public contract —
  third-party receivers break when a field changes.
- Scheduled tasks are registered in `services/cron/tasks*.go` with a config-driven schedule; a new
  task is registered there, not started with a bare goroutine.

## Recipes

**Add an event.** Add the method to `Notifier`, to `NullNotifier`, and to each notifier that should
react. Then call `notify_service.YourEvent(...)` from the service that performs the change, after
its transaction.

**Add a webhook platform.** A new file in `services/webhook/` implementing the payload conversion,
following `slack.go`, plus the form and admin UI.

**Add a scheduled job.** Register it in `services/cron/tasks_extended.go` with its default
schedule; make the handler safe to run concurrently with itself.

## Gotchas

- Because notifiers are fire-and-forget, a failure in one is logged rather than surfaced to the
  user. "The action worked but no email arrived" is a notifier bug, not a handler bug.
- Adding a `Notifier` method is a compile break across every notifier — expect to touch all of
  them in one commit.
- `services/task` (one-off background work) and `services/cron` (recurring) are easy to confuse.

## Related

- `services-issue.md` — the notify-after-commit rule
- `services-actions.md` — the notifier that starts CI
- `modules-infra.md` — the queues delivery runs on
