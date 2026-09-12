---
source: docs/models-user-org-perm.md
source-hash: 86c1ed1979ba9ae5
verified-at: c0092050a4
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

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/user/user.go` | The user row — people **and** organisations. |
| `models/organization/org.go`, `team.go` | The organisation view, and teams. |
| `models/organization/team_unit.go` | A team's permission per repository feature. |
| `models/perm/access_mode.go` | The permission scale. |
| `models/perm/access/repo_permission.go` | The computed permission for one user and one repository. |
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

**API tokens carry their own scopes.** A token is not simply "acting as its owner" — it may be
limited to a subset, which is what the API's scope guard enforces (`routers-api-v1.md`).

## How to actually do it

**Check whether someone may do something to a repository.** On a request, use
`ctx.Repo.Permission`. Outside one, build it through `models/perm/access`. Then compare with `>=`,
for the specific unit.

**Add a per-user setting.** There is a key/value settings table for exactly this. Prefer it to a new
column on the user row, which is wide and read constantly.

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

**You delete a user row directly and things break afterwards.** Deleting a user touches many tables.
Go through the service (`services-user-org-auth.md`); the model layer does not do the cleanup.

## Where to go next

- `docs/models-user-org-perm.md` (in this folder) — the reference page this was written from
- `services-user-org-auth.md` — accounts, teams and signing in
- `services-context.md` — where the computed permission is attached to a request
