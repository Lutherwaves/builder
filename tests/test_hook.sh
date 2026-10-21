#!/usr/bin/env bash
# Test the SessionStart hook: first run, quiet when onboarded, no python3, speed.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
hook="$root/hooks/session-start"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
ctx() { jq -r '.hookSpecificOutput.additionalContext'; }
run() { HOME="$tmp/home" BUILDER_HOME="$tmp/bh" CLAUDE_PLUGIN_ROOT="$root" "$@"; }
mkdir -p "$tmp/home"

echo "test_first_run"
out="$(run "$hook")"
text="$(jq -er '.hookSpecificOutput.additionalContext' <<<"$out")" && check "valid JSON" ok || check "valid JSON" bad
[ "$(jq -r '.hookSpecificOutput.hookEventName' <<<"$out")" = SessionStart ] && check "event name" ok || check "event name" bad
grep -q 'builder onboarding' <<<"$text" && check "onboarding marker present" ok || check "onboarding marker present" bad
grep -q 'builder recommends' <<<"$text" && check "no recommendation block" bad || check "no recommendation block" ok
grep -q 'Build the right thing' <<<"$text" && check "soul present" ok || check "soul present" bad
grep -q 'ledger.py" onboarded' <<<"$text" && check "onboarded command named" ok || check "onboarded command named" bad
grep -qF "To find the next due step, run: python3 \"$root/lib/ledger.py\" next" <<<"$text" && check "next-step command named" ok || check "next-step command named" bad
grep -qF 'Never invent a step' <<<"$text" && check "no invented steps" ok || check "no invented steps" bad
grep -qF 'This is required, even when the request is one line.' <<<"$text" && check "intro is required" ok || check "intro is required" bad
grep -qF 'builder: you decide. I propose, build, and end each piece of work with a report.' <<<"$text" && check "literal intro given" ok || check "literal intro given" bad

echo "test_onboarded_quiet"
ids="$(jq -r '.[].id' "$root/lib/recommendations.json")"
{ printf '{"onboarded": true, "answers": {'; sep=''
  for i in $ids; do printf '%s"%s": {"verdict": "never", "at": 1}' "$sep" "$i"; sep=','; done
  printf '}}'; } > "$tmp/ledger.json"
mkdir -p "$tmp/bh"; cp "$tmp/ledger.json" "$tmp/bh/ledger.json"
text="$(run "$hook" | ctx)"
grep -q 'Build the right thing' <<<"$text" && check "soul present" ok || check "soul present" bad
grep -q 'builder recommends' <<<"$text" && check "no recommendation" bad || check "no recommendation" ok
grep -q 'builder onboarding' <<<"$text" && check "no onboarding" bad || check "no onboarding" ok
grep -qF 'Conventional Commit' <<<"$text" && grep -qF -- '--no-verify' <<<"$text" && grep -qF 'WIP' <<<"$text" && grep -qiF 'signed' <<<"$text" \
  && check "commit rules in the soul" ok || check "commit rules in the soul" bad

echo "test_recommendation_block"
printf '{"onboarded": true, "answers": {}}' > "$tmp/bh/ledger.json"
text="$(run "$hook" | ctx)"
grep -q 'builder recommends: ' <<<"$text" && grep -q 'ledger.py" answer ' <<<"$text" && check "block with answer command" ok || check "block with answer command" bad
[ "$(grep -c 'builder recommends' <<<"$text")" -eq 1 ] && check "at most one block" ok || check "at most one block" bad

echo "test_hook_without_python"
rm -f "$tmp/bh/ledger.json"
out="$(HOME="$tmp/home" BUILDER_HOME="$tmp/bh" CLAUDE_PLUGIN_ROOT="$root" PATH=/nonexistent /bin/bash "$hook")"
text="$(jq -er '.hookSpecificOutput.additionalContext' <<<"$out")" && check "valid JSON" ok || check "valid JSON" bad
grep -q 'Build the right thing' <<<"$text" && check "soul present" ok || check "soul present" bad

echo "test_without_plugin_root"
out="$(env -u CLAUDE_PLUGIN_ROOT HOME="$tmp/home" BUILDER_HOME="$tmp/bh" bash "$hook")"
jq -er '.hookSpecificOutput.additionalContext' <<<"$out" | grep -q 'Build the right thing' && check "works from \$0" ok || check "works from \$0" bad

echo "test_hook_fast"
mkdir -p "$tmp/fakebin"
printf '#!/bin/sh\nsleep 5\n' > "$tmp/fakebin/matchblox"; chmod +x "$tmp/fakebin/matchblox"
s=$(python3 -c 'import time;print(time.time())')
out="$(PATH="$tmp/fakebin:$PATH" run "$hook")"
e=$(python3 -c 'import time;print(time.time())')
jq -e . >/dev/null <<<"$out" && check "valid JSON" ok || check "valid JSON" bad
python3 -c "import sys;sys.exit(0 if $e-$s < 1.0 else 1)" && check "under 1 s" ok || check "under 1 s" bad

[ "$fails" -eq 0 ] && echo "✓ hook tests passed" || { echo "✗ hook tests failed"; exit 1; }
