#!/usr/bin/env bash
# Validates .claude-students against the mechanical rules in MAINTENANCE.md.
# Dependency-free. Run from anywhere: ./.claude-students/check.sh
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

DIR=".claude-students"
TEMPLATE="$DIR/_TEMPLATE.md"
INDEX="$DIR/INDEX.md"
MAX_LINES=120
MAX_INDEX_LINES=60
MAX_FENCE=10

fail=0
err() {
  printf '%s: %s\n' "$1" "$2" >&2
  fail=1
}

for required in "$TEMPLATE" "$INDEX" "$DIR/MAINTENANCE.md" "$DIR/README.md"; do
  [ -f "$required" ] || err "$required" "missing"
done
[ "$fail" -eq 0 ] || exit 1

check_whitespace() {
  local file="$1" line
  line=$(grep -n '[[:space:]]$' "$file" | head -1 | cut -d: -f1)
  [ -z "$line" ] || err "$file:$line" "trailing whitespace (rule 10)"
  [ -z "$(tail -c 1 "$file")" ] || err "$file" "no final newline (rule 10)"
}

for meta in "$TEMPLATE" "$INDEX" "$DIR/MAINTENANCE.md" "$DIR/README.md"; do
  check_whitespace "$meta"
done

want_h2=$(grep '^## ' "$TEMPLATE")

index_lines=$(wc -l < "$INDEX")
if [ "$index_lines" -gt "$MAX_INDEX_LINES" ]; then
  err "$INDEX" "$index_lines lines, cap is $MAX_INDEX_LINES (rule 6)"
fi

claimed=""
found_doc=0

for doc in "$DIR"/docs/*.md; do
  [ -e "$doc" ] || break
  found_doc=1
  name="${doc##*/}"

  lines=$(wc -l < "$doc")
  if [ "$lines" -gt "$MAX_LINES" ]; then
    err "$doc" "$lines lines, cap is $MAX_LINES - split it (rule 6)"
  fi

  check_whitespace "$doc"

  if [ "$(head -1 "$doc")" != "---" ]; then
    err "$doc:1" "must open with a --- frontmatter block (rule 5)"
  fi

  scope_line=$(sed -n '2,10{/^scope:/p;}' "$doc")
  [ -n "$scope_line" ] || err "$doc" "frontmatter has no scope: (rule 14)"
  [ -n "$(sed -n '2,10{/^verified-at:/p;}' "$doc")" ] ||
    err "$doc" "frontmatter has no verified-at: (rule 13)"

  got_h2=$(grep '^## ' "$doc")
  if [ "$got_h2" != "$want_h2" ]; then
    err "$doc" "## headings must match _TEMPLATE.md exactly and in order (rule 5)"
  fi

  long_fence=$(awk -v max="$MAX_FENCE" '
    /^```/ {
      if (open) { if (NR - start - 1 > max) print start; open = 0 }
      else { open = 1; start = NR }
    }
  ' "$doc" | head -1)
  [ -z "$long_fence" ] ||
    err "$doc:$long_fence" "code fence over $MAX_FENCE lines - cite the file instead (rule 8)"

  lineref=$(grep -nE '\.(go|ts|vue|tmpl|css|md|yml|json):[0-9]+' "$doc" | head -1 | cut -d: -f1)
  [ -z "$lineref" ] ||
    err "$doc:$lineref" "cites a line number - use the symbol name instead (rule 8)"

  abspath=$(grep -nE '(^|[[:space:]`(])(/home/|/Users/|/root/|[A-Za-z]:\\)' "$doc" |
    head -1 | cut -d: -f1)
  [ -z "$abspath" ] || err "$doc:$abspath" "absolute path - use a repo-relative one (rule 9)"

  grep -q "docs/$name" "$INDEX" || err "$INDEX" "no row pointing at docs/$name (rule 14)"

  while read -r path; do
    [ -n "$path" ] || continue
    [ -e "$path" ] || err "$doc" "scope path does not exist: $path (rule 7)"
    owner=$(printf '%s' "$claimed" | awk -F'|' -v p="$path" '$1 == p { print $2; exit }')
    if [ -n "$owner" ]; then
      err "$doc" "scope path $path is already owned by docs/$owner (rule 2)"
    else
      claimed="${claimed}${path}|${name}
"
    fi
  done < <(printf '%s\n' "${scope_line#scope:}" | tr ',' '\n' |
    sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
done

[ "$found_doc" -eq 1 ] || err "$DIR/docs" "contains no docs"

while read -r ref; do
  [ -f "$DIR/$ref" ] || err "$INDEX" "points at $ref, which does not exist (rule 14)"
done < <(grep -oE 'docs/[a-z0-9.-]+\.md' "$INDEX" | sort -u)

# ---------------------------------------------------------------- guide/ (rules 16-22)
GUIDE_TEMPLATE="$DIR/guide/_TEMPLATE.md"
MAX_GUIDE_LINES=200
MAX_GUIDE_FENCE=30
guide_count=0

if [ -f "$GUIDE_TEMPLATE" ]; then
  want_guide_h2=$(grep '^## ' "$GUIDE_TEMPLATE")
  check_whitespace "$GUIDE_TEMPLATE"

  for g in "$DIR"/guide/*.md; do
    [ -e "$g" ] || break
    gname="${g##*/}"
    case "$gname" in _TEMPLATE.md) continue ;; esac

    check_whitespace "$g"

    lines=$(wc -l < "$g")
    if [ "$lines" -gt "$MAX_GUIDE_LINES" ]; then
      err "$g" "$lines lines, cap is $MAX_GUIDE_LINES (rule 20)"
    fi

    long_fence=$(awk -v max="$MAX_GUIDE_FENCE" '
      /^```/ {
        if (open) { if (NR - start - 1 > max) print start; open = 0 }
        else { open = 1; start = NR }
      }
    ' "$g" | head -1)
    [ -z "$long_fence" ] ||
      err "$g:$long_fence" "code fence over $MAX_GUIDE_FENCE lines (rule 20)"

    # START-HERE.md has no source doc: exempt from rules 17, 18, 21 (rule 22)
    case "$gname" in START-HERE.md) continue ;; esac

    got_guide_h2=$(grep '^## ' "$g")
    if [ "$got_guide_h2" != "$want_guide_h2" ]; then
      err "$g" "## headings must match guide/_TEMPLATE.md exactly and in order (rule 19)"
    fi

    src=$(sed -n '2,10{/^source:/p;}' "$g" | sed 's/^source: *//')
    stored=$(sed -n '2,10{/^source-hash:/p;}' "$g" | sed 's/^source-hash: *//')
    [ -n "$(sed -n '2,10{/^verified-at:/p;}' "$g")" ] ||
      err "$g" "frontmatter has no verified-at: (rule 16)"
    if [ -z "$src" ] || [ -z "$stored" ]; then
      err "$g" "frontmatter needs source: and source-hash: (rule 16)"
      continue
    fi
    if [ ! -f "$DIR/$src" ]; then
      err "$g" "source: $src does not exist (rule 17)"
      continue
    fi
    if [ "${src##*/}" != "$gname" ]; then
      err "$g" "source: $src must have the same filename as the guide (rule 17)"
    fi

    actual=$(sha256sum "$DIR/$src" | cut -c1-16)
    if [ "$actual" != "$stored" ]; then
      err "$g" "DRIFT: $src changed since this guide was written (hash $stored, now $actual) - regenerate it (rule 18)"
    fi

    # rule 21: a guide may only cite paths its source doc already cites.
    # The doc's citations are its backticked tokens plus its scope: paths; trailing
    # slashes are normalised, and citing a directory whose files the doc cites is allowed.
    src_paths=$({ grep -oE '`[A-Za-z0-9_./-]+`' "$DIR/$src" | tr -d '`'
                  sed -n '2,10{/^scope:/p;}' "$DIR/$src" | sed 's/^scope: *//' | tr ',' '\n'
                } | sed 's/^[[:space:]]*//; s/[[:space:]]*$//; s|/*$||' | sort -u)
    while read -r path; do
      [ -n "$path" ] || continue
      printf '%s\n' "$src_paths" | grep -qxF "$path" && continue
      printf '%s\n' "$src_paths" | grep -qE "^${path//./\\.}/" && continue
      err "$g" "cites $path, which $src does not - the guide adds explanation, not facts (rule 21)"
    done < <(grep -oE '`[A-Za-z0-9_./-]+`' "$g" | tr -d '`' | sed 's|/*$||' | sort -u |
      while read -r c; do
        case "$c" in */*) [ -e "$c" ] && printf '%s\n' "$c" ;; esac
      done)

    guide_count=$((guide_count + 1))
  done
fi

if [ "$fail" -eq 0 ]; then
  ndocs=$(find "$DIR/docs" -name '*.md' | wc -l | tr -d ' ')
  printf 'check.sh: ok (%s docs, guide %s/%s)\n' "$ndocs" "$guide_count" "$ndocs"
fi
exit "$fail"
