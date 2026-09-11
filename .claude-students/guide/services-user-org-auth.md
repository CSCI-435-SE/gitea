---
source: docs/services-user-org-auth.md
source-hash: 9bf30ebf079e43ba
verified-at: c0092050a4
---

<!-- Derived from docs/services-user-org-auth.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Accounts, teams and signing in, in plain English

**In one sentence:** creating and deleting accounts, managing organisations, and the chain of
methods that works out who is making a request.
**Come here when:** you are changing account handling, teams, or any way of authenticating.

## What this is

The logic on top of the identity tables: account lifecycle, organisations and teams, and
authentication — including the external sources like LDAP and OAuth2, and Gitea acting as an OAuth2
provider itself.

## Why it exists

**Authentication has to handle many kinds of request at once.** A browser has a session cookie. A
`git push` over HTTP sends a username and password. A script sends an API token. A company
deployment might have a proxy in front that has already authenticated the user.

Each of those identifies the same person differently. Rather than every handler checking all of
them, there is a **chain**: each method is asked in turn whether it recognises this request, and the
first that does wins. A handler just reads who the user is.

**Deleting a user is where this gets serious.** A user is referenced from almost every table in the
schema — their repositories, issues, comments, reviews, organisation memberships, keys, tokens,
notifications. Deleting the row alone leaves every one of those pointing at nothing. So deletion is
a long, careful operation in one place, and never something you do directly.

## Words you'll meet

- **authentication** — working out *who* is making a request. (Distinct from authorisation, which is
  what they may do.)
- **method chain** — the ordered list of ways to identify a request.
- **login source** — an external system that can vouch for users: LDAP, OAuth2, SMTP.
- **OAuth2 provider** — Gitea letting other applications sign users in with a Gitea account.
- **group sync** — mapping groups from an external system onto Gitea teams.
- **reserved name** — a username that cannot be taken because it would collide with a URL.
- **activation** — confirming an email address actually belongs to the person.

## What's in these files

| Where | What it is for |
| --- | --- |
| `services/user/user.go`, `delete.go`, `update.go`, `email.go`, `block.go` | Account operations. |
| `services/org/org.go`, `team.go`, `team_invite.go` | Organisations, teams and invitations. |
| `services/auth/auth.go`, `interface.go` | The authentication chain and the interface a method implements. |
| `services/auth/basic.go`, `session.go`, `oauth2.go`, `reverseproxy.go`, `httpsign.go` | The individual methods. |
| `services/auth/source/ldap/`, `oauth2/`, `smtp/` | The external sources. |
| `services/auth/group.go`, `services/auth/source/source_group_sync.go` | Mapping external groups onto teams. |

## The rules, and why

**Authentication is a chain, and you extend the chain.** A new way of identifying requests is a new
method added to it — never a special case inside a router. A router that does its own authentication
is a router that will disagree with the rest of Gitea about who you are.

**Order matters: the first method that claims a request stops the chain.** So adding a method is not
neutral; where it sits decides what it can shadow.

**A login source is a row plus an implementation.** Adding a provider means implementing the source
interface, not adding a branch to the login handler.

**Never delete a user row directly.** Go through the deletion service. It is long because it has to
be — it fans out across most of the schema. Deleting the row yourself leaves broken references
everywhere, and you will not notice until something unrelated fails much later.

**Renaming a user leaves a redirect**, just as renaming a repository does, so existing links keep
working.

**Team permissions are per repository feature**, so granting access means writing those rows, not
setting one flag (`models-user-org-perm.md`).

**Email addresses are a separate table** with a primary flag and an activation state. Changing the
email field on the user row alone is not enough and leaves the two disagreeing.

**Mutate, then notify — after the transaction**, the same rule as everywhere else
(`services-issue.md`).

## How to actually do it

**Add a login source type.** Implement the source interface in its own folder, register it, and add
its settings form (`services-forms-and-convert.md`) plus the admin UI templates. Three places, all
required.

**Create a user from code.** Go through the service. It applies the password policy, the
reserved-name check and the initial email row — none of which the model layer does, so a
model-level insert produces an account that is subtly broken.

**Sync teams from an external provider.** The group-sync code, driven from the auth package.

## Traps, and what they look like

**You add a new top-level route and some users cannot reach their profile.** Usernames and top-level
URLs share a namespace, which is why there is a reserved-name list. Adding a route called
`/dashboard` collides with any existing user named `dashboard` — and the reserved list only stops
*new* registrations, so existing accounts are already out there.

**Your new authentication method never runs.** Something earlier in the chain already claimed the
request.

**A function named for users behaves oddly on organisations, or the reverse.** Organisations are
user rows, so the two share machinery. Check which one a function is actually written for.

**You created a user and login fails, or the email never activates.** The account was created
through the model layer, skipping the policy checks and the initial email row.

**Something references a deleted user and crashes.** The row was deleted directly instead of through
the deletion service.

## Where to go next

- `docs/services-user-org-auth.md` (in this folder) — the reference page this was written from
- `models-user-org-perm.md` — the rows, and the access-mode rules
- `services-context.md` — how the authenticated user arrives as `ctx.Doer`
