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

out="${BUILDER_EVAL_OUT:-$HOME/builder-design/evals}/$skill/$(date +%Y%m%d-%H%M%S)"
# The prompt goes in on stdin, so a prompt that starts with '-' never parses as an option.
# acceptEdits lets the agent edit files in the temp dir; the deny list blocks push and network tools.
# An optional allow file adds tool rules (one per line), e.g. the fixture's test command.
allow=()
if [ -f "$fx/allow" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in ''|'#'*) continue ;; esac
    allow+=("$line")
  done < "$fx/allow"
fi
base=(claude -p --output-format text --permission-mode acceptEdits
      ${allow[@]+--allowedTools "${allow[@]}"}
      --disallowedTools "Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch")

show() { printf '%q ' "$@"; echo; }

if [ "$dry" -eq 1 ]; then
  echo "# transcripts: $out/{baseline,with}.md (stderr: .err)"
  echo "# baseline (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}"
  echo "# with     (cwd: fresh temp copy of $fx/repo; stdin: prompt.md)"; show "${base[@]}" --plugin-dir "$root"
  exit 0
fi

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out" || exit 1
workdir() { # $1 = name; makes a fresh copy of repo/ or an empty git repo
  mkdir -p "$tmp/$1"
  if [ -d "$fx/repo" ]; then cp -R "$fx/repo/." "$tmp/$1/"; else git -C "$tmp/$1" init -q; fi
}
workdir baseline; workdir with
(cd "$tmp/baseline" && "${base[@]}" < "$fx/prompt.md" > "$out/baseline.md" 2> "$out/baseline.err") \
  || echo "note: baseline run exited non-zero (see $out/baseline.err)" >&2
(cd "$tmp/with" && "${base[@]}" --plugin-dir "$root" < "$fx/prompt.md" > "$out/with.md" 2> "$out/with.err") \
  || echo "note: with run exited non-zero (see $out/with.err)" >&2

for part in "expect.md:Expected" "$out/baseline.md:Baseline (no skill)" "$out/with.md:With skill"; do
  f="${part%%:*}"; [ "$f" = "expect.md" ] && f="$fx/expect.md"
  printf '\n===== %s =====\n' "${part#*:}"; cat "$f"
done
echo; echo "transcripts: $out"
