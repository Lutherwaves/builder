#!/usr/bin/env bash
# Skill eval runner. See tests/evals/README.md.
#   run.sh --check              validate every fixture (no API call)
#   run.sh [--harness claude|codex|cursor|opencode] <skill> [--dry-run]
#                               run baseline and with-builder, side by side (default harness: claude)
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
evals="${BUILDER_EVAL_DIR:-$here}"

check() {
  local n=0 bad=0 d name bullets
  for d in "$evals"/*/; do
    [ -d "$d" ] || continue
    name="$(basename "$d")"
    [ "$name" = results ] && continue   # graded results (support-table.py), not a fixture
    n=$((n + 1))
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

usage() { echo "usage: run.sh --check | run.sh [--harness claude|codex|cursor|opencode] <skill> [--dry-run]" >&2; exit 2; }
skill=""; dry=0; harness=claude
while [ $# -gt 0 ]; do
  case "$1" in
    --harness) [ $# -ge 2 ] || usage; harness="$2"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    -*) usage ;;
    *) [ -z "$skill" ] || usage; skill="$1"; shift ;;
  esac
done
case "$harness" in claude|codex|cursor|opencode) ;; *) usage ;; esac
if [ -z "$skill" ] || [ ! -d "$evals/$skill" ]; then usage; fi
fx="$evals/$skill"
for f in prompt.md expect.md; do [ -s "$fx/$f" ] || { echo "✗ $skill: $f is missing or empty (try --check)" >&2; exit 2; }; done

out="${BUILDER_EVAL_OUT:-${XDG_STATE_HOME:-$HOME/.local/state}/builder/evals}/$skill/$(date +%Y%m%d-%H%M%S)"
. "$here/lib.sh"
"eval_base_$harness"

show() { printf '%q ' "$@"; echo; }

if [ "$dry" -eq 1 ]; then
  echo "# transcripts: $out/{baseline,with}.md (stderr: .err)"
  echo "# env: BUILDER_HOME=<temp dir, onboarded, every recommendation answered never>"
  "eval_notes_$harness"
  echo "# baseline (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}"
  echo "# with     (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}" ${with_extra[@]+"${with_extra[@]}"}
  exit 0
fi

tmp="$(mktemp -d)"
[ -n "$tmp" ] && [ -d "$tmp" ] || { echo "eval: cannot make a temp dir" >&2; exit 1; }
eval_init
# Only Claude Code keeps a session folder named after the work dir.
forget() { if [ "$harness" = claude ]; then forget_sessions "$1"; fi; }
# Leave nothing behind, also when the run is interrupted.
trap 'forget "$tmp/baseline"; forget "$tmp/with"; rm -rf "$tmp"' EXIT
trap 'exit 130' INT TERM
mkdir -p "$out" || exit 1
if [ "$harness" = claude ]; then eval_shims; else eval_shims strict; fi
workdir baseline && workdir with || { echo "✗ cannot prepare the work dirs" >&2; exit 1; }
eval_builder_home "$root" || exit 1
version="$(eval_version "${base[0]}")"

# run_one <baseline|with>: prepare that run's config, then run it in its own work dir.
run_one() {
  local mode="$1"
  "eval_prepare_$harness" "$mode" "$tmp/$mode" || return 1
  local cmd=("${base[@]}")
  if [ "$mode" = with ]; then cmd+=(${with_extra[@]+"${with_extra[@]}"}); fi
  (cd "$tmp/$mode" && env ${runenv[@]+"${runenv[@]}"} "${cmd[@]}" < "$fx/prompt.md" > "$out/$mode.md" 2> "$out/$mode.err") \
    || echo "note: $mode run exited non-zero (see $out/$mode.err)" >&2
  forget "$tmp/$mode"
}
run_one baseline || exit 1
run_one with || exit 1
eval_verdict "$harness" "$version" "$skill" "$fx/expect.md" "$out" || echo "note: cannot write verdict.json" >&2

for part in "expect.md:Expected" "$out/baseline.md:Baseline (no skill)" "$out/with.md:With skill"; do
  f="${part%%:*}"; [ "$f" = "expect.md" ] && f="$fx/expect.md"
  printf '\n===== %s =====\n' "${part#*:}"; cat "$f"
done
echo; echo "transcripts: $out"
