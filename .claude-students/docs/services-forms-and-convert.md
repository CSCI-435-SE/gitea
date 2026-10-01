---
scope: services/forms, services/convert
verified-at: 32ac0cbb78
---

# services/forms and services/convert — request in, DTO out

**Read when:** adding a form POST to a web page, or returning a model through the API.
**Not here:** route registration -> `routers-web.md` / `routers-api-v1.md`; the DTO structs themselves live in `modules/structs`.

## Responsibilities

- `services/forms/` — one struct per HTML form, with binding and validation rules. Used only by
  the web layer.
- `services/convert/` — turns model structs into the `modules/structs` API types. Used by the API
  layer, and anywhere else an API-shaped payload is needed (webhooks, notifications).

The API layer does **not** use `services/forms`: it binds `modules/structs` option types directly.

## Key files

| Path | What it holds |
| --- | --- |
| `services/forms/repo_form.go` | the repo/issue/PR forms, e.g. `CreateIssueForm` |
| `services/forms/user_form.go`, `auth_form.go`, `org.go`, `admin.go` | forms for the other areas |
| `services/convert/convert.go` | the general converters (`ToEmail`, `ToBranch`, `ToTag`, ...) |
| `services/convert/issue.go` | `ToAPIIssue`, `ToAPIIssueList`, `ToLabel`, `ToTrackedTime` |
| `services/convert/pull.go`, `pull_review.go`, `repository.go`, `user.go` | one file per domain |
| `modules/web/middleware/binding.go` | `Validate`, which every form's `Validate` delegates to |
| `modules/validation/binding.go` | `AddBindingRules` — registers the custom binding rules |

## Conventions & invariants

**Forms**

- A form is a plain struct with `binding:"..."` tags: `binding:"Required;MaxSize(255)"`. Field
  names map from the request through `util.ToSnakeCase` (`modules/web/middleware/binding.go`), so
  `CloseReasonText` binds `close_reason_text`. A plural acronym does not survive it —
  `AssigneeIDs` would become `assignee_i_ds` — which is why that field carries `form:"assignee_ids"`.
- A choice among named options arrives as a string, and a method on the form maps it to the
  model's integer enum, falling back to an unknown value that is then rejected:
  `SubmitReviewForm.ReviewType` (`services/forms/repo_form.go`). When other code needs the same
  names, the model owns them and the form calls it: `CreateCommentForm.CloseReasonOptions` uses
  `AsCloseReason` (`models/issues/issue_close_reason.go`). The stored numbers never appear in the
  request.
- A form that needs the standard error handling implements
  `Validate(req *http.Request, errs binding.Errors) binding.Errors` and returns
  `middleware.Validate(errs, ctx.Data, f, ctx.Locale)` — copy the body from a neighbouring form
  verbatim; it is boilerplate, not a decision.
- Validation messages are locale keys, resolved through `ctx.Locale` by `middleware.Validate`.
  `validation.AddBindingRules` is called once from `registerWebRoutes` and registers the custom
  rules; a new rule goes there, not into the form.
- The route wires the form with `web.Bind(forms.XxxForm{})` and the handler reads it back with
  `web.GetForm(ctx)`, type-asserted to the form type.

**Converters**

- Naming is `To<Thing>` for the plain conversion and `To<Thing>List` for a slice. Where both an
  internal and an API-facing variant exist, the API one is `ToAPI<Thing>`.
- A converter whose output depends on permissions or needs more queries takes
  `(ctx context.Context, doer *user_model.User, model *T)` — `ToAPIIssue` is the pattern. Pass the
  real doer, never `nil`, unless the neighbouring call sites do. A pure value conversion takes just
  the model (`ToEmail`, `ToTag`).
- Converters return `modules/structs` types. They must not query for anything a caller could have
  loaded; a converter that fans out queries per item turns a list endpoint into an N+1.

## Recipes

**Add a field to a form.** Add the struct field with its `binding` tag, add the locale key for any
new validation message to `options/locale/locale_en-US.json`, then read it in the handler from
`web.GetForm(ctx)`. Existing forms have tests in `services/forms/repo_form_test.go` — extend them
rather than adding an integration test.

**Expose a new model field over the API.** Add the field to the `modules/structs` type, set it in
the matching `services/convert` function, and check whether the swagger response registration in
`routers/api/v1/swagger/` already covers the struct (it usually does). Then
`make generate-swagger`.

## Gotchas

- Two binding mechanisms with similar names: `web.Bind` (web, `services/forms` types) and the
  local `bind` helper in `routers/api/v1/api.go` (API, `modules/structs` types). Using the web one
  on an API route silently binds nothing useful.
- `web.GetForm` returns `any`. A wrong type assertion panics at request time, not compile time —
  match the type in `web.Bind` exactly.
- Adding a field to a `modules/structs` type without setting it in the converter ships a field
  that is always the zero value, and swagger will still document it as present.
- A binding error message starts with the field's label: the `locale:"…"` tag, or else the key
  `form.<FieldName>` (`modules/web/middleware/binding.go` → `Validate`). A field with a rule but no
  tag or key shows the raw key to the person; `CreateCommentForm.CloseReasonText` sets the tag.

## Related

- `routers-web.md` — where `web.Bind` is attached
- `routers-api-v1.md` — the swagger registration these DTOs need
