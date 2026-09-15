---
source: docs/models-repo-and-git.md
source-hash: b780e9936623568c
verified-at: c0092050a4
---

<!-- Derived from docs/models-repo-and-git.md. Do not edit by hand: fix the reference doc and
     regenerate this file. See MAINTENANCE.md rules 16-22. -->

# Repositories in the database, in plain English

**In one sentence:** the repository row and everything attached to it — plus a second set of tables
that are only a *cache* of what git on disk already knows.
**Come here when:** you are querying repository data, checking whether a repository has a feature
turned on, or touching branch and commit-status records.

## What this is

Two related groups of tables.

`models/repo` is the `Repository` row and its satellites: collaborators, watchers, stars, forks,
releases, attachments, topics, mirrors, redirects, transfers, uploads, the wiki, licences and
language statistics.

`models/git` is different in kind. Its rows — branches, protected branches, protected tags, commit
statuses, LFS objects and locks — mostly **describe things that already exist in git**.

Two small packages complete the picture: `models/unit` names each repository feature, and
`models/gituser` maps git author identities onto Gitea users.

## Why it exists

**Why store git information in a database at all?** Because git is bad at the questions a web UI
asks. "List the twenty most recently updated branches across these repositories, with who pushed
them" would mean opening every repository and walking its refs. Storing that in a table makes it a
single query.

The price is that you now have the same information in two places, which is why the most important
rule here is knowing which one is right.

**Why are a repository's features rows rather than columns?** Because a feature is not just on or
off — it carries configuration, and permissions are granted per feature. A team can have write
access to issues and read access to code. Booleans cannot express that; rows can.

## Words you'll meet

- **unit** — one repository feature: code, issues, pull requests, wiki, actions.
- **repo unit row** — the record saying a feature is enabled here, with its settings.
- **composite unique index** — uniqueness across two columns together. Here: an owner may not have
  two repositories with the same name, but two owners may.
- **denormalised** — deliberately duplicated for speed, and therefore needing to be kept in step.
- **counter column** — a stored count, like a repository's number of issues.
- **cache** — a fast copy of something that lives authoritatively elsewhere.
- **source of truth** — the copy that is right when copies disagree.
- **ref** — a git branch or tag.
- **redirect row** — the record that keeps an old URL working after a rename.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/repo/repo.go` | The `Repository` struct, and the methods for asking which features are on. |
| `models/repo/repo_unit.go` | One row per enabled feature. |
| `models/repo/repo_list.go` | Searching and listing repositories. |
| `models/repo/redirect.go` | Keeping old URLs alive after a rename or transfer. |
| `models/unit/unit.go` | The list of feature types: code, issues, pull requests, wiki, actions and the rest. |
| `models/git/branch.go` | Branch records, kept in step with real git refs. |
| `models/git/protected_branch.go` | Branch protection rules. |
| `models/git/commit_status.go` | CI results per commit. |

## The rules, and why

**A repository's features are rows, so test them with a method.** Ask
`repo.UnitEnabled(ctx, unit.TypeIssues)`. Two ways to fetch one: `repo.GetUnit` returns an error
when it is absent, `repo.MustGetUnit` returns a usable empty value. Choose deliberately — the second
is convenient and will quietly give you default settings for a feature that is switched off.
`repo.LoadUnits(ctx)` fills them in.

**A repository is identified by owner plus lowercase name together.** `Name` keeps the casing to
display; `LowerName` is what lookups match, which is what makes repository names
case-insensitive.

**`Owner` is not loaded by default.** `OwnerID` is always there and `OwnerName` is stored beside it
for convenience, but the `Owner` field is empty until something loads it. A nil `Owner` is not an
error, it is an unloaded field.

**Counter columns are stored; the "open" ones are calculated.** `NumIssues`, `NumClosedIssues`,
`NumPulls`, `NumMilestones`, `NumProjects`, `NumActionRuns`, `NumWatches`, `NumStars` and
`NumForks` are real columns. Their `NumOpen*` counterparts are **not** — they are worked out as
total minus closed. So they are only right when the stored pair is right, and writing to one does
nothing at all. Milestones and labels work the same way.

**Git is the source of truth; `models/git` is the cache.** The real branches and tags are in the git
repository on disk. A row here can be stale, or missing, or describe a branch that was deleted
seconds ago. Code that merely lists things can use the rows. Code that must be *correct* — deciding
whether a merge is allowed, say — reads git.

**Errors follow the same three-part shape as everywhere else**: the error struct, an `IsErr...`
test, and an `Unwrap` to a shared sentinel so callers can recognise the kind without importing the
package.

## How to actually do it

**Check a feature before acting on it.** `repo.UnitEnabled(ctx, unit.TypeIssues)` works anywhere.
But on a route, prefer the middleware that already does it — `mustEnableIssues` on the API side,
`RequireUnitReader` or `RequireUnitWriter` on the web side. Then a disabled feature produces a
proper HTTP response instead of a confusing empty page.

**Add a column to `Repository`.** Add the field with its tag, write a migration, update
`models/fixtures/repository.yml`, and if it is a counter, extend the consistency check too.

**Handle an old repository URL.** A rename or an owner transfer leaves a redirect row behind. Before
returning a 404 for an owner/name pair, check it — otherwise every link to that repository written
before the rename breaks.

## Traps, and what they look like

**`repo.Owner` is nil and you expected a user.** It is unloaded, not missing. Call the loader.

**You set a `NumOpen*` field and nothing happens.** It is computed, not stored. Fix the stored pair
it is derived from.

**Your counts drift.** Something changed issue state without updating the repository's counters, or
a fixture row was added without adjusting them. A consistency test will usually catch it, often in a
test you did not touch.

**Your branch list shows a branch that no longer exists.** The cached row is stale. That is
expected — the question is whether the code reading it should have read git instead.

**You add a new `unit.Type` and half the UI ignores it.** Adding a feature type touches the enum,
the defaults for newly created repositories, the settings UI and the route guards. It is never a
one-line change.

**You open `models/repo/issue.go` looking for issue rows.** That file holds the repository's issue
*settings* — whether time tracking and dependencies are enabled. The issues themselves are in
`models/issues`.

## Where to go next

- `docs/models-repo-and-git.md` (in this folder) — the reference page this was written from
- `models-issues.md` — the issues those counters count
- `models-db.md` — transactions and generic queries
- `services-context.md` — how `ctx.Repo` wraps all this with permissions
