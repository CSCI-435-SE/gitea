---
scope: models/repo, models/git, models/unit, models/gituser
verified-at: c0092050a4
---

# models/repo and models/git — repositories, units, and git-derived rows

**Read when:** querying or changing repository rows, enabling/checking a repo feature, or touching
branch, tag-protection or commit-status records.
**Not here:** repo creation and git operations -> `services/repository`; the git CLI wrapper -> `modules/git`.

## Responsibilities

| Package | Owns |
| --- | --- |
| `models/repo` | `Repository` and everything hanging off it: collaboration, watch, star, fork, release, attachment, topic, mirror, pushmirror, redirect, transfer, upload, wiki, license, language stats, archiver, repo units |
| `models/git` | rows derived from git state: `branch.go`, `protected_branch.go`, `protected_tag.go`, `commit_status.go`, `commit_status_summary.go`, `lfs.go`, `lfs_lock.go` |
| `models/unit` | the `unit.Type` enum naming each repo feature |
| `models/gituser` | mapping git author identities to Gitea users |

## Key files

| Path | What it holds |
| --- | --- |
| `models/repo/repo.go` | `Repository`, `LoadUnits`, `UnitEnabled`, `GetUnit`, `MustGetUnit` |
| `models/repo/repo_unit.go` | `RepoUnit` — one row per enabled feature |
| `models/repo/repo_list.go` | the search/list options for repositories |
| `models/repo/redirect.go` | `Redirect`, `NewRedirect` — keeps old URLs working after a rename or transfer |
| `models/unit/unit.go` | `TypeCode`, `TypeIssues`, `TypePullRequests`, `TypeWiki`, `TypeActions`, ... |
| `models/git/branch.go` | `Branch` rows and their sync with real git refs |
| `models/git/protected_branch.go` | branch protection rules |
| `models/git/commit_status.go` | CI status per commit |

## Conventions & invariants

- **A repo's features are rows, not booleans.** Whether issues are enabled is a `RepoUnit` row, so
  test it with `repo.UnitEnabled(ctx, unit.TypeIssues)`. `repo.GetUnit` returns an error when
  absent; `repo.MustGetUnit` returns a usable zero value — pick deliberately.
  `repo.LoadUnits(ctx)` populates them.
- Repository identity is the composite unique index on (`OwnerID`, `LowerName`). `Name` keeps the
  display casing, `LowerName` is what look-ups match.
- `OwnerName` is denormalised beside `OwnerID`, and `Owner` is `xorm:"-"` — a loaded struct may
  have `OwnerID` set and `Owner` nil.
- Counter columns (`NumIssues`, `NumClosedIssues`, `NumPulls`, `NumMilestones`, `NumProjects`,
  `NumActionRuns`, `NumWatches`, `NumStars`, `NumForks`) are stored. Their `NumOpen*` counterparts
  are `xorm:"-"` and computed in `models/repo/repo.go` as stored-total minus stored-closed, so
  they are only correct when the stored pair is. `Milestone` and `Label` do the same.
- **`models/git` rows are a cache of git, not the source of truth.** The real branches and tags
  live in the git repository on disk (`modules/git`, `modules/gitrepo`). A row can be stale or
  missing; code that must be correct reads git.
- Error types use the same three-part shape as elsewhere: `ErrRedirectNotExist`,
  `IsErrRedirectNotExist`, and `Unwrap` to a `modules/util` sentinel.

## Recipes

**Check a feature before acting.** `repo.UnitEnabled(ctx, unit.TypeIssues)` — and on a route,
prefer the middleware that already does it (`mustEnableIssues` for the API in
`routers-api-v1.md`, `RequireUnitReader`/`RequireUnitWriter` for the web in
`services-context.md`), so the failure is a proper HTTP response.

**Add a column to `Repository`.** Add the field with its xorm tag, add a migration
(`models-migrations.md`), and update `models/fixtures/repository.yml` plus the consistency check if
it is a counter (`testing.md`).

**Resolve an old repo URL.** A rename or an owner transfer inserts a `Redirect` row (see
`services/repository/transfer.go`), so a 404 on an owner/name pair should consult it before
failing.

## Gotchas

- Adding a `unit.Type` value means touching the enum, the defaults for new repos, the settings UI
  and the route guards — it is never a one-line change.
- `repo.Owner` being nil is not an error state, it is an unloaded field. Call the loader.
- A `NumOpen*` field looks like a column but is computed; writing to it does nothing.
- `models/repo/issue.go` holds the repo's issue *settings* accessors (`IsTimetrackerEnabled`,
  `IsDependenciesEnabled`), not issue rows — those are in `models/issues` (`models-issues.md`).

## Related

- `models-issues.md` — the counters above and the issue rows they count
- `models-db.md` — transactions and generic queries
- `services-context.md` — `ctx.Repo` wraps `Repository` with permissions
