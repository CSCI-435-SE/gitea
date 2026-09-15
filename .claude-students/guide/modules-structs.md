---
source: docs/modules-structs.md
source-hash: d8f4dbe1db8c4d42
verified-at: c0092050a4
---

<!-- Derived from docs/modules-structs.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# The API's data shapes, in plain English

**In one sentence:** plain Go structs describing exactly what Gitea's API sends and accepts — a
public contract, which is why the rules here are about what you must *not* change.
**Come here when:** you are adding or changing a field on any API request or response, or on a
webhook payload.

## What this is

Every JSON shape Gitea speaks, as plain structs with no behaviour: the things the API returns
(issues, repositories, pull requests), the option types it accepts when you create or edit
something, and the payloads sent to webhooks.

It is a leaf package — it imports nothing from Gitea — which is why both `models/` and `services/`
can use it without breaking the dependency direction.

## Why it exists

**Because the database's shape and the API's shape must be free to differ.**

If the API returned database rows directly, then renaming a column would break everyone's scripts,
and internal columns nobody should see would leak out. Instead there is a deliberate translation
step, and these structs are the target of it: the promise Gitea makes about what its API looks like.

Once you see them as a *promise*, the rules stop feeling restrictive. Somebody's deployment script
reads a field. A dashboard parses a webhook payload. Those break the moment the shape changes — and
they break at their end, with no warning, in a way you will never see.

That is the whole reason the strictest rule here is "never remove or rename a field".

## Words you'll meet

- **DTO** — a struct that exists to be converted to JSON, with no logic.
- **JSON tag** — the annotation setting the name a field gets in JSON.
- **wire shape** — what actually travels over the network.
- **option type** — the struct describing a request body, usually named `XxxOption`.
- **swagger model** — a struct marked to appear in the generated API specification.
- **RFC 3339** — the standard timestamp format the API emits.
- **zero value** — Go's default for an unset field. Here, the thing that ships when you forget a
  step.
- **public contract** — a shape other people's software depends on.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/structs/issue.go`, `issue_comment.go`, `issue_label.go`, `issue_milestone.go` | The issue domain. |
| `modules/structs/repo.go`, `repo_file.go`, `repo_branch.go`, `repo_tag.go` | Repositories. |
| `modules/structs/pull.go`, `pull_review.go` | Pull requests and reviews. |
| `modules/structs/user.go`, `org.go`, `org_team.go` | Users and organisations. |
| `modules/structs/hook.go` | Webhook payloads. |
| `modules/structs/doc.go` | The package's documentation header. |

## The rules, and why

**A response type carries a marker comment putting it in the generated specification.** Without it
the type exists in Go and does not exist as far as the API documentation is concerned.

**Field names follow GitHub's API** wherever an equivalent exists. That compatibility is the reason
tools written for GitHub work against Gitea at all, and deviating without a reason is a review
rejection.

**Never remove or rename a field.** Version 1 of the API is frozen. A field that should not exist
gets a comment noting it for a hypothetical version 2 — it does not get deleted. This feels wrong
the first time; it is what "we do not break people's software" means in practice.

**Times use the standard format.** Do not introduce a second one.

**These structs have no behaviour.** No methods that query anything, and nothing imported from
`models/`. Adding either would invert the dependency direction (`architecture.md`) and make this
package impossible to use from both sides.

**Adding a type here is only step one.** Request bodies must also be listed in the swagger options
file, and responses registered by category, or the specification will not include them
(`routers-api-v1.md`).

## How to actually do it

**Expose a new field.** Three steps, and the middle one is the one people forget:

1. Add it here with the right JSON tag.
2. **Set it in the matching conversion function** (`services-forms-and-convert.md`).
3. Run `make generate-swagger` and commit the regenerated specification.

**Add a request option type.** Define it here, add it to the swagger options list, and bind it on
the route with the API-side bind helper.

**Find the type behind an endpoint.** The handler's swagger comment names it in its responses
block; the struct is in the file matching its domain.

## Traps, and what they look like

**Your new field is always empty — `0`, `""` or `false` — and the API docs say it exists.** You
added the field but never set it in the conversion function. This is the characteristic failure
here, and it is worse than an error: consumers see a documented field that is permanently blank, so
they reasonably conclude the data is missing rather than that Gitea has a bug.

**Your type does not appear in the API documentation.** It was never registered in the swagger
files.

**A reviewer asks why your field is not named what GitHub calls it.** Because compatibility is the
point. If GitHub has an equivalent, use its name.

**You use a different import alias and the file becomes inconsistent.** The convention is `api`.
Match what the file already does.

**You change a webhook payload and something outside Gitea breaks.** Webhook payloads are just as
public as API responses. Somebody's automation is parsing that JSON.

## Where to go next

- `docs/modules-structs.md` (in this folder) — the reference page this was written from
- `services-forms-and-convert.md` — the conversion functions that fill these structs
- `routers-api-v1.md` — registering them, and the status-code rules
