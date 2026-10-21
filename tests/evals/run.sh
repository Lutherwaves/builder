#!/usr/bin/env bash
# Skill eval runner. See tests/evals/README.md.
#   run.sh --check              validate every fixture (no API call)
#   run.sh <skill> [--dry-run]  run baseline and with-skill, side by side
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
evals="${BUILDER_EVAL_DIR:-$here}"

check() {
  local n=0 bad=0 d name bullets
  for d in "$evals"/*/; do
    [ -d "$d" ] || continue
    name="$(basename "$d")"; n=$((n + 1))
    [ -s "$d/prompt.md" ] || { echo "✗ $name: prompt.md is missing or empty" >&2; bad=$((bad + 1)); continue; }
    if [ ! -f "$d/expect.md" ]; then echo "✗ $name: expect.md is missing" >&2; bad=$((bad + 1)); continue; fi
    bullets="$(grep -cE '^[[:space:]]*[-*][[:space:]]+[^[:space:]]' "$d/expect.md")"
    if [ "$bullets" -lt 3 ] || [ "$bullets" -gt 6 ]; then
      echo "✗ $name: expect.md has $bullets bullet lines (need 3 to 6)" >&2; bad=$((bad + 1))
    fi
    if [ -e "$d/repo" ] && [ ! -d "$d/repo" ]; then
      echo "✗ $name: repo exists but is not a directory" >&2; bad=$((bad + 1))
    fi
  done
  [ "$bad" -eq 0 ] || return 1
  if [ "$n" -eq 0 ]; then echo "fixtures ok (0)"; else echo "fixtures ok"; fi
}

[ "${1:-}" = "--check" ] && { check; exit $?; }

skill="${1:-}"; dry=0
[ "${2:-}" = "--dry-run" ] && dry=1
if [ -z "$skill" ] || [ "${skill#-}" != "$skill" ] || [ ! -d "$evals/$skill" ]; then
  echo "usage: run.sh --check | run.sh <skill> [--dry-run]" >&2; exit 2
fi
fx="$evals/$skill"
for f in prompt.md expect.md; do [ -s "$fx/$f" ] || { echo "✗ $skill: $f is missing or empty (try --check)" >&2; exit 2; }; done

out="${BUILDER_EVAL_OUT:-${XDG_STATE_HOME:-$HOME/.local/state}/builder/evals}/$skill/$(date +%Y%m%d-%H%M%S)"
. "$here/lib.sh"
eval_base

show() { printf '%q ' "$@"; echo; }

if [ "$dry" -eq 1 ]; then
  echo "# transcripts: $out/{baseline,with}.md (stderr: .err)"
  echo "# env: BUILDER_HOME=<temp dir, onboarded, every recommendation answered never>"
  echo "# baseline (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}"
  echo "# with     (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}" --plugin-dir "$root"
  exit 0
fi

tmp="$(mktemp -d)"
[ -n "$tmp" ] && [ -d "$tmp" ] || { echo "eval: cannot make a temp dir" >&2; exit 1; }
eval_init
# Leave nothing behind, also when the run is interrupted.
trap 'forget_sessions "$tmp/baseline"; forget_sessions "$tmp/with"; rm -rf "$tmp"' EXIT
trap 'exit 130' INT TERM
mkdir -p "$out" || exit 1
eval_shims
workdir baseline && workdir with || { echo "✗ cannot prepare the work dirs" >&2; exit 1; }
eval_builder_home "$root" || exit 1
(cd "$tmp/baseline" && "${base[@]}" < "$fx/prompt.md" > "$out/baseline.md" 2> "$out/baseline.err") \
  || echo "note: baseline run exited non-zero (see $out/baseline.err)" >&2
forget_sessions "$tmp/baseline"
(cd "$tmp/with" && "${base[@]}" --plugin-dir "$root" < "$fx/prompt.md" > "$out/with.md" 2> "$out/with.err") \
  || echo "note: with run exited non-zero (see $out/with.err)" >&2
forget_sessions "$tmp/with"

for part in "expect.md:Expected" "$out/baseline.md:Baseline (no skill)" "$out/with.md:With skill"; do
  f="${part%%:*}"; [ "$f" = "expect.md" ] && f="$fx/expect.md"
  printf '\n===== %s =====\n' "${part#*:}"; cat "$f"
done
echo; echo "transcripts: $out"
