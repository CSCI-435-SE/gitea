---
source: docs/architecture.md
source-hash: 21e4a1af11b2bdcd
verified-at: c0092050a4
---

<!-- Derived from docs/architecture.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# How the codebase is laid out, in plain English

**In one sentence:** Gitea is split into five layers that are only allowed to depend on each other
in one direction, and knowing that direction tells you where your code belongs.
**Come here when:** you do not know which folder your change goes in, an import will not compile, or
you cannot work out which file produces the page you are looking at.

## What this is

Gitea's Go code sits in five top-level folders — `cmd/`, `routers/`, `services/`, `models/` and
`modules/` — plus `templates/` for HTML, `web_src/` for the frontend and `options/` for shipped
data like translations.

The folders are not just filing. They are a stack with a rule: **a folder may only use the ones to
its right.**

```text
cmd → routers → services → models → modules
```

`routers/` may call `services/`. `services/` may call `models/`. But `models/` may never call
`services/`, and `modules/` may call none of them. Break that and the code will often still
compile, and a reviewer will still send it back — it is the most common structural rejection.

## Why it exists

Without the rule, everything ends up importing everything, and you get two problems that get worse
every month.

The first is **circular imports**, which Go refuses to compile at all. In a tangled codebase you
discover this only after writing the code.

The second is that **nothing can be tested or reasoned about alone.** Because `modules/` depends on
nothing, you can test a helper there without a database. Because `models/` does not know about HTTP,
you can call it from a CLI command as easily as from a web page — which is exactly how
`cmd/` works.

The direction also answers the question you will ask most often: *where does this code go?* You do
not have to guess. Walk the stack.

## Words you'll meet

- **layer** — one of the five folders above, and its position in the dependency order.
- **handler** — a function in `routers/` that answers one URL.
- **XORM bean** — a Go struct in `models/` that maps onto a database table.
- **request context (`ctx`)** — the object carrying everything about the current request: who is
  signed in, which repo they are viewing, and the bag of data the template will read.
- **middleware** — code that runs before your handler, to log the user in, load the repo, or check
  permissions.
- **transaction** — a group of database writes that must all succeed or all be undone.
- **import alias** — a second name for an imported package, used when two packages share a name.
- **vite** — the tool that compiles `web_src/` into the browser assets in `public/assets/`.

## What's in these files

| Where | What it is for |
| --- | --- |
| `cmd/` | The command-line subcommands — `web` (start the server), `serv`, `hook`, `doctor`, the `admin` tools. |
| `routers/` | One function per URL. Split into `web` (HTML pages), `api` (JSON), `private` (internal, called by git hooks), `install`, and shared `common`. |
| `services/` | The actual business logic. Anything that coordinates several database calls, or sends mail, or fires a webhook. |
| `models/` | The database. Structs that map to tables, and the queries over them. |
| `modules/` | Self-contained helpers that depend on nothing — string handling, git command wrappers, config. |
| `templates/` | The HTML. Go template files, also used for the emails Gitea sends. |
| `web_src/` | TypeScript, Vue and CSS, compiled into `public/assets/`. |
| `options/` | Data shipped with Gitea: translations, `.gitignore` templates, licence texts. |

The four files worth knowing by name:

- `main.go` — where the program starts.
- `routers/init.go` — decides what lives at `/`, `/api/v1`, `/api/internal`, `/api/packages` and
  `/api/actions`.
- `routers/web/web.go` — the entire list of web URLs. Long, but it is the index of the whole UI.
- `services/context/context.go` — defines the `ctx` every handler receives.

## The rules, and why

**Dependencies only flow right.** Add an import that points left and the reviewer will reject it.
The fix is never to add the import — it is to move the code to the layer that is allowed to do the
work.

**Top-level folders are plural, subfolders are singular.** `services/user`, `models/repo`. Cosmetic,
but consistent, and a mismatch looks like a mistake.

**When two packages share a name, alias the import.** The same concept usually exists in two
layers at once, so files give each a snake_case alias saying which layer it came from —
`issues_model`, `user_service`, `access_model`, and `api` for `modules/structs`. Copy whatever
aliases the file you are editing already uses; a file with two conventions in it is hard to read.

**The module path is `gitea.dev`, not `code.gitea.io/gitea`.** This fork renamed it. Every import
path you copy from the official Gitea docs, a GitHub code search or an AI answer will be wrong and
will not compile. Copy from a neighbouring file instead. This one costs people an hour the first
time.

**There is no `modules/context` here.** The request context lives in `services/context`, because it
needs to reach models and services — which `modules/` is not allowed to do. Upstream Gitea moved it
for exactly that reason, so old references to `modules/context` are stale.

**Functions in a transaction take `context.Context` first.** That parameter is how the database
layer knows which transaction you are in. Pass a `*xorm.Session` around instead and your writes
quietly land outside the transaction, so a failure half-way leaves the database inconsistent.

## How to actually do it

**Work out where new code goes.** Ask, in order:

1. Does it read or write the database? → `models/`
2. Does it coordinate several of those calls, or send mail, notifications or webhooks? →
   `services/`
3. Does it read the request or write the response? → `routers/`
4. Is it a pure helper that needs nothing from Gitea? → `modules/`

Most real changes touch two or three of these. A feature is usually a `models/` change, a
`services/` change and a `routers/` change, in that order.

**Follow a web request from URL to page.**

1. `routers/init.go` decides the request belongs to the web router.
2. `routers/web/web.go` runs the middleware — sign-in, loading the repo — then matches the path.
3. Your handler runs. Its signature is `func(ctx *context.Context)`.
4. It calls into `services/`, which calls into `models/`.
5. It puts results in `ctx.Data[...]` and renders a template.

**Follow an API request.** Identical until `routers/api/v1/api.go`. Then the handler takes
`*context.APIContext`, converts the database structs into `modules/structs` types, and returns JSON.

## Traps, and what they look like

**"import cycle not allowed".** Go is telling you that you broke the direction rule. Do not try to
outsmart it with an interface — move the code to the correct layer.

**Your import does not resolve, and the path looks right.** You copied `code.gitea.io/gitea/...`
from somewhere. It must be `gitea.dev/...`.

**Your template renders a blank where a value should be.** A template can only use what the handler
put in `ctx.Data`. A blank almost always means the handler did not set that key, or set it under a
different name. Templates do not error on a missing key.

**Your change works locally, then breaks for everyone on upgrade.** You changed a struct in
`models/` that is stored in the database, without adding a migration under `models/migrations/`. A
fresh database gets the new shape; an existing one does not.

**Your PR gets sent back for being too big.** `docs/guidelines-refactoring.md` asks for tight PRs:
fix the root cause, keep the scope small, split large refactors up.

Before you push, run `make fmt`, then `make lint-go` or `make lint-js` for whichever you touched,
and `make tidy` if you changed dependencies. `make help` lists everything.

## Where to go next

- `docs/architecture.md` (in this folder) — the reference page this was written from
- `testing.md` — which kind of test to write, and how to run it
- Gitea's own `docs/guidelines-backend.md` and `docs/guidelines-frontend.md` — the project's
  rules, which win over anything here
