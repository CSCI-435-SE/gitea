---
source: docs/modules-util-and-generics.md
source-hash: e5fcecea66f4202a
verified-at: c0092050a4
---

<!-- Derived from docs/modules-util-and-generics.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Errors and small helpers, in plain English

**In one sentence:** a handful of tiny packages, one of which carries the contract that lets a
missing database row become a 404 rather than a 500.
**Come here when:** you are returning an error from a model or service, or reaching for a small
generic helper.

## What this is

Several small packages that everything else uses:

- `modules/util` — shared error values and the helpers for making them. The important one.
- `modules/optional` — a type meaning "a value, or deliberately nothing".
- `modules/container` — sets and slice filtering.
- `modules/timeutil` — the timestamp type every stored struct uses.
- `modules/json` — the single JSON entry point, used instead of the standard library's.

## Why it exists

The error part is the interesting one, and it solves a real layering problem.

A router needs to know whether an error means "not found", so it can return 404 instead of 500. But
errors come from deep inside `models/`, where there are dozens of error types — one per thing that
can be missing.

The router could import every one of them and check each. That would be absurd, and it would mean
`routers/` depending on every model package.

Instead, each specific error **unwraps to a shared sentinel**. `ErrIssueNotExist` unwraps to
`util.ErrNotExist`. So does `ErrRepoNotExist`, and every other one. The router asks a single
question — "is this a not-found error?" — and gets the right answer without knowing what was
missing.

That is why the three-part error pattern exists across `models/`, and why the third part is the one
that matters.

## Words you'll meet

- **sentinel error** — a shared error value used as a marker.
- **unwrap** — the method letting Go see a "more general" error behind a specific one.
- **`errors.Is`** — the check that follows unwrapping to ask whether an error is a particular kind.
- **typed error** — an error struct carrying extra fields, like which id was missing.
- **translatable error** — one whose message is shown to a user, so it must be localisable.
- **optional value** — one that may be deliberately absent, distinct from being zero.
- **zero value** — Go's default: `0`, `""`, `false`. Distinct from "not provided".
- **unix timestamp** — a time as a count of seconds.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/util/error.go` | The shared sentinels, the constructors for one-off errors, and the wrapper for user-facing messages. |
| `modules/optional/option.go` | The optional type and its constructors and accessors. |
| `modules/container/set.go`, `filter.go` | Sets and slice filtering. |
| `modules/timeutil/` | The timestamp type stored structs use. |

## The rules, and why

**The three-part error contract.** A typed model error has:

1. the struct, carrying whatever detail is useful,
2. an `IsErrXxx(err) bool` helper,
3. **an `Unwrap()` returning the matching shared sentinel.**

Part three is the one people forget, and it is the one that makes the other layers work. The
response helper described in `services-context.md` relies on exactly this to choose between a 404
and a 500.

**Use the constructors for one-off errors.** Define a typed error only when a caller genuinely needs
to read its fields. A struct nobody inspects is ceremony.

**Errors whose text reaches a user go through the translatable wrapper.** Ordinary error messages
are for logs, and logs are in English. Anything a user reads must be translatable, like every other
string (`i18n.md`).

**The optional type is for "not provided" versus "provided as zero".** This matters most on API edit
endpoints, where every field is optional (`routers-api-v1.md`). Without it you cannot tell "leave
the description alone" from "set the description to empty" — both arrive as `""`. Prefer it to a
pointer in new code.

**Times are stored as a unix-seconds type, and the timestamps maintain themselves.** Struct tags
keep the created and updated fields current automatically. Setting them by hand fights the database
layer.

**Use the project's JSON package, not the standard library's.** This keeps the implementation
swappable.

## How to actually do it

**Return "not found" from a model.** Use the constructor for a one-off. Use the three-part typed
error when callers must read its fields.

**Make a new typed error work across layers.** Give it an `Unwrap` returning the right sentinel.
Without that the error is correct and unrecognisable.

**Accept an optional field on an API endpoint.** Use the optional type in the option struct, and
check whether it has a value before applying it.

## Traps, and what they look like

**A missing thing produces a 500 instead of a 404.** Your error has no `Unwrap`, so the layer above
cannot tell it is a not-found error. The message in the log reads perfectly, which is what makes
this hard to spot: the error is *correct*, just unrecognisable. This is the classic bug in this
area.

**An API edit endpoint blanks a field the caller did not send.** A plain type was used where the
optional type was needed, so "absent" and "empty" arrived identically.

**One struct uses both pointers and the optional type for optional fields.** Both compile, and the
next reader cannot tell what "unset" means. Pick one per struct.

**A date shows as 1970, or as the far future.** Something converted through the wrong unit. The
stored type is **seconds**, not milliseconds, and not Go's own time type.

**A user sees an English error message in a non-English interface.** The error was built with a
plain formatter rather than the translatable wrapper.

## Where to go next

- `docs/modules-util-and-generics.md` (in this folder) — the reference page this was written from
- `services-context.md` — how these sentinels become HTTP status codes
- `models-db.md` — database-level errors and transactions
