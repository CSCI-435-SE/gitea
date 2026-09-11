---
source: docs/services-forms-and-convert.md
source-hash: d6cd1f5878804697
verified-at: c0092050a4
---

<!-- Derived from docs/services-forms-and-convert.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Forms in, JSON out, in plain English

**In one sentence:** two small packages at the edges of Gitea — one turns a submitted form into a
checked Go struct, the other turns database structs into the JSON the API returns.
**Come here when:** you are adding a field to a form on a page, or exposing a new field over the
API.

## What this is

Two packages that sit on opposite sides of a handler.

`services/forms` handles data coming **in** from an HTML form: one struct per form, with the
validation rules attached as tags.

`services/convert` handles data going **out**: functions that take a database struct and return the
public JSON shape.

The API does **not** use `services/forms`. It binds its own option types from `modules/structs`
directly. Forms are a web-pages-only mechanism.

## Why it exists

**For forms:** anything arriving from a browser is untrusted. It may be missing, too long, the
wrong type, or hostile. Checking all of that inside each handler would mean every handler
re-implementing the same validation slightly differently — and the one that forgets is the security
bug. So the rules live on the struct as tags, and the checking happens before your handler runs.

**For converters:** the database struct and the public JSON shape must be allowed to differ. The
database row has internal columns, foreign keys, and fields that depend on who is asking. If the API
returned database rows directly, then every column rename would break somebody's script, and you
would leak fields nobody should see. The converter is the deliberate seam between "how we store it"
and "what we promise".

## Words you'll meet

- **binding** — filling a Go struct from an HTTP request automatically.
- **validation** — checking those values are acceptable before the handler runs.
- **tag** — the backtick-quoted annotation after a struct field that carries these rules.
- **DTO** — a struct whose only job is to be turned into JSON.
- **converter** — a function turning a database struct into a DTO.
- **doer** — the user performing the request; output often depends on who they are.
- **N+1** — the performance trap where fetching a list of *n* things runs *n* extra queries, one
  per item.
- **zero value** — Go's default for an unset field: `0`, `""`, `false`.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/forms/repo_form.go` | The repository, issue and pull-request forms. |
| `services/forms/user_form.go`, `auth_form.go`, `org.go`, `admin.go` | Forms for the other areas. |
| `services/convert/convert.go` | General converters. |
| `services/convert/issue.go` | The issue converters — the pattern worth copying. |
| `services/convert/pull.go`, `pull_review.go`, `repository.go`, `user.go` | One file per domain. |
| `modules/web/middleware/binding.go` | The shared validation every form calls into. |
| `modules/validation/binding.go` | Where custom validation rules are registered. |

## The rules, and why

### Forms

**A form is a plain struct with tags.** `binding:"Required;MaxSize(255)"` says the field must be
present and at most 255 characters. Fields map from the request by name automatically; add
`form:"assignee_ids"` only when the incoming name differs from the Go field name.

**The `Validate` method is boilerplate — copy it.** A form that wants standard error handling
implements `Validate(req *http.Request, errs binding.Errors) binding.Errors` and returns
`middleware.Validate(errs, ctx.Data, f, ctx.Locale)`. Copy that body verbatim from a neighbouring
form. It is not a decision you need to make.

**Validation messages are locale keys, not English.** They are resolved through `ctx.Locale`, which
is how the error shows up in the user's language. Custom rules are registered once by
`validation.AddBindingRules` from `registerWebRoutes` — a new rule goes there, not inside the form.

**The route names the form; the handler reads it back.** The route says
`web.Bind(forms.XxxForm{})`, the handler calls `web.GetForm(ctx)` and asserts it to that type.

### Converters

**Names follow a pattern.** `ToThing` for one, `ToThingList` for a slice, and `ToAPIThing` when both
an internal and an API-facing variant exist.

**The signature tells you what the conversion needs.** A converter whose output depends on
permissions, or that needs extra queries, takes `(ctx, doer, model)` — `ToAPIIssue` is the example.
Pass the real doer, never `nil`, unless the surrounding call sites do. A pure value conversion takes
only the model, like `ToEmail` or `ToTag`.

**A converter must not query for things the caller could have loaded.** This is the N+1 rule. A
converter that runs a query per item turns a list endpoint returning fifty issues into fifty-one
database round trips. The caller should load what is needed in one go and hand it over.

## How to actually do it

**Add a field to a form.**

1. Add the struct field with its `binding` tag.
2. If it introduces a new validation message, add the locale key to
   `options/locale/locale_en-US.json`.
3. Read it in the handler from `web.GetForm(ctx)`.
4. Extend the existing tests in `services/forms/repo_form_test.go` rather than writing an
   integration test — the validation logic is testable on its own, which is cheaper and clearer.

**Expose a new model field over the API.**

1. Add the field to the type in `modules/structs`.
2. **Set it** in the matching `services/convert` function. This is the step people forget.
3. Check whether the swagger registration under `routers/api/v1/swagger/` already covers that
   struct — it usually does.
4. Run `make generate-swagger` and commit the result.

## Traps, and what they look like

**Your new API field is always empty — `0`, or `""`, or `false`.** You added it to the struct but
never set it in the converter. The field ships as its zero value, and the API documentation still
claims it is there, so consumers see a field that is permanently blank. This is the most common
mistake in this area and nothing warns you about it.

**Your form binds as empty on an API route.** You used `web.Bind`, which is the web mechanism over
`services/forms` types. API routes use the local `bind` helper with `modules/structs` types. The two
look alike and are not interchangeable.

**Your handler panics the moment someone submits the form.** `web.GetForm` returns an untyped value,
so a wrong type assertion is not caught at compile time — it explodes at request time. The type in
`web.GetForm` must match the type in `web.Bind` exactly.

**A list endpoint is mysteriously slow.** A converter in the loop is querying per item. Load the
data once before converting.

## Where to go next

- `docs/services-forms-and-convert.md` (in this folder) — the reference page this was written from
- `routers-web.md` — where `web.Bind` is attached to a route
- `routers-api-v1.md` — registering these types so they appear in the API spec
