---
scope: models/packages, services/packages, routers/api/packages, modules/packages
verified-at: c0092050a4
---

# packages — the package registries

**Read when:** working on any package registry (npm, NuGet, Maven, PyPI, container, Cargo, Debian,
RPM, ...).
**Not here:** the generic API -> `routers-api-v1.md`.

## Responsibilities

Gitea implements ~20 package-registry protocols. The work is split by layer, with one subpackage
per ecosystem at each layer:

| Layer | Path | Role |
| --- | --- | --- |
| Routes | `routers/api/packages/<ecosystem>/` | the ecosystem's own HTTP protocol |
| Logic | `services/packages/` | upload, download, cleanup, shared across ecosystems |
| Metadata | `modules/packages/<ecosystem>/` | parsing that ecosystem's package format |
| Rows | `models/packages/` | `Package`, `PackageVersion`, `PackageFile`, `PackageBlob`, `PackageProperty` |

## Key files

| Path | What it holds |
| --- | --- |
| `models/packages/package.go`, `package_version.go`, `package_file.go`, `package_blob.go` | the four-level row model |
| `models/packages/package_property.go` | ecosystem-specific key/value metadata |
| `models/packages/package_cleanup_rule.go` | retention rules |
| `models/packages/descriptor.go` | the assembled view of a package version |
| `routers/api/packages/helper/` | shared route helpers |
| `services/packages/` | the shared upload/download path |

## Conventions & invariants

- **Four levels, and they matter:** `Package` (name in an owner's namespace) → `PackageVersion` →
  `PackageFile` → `PackageBlob`. The blob is the deduplicated content, so the same bytes uploaded
  twice are stored once and referenced by two files.
- Ecosystem-specific metadata does not get new columns — it goes in `PackageProperty` rows. That
  is why adding a registry rarely needs a migration.
- Each registry speaks **its own protocol**, not Gitea's REST conventions. The swagger and
  status-code rules in `routers-api-v1.md` do not apply; match the upstream tool's expectations
  (what `npm`, `cargo` or `docker` actually sends).
- Package files live in `modules/storage`, not on a hard-coded path (`modules-infra.md`).
- Authentication is per-ecosystem too — most use basic auth or a token in the form the client tool
  supports, guarded by `reqPackageAccess` (`routers-api-v1.md`).

## Recipes

**Add an ecosystem.** A metadata parser in `modules/packages/<name>/`, route handlers in
`routers/api/packages/<name>/`, mounting in the packages router, and the package type constant in
`models/packages`. Copy the closest existing ecosystem.

**Add metadata to an existing ecosystem.** Extend its parser in `modules/packages/<name>/` and
store the result as `PackageProperty` rows.

**Debug a failing client.** The client tool's exact HTTP dialogue is the specification — compare
against the ecosystem's handler rather than against Gitea's other APIs.

## Gotchas

- Blob deduplication means deleting a package version must not delete the blob while another file
  still references it.
- The registries are the most protocol-sensitive code in Gitea: a response shape that looks
  harmless can break one specific client version.
- `models/packages/container/` and the other ecosystem subdirectories under `models/` hold
  ecosystem-specific *queries*, not more tables.

## Related

- `modules-infra.md` — blob storage
- `routers-api-v1.md` — the separate, conventional API
