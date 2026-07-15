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

# Confirm the graph actually exists (registry could be ahead of a wiped cache).
STATUS="$("$BIN" cli index_status "{\"project\":\"$PROJECT\"}" 2>/dev/null \
  | grep -vE 'mem.init|deprecated|supervisor' \
  | grep -oE '"status":"[a-z_]+"' | head -1 | cut -d'"' -f4 || true)"
if [ "${STATUS:-}" != "ready" ]; then
  echo "VERDICT=NOT_INDEXED project=$PROJECT repo=$REPO"; exit 0
fi

# Drift = committed (index-time sha ≠ live HEAD) and/or uncommitted (dirty tree).
LIVE="$(git -C "$REPO" rev-parse HEAD 2>/dev/null || true)"
DIRTY="$(git -C "$REPO" status --porcelain 2>/dev/null \
  | grep -vE '(\.mcp\.json|\.codebase-memory/)' | grep -c . || true)"
DIRTY="${DIRTY:-0}"

REASON=""
if [ -n "$STORED" ] && [ -n "$LIVE" ] && [ "$STORED" != "$LIVE" ]; then REASON="commits"; fi
if [ "$DIRTY" -gt 0 ]; then REASON="${REASON:+$REASON+}uncommitted:${DIRTY}"; fi

if [ -z "$REASON" ]; then
  echo "VERDICT=FRESH project=$PROJECT changed=0 head=${LIVE:0:7}"; exit 0
fi
echo "VERDICT=STALE project=$PROJECT reason=$REASON indexed=${STORED:0:7} head=${LIVE:0:7} repo=$REPO"
