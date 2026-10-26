#!/usr/bin/env bash
# Parity: the Go binary gives the answers the Python gives, on the same inputs.
# Text output is compared byte for byte; JSON output is compared after jq -S.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
fx="$root/tests/parity"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
command -v go >/dev/null 2>&1 || { echo "  go: skipped (no go)"; exit 0; }
command -v jq >/dev/null 2>&1 || { echo "  ✗ parity needs jq"; exit 1; }
py="$(command -v python3)"
(cd "$root" && go build -o "$tmp/builder" ./cmd/builder) || { echo "  ✗ go build failed"; exit 1; }

# A PATH with no git, gh, ps or matchblox, so both languages see the same signals.
# The shim needs bash, sed, head and dirname; none of them is a signal.
mkdir -p "$tmp/path" "$tmp/mb" "$tmp/work"
for t in bash sed head dirname; do ln -s "$(command -v $t)" "$tmp/path/$t"; ln -s "$(command -v $t)" "$tmp/mb/$t"; done
printf '#!/bin/sh\n/bin/cat "%s/usage.json"\n' "$fx" > "$tmp/mb/matchblox"; chmod +x "$tmp/mb/matchblox"

n=0
# side <lang> <home> <cmd...>: run one side in its own home; prints stdout, then "rc=<n>".
side() {
  local lang="$1" home="$2"; shift 2
  mkdir -p "$home"
  ( cd "$tmp/work" && env -i HOME="$home" BUILDER_HOME="$home/b" PATH="${XPATH:-$tmp/path}" TZ=UTC \
      BUILDER_BIN="$tmp/builder" "$@" ) 2>/dev/null
  echo "rc=$?"
}
gocmd() { echo "$root/bin/builder"; }

# same <kind:text|json> <name> <python argv...> -- <go argv...>
same() {
  local kind="$1" name="$2"; shift 2
  local pyargs=() goargs=()
  while [ "$1" != "--" ]; do pyargs+=("$1"); shift; done; shift
  goargs=("$@")
  n=$((n + 1))
  local a b
  a="$(side py "$tmp/h$n-py" "$py" "${pyargs[@]}")"
  b="$(side go "$tmp/h$n-go" "$root/bin/builder" "${goargs[@]}")"
  a="${a//$tmp\/h$n-py/<home>}"; b="${b//$tmp\/h$n-go/<home>}"
  if [ "$kind" = json ]; then
    local ra="${a##*rc=}" rb="${b##*rc=}"
    a="$(printf '%s' "${a%rc=*}" | jq -S . 2>&1)rc=$ra"; b="$(printf '%s' "${b%rc=*}" | jq -S . 2>&1)rc=$rb"
  fi
  if [ "$a" = "$b" ]; then check "$name" ok; else check "$name" bad; diff <(echo "$a") <(echo "$b") | head -20; fi
}

echo "budget"
same text "budget positional" "$root/skills/budget/burn-proj.py" 19 1783504800 1782990571 -- budget 19 1783504800 1782990571
XPATH="$tmp/mb" same text "budget --from-matchblox" "$root/skills/budget/burn-proj.py" --from-matchblox 1782990571 -- budget --from-matchblox 1782990571

echo "security"
same text "security cvss" "$root/lib/security.py" cvss CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H -- security cvss CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H
same text "security cvss bad vector" "$root/lib/security.py" cvss CVSS:3.1/AV:N -- security cvss CVSS:3.1/AV:N
same json "security score" "$root/lib/security.py" score "$fx/findings.json" -- security score "$fx/findings.json"
for lang in py go; do
  if [ $lang = py ]; then cmd=("$py" "$root/lib/security.py"); else cmd=("$root/bin/builder" security); fi
  side $lang "$tmp/sec-$lang" "${cmd[@]}" baseline save "$fx/findings.json" --today 2026-10-01 >/dev/null
  side $lang "$tmp/sec-$lang" "${cmd[@]}" baseline diff "$fx/findings.json" --today 2026-10-08 > "$tmp/sec-$lang.out"
done
sa="$(head -n -1 "$tmp/sec-py.out" | jq -S .)"; sb="$(head -n -1 "$tmp/sec-go.out" | jq -S .)"
[ "$(tail -n 1 "$tmp/sec-py.out")" = rc=0 ] && [ "$(tail -n 1 "$tmp/sec-go.out")" = rc=0 ] && [ -n "$sa" ] && [ "$sa" = "$sb" ] \
  && check "security baseline save then diff" ok || check "security baseline save then diff" bad
ba="$(jq -S . "$tmp/sec-py/b/security/"*.json 2>/dev/null)"; bb="$(jq -S . "$tmp/sec-go/b/security/"*.json 2>/dev/null)"
[ -n "$ba" ] && [ "$ba" = "$bb" ] && check "security baseline file" ok || check "security baseline file" bad

echo "reach"
same json "reach check teammate" "$root/lib/reach.py" check --to teammate --kind waiting --text hi --recipient Ana -- reach check --to teammate --kind waiting --text hi --recipient Ana
same json "reach check self" "$root/lib/reach.py" check --to self --kind waiting --text hi -- reach check --to self --kind waiting --text hi
same text "reach channel needs a room" "$root/lib/reach.py" check --to channel --kind x --text hi -- reach check --to channel --kind x --text hi

echo "reach files across languages"
reach_run() { local lang="$1" home="$2"; shift 2
  if [ "$lang" = py ]; then side py "$home" "$py" "$root/lib/reach.py" "$@"; else side go "$home" "$root/bin/builder" reach "$@"; fi; }
for w in py go; do
  r=go; [ $w = go ] && r=py
  # Python's casefold folds both sigmas to σ: a message logged as ΟΔΟΣ is a duplicate of οδος.
  reach_run $w "$tmp/r-$w" log --to teammate --recipient Ana --kind fyi --text "Build ΟΔΟΣ is green" --verdict send >/dev/null
  out="$(reach_run $r "$tmp/r-$w" check --to teammate --recipient ana --kind fyi --text "build  οδος is GREEN")"
  [ "$(printf '%s' "${out%rc=*}" | jq -r .reason)" = duplicate ] && check "$r dedupes a send that $w logged" ok || check "$r dedupes a send that $w logged ($out)" bad
  reach_run $w "$tmp/c-$w" channel Slack >/dev/null
  printf '{"to_teammate": {"allow": ["fyi"]}}' > "$tmp/c-$w.tmp"
  jq -s '.[0] * .[1]' "$tmp/c-$w/b/reach.json" "$tmp/c-$w.tmp" > "$tmp/c-$w/b/reach.json.new" && mv "$tmp/c-$w/b/reach.json.new" "$tmp/c-$w/b/reach.json"
  reach_run $r "$tmp/c-$w" channel teams >/dev/null
  out="$(reach_run $w "$tmp/c-$w" check --to teammate --recipient Ana --kind fyi --text hi)"
  [ "$(jq -r .channel "$tmp/c-$w/b/reach.json")" = teams ] && [ "$(printf '%s' "${out%rc=*}" | jq -r .verdict)" = send ] \
    && check "reach.json: $r keeps what $w wrote" ok || check "reach.json: $r keeps what $w wrote" bad
done

echo "ledger and hook"
same json "ledger next, empty home" "$root/lib/ledger.py" next --harness claude -- ledger next --harness claude
same json "ledger next, codex" "$root/lib/ledger.py" next --harness codex -- ledger next --harness codex
for h in claude cursor opencode codex; do
  a="$(side py "$tmp/hook-py-$h" "$py" "$root/hooks/session_start.py" --harness $h --format text)"
  b="$(side go "$tmp/hook-go-$h" "$root/bin/builder" hook --harness $h --format text)"
  a="${a//python3 \"$root\/lib\/ledger.py\"/\"$root/bin/builder\" ledger}"
  [ "$a" = "$b" ] && check "hook text, $h" ok || { check "hook text, $h" bad; diff <(echo "$a") <(echo "$b") | head -10; }
done
a="$(side py "$tmp/hookj-py" "$py" "$root/hooks/session_start.py" --harness cursor)"
b="$(side go "$tmp/hookj-go" "$root/bin/builder" hook --harness cursor)"
a="${a%rc=*}"; b="${b%rc=*}"
[ "$(jq -r .additional_context <<<"$a" | sed "s|python3 \"$root/lib/ledger.py\"|\"$root/bin/builder\" ledger|g")" = "$(jq -r .additional_context <<<"$b")" ] \
  && check "hook JSON, cursor" ok || check "hook JSON, cursor" bad

for lang in py go; do
  if [ $lang = py ]; then run=("$py" "$root/lib/ledger.py"); hk=("$py" "$root/hooks/session_start.py"); else run=("$root/bin/builder" ledger); hk=("$root/bin/builder" hook); fi
  side $lang "$tmp/on-$lang" "${run[@]}" onboarded >/dev/null
  side $lang "$tmp/on-$lang" "${hk[@]}" --harness claude --format text > "$tmp/on-$lang.out"
done
a="$(sed "s|python3 \"$root/lib/ledger.py\"|\"$root/bin/builder\" ledger|g" "$tmp/on-py.out")"
grep -q 'builder recommends:' "$tmp/on-go.out" && [ "$a" = "$(cat "$tmp/on-go.out")" ] \
  && check "hook text after onboarding" ok || { check "hook text after onboarding" bad; diff <(echo "$a") "$tmp/on-go.out" | head -10; }

echo "cross-writer ledger"
run_ledger() { local lang="$1" home="$2"; shift 2
  if [ "$lang" = py ]; then side py "$home" "$py" "$root/lib/ledger.py" "$@"; else side go "$home" "$root/bin/builder" ledger "$@"; fi; }
for w in py go; do for r in py go; do
  run_ledger $w "$tmp/x-$w$r" answer gh later >/dev/null
  run_ledger $w "$tmp/x-$w$r" answer superpowers never >/dev/null
  out="$(run_ledger $r "$tmp/x-$w$r" next --harness claude)"
  printf '%s' "${out%rc=*}" | jq -S . > "$tmp/x-$w$r.json"
done; done
[ "$(cat "$tmp/x-pygo.json")" = "$(cat "$tmp/x-pypy.json")" ] && check "Go reads a ledger Python wrote" ok || check "Go reads a ledger Python wrote" bad
[ "$(cat "$tmp/x-gopy.json")" = "$(cat "$tmp/x-gogo.json")" ] && check "Python reads a ledger Go wrote" ok || check "Python reads a ledger Go wrote" bad
[ "$(cat "$tmp/x-pypy.json")" = "$(cat "$tmp/x-gogo.json")" ] && check "both languages pick the same next step" ok || check "both languages pick the same next step" bad

echo "mcp-merge and support table"
# the Python merge that setup.sh used to carry, kept as the reference until the Python goes
heredoc=$(cat <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
if not isinstance(data, dict):
    sys.exit(1)
servers = data.get("mcpServers") if isinstance(data.get("mcpServers"), dict) else {}
servers["codebase-memory"] = {"command": sys.argv[2], "args": ["--ui=false"], "env": {}}
data["mcpServers"] = servers
print(json.dumps(data, indent=2))
PY
)
a="$(printf '%s\n' "$heredoc" | "$py" - "$fx/mcp.json" /bin/cbm | jq -S .)"
b="$("$tmp/builder" mcp-merge "$fx/mcp.json" /bin/cbm | jq -S .)"
[ -n "$a" ] && [ "$a" = "$b" ] && check "mcp-merge" ok || check "mcp-merge" bad
a="$(cd "$root" && "$py" scripts/support-table.py render)"
b="$(cd "$root" && go run ./cmd/support-table render)"
[ "$a" = "$b" ] && check "support-table render" ok || check "support-table render" bad

exit "$fails"
