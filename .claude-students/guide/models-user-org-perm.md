---
source: docs/models-user-org-perm.md
source-hash: 51b4b7b78979621d
verified-at: 2629e98ef9
---

<!-- Derived from docs/models-user-org-perm.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Users, organisations and permissions, in plain English

**In one sentence:** organisations are stored as users, permissions are an ordered scale rather than
a yes/no, and both facts cause bugs if nobody tells you.
**Come here when:** you are querying users, organisations or teams, or deciding whether somebody is
allowed to do something.

## What this is

Identity and access. Who exists, which organisations and teams they belong to, what they may do, how
they authenticate, and their SSH and GPG keys.

## Why it exists

**Why is an organisation a user?** Because they need almost all the same things. An organisation has
a name in the same namespace as users — you cannot have a user and an organisation both called
`gitea` — owns repositories, has an avatar, and appears in URLs the same way. Storing them
separately would mean duplicating all of that and then constantly checking both tables.

So there is one table with a field saying which kind of row it is. Convenient, and it means **every
query over users includes organisations unless you exclude them**.

**Why is permission a scale rather than a boolean?** Because "can write" and "is an admin" are not
independent — an admin can do everything a writer can. Modelling them as separate flags means
checking several every time and getting it wrong occasionally. An ordered scale lets every check be
a single comparison.

**Why is permission per feature?** Because real teams need it. A documentation team should be able
to file issues without pushing code. A single per-repository permission cannot express that.

## Words you'll meet

- **user type** — the field distinguishing a person from an organisation.
- **team** — a group within an organisation, holding the permissions.
- **access mode** — the permission scale: none, read, write, admin, owner.
- **unit** — one repository feature (code, issues, wiki, actions) — see `models-repo-and-git.md`.
- **computed permission** — the worked-out answer for one user and one repository.
- **token scope** — how much an API token may do, independent of what its owner may do.
- **hash algorithm** — how passwords are stored so they cannot be read back.
- **2FA (two-factor authentication)** — a second proof at sign-in besides the password: a code from an
  authenticator app (TOTP) or a security key (WebAuthn).
- **fail-closed** — when in doubt, deny. A new piece of code gets the safe answer without having to
  remember to ask for it.
- **SQL twin** — the same rule written as a database condition, for lists and searches that never
  compute a permission one row at a time.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/user/user.go` | The user row — people **and** organisations. |
| `models/organization/org.go`, `team.go` | The organisation view, and teams. |
| `models/organization/team_unit.go` | A team's permission per repository feature. |
| `models/perm/access_mode.go` | The permission scale. |
| `models/perm/access/repo_permission.go` | The computed permission for one user and one repository. `GetDoerRepoPermission` is the one request code uses. |
| `models/organization/org_two_factor.go` | An organisation's "require 2FA" rule: who it blocks, and the opt-out for clean-up code. |
| `models/user/two_factor_policy.go` | The same rule as SQL, so repository lists in `models/repo/repo_list.go` agree with it. |
| `models/auth/access_token.go`, `access_token_scope.go` | API tokens and their scopes. |
| `models/asymkey/ssh_key.go`, `gpg_key.go` | SSH and GPG keys. |

## The rules, and why

**An organisation is a user row.** A field records which it is. Any query over users that forgets to
filter on it silently includes organisations — which is how "how many users do we have" quietly
becomes wrong.

**The lowercase name is the unique key**, so names are case-insensitive for uniqueness while keeping
their display casing — the same arrangement as repositories.

**The access mode is ordered, so compare with `>=`, never `==`.** Write
`perm.AccessMode >= AccessModeWrite`. An equality check for "write" **fails for an owner**, who
obviously should be allowed. This is a genuine security bug that reads as a reasonable line of code,
and it fails in the safe direction only by luck.

**The computed permission is built for you.** It is produced by the middleware and reached as
`ctx.Repo.Permission` (`services-context.md`). Do not construct one by hand in a handler — the
computation considers ownership, team membership, visibility and per-unit rules, and a hand-rolled
version will miss one.

**Permission is per unit, not per repository.** A team can have write on issues and read on code. A
check that ignores which feature it is asking about is wrong even when it happens to give the right
answer.

**Passwords are hashed and never compared directly.** The algorithm is recorded per user, so old
accounts can be upgraded. There is also a flag forcing a password change after an
administrator-created account.

**An organisation can require 2FA, and then a member without it counts as a stranger.** Members and
outside collaborators who have neither an authenticator app nor a security key lose everything that
membership gave them: private repositories, team permissions, member-only pages, repository creation.
The rule is applied fail-closed in the shared places — the permission calculation, organisation
visibility, organisation unit permissions, the "can create a repository here" check and the
repository list conditions — so a new page or API endpoint inherits it. Site admins, and an
organisation pushing through its own deploy key, are never blocked. Nothing is deleted, which is why
setting up 2FA gives everything back on the next request.

**API tokens carry their own scopes.** A token is not simply "acting as its owner" — it may be
limited to a subset, which is what the API's scope guard enforces (`routers-api-v1.md`).

## How to actually do it

**Check whether someone may do something to a repository.** On a request, use
`ctx.Repo.Permission`. Outside one, build it through `models/perm/access`. Then compare with `>=`,
for the specific unit.

**Add a per-user setting.** There is a key/value settings table for exactly this. Prefer it to a new
column on the user row, which is wide and read constantly.

**Remove something because a user lost access** — unassign them, unwatch, trim an allowlist. Check
access inside `organization.IgnoreTwoFactorPolicy(ctx)`, as `services/repository/collaboration.go`
does. Without it, a user who is only blocked until they set up 2FA would lose those rows for good.

**List an organisation's members.** Go through the organisation package rather than joining the user
table yourself — membership is not a column on the user.

## Traps, and what they look like

**A count is too high.** Organisations were included because the query did not filter on user type.

**An owner is denied something a writer may do.** Somebody compared the access mode with `==`
instead of `>=`.

**Your permission check is right for code and wrong for issues.** It ignored the unit.

**Two imports look like the same package.** The permission *scale* and the permission *computation*
are different packages, usually aliased to something like `perm_model` and `access_model`. Match
whatever the file already uses.

**You look for `GetUserRepoPermission` and it is not there.** This fork splits it in two: request
code calls `GetDoerRepoPermission`, which also understands Actions' temporary users, and
`GetIndividualUserRepoPermission` takes any explicit user.

**You delete a user row directly and things break afterwards.** Deleting a user touches many tables.
Go through the service (`services-user-org-auth.md`); the model layer does not do the cleanup.

## Where to go next

- `docs/models-user-org-perm.md` (in this folder) — the reference page this was written from
- `services-user-org-auth.md` — accounts, teams and signing in
- `services-context.md` — where the computed permission is attached to a request
