---
scope: modules/setting
verified-at: c0092050a4
---

# modules/setting — configuration

**Read when:** adding a config option, or reading one at runtime.
**Not here:** the installer that writes the initial config -> `routers/install`.

## Responsibilities

Parses `app.ini` into typed package-level variables that the rest of Gitea reads. One file per
config section, each exposing a struct or a set of variables plus a `loadXxxFrom(cfg)` function.

## Key files

| Path | What it holds |
| --- | --- |
| `modules/setting/setting.go` | `LoadSettings`, `LoadCommonSettings`, `LoadSettingsForInstall` — the entry points |
| `modules/setting/config_provider.go` | the ini abstraction every loader receives |
| `modules/setting/config_env.go` | the `GITEA__SECTION__KEY` environment-variable override mechanism |
| `modules/setting/server.go`, `database.go`, `repository.go`, `service.go` | the big sections |
| `modules/setting/actions.go`, `indexer.go`, `mailer.go`, `lfs.go`, `log.go`, `cron.go`, `git.go` | one file per feature area |
| `modules/setting/config/` | the runtime-editable settings stored in the database |

## Conventions & invariants

- Settings are **package-level variables**, read directly as `setting.Service.RequireSignInView`,
  `setting.Actions.Enabled`, `setting.RepoRootPath`. There is no getter and no injection.
- Each section file defines its struct with `ini` mapping tags and a `loadXxxFrom(rootCfg
  ConfigProvider)` called from the central load. A new section follows the same shape and is
  registered there — a loader that nothing calls silently leaves defaults in place.
- **Defaults live in the struct literal**, not in the caller. Code reading a setting should never
  need `if x == "" { x = "default" }`.
- Every key is overridable by environment variable as `GITEA__<SECTION>__<KEY>`
  (`config_env.go`), which is how the Docker image is configured. A setting that bypasses the
  config provider loses this.
- `modules/setting` sits at the far right of the dependency direction (`architecture.md`): it must
  not import models or services. Config that depends on database state belongs in
  `modules/setting/config/`, which is the runtime-editable layer.
- Tests load settings explicitly; a package-level variable read at init time before settings load
  gets the zero value.

## Recipes

**Add a config option.** Add the field to the section struct with its ini tag and default, read it
where needed as `setting.Section.Field`, and document it in the upstream config cheat sheet if the
change is user-facing.

**Add a whole section.** New `modules/setting/<name>.go` with the struct and a `loadNameFrom`,
called from the central load in `setting.go`.

**Check what a setting resolves to.** Settings are plain variables, so the fastest check is reading
the `loadXxxFrom` for its default and the ini key name.

## Gotchas

- Reading a setting from an `init()` function runs before settings are loaded. Read them from a
  function called after startup instead.
- `setting.AppSubURL` (and friends) mean URLs must be built through the helpers, not by
  concatenating `/` paths, or Gitea breaks when hosted under a subpath.
- The runtime-editable settings under `modules/setting/config/` are *not* the same as the ini
  values; changing one does not change the other.

## Related

- `architecture.md` — why this package may not import upward
- `modules-infra.md` — the subsystems configured from here
