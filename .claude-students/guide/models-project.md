---
source: docs/models-project.md
source-hash: 6b34d82ffd28a54e
verified-at: c0092050a4
---

<!-- Derived from docs/models-project.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Project boards (Kanban), in plain English

**In one sentence:** three tables — a board, its columns, and which issues sit in which column —
carrying one badly-named legacy column that will trip you up if nobody warns you.
**Come here when:** you are working on project boards, their columns, or the cards on them.

## What this is

The database side of Gitea's Kanban boards: the drag-and-drop view where issues become cards in
columns like "To do", "In progress" and "Done".

A board belongs **either** to a repository **or** to a user or organisation — never both. A
repository board holds that repository's issues; an organisation board can span several.

## Why it exists

Issues on their own are a list. A board is a *view* of some of those issues, arranged in an order a
human chose.

That last part is why this needs its own tables. "Which column is this issue in" and "where in that
column" are not facts about the issue — the same issue can sit on several boards in different
places. So the position lives in a join table between the issue and the column, and the manual order
is a number on that row.

## Words you'll meet

- **board / project** — the Kanban view. The code says "project"; the UI says both.
- **column** — a vertical lane on the board.
- **card** — one issue as it appears on the board.
- **join table** — a table whose job is to connect two others.
- **sorting value** — a number giving cards their manual order.
- **default column** — where an issue lands if it is added to a board without choosing one.
- **template** — a preset set of columns for a new board.
- **sentinel value** — a fixed special value meaning something other than a real id.
- **computed field** — one worked out when loaded, not stored.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/project/project.go` | The board itself: the struct, its enums, and creating, fetching, updating, closing and deleting one. |
| `models/project/column.go` | Columns, and validating how cards should look. |
| `models/project/issue.go` | The join between issues and columns, and working out the next sorting value. |
| `models/project/template.go` | The presets a new board can start from. |

## The rules, and why

**The `TemplateType` field is stored in a column called `board_type`.** This is the one thing to
remember from this page.

The Go field is `TemplateType`. The database column is `board_type`, from an earlier version where
boards were called something else. There is a standing note in the code to rename it, and until
somebody does, **anything touching the database directly — a migration, raw SQL, a test fixture —
must say `board_type`**, while your Go code says `TemplateType`.

Getting this wrong produces a migration that appears to work and silently affects nothing, or a
fixture that fails to load for reasons that look unrelated.

**A board belongs to a repository or to an owner, not both.** One of the two id fields is set, and a
separate field records which shape it is. That means **looking a board up by id alone crosses
ownership boundaries** — you can fetch an organisation's board while serving a repository URL. That
is why there are lookup functions that take the owner or the repository as well, and why you should
prefer them on anything user-facing. The plain by-id lookup is for code that has already established
who is allowed to see it.

**The issue counts on a board are computed, not stored.** Unlike a repository, a board has no
counter columns to keep in step. They are worked out when the board is loaded.

**Exactly one column is the default.** An issue added to a board without a chosen column lands
there. This is why deleting a column is not just a delete — its issues have to go somewhere.

**Card order comes from a helper, not from the caller.** Ask for the next sorting value rather than
computing "highest plus one" yourself. Two people dragging cards at the same moment will otherwise
pick the same number and the order becomes unstable.

**There is a sentinel id for a deleted board**, so something still referring to a board that has
been removed has a defined value to hold rather than a dangling id.

**Errors use the same three-part shape as the rest of `models/`** — the error struct, an `IsErr...`
test, and an unwrap to a shared sentinel.

## How to actually do it

**Add a field to a board.** Add it to the struct with its tag, write the migration, extend the
update function so the value is actually saved, and update the fixtures. Note the fixture files
still carry the old names — the column and join fixtures are `project_board.yml` and
`project_issue.yml`, not what you would guess from the Go package.

**Move an issue between columns.** Go through the join in `models/project/issue.go`, and take the
new position from the sorting helper.

**Add a board preset.** A new template type, its entry in the template configuration, and the set of
columns it should create.

## Traps, and what they look like

**Your migration or fixture does nothing, silently.** You wrote `template_type`. The column is
`board_type`.

**You used the wrong enum.** There are three small enums on a board and they are easy to confuse:
one records repository-versus-owner ownership, one is the board preset, and one is how cards render.
They are all short names on the same struct.

**Your permission check passes on one URL shape and not the other.** Boards are reachable from both
a repository URL and an organisation URL. A check written with only one in mind does not cover the
other, and this is a genuine access bug rather than a cosmetic one.

**You changed the join and half of it still behaves oddly.** The issue side of the relationship also
has code in `models/issues/issue_project.go`. The join is written from both packages, so a change to
its shape means checking both.

**Cards jump around after a drag.** Something computed a sorting value itself instead of asking for
the next one.

## Where to go next

- `docs/models-project.md` (in this folder) — the reference page this was written from
- `models-issues.md` — the issues a board displays
- `models-db.md` — transactions, which a multi-row column move needs
