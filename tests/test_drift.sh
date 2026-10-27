#!/usr/bin/env bash
# Test skills/harness/drift.sh with a fake codebase-memory-mcp binary.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

# A fake binary: every index_status call says the graph is ready.
printf '#!/bin/sh\necho %s\n' "'{\"status\":\"ready\"}'" > "$tmp/cbm"; chmod +x "$tmp/cbm"
if [ -z "${BUILDER_BIN:-}" ]; then
  (cd "$root" && go build -o "$tmp/builder" ./cmd/builder) || { echo "go build failed"; exit 1; }
  export BUILDER_BIN="$tmp/builder"
fi
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

# setup.sh merges into an existing .mcp.json: with jq, with the builder binary only, and refuses with neither.
printf '#!/bin/sh\necho %s\n' "'{\"project\":\"proj\",\"nodes\":1,\"edges\":1}'" > "$tmp/cbm-index"; chmod +x "$tmp/cbm-index"
toolbin() { # $1 = dir; then the tools to link into it
  local d="$1" t; shift; mkdir -p "$d"
  for t in bash dirname mktemp mv cat grep head cut git awk mkdir touch wc tr rm cp "$@"; do ln -sf "$(command -v "$t")" "$d/$t"; done
}
toolbin "$tmp/tb-jq" jq; toolbin "$tmp/tb-bin"; toolbin "$tmp/tb-none"
setup() { # $1 = repo, $2 = PATH dir, then env assignments
  local r="$1" p="$2"; shift 2
  env -u BUILDER_BIN CBM_BIN="$tmp/cbm-index" PATH="$p" "$@" /bin/bash "$root/skills/harness/setup.sh" "$r" 2>&1
}
for kind in jq bin; do
  r="$tmp/mcp-$kind"; mkdir -p "$r"; echo '{"mcpServers": {"other": {"command": "/x/other"}}}' > "$r/.mcp.json"
  if [ "$kind" = bin ]; then out="$(setup "$r" "$tmp/tb-bin" BUILDER_BIN="$BUILDER_BIN")"; rc=$?; else out="$(setup "$r" "$tmp/tb-jq")"; rc=$?; fi
  [ "$rc" -eq 0 ] && jq -e --arg b "$tmp/cbm-index" '.mcpServers.other == {"command":"/x/other"} and .mcpServers["codebase-memory"].command == $b' "$r/.mcp.json" >/dev/null \
    && check "setup.sh keeps other servers ($kind)" ok || check "setup.sh keeps other servers ($kind: $rc $out)" bad
done
r="$tmp/mcp-order"; mkdir -p "$r"; echo '{"z":1,"mcpServers":{"other":{"command":"/x/other"}}}' > "$r/.mcp.json"
out="$(setup "$r" "$tmp/tb-bin" BUILDER_BIN="$BUILDER_BIN")"; rc=$?
[ "$rc" -eq 0 ] && [ "$(jq -r 'keys_unsorted|join(",")' "$r/.mcp.json")" = "z,mcpServers" ] \
  && check "setup.sh keeps key order (bin)" ok || check "setup.sh keeps key order (bin: $rc $out)" bad
r="$tmp/mcp-none"; mkdir -p "$r"; echo '{"mcpServers": {"other": {"command": "/x/other"}}}' > "$r/.mcp.json"; before="$(cat "$r/.mcp.json")"
out="$(setup "$r" "$tmp/tb-none" BUILDER_HOME="$tmp/empty-home")"; rc=$?
[ "$rc" -ne 0 ] && [ "$(cat "$r/.mcp.json")" = "$before" ] && grep -q 'install jq or the builder binary' <<<"$out" \
  && check "setup.sh refuses to overwrite .mcp.json with neither jq nor the builder binary" ok || check "setup.sh refuses ($rc: $out)" bad
r="$tmp/mcp-bad"; mkdir -p "$r"; echo '{not json' > "$r/.mcp.json"
out="$(setup "$r" "$tmp/tb-bin" BUILDER_BIN="$BUILDER_BIN")"; rc=$?
[ "$rc" -ne 0 ] && [ "$(cat "$r/.mcp.json")" = '{not json' ] \
  && check "setup.sh leaves a broken .mcp.json alone" ok || check "setup.sh broken .mcp.json ($rc: $out)" bad

exit "$fails"
