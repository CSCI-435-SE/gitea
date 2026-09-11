---
scope: modules/util, modules/optional, modules/container, modules/timeutil, modules/json
verified-at: c0092050a4
---

# modules/util and friends — errors and small generic helpers

**Read when:** returning an error from a model or service, or reaching for a generic helper.
**Not here:** database errors -> `models-db.md`; HTTP error responses -> `services-context.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `modules/util` | the sentinel error values and their constructors, plus assorted small helpers |
| `modules/optional` | `Option[T]` — an explicit "value or not set" |
| `modules/container` | `Set[T]` and filter helpers |
| `modules/timeutil` | `TimeStamp`, the unix-seconds type every model stores |
| `modules/json` | the single JSON entry point the codebase uses instead of `encoding/json` directly |

## Key files

| Path | What it holds |
| --- | --- |
| `modules/util/error.go` | `ErrInvalidArgument`, `ErrPermissionDenied`, `ErrNotExist`, `ErrAlreadyExist`, and `NewNotExistErrorf`, `NewInvalidArgumentErrorf`, `NewPermissionDeniedErrorf`, `NewAlreadyExistErrorf`, `ErrorWrap`, `ErrorWrapTranslatable` |
| `modules/optional/option.go` | `None`, `Some`, `FromPtr`, `FromNonDefault`, `Has`, `Value`, `ValueOrDefault` |
| `modules/container/set.go`, `filter.go` | `Set[T]` and slice filtering |
| `modules/timeutil/` | `TimeStamp` and its conversions |

## Conventions & invariants

- **The error contract that ties the layers together.** A typed model error comes in three parts:
  the struct (`ErrIssueNotExist`), an `IsErrIssueNotExist(err) bool`, and an
  `Unwrap() error` returning the matching `modules/util` sentinel. Because of the `Unwrap`, a
  router can write `errors.Is(err, util.ErrNotExist)` without importing every model's error type —
  and `ctx.NotFoundOrServerError` (`services-context.md`) relies on exactly this.
- Use the constructors (`util.NewNotExistErrorf(...)`) for one-off errors; define a typed error
  only when callers need to inspect its fields.
- `ErrorWrapTranslatable` exists for errors whose message reaches the user, so it can be localised.
  Plain `fmt.Errorf` messages are for logs, not for users.
- `optional.Option[T]` is for parameters where "not provided" differs from "provided as zero" —
  common on API edit endpoints, where every field is optional (`routers-api-v1.md`). Prefer it to
  a `*T` in new code.
- Models store times as `timeutil.TimeStamp` (unix seconds), with `xorm:"created"` / `xorm:"updated"`
  tags maintaining `CreatedUnix` / `UpdatedUnix` automatically. Do not set them by hand.
- Use `modules/json` rather than `encoding/json` so the implementation stays swappable.

## Recipes

**Return "not found" from a model.** `util.NewNotExistErrorf("thing [id: %d] does not exist", id)`,
or the three-part typed error when callers must read its fields.

**Make a new typed error work across layers.** Give it `Unwrap() error` returning the right
`util.Err*` sentinel. Without that, `errors.Is` fails and the router turns a 404 into a 500.

**Accept an optional API field.** `optional.Option[string]` in the option struct, then
`opt.Has()` before applying it.

## Gotchas

- Forgetting `Unwrap` is the classic bug here: the error reads correctly in logs and produces the
  wrong HTTP status.
- `optional.Option[T]` and a plain pointer both compile; mixing them in one struct makes "unset"
  ambiguous for the next reader.
- `timeutil.TimeStamp` is seconds, not milliseconds, and not `time.Time`. Converting through the
  wrong unit produces dates in 1970 or the far future.

## Related

- `models-db.md` — database-level errors and transactions
- `services-context.md` — how these sentinels become HTTP status codes
