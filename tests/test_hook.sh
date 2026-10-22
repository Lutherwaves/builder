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

echo "test_next_command_names_the_harness"
for h in claude codex cursor opencode; do
  text="$(run "$hook" --harness "$h" --format text)"
  grep -qF "ledger.py\" next --harness $h" <<<"$text" && check "$h: next command carries --harness $h" ok || check "$h: next command carries --harness $h" bad
done
text="$(run "$hook" --harness unknown --format text)"
grep -qF 'ledger.py" next --harness claude' <<<"$text" && check "unknown: next command carries --harness claude" ok || check "unknown: next command carries --harness claude" bad

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
grep -q 'Build the right thing' <<<"$(jq -er '.hookSpecificOutput.additionalContext' <<<"$out")" && check "works from \$0" ok || check "works from \$0" bad

echo "test_hook_fast"
mkdir -p "$tmp/fakebin"
printf '#!/bin/sh\nsleep 5\n' > "$tmp/fakebin/matchblox"; chmod +x "$tmp/fakebin/matchblox"
s=$(python3 -c 'import time;print(time.time())')
out="$(PATH="$tmp/fakebin:$PATH" run "$hook")"
e=$(python3 -c 'import time;print(time.time())')
jq -e . >/dev/null <<<"$out" && check "valid JSON" ok || check "valid JSON" bad
python3 -c "import sys;sys.exit(0 if $e-$s < 1.0 else 1)" && check "under 1 s" ok || check "under 1 s" bad

echo "test_shape_cursor"
out="$(run "$hook" --harness cursor)"
jq -e '.additional_context' >/dev/null <<<"$out" && check "additional_context present" ok || check "additional_context present" bad
jq -e '.hookSpecificOutput' >/dev/null 2>&1 <<<"$out" && check "no hookSpecificOutput" bad || check "no hookSpecificOutput" ok
grep -q 'Build the right thing' <<<"$(jq -r '.additional_context' <<<"$out")" && check "soul present" ok || check "soul present" bad

echo "test_shape_codex"
out="$(run "$hook" --harness codex)"
grep -q 'Build the right thing' <<<"$(jq -er '.hookSpecificOutput.additionalContext' <<<"$out")" && check "hookSpecificOutput shape" ok || check "hookSpecificOutput shape" bad

echo "test_text_opencode"
out="$(run "$hook" --harness opencode)"
jq -e . >/dev/null 2>&1 <<<"$out" && check "not JSON" bad || check "not JSON" ok
grep -q 'Build the right thing' <<<"$out" && check "soul present" ok || check "soul present" bad
out="$(run "$hook" --harness claude --format text)"
jq -e . >/dev/null 2>&1 <<<"$out" && check "--format text on claude is plain" bad || check "--format text on claude is plain" ok

echo "test_cursor_inside_claude_env"
out="$(CLAUDECODE=1 CURSOR_VERSION=x run "$hook" --harness cursor)"
jq -e '.additional_context' >/dev/null <<<"$out" && check "cursor shape" ok || check "cursor shape" bad
out="$(CLAUDECODE=1 CURSOR_VERSION=x run "$hook" --harness auto)"
jq -e '.additional_context' >/dev/null <<<"$out" && check "auto detects cursor" ok || check "auto detects cursor" bad

echo "test_ledger_path_without_claude_root"
out="$(env -u CLAUDE_PLUGIN_ROOT HOME="$tmp/home" BUILDER_HOME="$tmp/bh" "$hook" --harness cursor)"
grep -qF "$root/lib/ledger.py" <<<"$(jq -r '.additional_context' <<<"$out")" && check "ledger path uses root" ok || check "ledger path uses root" bad

echo "test_no_python_each_shape"
nopy() { HOME="$tmp/home" BUILDER_HOME="$tmp/bh" PATH=/nonexistent /bin/bash "$hook" "$@"; }
for h in claude codex unknown auto; do
  out="$(nopy --harness $h)"
  grep -q 'Build the right thing' <<<"$(jq -er '.hookSpecificOutput.additionalContext' <<<"$out")" && check "$h: hookSpecificOutput" ok || check "$h: hookSpecificOutput" bad
done
out="$(nopy --harness cursor)"
grep -q 'Build the right thing' <<<"$(jq -er '.additional_context' <<<"$out")" && check "cursor: additional_context" ok || check "cursor: additional_context" bad
out="$(CURSOR_VERSION=x nopy --harness auto)"
grep -q 'Build the right thing' <<<"$(jq -er '.additional_context' <<<"$out")" && check "auto + CURSOR_VERSION: additional_context" ok || check "auto + CURSOR_VERSION: additional_context" bad
out="$(nopy --harness opencode)"
jq -e . >/dev/null 2>&1 <<<"$out" && check "opencode: plain text" bad || { grep -q 'Build the right thing' <<<"$out" && check "opencode: plain text" ok || check "opencode: plain text" bad; }
out="$(nopy --harness claude --format text)"
jq -e . >/dev/null 2>&1 <<<"$out" && check "--format text: plain text" bad || check "--format text: plain text" ok

echo "test_root_line"
text="$(run "$hook" --harness claude | ctx)"
grep -qF "builder root: $root. Where a builder skill says \${CLAUDE_PLUGIN_ROOT}, use this path." <<<"$text" && check "root line" ok || check "root line" bad

echo "test_mapping_block_max_15"
for h in claude codex cursor opencode; do
  f="$root/skills/builder/harnesses/$h.md"
  n="$(awk '/<!-- hook -->/{on=1;next} /<!-- \/hook -->/{on=0} on' "$f" 2>/dev/null | wc -l | tr -d ' ')"
  [ "$n" -ge 1 ] && [ "$n" -le 15 ] && check "$h block has $n lines" ok || check "$h block 1-15 lines (got $n)" bad
  block1="$(awk '/<!-- hook -->/{on=1;next} /<!-- \/hook -->/{on=0} on' "$f" 2>/dev/null | head -1)"
  [ -n "$block1" ] && grep -qF "$block1" <<<"$(run "$hook" --harness "$h")" && check "$h block in hook output" ok || check "$h block in hook output" bad
done

echo "test_speed_each_harness"
for h in claude codex cursor opencode unknown; do
  s=$(python3 -c 'import time;print(time.time())')
  run "$hook" --harness $h >/dev/null
  e=$(python3 -c 'import time;print(time.time())')
  python3 -c "import sys;sys.exit(0 if $e-$s < 1.0 else 1)" && check "$h under 1 s" ok || check "$h under 1 s" bad
done

echo "test_hook_manifests"
jq -e '.hooks.SessionStart[0].hooks[0].command == "\"${CLAUDE_PLUGIN_ROOT}/hooks/session-start\" --harness auto"' "$root/hooks/hooks.json" >/dev/null && check "hooks.json passes --harness auto" ok || check "hooks.json passes --harness auto" bad
jq -e '. == {"version":1,"hooks":{"sessionStart":[{"command":"./hooks/session-start --harness cursor"}]}}' "$root/hooks/hooks-cursor.json" >/dev/null && check "hooks-cursor.json exact" ok || check "hooks-cursor.json exact" bad

[ "$fails" -eq 0 ] && echo "✓ hook tests passed" || { echo "✗ hook tests failed"; exit 1; }
