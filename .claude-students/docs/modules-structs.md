---
scope: modules/structs
verified-at: c0092050a4
---

# modules/structs — the API data types

**Read when:** adding or changing a field on any API request or response.
**Not here:** the conversion from models -> `services-forms-and-convert.md`; route and swagger registration -> `routers-api-v1.md`.

## Responsibilities

Every JSON shape Gitea's API speaks, as plain Go structs: responses (`Issue`, `Repository`,
`PullRequest`), request option types (`CreateIssueOption`, `EditRepoOption`), and webhook payloads.
It is a leaf package with no Gitea dependencies, which is why both models and services can use it.

## Key files

| Path | What it holds |
| --- | --- |
| `modules/structs/issue.go`, `issue_comment.go`, `issue_label.go`, `issue_milestone.go` | the issue domain |
| `modules/structs/repo.go`, `repo_file.go`, `repo_branch.go`, `repo_tag.go` | the repository domain |
| `modules/structs/pull.go`, `pull_review.go` | pull requests and reviews |
| `modules/structs/user.go`, `org.go`, `org_team.go` | users and organisations |
| `modules/structs/hook.go` | webhook payload types |
| `modules/structs/doc.go` | the package's swagger documentation header |

## Conventions & invariants

- Response types carry a `// swagger:model` comment; that is what puts them in the generated spec.
- **Field names and JSON tags follow the GitHub API** wherever an equivalent exists
  (`docs/guidelines-backend.md`). Deviating without reason is a review rejection.
- **Never remove or rename a field.** v1 is frozen; a field that should not exist gets a comment
  pointing at a hypothetical v2 (`routers-api-v1.md`).
- Times are `time.Time` with `json:"..."` tags; the API emits RFC 3339. Do not introduce a second
  time format.
- A struct here is *only* a wire shape. No methods that query, no imports from `models/` — that
  would invert the dependency direction (`architecture.md`).
- Adding a type is only step one: request bodies must also be listed in
  `routers/api/v1/swagger/options.go` and responses registered by category under
  `routers/api/v1/swagger/`, or the spec will not contain them.

## Recipes

**Expose a new field.** Add it here with the right JSON tag, set it in the matching
`services/convert` function, then `make generate-swagger` and commit the regenerated spec.

**Add a request option type.** Define `XxxOption` here, add it to `swaggerParameterBodies` in
`routers/api/v1/swagger/options.go`, and bind it on the route with the API-local `bind` helper.

**Find the type behind an endpoint.** The handler's swagger `responses:` block names it; the type
is in the file matching its domain.

## Gotchas

- A field added here but never set in `services/convert` ships as the zero value while swagger
  documents it as present — silently wrong for every consumer.
- `api` is the conventional import alias (`api "gitea.dev/modules/structs"`); match it.
- Webhook payloads in `hook.go` are part of the public contract too — third-party receivers break
  on changes just as API clients do.

## Related

- `routers-api-v1.md` — swagger registration and the status-code rules
- `services-forms-and-convert.md` — what fills these structs
