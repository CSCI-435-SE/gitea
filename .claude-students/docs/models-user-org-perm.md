---
scope: models/user, models/organization, models/perm, models/auth, models/asymkey
verified-at: c0092050a4
---

# models/user, organization, perm — identities and who can do what

**Read when:** querying users, orgs, teams, or deciding whether someone may do something.
**Read also:** `services-user-org-auth.md` for the logic on top.

## Responsibilities

| Package | Owns |
| --- | --- |
| `models/user` | `User` (people *and* organisations), email addresses, settings, redirects, follows, blocks |
| `models/organization` | `Organization`, `Team`, team membership, `team_unit`, `team_repo`, invites |
| `models/perm` | the `AccessMode` enum; `models/perm/access` computes effective repo permissions |
| `models/auth` | login sources, access tokens and their scopes, OAuth2, WebAuthn, TOTP |
| `models/asymkey` | SSH keys, deploy keys, principals, GPG keys and commit verification |

## Key files

| Path | What it holds |
| --- | --- |
| `models/user/user.go` | `User`, `UserType`, the name/email uniqueness rules |
| `models/organization/org.go`, `team.go` | orgs and teams |
| `models/organization/team_unit.go` | a team's access mode per repo unit |
| `models/perm/access_mode.go` | `AccessModeNone`/`Read`/`Write`/`Admin`/`Owner` |
| `models/perm/access/repo_permission.go` | `Permission` and its `IsOwner`, unit-level accessors |
| `models/auth/access_token.go`, `access_token_scope.go` | API tokens and their scopes |
| `models/asymkey/ssh_key.go`, `gpg_key.go` | key storage and verification |

## Conventions & invariants

- **An organisation is a `User` row.** `User.Type` (`UserType`) distinguishes a person from an
  organisation, and `models/organization.Organization` is a view over the same table. Any query
  over users that forgets `Type` will include organisations.
- `LowerName` is the unique key, not `Name` — usernames are case-insensitive for uniqueness while
  preserving display casing, exactly as repositories do (`models-repo-and-git.md`).
- **`AccessMode` is ordered**, so permission checks are comparisons:
  `perm.AccessMode >= AccessModeWrite`. Never compare for equality — an owner would fail a
  `== AccessModeWrite` test.
- `Permission` in `models/perm/access` is the *computed* result for one user and one repository,
  with private fields per unit. It is built by the assignment middleware and reached as
  `ctx.Repo.Permission` (`services-context.md`); do not construct one by hand in a handler.
- Permission is per *unit*, not per repo: a team may have write on issues and read on code. A check
  that ignores the unit is wrong (`models-repo-and-git.md`).
- Passwords are hashed with the algorithm named in `PasswdHashAlgo` (default `argon2`), never
  compared directly. `MustChangePassword` gates login after admin-created accounts.
- API tokens carry scopes (`models/auth`), which is what `tokenRequiresScopes` enforces on API
  routes (`routers-api-v1.md`).

## Recipes

**Check whether a user may do something to a repo.** Use `ctx.Repo.Permission` on a request, or
build it through `models/perm/access` outside one — then compare access modes with `>=`, for the
specific unit.

**Add a per-user setting.** `models/user/setting.go` holds key/value user settings; prefer that to
a new column on `User`, which is a wide and hot table.

**Find an org's members.** Go through `models/organization` (`org_user.go`, `team_user.go`) rather
than joining the user table directly.

## Gotchas

- Because orgs are users, "user count" and "member count" queries are a classic source of wrong
  numbers.
- `models/perm` (the enum) and `models/perm/access` (the computation) are different packages that
  both get imported as something like `perm_model` and `access_model`. Match the file's aliases.
- Deleting a user is not a simple delete — it touches many tables; go through
  `services-user-org-auth.md`.

## Related

- `services-user-org-auth.md` — user/org lifecycle and authentication
- `services-context.md` — where `Permission` is attached to a request
