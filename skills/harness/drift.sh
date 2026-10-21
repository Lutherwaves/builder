#!/usr/bin/env bash
# drift.sh — deterministic staleness verdict for ONE indexed repo. Prints
# EXACTLY ONE machine-readable line and never aborts mid-way (a report script
# must always return a verdict, even when a probe comes back empty).
#
# Usage: drift.sh [repo-path]   (default: $PWD)
# Output: VERDICT=<FRESH|STALE|NOT_INDEXED|NO_BINARY|BAD_PATH> project=<name> ...
#
# The registry written by setup.sh (~/.claude/code-graph/repos.txt, tab-
# separated: <abs-path>\t<project>\t<indexed-sha>) is the source of truth:
#   - <project>     the AUTHORITATIVE name the tool assigned (paths get
#                   normalized in ways a naive slug can't reproduce).
#   - <indexed-sha> the git HEAD *at index time* — the only reliable way to
#                   detect COMMITTED drift, since the tool exposes neither an
#                   index-time sha (its head_sha is the live HEAD) nor a
#                   graph-vs-disk diff (detect_changes only sees git dirtiness).
set -u

REGISTRY="${HOME}/.claude/code-graph/repos.txt"

find_bin() {
  if [ -n "${CBM_BIN:-}" ] && [ -x "${CBM_BIN}" ]; then printf '%s' "${CBM_BIN}"; return 0; fi
  command -v codebase-memory-mcp 2>/dev/null && return 0
  for c in "${HOME}/dev/codebase-memory-mcp/build/c/codebase-memory-mcp" \
           "${HOME}/.local/bin/codebase-memory-mcp"; do
    [ -x "$c" ] && { printf '%s' "$c"; return 0; }
  done
  return 1
}
BIN="$(find_bin || true)"
[ -n "${BIN:-}" ] && [ -x "$BIN" ] || { echo "VERDICT=NO_BINARY"; exit 0; }

REPO="$(cd "${1:-$PWD}" 2>/dev/null && pwd || true)"
[ -n "$REPO" ] || { echo "VERDICT=BAD_PATH repo=${1:-$PWD}"; exit 0; }

# Resolve project + index-time sha from the registry (authoritative).
PROJECT="" ; STORED=""
if [ -f "$REGISTRY" ]; then
  PROJECT="$(awk -F'\t' -v p="$REPO" '$1==p{print $2; exit}' "$REGISTRY")"
  STORED="$(awk -F'\t'  -v p="$REPO" '$1==p{print $3; exit}' "$REGISTRY")"
fi
[ -n "$PROJECT" ] || PROJECT="$(printf '%s' "$REPO" | sed 's#^/##; s#/#-#g')"

# Confirm the graph exists. Extract the field directly — do NOT pre-filter by
# line: index_status returns one huge JSON line that can legitimately contain a
# word like "supervisor" in a file path, and a line-filter would nuke the whole
# payload. A field-specific match ignores the tool's log lines on its own.
STATUS="$("$BIN" cli index_status "{\"project\":\"$PROJECT\"}" 2>/dev/null \
  | grep -oE '"status":"[a-z_]+"' | head -1 | cut -d'"' -f4 || true)"
if [ "${STATUS:-}" != "ready" ]; then
  echo "VERDICT=NOT_INDEXED project=$PROJECT repo=$REPO"; exit 0
fi

# Drift = committed (index-time sha ≠ live HEAD) and/or edited since index.
# A dirty tree is NOT drift by itself — the graph was built from the working
# tree, dirty files included. What matters is whether any tracked file changed
# AFTER the graph was built, so compare tracked-file mtimes to the graph DB's
# mtime (a stable index-time marker: read queries don't touch it).
LIVE="$(git -C "$REPO" rev-parse HEAD 2>/dev/null || true)"
DB="${HOME}/.cache/codebase-memory-mcp/${PROJECT}.db"
DBMTIME="$(stat -c %Y "$DB" 2>/dev/null || echo 0)"
NEWEST="$(git -C "$REPO" ls-files -z 2>/dev/null \
  | ( cd "$REPO" 2>/dev/null && xargs -0 -r stat -c %Y 2>/dev/null ) | sort -rn | head -1 || true)"
NEWEST="${NEWEST:-0}"

REASON=""
if [ -n "$STORED" ] && [ -n "$LIVE" ] && [ "$STORED" != "$LIVE" ]; then REASON="commits"; fi
if [ "$NEWEST" -gt "$DBMTIME" ]; then REASON="${REASON:+$REASON+}edits"; fi

if [ -z "$REASON" ]; then
  echo "VERDICT=FRESH project=$PROJECT head=${LIVE:0:7}"; exit 0
fi
echo "VERDICT=STALE project=$PROJECT reason=$REASON indexed=${STORED:0:7} head=${LIVE:0:7} repo=$REPO"
