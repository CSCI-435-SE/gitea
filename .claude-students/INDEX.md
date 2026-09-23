# Context map (course fork)

**New to this codebase? Read `guide/START-HERE.md` first.** Every page below has a plain-English
version at `guide/<same-name>.md`.

Before exploring the tree, read the 1-3 docs below that match your task. Paths are relative to
`.claude-students/`. Found one wrong? Fix it in your PR — see `.claude-students/MAINTENANCE.md`.

| Working on | Read |
| --- | --- |
| Drafting an issue, PR, code review or sprint deliverable (course formats, deadlines) | `COURSE-WORKFLOW.md`, `AI-ASSISTANCE.md` |
| Which layer does this belong in? What may import what? | `docs/architecture.md` |
| Writing or running any test; needs a DB, context or fixture | `docs/testing.md` |
| An HTML page, a form POST, a web route or guard | `docs/routers-web.md` |
| An `/api/v1` endpoint, or a failing swagger check | `docs/routers-api-v1.md` |
| Anything on `ctx`: the doer, `ctx.Repo`, permissions, error responses | `docs/services-context.md` |
| Form binding and validation, or model-to-API conversion | `docs/services-forms-and-convert.md` |
| Any query, transaction, or a new table | `docs/models-db.md` |
| A schema change, or a failing migration test | `docs/models-migrations.md` |
| Issue, PR, comment, label, milestone or review rows | `docs/models-issues.md` |
| Repository rows, repo units/features, branch or commit-status rows | `docs/models-repo-and-git.md` |
| Project boards and their columns (Kanban) | `docs/models-project.md` |
| Issue create/edit/label/assign/close behaviour | `docs/services-issue.md` |
| PR lifecycle, merging, code review, the diff viewer | `docs/services-pull-and-gitdiff.md` |
| Repo create/fork/transfer, branches, file edits, push handling | `docs/services-repository.md` |
| Browser behaviour on a page, wiring a feature to the DOM | `docs/frontend-js.md` |
| A Vue component, or deciding whether something needs Vue | `docs/frontend-vue-components.md` |
| Styling, Tailwind `tw-` classes, theme colours | `docs/frontend-css.md` |
| Page markup, partials, template functions | `docs/templates.md` |
| Actions rows: runs, jobs, tasks, runners, artifacts | `docs/models-actions.md` |
| When a workflow triggers, job dispatch, approvals, re-runs | `docs/services-actions.md` |
| The runner protocol, artifact upload/download | `docs/routers-api-actions.md` |
| Running git, reading commits/refs/blobs | `docs/modules-git.md` |
| Adding or reading a config option | `docs/modules-setting.md` |
| API request/response struct fields | `docs/modules-structs.md` |
| Markdown, syntax highlighting, `#123` refs, emoji, sanitising | `docs/modules-markup.md` |
| Returning errors, `Option[T]`, `TimeStamp`, sets | `docs/modules-util-and-generics.md` |
| Background work, caching, blob storage, search indexes, locks | `docs/modules-infra.md` |
| Users, orgs, teams, access modes and permissions | `docs/models-user-org-perm.md` |
| Account lifecycle, teams, sign-in and login sources | `docs/services-user-org-auth.md` |
| Events, webhooks, email, notifications, scheduled jobs | `docs/services-notify-mailer-webhook.md` |
| Adding a user-visible string or a locale key | `docs/i18n.md` |
| Package registries (npm, NuGet, container, ...) | `docs/models-packages.md` |
| Git hooks, the internal API, the installer, CLI subcommands | `docs/routers-private-install-cmd.md` |
| A failing lint/build step, or which command to run | `docs/build-and-tooling.md` |

No doc for your path? Most remaining subpackages are small single-purpose leaf utilities. The
notable gaps are `services/{wiki,release,mirror,migrations,attachment,projects,indexer}` and
`models/{webhook,activities}` — explore those normally, then consider adding a doc under rule 14 in
`.claude-students/MAINTENANCE.md`.
