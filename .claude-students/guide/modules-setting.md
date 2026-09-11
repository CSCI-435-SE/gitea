---
source: docs/modules-setting.md
source-hash: d1a9565131cd856a
verified-at: c0092050a4
---

<!-- Derived from docs/modules-setting.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Configuration, in plain English

**In one sentence:** Gitea's settings are ordinary Go variables you read directly, filled in once at
startup from a config file — which is simple, and has exactly one sharp edge.
**Come here when:** you are adding a configuration option, or reading one.

## What this is

The package that reads Gitea's config file and turns it into typed values the rest of the program
uses. One file per section of the config, each defining a struct and the function that fills it.

There is also a second, smaller kind of setting: values an administrator can change from the web UI
at runtime, stored in the database rather than the file.

## Why it exists

A server needs configuration, and Gitea's approach is deliberately plain: **no getters, no
dependency injection, no configuration objects passed around.** You write
`setting.Service.RequireSignInView` and read a variable.

That is a real simplification. Passing a config object through every layer would add a parameter to
hundreds of functions to no benefit, since the values never change after startup.

The sharp edge follows directly: **the variables are empty until startup fills them.** Code that
runs before that — anything in an `init()` function — reads zeros, empty strings and falses. It does
not fail; it quietly believes every feature is disabled.

## Words you'll meet

- **ini file** — the config format: `[section]` headers with `KEY = value` beneath.
- **section** — one bracketed block, matching one struct here.
- **provider** — the abstraction over the parsed file, so loaders do not read it directly.
- **environment variable override** — setting a config value through the environment instead of the
  file.
- **package-level variable** — one belonging to the package, readable from anywhere that imports it.
- **`init()`** — a Go function running automatically when a package loads, **before `main`**.
- **zero value** — Go's default for something not set: `0`, `""`, `false`.
- **sub-path** — running Gitea at `example.com/gitea` rather than at the root.

## What's in these files

| Where | What it is for |
| --- | --- |
| `modules/setting/setting.go` | The entry points that load everything at startup. |
| `modules/setting/config_provider.go` | The abstraction over the parsed file. |
| `modules/setting/config_env.go` | Environment-variable overrides. |
| `modules/setting/server.go`, `database.go`, `repository.go`, `service.go` | The big sections. |
| `modules/setting/actions.go`, `indexer.go`, `mailer.go`, `lfs.go`, `log.go`, `cron.go`, `git.go` | One file per feature area. |
| `modules/setting/config/` | The runtime-editable settings kept in the database. |

## The rules, and why

**Settings are package-level variables, read directly.** `setting.Service.RequireSignInView`,
`setting.Actions.Enabled`, `setting.RepoRootPath`. There is nothing to construct and nothing to pass
in.

**Each section is a struct plus a loader, and the loader must be registered.** The struct carries
the mapping tags; the loader is called from the central startup load. A loader nothing calls fails
in the quietest possible way: no error, no warning, and every value in that section stays at its
default forever.

**Defaults belong in the struct, not in the reader.** Code using a setting should never contain
`if x == "" { x = "something" }`. If it does, the default now exists in two places and they will
disagree.

**Every key can be overridden by an environment variable**, following a `GITEA__SECTION__KEY`
pattern. This is how the Docker image is configured, so a setting that bypasses the provider and
reads the file itself silently loses that ability — and breaks for every containerised user without
breaking for you.

**This package may not import models or services.** It sits at the far right of the dependency
direction (`architecture.md`). Configuration that genuinely depends on database state belongs in the
runtime-editable layer instead.

**Tests load settings explicitly.** A variable read at package-initialisation time, before that
load, is the zero value.

## How to actually do it

**Add a config option.** Add the field to the right section struct with its mapping tag and its
default, then read it as `setting.Section.Field` wherever needed. If it is user-facing, document it
where the project documents configuration.

**Add a whole section.** A new file with the struct and its loader, and — the step to not forget —
register that loader in the central load.

**Find out what a setting actually defaults to.** Read its loader. Settings are plain variables, so
there is no indirection to chase; the struct and its loader tell you the key name and the default.

## Traps, and what they look like

**A feature behaves as though it is switched off, and the config says otherwise.** You read the
setting from an `init()` function, which runs before settings are loaded. You got the zero value:
`false`. Read settings from a function called after startup instead.

**Your new section has no effect at all.** You wrote the struct and the loader and never registered
the loader. Nothing calls it, so the defaults stand.

**It works from a config file and not in Docker.** Something bypassed the provider, so the
environment-variable override never applies.

**Gitea breaks when hosted under a sub-path.** URLs were built by pasting `/` and path segments
together instead of using the helpers that account for the configured sub-path. This is invisible in
development, where Gitea runs at the root, and immediately visible to anyone hosting it under a
prefix.

**You change a setting in the admin UI and the config file does not change, or the reverse.** The
runtime-editable settings and the file values are two different systems. Changing one does not
change the other.

## Where to go next

- `docs/modules-setting.md` (in this folder) — the reference page this was written from
- `architecture.md` — why this package may not import upward
- `modules-infra.md` — the subsystems these settings configure
