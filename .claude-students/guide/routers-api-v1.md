---
source: docs/routers-api-v1.md
source-hash: f9453e268dc5eda8
verified-at: c0092050a4
---

<!-- Derived from docs/routers-api-v1.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# The JSON API, in plain English

**In one sentence:** the endpoints under `/api/v1` that programs use instead of the website, with
much stricter rules than web pages because other people's software depends on them.
**Come here when:** you are adding or changing an API endpoint, or a swagger check is failing and
you do not know what it wants.

## What this is

The REST API. Same idea as the web routes — a URL, guards, a handler — but it returns JSON instead
of HTML, and it carries a documentation obligation that web pages do not.

- `routers/api/v1/api.go` — every endpoint, plus every guard middleware, in one file.
- `routers/api/v1/repo/`, `org/`, `user/`, `admin/`, `notify/`, `packages/`, `settings/`, `misc/`,
  `activitypub/`, `token/` — the handlers.
- `routers/api/v1/swagger/` — where you register your request and response types so they appear in
  the generated API documentation.
- `routers/api/v1/utils/` — shared helpers, including pagination.

## Why it exists

A website can change freely: a user reloads the page and sees the new layout. **An API cannot.**
Somebody's script, CI pipeline or mobile app is calling these endpoints, and it will break the
moment a field disappears or changes meaning.

That single fact explains every rule below. Gitea's API deliberately mirrors GitHub's, so tools
written for GitHub mostly work against Gitea. It is documented by a machine-readable specification
generated from comments in the code, which is why an undocumented endpoint is treated as a broken
one. And it is versioned `v1` — a version that is never allowed to break.

So the rules are not bureaucracy. They are the reason people can build against Gitea at all.

## Words you'll meet

- **REST endpoint** — one URL plus one HTTP method, doing one thing.
- **swagger / OpenAPI** — a machine-readable description of an API. Gitea generates it from the
  comments above each handler, and tools use it to build clients automatically.
- **swagger block** — that comment. It lists the parameters and every possible response.
- **DTO** — "data transfer object": a struct that exists to be turned into JSON, separate from the
  database struct.
- **guard** — middleware that refuses the request: not signed in, no permission, feature disabled.
- **token scope** — how much an API token is allowed to do. A token limited to reading issues must
  not be able to delete a repository.
- **operation id** — the unique name for one endpoint in the specification.

## What's in these files

| Where | What it is for |
| --- | --- |
| `routers/api/v1/api.go` | The route list and every `req*` / `must*` guard. Big, but it is the whole API at a glance. |
| `routers/api/v1/repo/issue.go` | The best file to copy from: real handlers with complete swagger blocks. |
| `routers/api/v1/swagger/options.go` | `swaggerParameterBodies`. Every request body type must be listed here or it will not appear in the spec. |
| `routers/api/v1/swagger/issue.go` | Response registration, one file per category. |
| `routers/api/v1/utils/hook.go` | A worked example of setting the total-count header on a list endpoint. |

## The rules, and why

These come from `docs/guidelines-backend.md`, which is the authority — read it before designing
anything new.

**Mirror GitHub's names where an equivalent exists.** Adding something GitHub does not have is fine.
Renaming or removing something that already exists is not.

**Never break v1.** If the right fix would be a breaking change, do not make it. Leave a comment
describing what a future v2 should do. This feels wrong the first time and is correct.

**Document every outcome in the swagger block** — success, failure, and each individual error. An
undocumented response is invisible to everyone generating a client from the spec.

**Request and response types live in `modules/structs/` and must be registered.** A request body
type goes in `swaggerParameterBodies` in `routers/api/v1/swagger/options.go`; a response type goes
in the matching file under `routers/api/v1/swagger/`. Miss the registration and your type simply
will not be in the specification, with no error to tell you.

**The HTTP method decides the status code.** GET returns 200 with the object. POST returns 201 with
the created object. PUT returns 204 and no body. PATCH returns 200 with the changed object. DELETE
returns 204 and no body. Clients rely on this; it is not a style preference.

**On an edit endpoint, every field is optional** except the ones identifying what to edit. A caller
updating one field must not have to resend the rest — and must not accidentally blank it.

**List endpoints take `page` and `limit`, and set the total count** via `ctx.SetTotalCountHeader`,
which produces the `X-Total-Count` header. Without it a client cannot tell how many pages there are.

**Handlers take `*context.APIContext`**, convert database structs with `services/convert`, and
return with `ctx.JSON`. The swagger comment sits directly above the handler body and starts with
`// swagger:operation <METHOD> <path> <tag> <operationId>` followed by `// ---`.

## How to actually do it

**Add an endpoint.**

1. If it accepts a body, define the option struct in `modules/structs/`, then add it to
   `swaggerParameterBodies` in `routers/api/v1/swagger/options.go`.
2. If it returns something new, define that struct in `modules/structs/` and register it in the
   matching file under `routers/api/v1/swagger/`.
3. Write the handler in the right subpackage, with the complete swagger block above it.
4. Register the route in `api.go`, inside the group whose guards you need, adding `reqToken()`,
   `tokenRequiresScopes(...)` and `bind(api.XxxOption{})` as appropriate.
5. Run `make generate-swagger` and **commit** the regenerated `templates/swagger/v1_json.tmpl`.

**Choose your guards.** They are grouped by what they check:

- Who you are: `reqToken`, `reqBasicOrRevProxyAuth`, `tokenRequiresScopes`
- Repository access: `repoAssignment()`, `reqRepoReader(unit)`, `reqRepoWriter(units...)`,
  `reqAnyRepoReader`
- Organisation access: `orgAssignment()`, `reqOrgMembership`, `reqOrgOwnership`,
  `reqTeamMembership`
- Administrative: `reqSiteAdmin`, `reqOwner`, `reqAdmin`, `reqSelfOrAdmin`
- Is the feature even on: `mustEnableIssues`, `mustAllowPulls`, `mustEnableWiki`,
  `mustNotBeArchived`, `mustEnableEditor`, `mustEnableAttachments`

## Traps, and what they look like

**CI fails on `make swagger-check` and you did not touch the spec.** You edited a swagger comment.
The committed specification no longer matches the comments. Run `make generate-swagger` and commit
`templates/swagger/v1_json.tmpl` along with your change. This catches almost everyone once.

**Your new field does not appear in the API docs.** You defined the struct but never registered it
under `routers/api/v1/swagger/`.

**You try to "fix" the odd annotation in `options.go`.** The `swagger:parameters parameterBodies`
annotation deliberately matches no real route — it exists purely to force the generator to emit
those type definitions. Add your type to the struct and leave the annotation alone.

**Your request body binds as empty.** There are two similarly named binding mechanisms. The API uses
the local `bind` helper in `api.go` with `modules/structs` types. The web layer uses `web.Bind` with
`services/forms` types. They are not interchangeable, and using the wrong one fails at request time
rather than compile time.

**You add a guard that was already there.** Guards on a group apply to every route inside it. Read
the group before adding a per-route guard — duplicating one is harmless but confusing, and it
usually means you misread which group you are in.

## Where to go next

- `docs/routers-api-v1.md` (in this folder) — the reference page this was written from
- `routers-web.md` — the HTML side, and the middleware both layers share
- `services-forms-and-convert.md` — `services/convert`, which builds these JSON structs
