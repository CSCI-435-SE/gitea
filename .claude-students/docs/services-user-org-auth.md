---
scope: services/user, services/org, services/auth, services/oauth2_provider, services/asymkey, services/externalaccount
verified-at: c0092050a4
---

# services/user, org, auth — accounts, teams and sign-in

**Read when:** changing account creation/deletion, org and team management, or any authentication
method.
**Not here:** the rows -> `models-user-org-perm.md`; route guards -> `routers-web.md` and `routers-api-v1.md`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `services/user` | create, update, delete, rename, email management, blocking |
| `services/org` | organisation and team lifecycle, membership |
| `services/auth` | the authentication methods and login-source management |
| `services/auth/source/{db,ldap,oauth2,pam,smtp,sspi}` | one subpackage per login source |
| `services/oauth2_provider` | Gitea acting as an OAuth2 *provider* |
| `services/asymkey` | adding and verifying SSH/GPG keys, commit signing |
| `services/externalaccount` | linking external identities to local accounts |

## Key files

| Path | What it holds |
| --- | --- |
| `services/user/user.go`, `delete.go`, `update.go`, `email.go`, `block.go` | the account operations |
| `services/org/org.go`, `team.go`, `team_invite.go` | org and team operations |
| `services/auth/auth.go`, `interface.go` | the authentication method chain and its interface |
| `services/auth/basic.go`, `session.go`, `oauth2.go`, `reverseproxy.go`, `httpsign.go` | the individual methods |
| `services/auth/source/ldap/`, `oauth2/`, `smtp/` | the external sources |
| `services/auth/group.go`, `services/auth/source/source_group_sync.go` | mapping external groups onto Gitea teams |

## Conventions & invariants

- **Authentication is a chain of methods**, each deciding whether it can identify the request
  (session, basic auth, API token, OAuth2, reverse proxy). A new method is added to that chain,
  not hard-coded into a router.
- A login source is a row plus a source-type implementation under `services/auth/source/`. Adding
  a provider means implementing that interface, not special-casing the login handler.
- **Deleting a user fans out across most of the schema** — repos, issues, comments, orgs, keys,
  tokens. Always go through `services/user/delete.go`; never delete the row directly.
- Renaming a user, like renaming a repo, must leave a redirect so old URLs resolve
  (`models-repo-and-git.md`).
- Team permissions are stored per unit (`team_unit`), so granting access means writing the unit
  rows, not a single flag (`models-user-org-perm.md`).
- Email addresses are their own table with a primary flag and activation state; changing
  `User.Email` alone is not enough.
- These services notify like every other (`services-issue.md`): mutate, then `notify_service.*`
  after the transaction.

## Recipes

**Add a login source type.** Implement the source interface under
`services/auth/source/<name>/`, register it, and add its settings form in `services/forms`
(`services-forms-and-convert.md`) plus the admin UI templates.

**Create a user programmatically.** `services/user` — it applies password policy, reserved-name
checks and the initial email row, none of which the model layer does.

**Sync teams from an external provider.** `services/auth/source/source_group_sync.go`, driven from
`services/auth/group.go`.

## Gotchas

- Reserved usernames overlap with route names (`admin`, `api`, `explore`, ...), so user creation
  validates against a reserved list. Adding a new top-level route can collide with existing
  usernames.
- An org is a `User` row, so "delete user" and "delete org" share machinery — check which one a
  function is actually for.
- Authentication ordering matters: a method that claims a request stops the chain.

## Related

- `models-user-org-perm.md` — the rows and the access-mode rules
- `services-context.md` — how the authenticated user reaches `ctx.Doer`
