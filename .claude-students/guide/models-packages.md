---
source: docs/models-packages.md
source-hash: f632ac51cedfe592
verified-at: c0092050a4
---

<!-- Derived from docs/models-packages.md. Do not edit by hand: fix the reference doc and regenerate
     this file. See MAINTENANCE.md rules 16-22. -->

# Package registries, in plain English

**In one sentence:** Gitea pretends to be about twenty different package registries at once, so this
is the one place where "what does the client tool expect" beats every convention in the rest of the
codebase.
**Come here when:** you are working on any package registry — npm, NuGet, Maven, PyPI, container
images, Cargo, Debian, RPM and the rest.

## What this is

Gitea can act as a package registry for many ecosystems. You point `npm` or `docker` at your Gitea
instance and it works, because Gitea implements each tool's own protocol.

The work is split across four layers, with one piece per ecosystem at each:

| Layer | Where | Job |
| --- | --- | --- |
| Routes | `routers/api/packages/<ecosystem>/` | Speaking that tool's HTTP protocol. |
| Logic | `services/packages/` | Upload and download, shared by all of them. |
| Metadata | `modules/packages/<ecosystem>/` | Parsing that ecosystem's package format. |
| Rows | `models/packages/` | The shared storage model. |

## Why it exists

**Because every package ecosystem invented its own protocol**, and none of them agreed. `npm`
expects certain URLs and JSON shapes. `docker` implements a completely different specification.
Maven expects something else again.

Gitea cannot ask those tools to adapt. It has to speak each protocol exactly, which is why this is
the one area where the project's own API conventions do not apply — **the client tool's actual
behaviour is the specification**.

What Gitea *can* do is share the storage underneath. All these ecosystems are ultimately "a named
thing, with versions, each having files". So there is one storage model beneath all of them, and the
per-ecosystem code is only protocol and metadata.

## Words you'll meet

- **registry** — a server that hosts packages for one ecosystem.
- **ecosystem** — one packaging world: npm, Maven, PyPI.
- **blob** — the raw bytes of a file, stored once regardless of how many things reference it.
- **deduplication** — storing identical content once.
- **property** — a key/value row holding ecosystem-specific metadata.
- **protocol** — the exact HTTP dialogue a client expects.
- **retention rule** — a policy for deleting old versions automatically.

## What's in these files

| Where | What it is for |
| --- | --- |
| `models/packages/package.go`, `package_version.go`, `package_file.go`, `package_blob.go` | The four-level storage model. |
| `models/packages/package_property.go` | Ecosystem-specific metadata as key/value rows. |
| `models/packages/package_cleanup_rule.go` | Retention rules. |
| `models/packages/descriptor.go` | A version assembled into one view. |
| `routers/api/packages/helper/` | Shared route helpers. |
| `services/packages/` | The shared upload and download path. |

## The rules, and why

**Four levels, and each earns its place.** A **package** is a name in an owner's namespace. A
**version** is one release of it. A **file** is one file in that version. A **blob** is the actual
bytes.

Splitting file from blob is what gives deduplication: upload the same content twice and it is stored
once, referenced by two files. For container images, where layers are shared across many images,
this is the difference between a workable registry and one that fills the disk.

**Ecosystem-specific metadata goes in property rows, not new columns.** This is why adding a whole
new registry usually needs no migration at all — the storage model already fits.

**Each registry speaks its own protocol, and the API conventions do not apply.** The swagger rules
and status-code conventions from `routers-api-v1.md` are for Gitea's own API. Here, match what the
tool actually sends and expects.

**Files go through the storage interface**, never a hard-coded path (`modules-infra.md`).

**Authentication is per-ecosystem too**, because each tool supports different credential forms.

## How to actually do it

**Add an ecosystem.** Four pieces: a metadata parser, route handlers speaking the protocol, mounting
in the packages router, and a package type constant. Copy whichever existing ecosystem is closest to
yours — that is far more reliable than working from the specification alone.

**Add metadata to an existing ecosystem.** Extend its parser and store the result as property rows.
No schema change needed.

**Debug a client that will not work.** Capture what the tool actually sends and compare it against
the handler. **The tool's behaviour is the specification** — not its documentation, and certainly
not how Gitea's other APIs behave.

## Traps, and what they look like

**Deleting a version deletes content another version still uses.** Blobs are shared. Removing one
because the file referencing it went away breaks every other file pointing at the same bytes. Any
deletion path has to check whether anything else still references the blob.

**A change that looks harmless breaks one specific tool version.** This is the most
protocol-sensitive code in Gitea. A field you thought was cosmetic, a status code you thought was
equivalent, a header you reordered — some client parses it strictly. Test with the real tool.

**You apply Gitea's API conventions here and the client rejects it.** Those conventions are for
Gitea's own API.

**You look for more tables in the ecosystem subfolders under `models/`.** Those hold
ecosystem-specific *queries* over the shared tables, not new tables.

## Where to go next

- `docs/models-packages.md` (in this folder) — the reference page this was written from
- `modules-infra.md` — where the blobs are actually stored
- `routers-api-v1.md` — Gitea's own API, and why its rules do not apply here
