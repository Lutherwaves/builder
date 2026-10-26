#!/usr/bin/env bash
# Test the install offer in the SessionStart hook when the binary is missing.
# Needs no Go and no jq. Runs under Bash 3.2.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
now="$(date +%s)"

# hook_run <plugin root> <answer or "-" for none> [hook args]: sets out, err, rc.
hook_run() {
  local plug="$1" answer="$2"; shift 2
  local bh="$tmp/bh"
  rm -rf "$bh"; mkdir -p "$bh"
  [ "$answer" = "-" ] || printf '%s\n' "$answer" > "$bh/install.answer"
  out="$(env -u BUILDER_BIN HOME="$tmp/home" BUILDER_HOME="$bh" PATH=/nonexistent /bin/bash "$plug/hooks/session-start" "$@" 2>"$tmp/stderr")"
  rc=$?
  err="$(<"$tmp/stderr")"
}
has() { grep -qF -- "$1" <<<"$out"; }
root_line() { echo "builder root: $1. Where a builder skill says \${CLAUDE_PLUGIN_ROOT}, use this path."; }
text=(--harness opencode --format text)

offer_ok() {
  has 'builder install: the helper binary is missing.' \
    && has "builder's helper binary is not installed. It runs the setup steps, send checks and forecasts. Install it now? yes, not now, or never." \
    && has "On yes, run: bash \"$1/scripts/install.sh\". It downloads from GitHub, so report it under ↗ Outside." \
    && has "On not now, run: \"$1/bin/builder\" install-answer later" \
    && has "On never, run: \"$1/bin/builder\" install-answer never" \
    && has 'Answer the person'"'"'s request first. At the end of that reply, ask this once, word for word:'
}
quiet_ok() { has 'Build the right thing' && has "$(root_line "$root")" && ! has 'builder install:' && [ "$rc" -eq 0 ] && [ -z "$err" ]; }

echo "no_answer_offers"
hook_run "$root" - "${text[@]}"
offer_ok "$root" && has "$(root_line "$root")" && has 'Build the right thing' && [ "$rc" -eq 0 ] && [ -z "$err" ] && check "offer block, root line, quiet" ok || check "offer block, root line, quiet (rc=$rc err=$err)" bad

echo "later_recent_quiet"
hook_run "$root" "later $((now - 60))" "${text[@]}"
quiet_ok && check "no block" ok || check "no block" bad

echo "later_old_offers"
hook_run "$root" "later $((now - 604800))" "${text[@]}"
offer_ok "$root" && check "offer after 7 days" ok || check "offer after 7 days" bad

echo "never_quiet"
hook_run "$root" "never $((now - 60))" "${text[@]}"
quiet_ok && check "no block" ok || check "no block" bad

echo "yes_says_install"
hook_run "$root" "yes $((now - 60))" "${text[@]}"
has 'builder install: the helper binary of this plugin version is missing, and the person said yes to it before.' \
  && has "In your first reply, run: bash \"$root/scripts/install.sh\". Report it under ↗ Outside." \
  && ! has 'ask this once' && has "$(root_line "$root")" && [ -z "$err" ] && check "yes block" ok || check "yes block" bad

echo "corrupt_answer_offers"
for a in "" "later" "later abc" "later 08" "later 0" "later $((now + 86400))" "maybe $now"; do
  hook_run "$root" "$a" "${text[@]}"
  offer_ok "$root" && [ -z "$err" ] && check "offer for '$a'" ok || check "offer for '$a'" bad
done

echo "spaced_root"
spaced="$tmp/plug in"
mkdir -p "$spaced"
cp -R "$root/hooks" "$root/bin" "$root/skills" "$root/.claude-plugin" "$spaced/"
hook_run "$spaced" - "${text[@]}"
offer_ok "$spaced" && has "$(root_line "$spaced")" && check "commands quote the spaced path" ok || check "commands quote the spaced path" bad

echo "claude_json_offer"
hook_run "$root" - --harness claude
has '"additionalContext"' && has 'install-answer later' && [ -z "$err" ] && check "escaped JSON holds the offer" ok || check "escaped JSON holds the offer" bad

exit "$fails"
