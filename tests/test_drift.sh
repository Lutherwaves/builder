#!/usr/bin/env bash
# Test skills/harness/drift.sh with a fake codebase-memory-mcp binary.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

# A fake binary: every index_status call says the graph is ready.
printf '#!/bin/sh\necho %s\n' "'{\"status\":\"ready\"}'" > "$tmp/cbm"; chmod +x "$tmp/cbm"
export HOME="$tmp/home" CBM_BIN="$tmp/cbm"
repo="$tmp/repo"; mkdir -p "$repo" "$HOME/.claude/code-graph" "$HOME/.cache/codebase-memory-mcp"
git -C "$repo" init -q; git -C "$repo" config user.email t@example.invalid; git -C "$repo" config user.name t
echo a > "$repo/a.txt"; git -C "$repo" add a.txt; git -C "$repo" commit -qm init
repo="$(cd "$repo" && pwd -P)"; sha="$(git -C "$repo" rev-parse HEAD)"
printf '%s\tproj\t%s\n' "$repo" "$sha" > "$HOME/.claude/code-graph/repos.txt"
db="$HOME/.cache/codebase-memory-mcp/proj.db"
drift() { bash "$root/skills/harness/drift.sh" "$repo"; }

touch -t 203001010000 "$db"   # the graph is newer than every file
out="$(drift)"
grep -q '^VERDICT=FRESH project=proj' <<<"$out" && check "fresh graph is FRESH" ok || check "fresh graph is FRESH ($out)" bad

touch -t 202001010000 "$db"   # a tracked file changed after the graph was built
out="$(drift)"
grep -q '^VERDICT=STALE project=proj reason=edits' <<<"$out" && check "edit after index is STALE edits" ok || check "edit after index is STALE edits ($out)" bad

touch -t 203001010000 "$db"; echo b >> "$repo/a.txt"; git -C "$repo" commit -qam two; touch -t 202501010000 "$repo/a.txt"
out="$(drift)"
grep -q '^VERDICT=STALE project=proj reason=commits ' <<<"$out" && check "new commit is STALE commits" ok || check "new commit is STALE commits ($out)" bad

out="$(CBM_BIN="$tmp/none" PATH=/usr/bin:/bin bash "$root/skills/harness/drift.sh" "$repo")"
[ "$out" = "VERDICT=NO_BINARY" ] && check "missing binary is NO_BINARY" ok || check "missing binary is NO_BINARY ($out)" bad

exit "$fails"
