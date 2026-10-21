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

# PATH shims. A fake "real" matchblox sits on the original PATH; the fake claude runs the probes.
mkdir -p "$tmp/real"
printf '#!/usr/bin/env bash\necho "REAL matchblox $*"\n' > "$tmp/real/matchblox"; chmod +x "$tmp/real/matchblox"
cat > "$tmp/bin/claude" <<'FAKE'
#!/usr/bin/env bash
cat >/dev/null
p() { echo "PROBE $1 => rc=$(shift; "$@" >/dev/null 2>&1; echo $?)"; }
p flags matchblox --json send x y
p env env matchblox send x y
p bashc bash -c "matchblox send x y"
p tmux tmux send -t x y
p tmuxL tmux -L s send-keys x
echo "READ: $(matchblox sessions --json 2>&1)"
FAKE
mkdir -p "$tmp/evals/shim"; echo "Do it." > "$tmp/evals/shim/prompt.md"; printf -- '- a\n- b\n- c\n' > "$tmp/evals/shim/expect.md"
HOME="$tmp/home" PATH="$tmp/bin:$tmp/real:$PATH" BUILDER_EVAL_OUT="$tmp/out4" run shim </dev/null >/dev/null 2>&1
d="$(ls -d "$tmp"/out4/shim/*/ 2>/dev/null | head -1)"
[ -n "$d" ] && [ "$(grep -c 'rc=3$' "$d/with.md")" -eq 5 ] && grep -q '^READ: REAL matchblox sessions --json' "$d/with.md" \
  && check "shims block steering and pass read-only matchblox" ok || check "shims block steering and pass read-only matchblox ($(cat "$d/with.md" 2>&1))" bad
rm -rf "$tmp/evals/shim"

# An optional allow file adds tool rules to --allowedTools; blank and # lines are skipped.
mkdir -p "$tmp/evals/allowed"
echo "Do it." > "$tmp/evals/allowed/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/allowed/expect.md"
printf '# tests\nBash(python3 -m unittest:*)\n\n' > "$tmp/evals/allowed/allow"
out="$(run allowed --dry-run 2>&1)"
deny_ok=1
for rule in 'Bash(matchblox send:*)' 'Bash(matchblox compact:*)' 'Bash(matchblox hold:*)' 'Bash(matchblox release:*)' \
  'Bash(matchblox kill:*)' 'Bash(matchblox new:*)' 'Bash(matchblox rename:*)' 'Bash(matchblox stoker:*)' \
  'Bash(tmux send-keys:*)' 'Bash(tmux kill-session:*)' 'Bash(git push:*)' 'WebFetch'; do
  q="$(printf '%q' "$rule")"
  [ "$(grep -cF -- "$q" <<<"$out")" -eq 2 ] || { deny_ok=0; echo "    missing: $rule"; }
done
[ "$deny_ok" -eq 1 ] && check "both runs deny session-steering and existing tools" ok || check "both runs deny session-steering and existing tools" bad
[ "$(grep -c -- '--allowedTools Bash\\(python3\\ -m\\ unittest:\\\*\\) --disallowedTools' <<<"$out")" -eq 2 ] \
  && ! grep -q -- '--allowedTools.*#' <<<"$out" \
  && check "allow file adds --allowedTools" ok || check "allow file adds --allowedTools ($out)" bad
out="$(run good --dry-run 2>&1)"
! grep -q -- '--allowedTools' <<<"$out" && check "no allow file, no --allowedTools" ok || check "no allow file, no --allowedTools" bad
rm -rf "$tmp/evals/allowed"

# parity.sh: the builder-led config, the auth gate, and two runs per fixture.
ph="$tmp/phome"; mkdir -p "$ph/.claude/plugins/cache/m/superpowers/1" "$ph/.claude/plugins/marketplaces/m" "$ph/.claude/skills/gstack"
cat > "$ph/.claude/settings.json" <<'JSON'
{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "/x/matchblox hook Stop"}, {"type": "command", "command": "/x/other-hook"}]}],
           "PostToolUse": [{"hooks": [{"type": "command", "command": "/x/other-hook"}]}]},
 "statusLine": {"type": "command", "command": "/x/matchblox hook statusline"},
 "enabledPlugins": {"superpowers@m": true, "extra@m": true}, "permissions": {"allow": ["Bash(ls:*)"]}}
JSON
printf '{"version":2,"plugins":{"superpowers@m":[{"installPath":"%s/.claude/plugins/cache/m/superpowers/1"}],"extra@m":[{}]}}' "$ph" > "$ph/.claude/plugins/installed_plugins.json"
printf '{"m":{"installLocation":"%s/.claude/plugins/marketplaces/m"},"other":{}}' "$ph" > "$ph/.claude/plugins/known_marketplaces.json"
printf '{"mcpServers":{"codebase-memory-mcp":{"command":"/x/cbm"},"other":{}},"oauthAccount":{"secret":1}}' > "$ph/.claude.json"
for f in builder shape; do mkdir -p "$tmp/pevals/$f"; echo "Do $f." > "$tmp/pevals/$f/prompt.md"; printf -- '- a\n- b\n- c\n' > "$tmp/pevals/$f/expect.md"; done
cat > "$tmp/bin/claude" <<'FAKE'
#!/usr/bin/env bash
if [ "$1" = auth ]; then echo "{\"loggedIn\": ${FAKE_LOGGED_IN:-false}}"; exit 0; fi
echo "STDIN: $(cat)"; echo "CFG: ${CLAUDE_CONFIG_DIR:-unset}"; echo "ARGS: $*"
FAKE
prun() { HOME="$ph" PATH="$tmp/bin:$PATH" BUILDER_EVAL_DIR="$tmp/pevals" bash "$root/tests/evals/parity.sh" "$@"; }
out="$(BUILDER_EVAL_OUT="$tmp/pout0" prun --dry-run 2>&1)"; rc=$?
sj="$(sed -n '/^# settings.json:/,/^# builder current/p' <<<"$out" | sed '1d;$d')"
[ "$rc" -eq 0 ] && [ ! -e "$tmp/pout0" ] \
  && [ "$(jq -c '.enabledPlugins' <<<"$sj")" = '{"superpowers@m":true}' ] \
  && [ "$(jq -r '.env.CLAUDE_AUTOCOMPACT_PCT_OVERRIDE' <<<"$sj")" = 85 ] \
  && [ "$(jq -c '[.hooks[][].hooks[].command]' <<<"$sj")" = '["/x/matchblox hook Stop"]' ] \
  && [ "$(jq -r '.statusLine.command' <<<"$sj")" = '/x/matchblox hook statusline' ] \
  && [ "$(jq 'has("permissions")' <<<"$sj")" = false ] \
  && [ "$(grep -c -- '--plugin-dir' <<<"$out")" -eq 2 ] && [ "$(grep -c '^CLAUDE_CONFIG_DIR=' <<<"$out")" -eq 2 ] \
  && check "parity dry-run: builder-led settings hold only builder's recommendations" ok || check "parity dry-run ($out)" bad
out="$(BUILDER_EVAL_OUT="$tmp/pout1" prun 2>&1)"; rc=$?
[ "$rc" -eq 4 ] && grep -q 'not signed in under CLAUDE_CONFIG_DIR' <<<"$out" && grep -q 'never copies credentials' <<<"$out" \
  && [ ! -e "$tmp/pout1" ] \
  && check "parity stops before any run when the config dir is not signed in" ok || check "parity auth gate ($rc: $out)" bad
mkdir -p "$tmp/pcfg"
out="$(FAKE_LOGGED_IN=true BUILDER_PARITY_CONFIG_DIR="$tmp/pcfg" BUILDER_EVAL_OUT="$tmp/pout2" prun 2>&1)"; rc=$?
d="$(ls -d "$tmp"/pout2/parity/*/ 2>/dev/null | head -1)"
[ "$rc" -eq 0 ] && [ -n "$d" ] && grep -q '^CFG: unset' "$d/builder/current.md" && grep -q "^CFG: $tmp/pcfg" "$d/shape/builder.md" \
  && grep -q -- '--plugin-dir' "$d/shape/builder.md" && ! grep -q -- '--plugin-dir' "$d/shape/current.md" \
  && grep -q '^STDIN: Do shape.' "$d/shape/builder.md" && grep -q '===== Builder-led setup =====' <<<"$out" \
  && check "parity runs each fixture in the current and the builder-led setup" ok || check "parity runs ($rc: $out)" bad
[ "$(jq -c '.mcpServers' "$tmp/pcfg/.claude.json")" = '{"codebase-memory-mcp":{"command":"/x/cbm"}}' ] \
  && ! grep -q oauthAccount "$tmp/pcfg/.claude.json" && [ -L "$tmp/pcfg/skills" ] && [ -L "$tmp/pcfg/plugins/cache" ] \
  && [ "$(jq -r '.plugins|keys|join(",")' "$tmp/pcfg/plugins/installed_plugins.json")" = 'superpowers@m' ] \
  && grep -q "$tmp/pcfg/plugins/cache/m/superpowers/1" "$tmp/pcfg/plugins/installed_plugins.json" \
  && [ "$(jq -r 'keys|join(",")' "$tmp/pcfg/plugins/known_marketplaces.json")" = m ] \
  && check "parity links plugins and skills, adds the code graph, copies no account" ok || check "parity config contents" bad

exit "$fails"
