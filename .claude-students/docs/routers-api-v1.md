---
scope: routers/api/v1
verified-at: c0092050a4
---

# routers/api/v1 — the REST API, its guards and its swagger spec

**Read when:** adding or changing a `/api/v1` endpoint, or a swagger check fails.
**Not here:** HTML routes -> `routers-web.md`; `ctx` itself -> `services-context.md`; DTO conversion -> `services-forms-and-convert.md`.

## Responsibilities

- `routers/api/v1/api.go` — `Routes` declares every v1 endpoint and defines every guard middleware.
- `routers/api/v1/{repo,org,user,admin,notify,packages,settings,misc,activitypub,token}/` — the
  handlers.
- `routers/api/v1/swagger/` — registers request and response structs so go-swagger emits their
  definitions.
- `routers/api/v1/utils/` — shared helpers, including pagination.

## Key files

| Path | What it holds |
| --- | --- |
| `routers/api/v1/api.go` | `Routes`, and all the `req*` / `must*` / `bind` middleware |
| `routers/api/v1/repo/issue.go` | representative handlers with full swagger comment blocks |
| `routers/api/v1/swagger/options.go` | `swaggerParameterBodies` — every request body must be listed here |
| `routers/api/v1/swagger/issue.go` | response registration, one file per category |
| `routers/api/v1/utils/hook.go` | example of `ctx.SetTotalCountHeader` on a list endpoint |

## Conventions & invariants

From `docs/guidelines-backend.md` (authoritative — read it before designing an endpoint):

- Mirror the GitHub API's endpoints and field names where possible. Extra endpoints and fields are
  fine when Gitea offers something GitHub does not; removing or renaming an existing field is not.
- **Never break v1.** If a fix would be breaking, leave a code comment for a hypothetical v2
  instead.
- **Every** result — success, failure and each error — must appear in the swagger comment block.
- Every JSON request body is a struct in `modules/structs/` **and** listed in
  `routers/api/v1/swagger/options.go`. Every JSON response is a struct in `modules/structs/` **and**
  registered by category under `routers/api/v1/swagger/`. Miss either and the spec is incomplete.
- Method to status: GET → 200 with the object; POST → 201 with the created object; PUT → 204 with
  no body; PATCH → 200 with the changed object; DELETE → 204 with no body.
- On an edit endpoint every parameter is optional except the ones identifying the object.
- List endpoints accept `page` and `limit` and call `ctx.SetTotalCountHeader` so `X-Total-Count`
  is set.

Handler shape: `func Foo(ctx *context.APIContext)`, converting models with `services/convert`
before `ctx.JSON`. The swagger block sits immediately above the handler body and opens with
`// swagger:operation <METHOD> <path> <tag> <operationId>` followed by `// ---`.

## Recipes

**Add an endpoint.**

1. If it takes a body: an option struct in `modules/structs/`, then add it to
   `swaggerParameterBodies` in `routers/api/v1/swagger/options.go`.
2. If it returns something new: a struct in `modules/structs/`, registered in the matching
   `routers/api/v1/swagger/<category>.go`.
3. The handler in the right subpackage, with the complete swagger comment block.
4. The route in `api.go`, inside the group whose guards you need, adding `reqToken()`,
   `tokenRequiresScopes(...)` and `bind(api.XxxOption{})` as appropriate.
5. `make generate-swagger`, and commit the regenerated `templates/swagger/v1_json.tmpl`.

**Pick guards.** Authentication: `reqToken`, `reqBasicOrRevProxyAuth`, `tokenRequiresScopes`.
Repo access: `repoAssignment()`, `reqRepoReader(unit)`, `reqRepoWriter(units...)`,
`reqAnyRepoReader`. Org access: `orgAssignment()`, `reqOrgMembership`, `reqOrgOwnership`,
`reqTeamMembership`. Admin: `reqSiteAdmin`, `reqOwner`, `reqAdmin`, `reqSelfOrAdmin`. Feature
gates: `mustEnableIssues`, `mustAllowPulls`, `mustEnableWiki`, `mustNotBeArchived`,
`mustEnableEditor`, `mustEnableAttachments`.

## Gotchas

- `make swagger-check` fails CI when the committed spec does not match the comments, so a swagger
  block edit always means regenerating and committing `templates/swagger/v1_json.tmpl`.
- The `swagger:parameters parameterBodies` annotation in `options.go` matches no real route on
  purpose — it exists only to force go-swagger to emit the definitions. Add to the struct; do not
  "fix" the annotation.
- `bind` here is the API-local helper in `api.go` over `modules/structs` types. The web layer's
  `web.Bind` over `services/forms` types is a different mechanism.
- A guard added to a group applies to every route in it — check the group before adding a
  per-route guard that is already there.

## Related

- `routers-web.md` — the HTML side and the shared middleware in `routers/common`
- `services-forms-and-convert.md` — `services/convert` turns models into these DTOs
