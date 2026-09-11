<!-- markdownlint-disable MD029 -->
<!-- Rules are numbered continuously 1-15 across the sections below; check.sh error
     messages and all 34 docs cite them by number, so the sequence must not restart. -->

# Maintenance rules

Five people maintain one shared set of docs. These rules exist so the set stays *one* set. Read
them before editing anything in `.claude-students/`. Rules marked **[checked]** are enforced by
`./.claude-students/check.sh`.

## Scope discipline

1. **Upstream wins.** `AGENTS.md`, `CONTRIBUTING.md` and `docs/guidelines-*.md` are authoritative.
   Docs here summarise them and link to them; they never restate a rule differently and never
   contradict one. A contradiction is a bug in this folder, not in the project.
2. **One fact, one home.** Every fact lives in exactly one doc — the one whose `scope:` owns that
   path. Everywhere else, link to it. Duplicated facts are how five maintainers end up with five
   different answers.
3. **Durability test.** Record a fact only if it (a) took more than one discovery step to find and
   (b) will still be true after the next upstream merge. A one-off workaround, a sprint detail or
   anything about your own task fails this test — that belongs in `ai-logs/`.
4. **No process notes.** No TODOs, no open questions, no first person, no student names, no dates
   other than `verified-at`. This is a map of the code, not a journal.

## Form

5. **Template only [checked].** Use the `## ` headings from `_TEMPLATE.md`, all of them, in that
   order, with nothing added. If a section has nothing to say, write one line saying so.
6. **120-line cap [checked].** Going over means the scope is too broad. Split the doc, and add the
   new `INDEX.md` row in the same commit. `INDEX.md` itself is capped at 60 lines.
7. **Cite or delete [checked, partially].** Every claim names a repo path. A claim with no path is
   unverifiable by the next reader — delete it.
8. **Symbols, not line numbers [checked].** Write `models/db/context.go` → `WithTx`, never
   `models/db/context.go:143`. Line numbers rot at the next upstream merge and a stale number is
   worse than no number. Code fences stay at 10 lines or fewer — point at the file instead of
   copying it.
9. **Repo-relative paths [checked].** `models/db/context.go`, never `/home/you/gitea/models/...`.
10. **Whitespace [checked].** No trailing whitespace, LF endings, final newline — same as the rest
    of the repo (`.editorconfig`).

## Change control

11. **One universal set.** These docs live on `main` and nowhere else. Never keep a personal copy,
    a per-student folder, or a long-lived branch of them.
12. **Changes land by PR**, titled `docs(context): <subject>` per `AGENTS.md` Conventional Commits,
    with an `Assisted-by: AGENT_NAME:MODEL_VERSION` trailer and no `Co-Authored-By` or
    `Signed-off-by`.
13. **Fix in place.** If you discover while working that a doc is wrong or incomplete, correct it
    in the *same* PR as your code change and update `verified-at` to the commit you checked
    against. Leaving a doc stale for someone else to hit is the exact failure this folder exists to
    prevent. This rule applies to AI sessions too.
14. **New docs need three things in one commit [checked]:** the file, its `INDEX.md` row, and a
    `scope:` that does not overlap any existing doc's. If the scope overlaps, you wanted to edit
    that doc, not add one.
15. **`./.claude-students/check.sh` passes before you push [checked].** It is dependency-free and
    finishes in under a second.

## The guide layer (rules 16-22)

`guide/` holds a plain-English page for each reference page in `docs/`, same filename. It exists so
a person new to Go and to this codebase can get oriented. It is **derived**, not independent.

16. Every guide carries `source:`, `source-hash:` and `verified-at:` in its frontmatter **[checked]**.
17. A guide's `source:` names a real `docs/*.md` with the **same filename** **[checked]**.
18. `source-hash:` is the first 16 characters of `sha256sum docs/<name>.md`. If the reference page
    changes and the guide is not regenerated, `check.sh` reports **drift** **[checked]**. Regenerating
    means: reread the reference page, rewrite the guide, update the hash and `verified-at`.
19. Guide headings match `guide/_TEMPLATE.md` exactly and in order **[checked]**.
20. Guides are capped at **200 lines**, code fences at **30** **[checked]** — teaching needs more room
    than reference does.
21. **A guide adds explanation, vocabulary and worked steps. It adds no new facts.** Every repo path
    a guide cites must also be cited by its source page **[checked]**. If you need a fact the
    reference page lacks, add it *there* first, which bumps the hash and forces the guide to be
    regenerated in the same commit — exactly the intended order.
22. `guide/START-HERE.md` has no source page. It is exempt from rules 17, 18 and 21, and is where
    cross-cutting material lives that no single reference page owns.

**Never fix a fact in `guide/`.** If the two layers disagree, the reference page is the truth and
the guide is stale: fix `docs/`, regenerate the guide, ship both in one PR. A disagreement is a bug
report against `docs/`, not an editing job in `guide/`.

**Known limitation:** a hand-edit to a guide that only rewords prose is not mechanically
detectable. Rule 21 catches invented facts; rule 18 catches a stale guide; review catches the rest.

## Updating `verified-at`

`verified-at` is the short SHA the doc was last checked against, from `git rev-parse --short HEAD`.
Bump it whenever you confirm or correct a doc. A doc whose `verified-at` is far behind `main` has
not been checked recently — treat its details as hints and confirm them in the code before relying
on them.
