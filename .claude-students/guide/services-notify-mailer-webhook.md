---
source: docs/services-notify-mailer-webhook.md
source-hash: e485337ab0106641
verified-at: a50a52cbc8
---

<!-- Derived from docs/services-notify-mailer-webhook.md. Do not edit by hand: fix the reference doc
     and regenerate this file. See MAINTENANCE.md rules 16-22. -->

# How one event becomes email, webhooks and CI, in plain English

**In one sentence:** Gitea announces that something happened, and a list of subscribers each decide
what to do about it — which is why closing an issue can send mail, call a webhook and start a build
without the closing code knowing about any of them.
**Come here when:** you are adding a new event, a webhook payload, an email, or a scheduled job.

## What this is

A publish-and-subscribe hub. Service code announces an event; every registered subscriber hears it.

| Subscriber | What it does with the event |
| --- | --- |
| `services/webhook` | Delivers HTTP webhooks, formatted per platform. |
| `services/mailer` | Sends email. |
| `services/uinotification` | Writes the in-app notifications. |
| `services/feed` | Writes the activity feed. |
| `services/actions` | Starts CI (`services-actions.md`). |

Two more packages sit alongside: `services/cron` for recurring scheduled work, and `services/task`
for one-off background jobs like imports.

## Why it exists

**Because the code that closes an issue should not know about email.**

Imagine the alternative. Closing an issue would call the mailer, then the webhook deliverer, then
the notification writer, then the activity feed, then CI. So would reopening it. So would every one
of the dozens of things that can happen. Adding a new subscriber would mean editing every one of
those places, and missing some.

Instead, closing an issue announces "this issue was closed". The subscribers were registered at
startup and each decides whether it cares. Adding a new kind of reaction means adding one
subscriber, not editing the whole codebase.

That is also why the "notify after the transaction" rule from `services-issue.md` matters so much
*here*: one announcement can send irreversible email, call somebody else's server, and start a
build.

## Words you'll meet

- **publish/subscribe** — announcing events without knowing who is listening.
- **notifier** — a subscriber.
- **event** — one thing that happened, as a method on a shared interface.
- **interface** — in Go, a list of methods a type must have.
- **embed** — including one type in another so its methods are inherited.
- **no-op** — a method that deliberately does nothing.
- **fire and forget** — the announcer does not wait for or check the result.
- **payload** — the JSON body sent to a webhook receiver.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/notify/notifier.go` | The interface listing every event Gitea can announce. |
| `services/notify/notify.go` | Registering a subscriber, and announcing events. |
| `services/notify/null.go` | A do-nothing base that subscribers embed. |
| `services/webhook/notifier.go`, `deliver.go` | Subscribing to events, and delivering the HTTP calls. |
| `services/webhook/slack.go`, `discord.go`, `matrix.go`, `dingtalk.go`, `feishu.go`, `msteams.go`, `packagist.go` | Formatting the payload each platform expects. |
| `services/webhook/payloader.go`, `general.go` | The shared payload machinery. |
| `services/cron/tasks.go`, `tasks_basic.go`, `tasks_extended.go`, `tasks_actions.go` | The scheduled tasks. |

## The rules, and why

**Adding an event means adding a method to the shared interface** — which every subscriber must
then have.

That sounds painful, and the do-nothing base is why it is not. Subscribers embed it and override
only the events they care about, so adding a method does not break them all. It is still a
compile-wide change, so expect to touch every subscriber in one commit — but most of those edits
are nothing.

**Registration order is not priority.** Do not write a subscriber that depends on another having
already run.

**Announcements happen after the transaction commits.** This package is the reason that rule exists
(`services-issue.md`): a single announcement can send mail, call a stranger's server, and start CI,
none of which can be taken back.

**A subscriber method runs for every matching event on the whole instance**, so it must be cheap and
hand real work to a queue (`modules-infra.md`). Webhook delivery is queued, not done inline —
otherwise a slow or unreachable receiver would hold up whoever triggered it.

**Webhook payloads are a public contract** (`modules-structs.md`). Somebody's automation parses that
JSON, and changing it breaks them silently.

**Scheduled tasks are registered with a configurable schedule.** A new recurring job goes there —
not started with a bare goroutine, which nobody can see, configure or disable.

## How to actually do it

**Add an event.** Add the method to the interface, to the do-nothing base, and to each subscriber
that should react. Then announce it from the service performing the change — after its transaction.

**Add a webhook platform.** A new file implementing the payload conversion, following an existing
one, plus its settings form and admin UI.

**Add a scheduled job.** Register it with its default schedule, and make the handler safe to run
alongside itself — a slow run can overlap the next.

## Traps, and what they look like

**The action worked but no email arrived.** Subscribers are fire-and-forget: a failure inside one is
logged, not returned to the user. So this is a subscriber bug, not a bug in the code that did the
thing. Look in the logs, not at the handler.

**Adding one interface method breaks the build in five places.** Expected. Add the method to the
do-nothing base and each real subscriber in the same commit.

**A webhook receiver outside Gitea suddenly breaks.** A payload field changed. Those payloads are as
public as the API.

**Your subscriber makes the whole instance slow.** It runs on every matching event repository-wide.
Move the work to a queue.

**You reach for the wrong package for background work.** Recurring work is `services/cron`; one-off
jobs are `services/task`. The names are close and the purposes are not.

**You trace a delivery back to a notifier and find none.** The test buttons on the webhook settings
page skip the announcement hub entirely: `routers/web/repo/setting/webhook.go` calls
`PrepareWebhook` and `PrepareWebhookPing` in `services/webhook/webhook.go` itself. The ping one also
skips the subscription and branch-filter checks on purpose, because a person asked for that single
delivery — so "this webhook is not subscribed to ping" is not a reason for it not to arrive.

## Where to go next

- `docs/services-notify-mailer-webhook.md` (in this folder) — the reference page this was written from
- `services-issue.md` — where the notify-after-commit rule is stated
- `services-actions.md` — the subscriber that starts CI
- `modules-infra.md` — the queues delivery runs on
