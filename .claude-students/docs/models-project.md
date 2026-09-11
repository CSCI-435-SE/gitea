---
scope: models/project
verified-at: c0092050a4
---

# models/project — project boards (Kanban)

**Read when:** working on project boards, their columns, or which issues sit in them.
**Not here:** the issues themselves -> `models-issues.md`; the board UI logic -> `services/projects`.

## Responsibilities

Three tables: a `Project` (a board), its `Column`s, and the join rows placing issues into columns.
A project belongs either to a repository or to an owner (user or org), not both.

## Key files

| Path | What it holds |
| --- | --- |
| `models/project/project.go` | `Project`, `Type`, `CardType`, `NewProject`, `GetProjectByID`, `UpdateProject`, `ChangeProjectStatus`, `DeleteProjectByID` |
| `models/project/column.go` | `Column`, `IsCardTypeValid` |
| `models/project/issue.go` | the issue-to-column join, `GetColumnIssueNextSorting`, `DeleteAllProjectIssueByIssueIDsAndProjectIDs` |
| `models/project/template.go` | `TemplateType`, `GetTemplateConfigs`, `IsTemplateTypeValid` |

## Conventions & invariants

- **`Project.TemplateType` is stored in a column named `board_type`.** The xorm tag is
  `xorm:"'board_type'"`, with a standing TODO to rename it. Any raw SQL, migration or fixture must
  use `board_type`; the Go field is `TemplateType`. This mismatch is the single most likely thing
  to bite you here.
- Ownership is one of two shapes: `RepoID` set for a repository board, or `OwnerID` set for a
  user/org board. `Type` records which. Queries must filter on the right one — a look-up by `ID`
  alone crosses ownership boundaries, which is why `GetProjectByIDAndOwner` and
  `GetProjectForRepoByID` exist. Prefer those over `GetProjectByID` on any user-facing path.
- `Project.NumIssues`, `NumOpenIssues` and `NumClosedIssues` are all `xorm:"-"` — computed on
  load, never stored. Unlike `Repository`, there is no counter column to keep consistent here.
- Exactly one `Column` per project has `Default` true: issues added to the board without a column
  land there. Deleting a column has to rehome its issues.
- `Column.Sorting` is the manual card order. New placements get their value from
  `GetColumnIssueNextSorting` rather than a naive max+1 computed in the caller.
- `GhostProjectID` is the sentinel for a deleted project referenced by something still alive.
- The usual error triad applies: `ErrProjectNotExist` / `IsErrProjectNotExist` and
  `ErrProjectColumnNotExist` / `IsErrProjectColumnNotExist`.

## Recipes

**Add a field to a board.** Add it to `Project` with its xorm tag, write the migration
(`models-migrations.md`), extend `UpdateProject`, and update `models/fixtures/project.yml`. Note
the column and join fixtures carry the legacy names `project_board.yml` and `project_issue.yml`.

**Move an issue between columns.** Go through `models/project/issue.go` and take the new sorting
value from `GetColumnIssueNextSorting` — the board's ordering is otherwise not stable.

**Add a board template.** A `TemplateType` value plus its entry in `GetTemplateConfigs`, and the
column set it should create.

## Gotchas

- Two similar names: `Type` is repo-vs-owner ownership, `TemplateType` is the board preset, and
  `CardType` is how cards render. All three are small enums on `Project` and are easy to swap by
  accident.
- Boards are reachable from both a repo URL and an org URL, so a permission check written for one
  shape does not cover the other.
- The issue side of the relationship also has `models/issues/issue_project.go`; the join is
  written from both packages, so check both when changing its shape.

## Related

- `models-issues.md` — the issues a board holds
- `models-db.md` — transactions for multi-row column moves
