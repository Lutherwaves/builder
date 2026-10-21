#!/usr/bin/env bash
# setup.sh — provision codebase-memory-mcp for ONE repo: locate the binary,
# wire a project-scoped .mcp.json, index the tree, and register it for the
# staleness watcher. Idempotent — safe to re-run; re-runs refresh the index.
#
# Usage: setup.sh [repo-path]   (default: $PWD)
# Env:   CBM_BIN=/path/to/codebase-memory-mcp  to force a specific binary.
set -euo pipefail

STATE_DIR="${HOME}/.claude/code-graph"
REGISTRY="${STATE_DIR}/repos.txt"

# ── Locate the binary (never auto-compile — building is a 279MB C build the
#    user should opt into explicitly) ─────────────────────────────────────
find_bin() {
  if [ -n "${CBM_BIN:-}" ] && [ -x "${CBM_BIN}" ]; then printf '%s' "${CBM_BIN}"; return 0; fi
  if command -v codebase-memory-mcp >/dev/null 2>&1; then command -v codebase-memory-mcp; return 0; fi
  for c in \
    "${HOME}/dev/codebase-memory-mcp/build/c/codebase-memory-mcp" \
    "${HOME}/.local/bin/codebase-memory-mcp"; do
    [ -x "$c" ] && { printf '%s' "$c"; return 0; }
  done
  return 1
}

BIN="$(find_bin)" || {
  echo "error: codebase-memory-mcp binary not found." >&2
  echo "Build it from a source checkout (audited MIT, C):" >&2
  echo "  git clone https://github.com/DeusData/codebase-memory-mcp ~/dev/codebase-memory-mcp" >&2
  echo "  cd ~/dev/codebase-memory-mcp && scripts/build.sh   # needs gcc/g++/zlib" >&2
  echo "Then re-run, or set CBM_BIN=/path/to/binary." >&2
  exit 2
}

REPO="$(cd "${1:-$PWD}" && pwd)"
echo "→ binary: $BIN"
echo "→ repo:   $REPO"

# ── Wire a project-scoped .mcp.json (merge-safe — never clobber other servers)
MCP="${REPO}/.mcp.json"
if [ -f "$MCP" ] && command -v jq >/dev/null 2>&1; then
  tmp="$(mktemp)"
  jq --arg bin "$BIN" \
     '.mcpServers = (.mcpServers // {}) | .mcpServers["codebase-memory"] = {command:$bin, args:["--ui=false"], env:{}}' \
     "$MCP" > "$tmp" && mv "$tmp" "$MCP"
  echo "→ merged codebase-memory into existing .mcp.json"
else
  cat > "$MCP" <<EOF
{
  "mcpServers": {
    "codebase-memory": {
      "command": "$BIN",
      "args": ["--ui=false"],
      "env": {}
    }
  }
}
EOF
  echo "→ wrote $MCP"
fi

# ── Index the tree. No --name: the tool derives the project from the absolute
#    path, so every clone/worktree gets an ISOLATED graph automatically. ────
echo "→ indexing (moderate mode)…"
# Keep the raw output for error reporting; extract fields with specific matches
# (a line pre-filter would nuke a JSON payload that happens to contain a log
# keyword like "supervisor" in a file path).
OUT="$("$BIN" cli index_repository --repo-path "$REPO" --mode moderate 2>&1 || true)"
PROJECT="$(printf '%s' "$OUT" | grep -oE '"project":"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
NODES="$(printf '%s' "$OUT" | grep -oE '"nodes":[0-9]+' | head -1 | cut -d: -f2 || true)"
EDGES="$(printf '%s' "$OUT" | grep -oE '"edges":[0-9]+' | head -1 | cut -d: -f2 || true)"
[ -n "$PROJECT" ] || { echo "error: indexing failed:" >&2; printf '%s\n' "$OUT" >&2; exit 1; }
echo "→ indexed project '$PROJECT' — ${NODES:-?} nodes / ${EDGES:-?} edges"

# ── Register for the staleness watcher. Store the AUTHORITATIVE project name
#    and the index-time git HEAD (tab-separated) so drift.sh can detect both
#    committed and uncommitted drift without re-deriving anything. Dedup by path.
IDX_SHA="$(git -C "$REPO" rev-parse HEAD 2>/dev/null || true)"
mkdir -p "$STATE_DIR"
touch "$REGISTRY"
if [ -s "$REGISTRY" ]; then
  awk -F'\t' -v p="$REPO" '$1!=p' "$REGISTRY" > "${REGISTRY}.tmp" && mv "${REGISTRY}.tmp" "$REGISTRY"
fi
printf '%s\t%s\t%s\n' "$REPO" "$PROJECT" "$IDX_SHA" >> "$REGISTRY"
echo "→ registered in $REGISTRY ($(wc -l < "$REGISTRY" | tr -d ' ') repo(s) watched)"

echo "✓ done. In a Claude Code session rooted at $REPO, approve the 'codebase-memory' MCP server once, then ask structural questions instead of grepping."
