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
# The prompt goes in on stdin, so a prompt that starts with '-' never parses as an option.
# acceptEdits lets the agent edit files in the temp dir; the deny list blocks push, network tools,
# and the commands that steer the person's real agent sessions (matchblox, tmux).
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
      --disallowedTools "Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch"
      "Bash(matchblox send:*)" "Bash(matchblox compact:*)" "Bash(matchblox hold:*)"
      "Bash(matchblox release:*)" "Bash(matchblox kill:*)" "Bash(matchblox new:*)"
      "Bash(matchblox rename:*)" "Bash(matchblox stoker:*)"
      "Bash(tmux send-keys:*)" "Bash(tmux kill-session:*)")

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
# Leave nothing behind, also when the run is interrupted.
trap 'forget_sessions "$tmp/baseline"; forget_sessions "$tmp/with"; rm -rf "$tmp"' EXIT
trap 'exit 130' INT TERM
mkdir -p "$out" || exit 1
workdir() { # $1 = name; a fresh copy of repo/ (or an empty dir), made a git repo with one commit
  mkdir -p "$tmp/$1"
  if [ -d "$fx/repo" ]; then cp -R "$fx/repo/." "$tmp/$1/"; fi
  git -C "$tmp/$1" init -q || return 1
  git -C "$tmp/$1" config user.name "Eval"
  git -C "$tmp/$1" config user.email "eval@example.invalid"
  git -C "$tmp/$1" config commit.gpgsign false
  git -C "$tmp/$1" add -A \
    && git -C "$tmp/$1" commit -q --allow-empty -m "initial fixture" || return 1
}
# Claude Code names a session folder after its cwd, every char not [A-Za-z0-9] turned into "-".
slug() { printf '%s' "$1" | sed 's/[^A-Za-z0-9]/-/g'; }
tmp_slug="$(slug "$(cd "$tmp" && pwd -P)")"
# Remove the session folder of one run: exactly the folder named like the work dir, nothing else.
forget_sessions() { # $1 = work dir
  local real s dir
  real="$(cd "$1" 2>/dev/null && pwd -P)" || return 0
  s="$(slug "$real")"
  case "$s" in "$tmp_slug"?*) ;; *) echo "note: not removing session folder for $1 (slug outside temp root)" >&2; return 0 ;; esac
  dir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/projects/$s"
  if [ -d "$dir" ] && [ ! -L "$dir" ]; then rm -rf "$dir"; fi
}
# PATH shims: the deny list matches command text, so it misses `matchblox --json send`, `env matchblox send`
# or `bash -c`. The shims stop those at the program. Read-only matchblox commands pass to the real one.
real_matchblox="$(command -v matchblox 2>/dev/null || true)"
mkdir -p "$tmp/bin"
cat > "$tmp/bin/matchblox" <<SHIM
#!/usr/bin/env bash
real='$real_matchblox'
cmd=""
for a in "\$@"; do case "\$a" in -*) ;; *) cmd="\$a"; break ;; esac; done
case "\$cmd" in
  sessions|ships|usage|peek|status|version|help|"")
    if [ -z "\$cmd" ] && [ "\$#" -gt 0 ]; then
      for a in "\$@"; do case "\$a" in --help|-h) ;; --json) ;; *) echo "eval: matchblox \$a is blocked in evals" >&2; exit 3 ;; esac; done
    fi
    [ -n "\$real" ] && [ -x "\$real" ] || { echo "matchblox not installed" >&2; exit 127; }
    exec "\$real" "\$@" ;;
  *) echo "eval: matchblox \$cmd is blocked in evals" >&2; exit 3 ;;
esac
SHIM
printf '#!/usr/bin/env bash\necho "eval: tmux is blocked in evals" >&2\nexit 3\n' > "$tmp/bin/tmux"
chmod +x "$tmp/bin/matchblox" "$tmp/bin/tmux"
export PATH="$tmp/bin:$PATH"
workdir baseline && workdir with || { echo "✗ cannot prepare the work dirs" >&2; exit 1; }
# A temp builder home: onboarded, every recommendation answered "never". The hook then
# injects only the soul, so no setup question pollutes the transcript.
export BUILDER_HOME="$tmp/builder-home"
python3 "$root/lib/ledger.py" onboarded >/dev/null || exit 1
for id in $(python3 -c 'import json,sys; print(" ".join(i["id"] for i in json.load(open(sys.argv[1]))))' "$root/lib/recommendations.json"); do
  python3 "$root/lib/ledger.py" answer "$id" never >/dev/null || exit 1
done
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
