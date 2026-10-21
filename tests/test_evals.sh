#!/usr/bin/env bash
# Test tests/evals/run.sh --check and --dry-run.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
run() { BUILDER_EVAL_DIR="$tmp/evals" bash "$root/tests/evals/run.sh" "$@"; }

mkdir -p "$tmp/evals"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "fixtures ok (0)" ] && check "zero fixtures" ok || check "zero fixtures ($out)" bad

mkdir -p "$tmp/evals/good"
echo "Do the thing." > "$tmp/evals/good/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/good/expect.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "fixtures ok" ] && check "good fixture passes" ok || check "good fixture passes ($out)" bad

mkdir -p "$tmp/evals/nope"; echo "x" > "$tmp/evals/nope/prompt.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -ne 0 ] && grep -q 'nope: expect.md is missing' <<<"$out" && check "missing expect.md fails" ok || check "missing expect.md fails" bad

rm -rf "$tmp/evals/nope"; mkdir -p "$tmp/evals/few"; echo x > "$tmp/evals/few/prompt.md"; printf -- '- a\n- b\n' > "$tmp/evals/few/expect.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -ne 0 ] && grep -q 'few: expect.md has 2 bullet' <<<"$out" && check "too few bullets fails" ok || check "too few bullets fails" bad
rm -rf "$tmp/evals/few"

out="$(BUILDER_EVAL_OUT="$tmp/out" run good --dry-run 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$(grep -c '^claude ' <<<"$out")" -eq 2 ] \
  && grep -q -- '--plugin-dir' <<<"$out" && grep -q "$tmp/out/good/" <<<"$out" \
  && ! grep -q -- '--bare' <<<"$out" && [ ! -e "$tmp/out" ] \
  && check "dry-run prints two commands, writes nothing" ok || check "dry-run prints two commands, writes nothing" bad

# A fake claude on PATH: echoes stdin and its args to stdout, writes a marker to stderr.
mkdir -p "$tmp/bin"
cat > "$tmp/bin/claude" <<'FAKE'
#!/usr/bin/env bash
echo "ARGS: $*"; echo "STDIN: $(cat)"; echo "fake-stderr" >&2
echo "HOME_DIR: ${BUILDER_HOME:-unset}"
echo "HOOK: $("$HOOK" 2>/dev/null)"
echo "GITLOG: $(git log --oneline 2>&1 | wc -l | tr -d ' ') $(git config user.email) $(git config commit.gpgsign) $([ -d .git ] && echo dotgit)"
# Like Claude Code: leave a session folder named after the cwd (and a decoy that must survive).
pdir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/projects"
mkdir -p "$pdir/$(pwd -P | sed 's/[^A-Za-z0-9]/-/g')" "$pdir/-decoy-project"
FAKE
chmod +x "$tmp/bin/claude"
mkdir -p "$tmp/evals/dash"
printf -- '- leading dash prompt\n' > "$tmp/evals/dash/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/dash/expect.md"
mkdir -p "$tmp/home"
HOME="$tmp/home" HOOK="$root/hooks/session-start" PATH="$tmp/bin:$PATH" BUILDER_EVAL_OUT="$tmp/out2" run dash </dev/null >/dev/null 2>&1
d="$(ls -d "$tmp"/out2/dash/*/ 2>/dev/null | head -1)"
[ -n "$d" ] && grep -q '^STDIN: - leading dash prompt' "$d/baseline.md" && grep -q '^STDIN: - leading dash prompt' "$d/with.md" \
  && ! grep -q 'leading dash' <(grep '^ARGS:' "$d/baseline.md") \
  && check "prompt goes to claude on stdin, not as an argument" ok || check "prompt goes to claude on stdin, not as an argument" bad
[ -n "$d" ] && grep -q 'fake-stderr' "$d/baseline.err" && grep -q 'fake-stderr' "$d/with.err" \
  && ! grep -q 'fake-stderr' "$d/baseline.md" "$d/with.md" \
  && check "stderr goes to .err files, not the transcript" ok || check "stderr goes to .err files, not the transcript" bad
# Eval runs use a temp, onboarded builder home: no onboarding text and no recommendation reach the agent.
[ -n "$d" ] && grep -q '^HOME_DIR: /' "$d/with.md" && ! grep -q "^HOME_DIR: $HOME/.config/builder" "$d/with.md" \
  && grep -q 'Build the right thing' "$d/with.md" \
  && ! grep -qi 'onboarding\|builder recommends' "$d/with.md" \
  && check "eval runs use an onboarded temp BUILDER_HOME" ok || check "eval runs use an onboarded temp BUILDER_HOME" bad
# The work dir is a git repo with exactly one commit and a local throwaway identity.
[ -n "$d" ] && [ "$(grep -c '^GITLOG: 1 eval@example.invalid false dotgit' "$d/baseline.md")" -eq 1 ] \
  && grep -q '^GITLOG: 1 eval@example.invalid false dotgit' "$d/with.md" \
  && check "work dir is a git repo with one commit" ok || check "work dir is a git repo with one commit" bad
# Session folders of the work dirs are removed; other projects stay.
[ -d "$tmp/home/.claude/projects/-decoy-project" ] \
  && [ "$(ls "$tmp/home/.claude/projects" | wc -l | tr -d ' ')" -eq 1 ] \
  && check "run removes its own session folders only" ok || check "run removes its own session folders only ($(ls "$tmp/home/.claude/projects" 2>&1))" bad
# CLAUDE_CONFIG_DIR is honoured.
CLAUDE_CONFIG_DIR="$tmp/cfg" HOME="$tmp/home" HOOK="$root/hooks/session-start" PATH="$tmp/bin:$PATH" BUILDER_EVAL_OUT="$tmp/out3" run dash </dev/null >/dev/null 2>&1
[ -d "$tmp/cfg/projects/-decoy-project" ] && [ "$(ls "$tmp/cfg/projects" | wc -l | tr -d ' ')" -eq 1 ] \
  && check "CLAUDE_CONFIG_DIR projects dir is cleaned" ok || check "CLAUDE_CONFIG_DIR projects dir is cleaned" bad
rm -rf "$tmp/evals/dash"

# An optional allow file adds tool rules to --allowedTools; blank and # lines are skipped.
mkdir -p "$tmp/evals/allowed"
echo "Do it." > "$tmp/evals/allowed/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/allowed/expect.md"
printf '# tests\nBash(python3 -m unittest:*)\n\n' > "$tmp/evals/allowed/allow"
out="$(run allowed --dry-run 2>&1)"
for rule in 'Bash(matchblox send:*)' 'Bash(matchblox compact:*)' 'Bash(matchblox hold:*)' 'Bash(matchblox release:*)' \
  'Bash(matchblox kill:*)' 'Bash(matchblox new:*)' 'Bash(matchblox rename:*)' 'Bash(matchblox stoker:*)' \
  'Bash(tmux send-keys:*)' 'Bash(tmux kill-session:*)' 'Bash(git push:*)' 'WebFetch'; do
  q="$(printf '%q' "$rule")"
  [ "$(grep -cF -- "$q" <<<"$out")" -eq 2 ] || { check "both runs deny $rule" bad; }
done
check "both runs deny session-steering and existing tools" ok
[ "$(grep -c -- '--allowedTools Bash\\(python3\\ -m\\ unittest:\\\*\\) --disallowedTools' <<<"$out")" -eq 2 ] \
  && ! grep -q -- '--allowedTools.*#' <<<"$out" \
  && check "allow file adds --allowedTools" ok || check "allow file adds --allowedTools ($out)" bad
out="$(run good --dry-run 2>&1)"
! grep -q -- '--allowedTools' <<<"$out" && check "no allow file, no --allowedTools" ok || check "no allow file, no --allowedTools" bad
rm -rf "$tmp/evals/allowed"

exit "$fails"
