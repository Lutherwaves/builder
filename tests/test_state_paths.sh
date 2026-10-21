#!/usr/bin/env bash
# Test that builder state lives in $BUILDER_HOME and the old ~/.claude place is still read.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

printf '#!/bin/sh\necho %s\n' "'{\"status\":\"ready\"}'" > "$tmp/cbm"; chmod +x "$tmp/cbm"
printf '#!/bin/sh\necho %s\n' "'{\"project\":\"proj\",\"nodes\":1,\"edges\":1}'" > "$tmp/cbm-index"; chmod +x "$tmp/cbm-index"
export CBM_BIN="$tmp/cbm"
unset BUILDER_HOME
repo="$tmp/repo"; mkdir -p "$repo"
git -C "$repo" init -q; git -C "$repo" config user.email t@example.invalid; git -C "$repo" config user.name t
echo a > "$repo/a.txt"; git -C "$repo" add a.txt; git -C "$repo" commit -qm init
repo="$(cd "$repo" && pwd -P)"; sha="$(git -C "$repo" rev-parse HEAD)"
echo b >> "$repo/a.txt"; git -C "$repo" commit -qam two   # live HEAD differs from $sha
fresh_home() { export HOME="$tmp/home-$1"; rm -rf "$HOME"; mkdir -p "$HOME/.cache/codebase-memory-mcp"; touch -t 203001010000 "$HOME/.cache/codebase-memory-mcp/proj.db"; }
drift() { bash "$root/skills/harness/drift.sh" "$repo"; }

fresh_home old
mkdir -p "$HOME/.claude/code-graph"; printf '%s\tproj\t%s\n' "$repo" "$sha" > "$HOME/.claude/code-graph/repos.txt"
out="$(drift)"
grep -q "^VERDICT=STALE project=proj reason=commits indexed=${sha:0:7}" <<<"$out" \
  && check "old_registry_only_is_read" ok || check "old_registry_only_is_read ($out)" bad

fresh_home both
mkdir -p "$HOME/.claude/code-graph" "$HOME/.config/builder/code-graph"
printf '%s\tproj\t%s\n' "$repo" "0000000000000000000000000000000000000000" > "$HOME/.claude/code-graph/repos.txt"
printf '%s\tnewproj\t%s\n' "$repo" "$(git -C "$repo" rev-parse HEAD)" > "$HOME/.config/builder/code-graph/repos.txt"
touch -t 203001010000 "$HOME/.cache/codebase-memory-mcp/newproj.db"
out="$(drift)"
grep -q '^VERDICT=FRESH project=newproj' <<<"$out" && check "new_wins_over_old" ok || check "new_wins_over_old ($out)" bad

fresh_home setup
mkdir -p "$HOME/.claude/code-graph"; printf '/elsewhere\tother\tabc\n' > "$HOME/.claude/code-graph/repos.txt"
cp "$HOME/.claude/code-graph/repos.txt" "$tmp/old-copy"
out="$(CBM_BIN="$tmp/cbm-index" bash "$root/skills/harness/setup.sh" "$repo" 2>&1)"; rc=$?
new="$HOME/.config/builder/code-graph/repos.txt"
[ "$rc" -eq 0 ] && grep -q "^$repo	proj	" "$new" && grep -q '^/elsewhere	other	abc$' "$new" && cmp -s "$tmp/old-copy" "$HOME/.claude/code-graph/repos.txt" \
  && check "setup_writes_new_keeps_old" ok || check "setup_writes_new_keeps_old ($rc: $out)" bad

fresh_home space
export BUILDER_HOME="$tmp/b h"
out="$(CBM_BIN="$tmp/cbm-index" bash "$root/skills/harness/setup.sh" "$repo" 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "^$repo	proj	" "$BUILDER_HOME/code-graph/repos.txt" && [ ! -e "$HOME/.config/builder" ] \
  && check "builder_home_with_space (setup)" ok || check "builder_home_with_space setup ($rc: $out)" bad
out="$(drift)"
grep -q '^VERDICT=FRESH project=proj' <<<"$out" \
  && check "builder_home_with_space (drift)" ok || check "builder_home_with_space drift ($out)" bad

exit "$fails"
