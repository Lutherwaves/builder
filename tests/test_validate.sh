#!/usr/bin/env bash
# Test the checks and guards of scripts/validate.sh.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

# Every temp repo gets its own gitdir: copy the tree without .git (a worktree's .git is a pointer
# to the shared gitdir), init, commit, and give it the origin/main that validate.sh looks for.
mkrepo() {
  mkdir -p "$2"
  (cd "$1" && tar --exclude=.git -cf - .) | tar -xf - -C "$2"
  git -C "$2" init -q
  git -C "$2" add -A
  git -C "$2" -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit -q -m "chore: base"
  git -C "$2" update-ref refs/remotes/origin/main HEAD
}
snapshot() { { git -C "$root" rev-parse HEAD; git -C "$root" for-each-ref; git -C "$root" status --porcelain; } 2>&1; }
before="$(snapshot)"

term="secret""corp"; printf '%s\n' "$term" > "$tmp/deny.txt"

# Case 1: a deny-listed term in a tracked file fails the scrub.
mkrepo "$root" "$tmp/dirty"
echo "$term" >> "$tmp/dirty/README.md"
git -C "$tmp/dirty" add -A >/dev/null
out="$(cd "$tmp/dirty" && BUILDER_DENYLIST="$tmp/deny.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && check "dirty repo exits 1" ok || check "dirty repo exits 1 (got $rc)" bad
grep -q '✗ scrub: README.md' <<<"$out" && check "output names README.md" ok || check "output names README.md" bad

# Case 2: the clean repo passes.
rm -rf "$tmp/clean"; mkrepo "$root" "$tmp/clean"
git -C "$tmp/clean" add -A >/dev/null
out="$(cd "$tmp/clean" && BUILDER_DENYLIST="$tmp/deny.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && check "clean repo exits 0" ok || check "clean repo exits 0 (got $rc)" bad

# Case 3: no deny-list skips the step and passes.
out="$(cd "$tmp/clean" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
grep -q 'scrub: skipped (no BUILDER_DENYLIST)' <<<"$out" && [ "$rc" -eq 0 ] \
  && check "absent deny-list skips" ok || check "absent deny-list skips" bad

# Case 4: set but missing deny-list file fails.
out="$(cd "$tmp/clean" && BUILDER_DENYLIST="$tmp/nope.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ scrub:' <<<"$out" && check "missing deny-list file fails" ok || check "missing deny-list file fails" bad

# Case 5: blank lines in the deny-list do not match every line.
printf '\n   \n%s\n' "no""match""term" > "$tmp/blank.txt"
out="$(cd "$tmp/clean" && BUILDER_DENYLIST="$tmp/blank.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && check "blank lines ignored" ok || check "blank lines ignored (got $rc)" bad

# Case 6: a deny-list with only blank lines fails.
printf '\n\n' > "$tmp/empty.txt"
out="$(cd "$tmp/clean" && BUILDER_DENYLIST="$tmp/empty.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && check "blank-only deny-list fails" ok || check "blank-only deny-list fails" bad

# Case 7: a real git grep error (not a git repo) fails instead of passing.
rm -rf "$tmp/nogit"; mkrepo "$tmp/clean" "$tmp/nogit"; rm -rf "$tmp/nogit/.git"
out="$(cd "$tmp/nogit" && GIT_CEILING_DIRECTORIES="$tmp" BUILDER_DENYLIST="$tmp/deny.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ scrub: git grep failed' <<<"$out" && check "git grep error fails" ok || check "git grep error fails" bad

# Case 8 (test_skill_without_self_check_fails): a skill with no "## Self-check" fails the shape check.
rm -rf "$tmp/shape"; mkrepo "$tmp/clean" "$tmp/shape"
sed -i.bak 's/^## Self-check$/## Checks/' "$tmp/shape/skills/budget/SKILL.md"; rm -f "$tmp/shape/skills/budget/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/budget/SKILL.md has no "## Self-check"' <<<"$out" \
  && check "skill without self-check fails" ok || check "skill without self-check fails (got $rc)" bad

# Case 9: a description that does not start "Use when" fails.
rm -rf "$tmp/shape"; mkrepo "$tmp/clean" "$tmp/shape"
sed -i.bak 's/^description: Use when/description: Handy when/' "$tmp/shape/skills/intake/SKILL.md"; rm -f "$tmp/shape/skills/intake/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/intake/SKILL.md description does not start "Use when"' <<<"$out" \
  && check "description without Use when fails" ok || check "description without Use when fails (got $rc)" bad

# Case 10: a skill outside 60 to 150 lines fails; one without "## Common mistakes" fails.
rm -rf "$tmp/shape"; mkrepo "$tmp/clean" "$tmp/shape"
head -40 "$tmp/clean/skills/harness/SKILL.md" > "$tmp/shape/skills/harness/SKILL.md"
sed -i.bak 's/^## Common mistakes$/## Pitfalls/' "$tmp/shape/skills/reach/SKILL.md"; rm -f "$tmp/shape/skills/reach/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/harness/SKILL.md has 40 lines (need 60 to 150)' <<<"$out" \
  && grep -q '✗ skills/reach/SKILL.md has no "## Common mistakes"' <<<"$out" \
  && check "short skill and missing common mistakes fail" ok || check "short skill and missing common mistakes fail (got $rc)" bad

# Case 11: commit subjects in origin/main..HEAD must follow Conventional Commits.
commit_in() { git -C "$1" -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit -q --allow-empty -m "$2"; }
run_in() { (cd "$1" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1); }
rm -rf "$tmp/cc"; mkrepo "$tmp/clean" "$tmp/cc"
commit_in "$tmp/cc" "feat(scope)!: a person can do a new thing"
commit_in "$tmp/cc" "chore: release 1.2.3"
out="$(run_in "$tmp/cc")"; rc=$?
[ "$rc" -eq 0 ] && check "good and release subjects pass" ok || check "good and release subjects pass (got $rc)" bad
commit_in "$tmp/cc" "wip stuff"
out="$(run_in "$tmp/cc")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ commit subject is not a Conventional Commit: wip stuff' <<<"$out" \
  && check "bad subject fails" ok || check "bad subject fails (got $rc)" bad

commit_in "$tmp/cc" "Merge pull request #12 from someone/branch"
out="$(run_in "$tmp/cc")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ commit subject is not a Conventional Commit: Merge pull request' <<<"$out" \
  && check "merge subject fails" ok || check "merge subject fails (got $rc)" bad

# Case 12: without origin/main the commit check skips with a message.
git -C "$tmp/cc" update-ref -d refs/remotes/origin/main
out="$(run_in "$tmp/cc")"; rc=$?
[ "$rc" -eq 0 ] && grep -q 'commit subjects: skipped (no origin/main)' <<<"$out" \
  && check "no origin/main skips" ok || check "no origin/main skips (got $rc)" bad

# Case 13 (guard_flags_tool_in_skill): a harness tool name in a skill body fails and names the file.
rm -rf "$tmp/guard"; mkrepo "$tmp/clean" "$tmp/guard"
printf '%s\n' 'Use the PushNotification tool.' >> "$tmp/guard/skills/reach/SKILL.md"
printf '%s\n' 'Run /loop 30m now.' >> "$tmp/guard/skills/fleet/SKILL.md"
out="$(run_in "$tmp/guard")"; rc=$?
[ "$rc" -eq 1 ] && grep -qE '✗ guard: skills/reach/SKILL.md:[0-9]+' <<<"$out" \
  && grep -qE '✗ guard: skills/fleet/SKILL.md:[0-9]+' <<<"$out" \
  && check "guard flags tool and slash command in skill" ok || check "guard flags tool and slash command in skill (got $rc)" bad

# Case 14 (guard_allows_mapping_dir): the mapping files may name harness tools.
rm -rf "$tmp/guard"; mkrepo "$tmp/clean" "$tmp/guard"
printf '%s\n' '| send a notification | PushNotification, /loop, /usage |' >> "$tmp/guard/skills/builder/harnesses/claude.md"
out="$(run_in "$tmp/guard")"; rc=$?
[ "$rc" -eq 0 ] && check "guard allows the mapping dir" ok || check "guard allows the mapping dir (got $rc)" bad

# Case 15 (guard_allows_gh_tmux_matchblox): ordinary tools and paths are not flagged.
rm -rf "$tmp/guard"; mkrepo "$tmp/clean" "$tmp/guard"
printf '%s\n' 'Run `gh pr list`, `tmux send-keys`, `matchblox usage --json`, see https://example.com/loop and src/usage/x.' \
  'The monitor of a room and an artifact in prose stay fine in lower case.' >> "$tmp/guard/skills/reach/SKILL.md"
out="$(run_in "$tmp/guard")"; rc=$?
[ "$rc" -eq 0 ] && ! grep -q 'guard:' <<<"$out" && check "guard allows gh, tmux, matchblox, paths" ok || check "guard allows gh, tmux, matchblox, paths (got $rc)" bad

# Case 15b (guard_flags_slash_in_neutral_how): a string-valued `how` with a slash command fails; a map-valued `how` may have one.
rm -rf "$tmp/guard"; mkrepo "$tmp/clean" "$tmp/guard"
jq '(.[] | select(.id == "fleet") | .how) = "Ask me: /builder:fleet"' "$tmp/guard/lib/recommendations.json" > "$tmp/guard/r.json" \
  && mv "$tmp/guard/r.json" "$tmp/guard/lib/recommendations.json"
out="$(run_in "$tmp/guard")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ guard: lib/recommendations.json: fleet how' <<<"$out" && ! grep -q 'guard: lib/recommendations.json: superpowers' <<<"$out" \
  && check "guard flags slash command in a string how, allows it in a map how" ok || check "guard flags slash command in a string how, allows it in a map how (got $rc)" bad

# Case 17 (manifest_version_drift_fails): a manifest whose version differs from plugin.json fails.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
sed -i.bak 's/"version": "[^"]*"/"version": "9.9.9"/' "$tmp/man/.codex-plugin/plugin.json"; rm -f "$tmp/man/.codex-plugin/plugin.json.bak"
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ .*codex-plugin/plugin.json version' <<<"$out" \
  && check "manifest version drift fails" ok || check "manifest version drift fails (got $rc)" bad

# Case 18 (manifest_description_drift_fails): a package.json description that differs fails.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
sed -i.bak 's/"description": "[^"]*"/"description": "other"/' "$tmp/man/package.json"; rm -f "$tmp/man/package.json.bak"
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ .*package.json description' <<<"$out" \
  && check "manifest description drift fails" ok || check "manifest description drift fails (got $rc)" bad

# Case 19 (manifest_missing_fails): a missing manifest fails.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
git -C "$tmp/man" rm -q -f .cursor-plugin/plugin.json
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ .cursor-plugin/plugin.json is not valid JSON' <<<"$out" \
  && check "missing manifest fails" ok || check "missing manifest fails (got $rc)" bad

# Case 20 (cursor_hooks_extra_key_fails): one invalid key disables every Cursor hook, so the file may hold only version and hooks.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
jq '. + {description: "x"}' "$tmp/man/hooks/hooks-cursor.json" > "$tmp/man/h.json" && mv "$tmp/man/h.json" "$tmp/man/hooks/hooks-cursor.json"
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ hooks/hooks-cursor.json has a key besides version and hooks: description' <<<"$out" \
  && check "cursor hooks extra key fails" ok || check "cursor hooks extra key fails (got $rc)" bad

# Case 21 (opencode_syntax_error_fails): a syntax error in the adapter fails node --check.
if command -v node >/dev/null 2>&1; then
  rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
  echo 'export const = ;' >> "$tmp/man/.opencode/plugins/builder.js"
  out="$(run_in "$tmp/man")"; rc=$?
  [ "$rc" -eq 1 ] && grep -q '✗ .*builder.js' <<<"$out" \
    && check "opencode syntax error fails" ok || check "opencode syntax error fails (got $rc)" bad
fi

# Case 22 (support_table_drift_fails): a README table that no results file backs fails validation.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
awk '
  $0 == "<!-- support:start -->" { print; print "| Harness | Version | Eval date | Status |"; print "| --- | --- | --- | --- |"; print "| Codex CLI | 1 | 2026-10-22 | supported |"; skip = 1; next }
  $0 == "<!-- support:end -->" { skip = 0 }
  !skip
' "$tmp/man/README.md" > "$tmp/man/r.md" && mv "$tmp/man/r.md" "$tmp/man/README.md"
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ README support table is out of date' <<<"$out" \
  && check "support table with no results behind it fails" ok || check "support table with no results behind it fails (got $rc)" bad

# Case 23 (hedge_fails): a hedge in a user-facing doc fails validation and names the file and phrase.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
echo 'Only the local path was proven.' >> "$tmp/man/README.md"
git -C "$tmp/man" add -A >/dev/null
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ hedge: README.md: was proven' <<<"$out" \
  && check "hedge in README fails" ok || check "hedge in README fails (got $rc)" bad

# Case 24 (not_proven_is_a_report_field): the report's "Not proven" line is not a hedge.
# CONTRIBUTING, not README: a second report line in README breaks the example-order test.
rm -rf "$tmp/man"; mkrepo "$tmp/clean" "$tmp/man"
echo '> **⚠ Not proven:** the email in a real inbox.' >> "$tmp/man/CONTRIBUTING.md"
git -C "$tmp/man" add -A >/dev/null
out="$(run_in "$tmp/man")"; rc=$?
[ "$rc" -eq 0 ] && check "Not proven in a user-facing doc passes" ok || check "Not proven in a user-facing doc passes (got $rc)" bad

# Case 25 (hedge_grep_error_fails): a git grep error in the hedge step fails instead of passing.
rm -rf "$tmp/nogit"; mkrepo "$tmp/clean" "$tmp/nogit"; rm -rf "$tmp/nogit/.git"
out="$(cd "$tmp/nogit" && GIT_CEILING_DIRECTORIES="$tmp" env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ hedge: git grep failed' <<<"$out" \
  && check "hedge git grep error fails" ok || check "hedge git grep error fails (got $rc)" bad

# Case 26 (python_file_fails): a tracked .py file outside an eval fixture fails.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
echo 'print(1)' > "$tmp/py/skills/reach/x.py"
git -C "$tmp/py" add -A >/dev/null
out="$(run_in "$tmp/py")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ python: skills/reach/x.py' <<<"$out" \
  && check "python file fails" ok || check "python file fails (got $rc)" bad

# Case 27 (python3_word_fails): the word python3 in a skill fails.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
printf '%s\n' 'Run `python3 x`.' >> "$tmp/py/skills/reach/SKILL.md"
out="$(run_in "$tmp/py")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ python: skills/reach/SKILL.md:' <<<"$out" \
  && check "python3 word fails" ok || check "python3 word fails (got $rc)" bad

# Case 28 (fixture_python_allowed): an eval fixture's sample project may hold Python.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
printf '%s\n' '#!/usr/bin/env python3' 'print(1)' > "$tmp/py/tests/evals/review/repo/y.py"
git -C "$tmp/py" add -A >/dev/null
out="$(run_in "$tmp/py")"; rc=$?
! grep -q 'python:' <<<"$out" && check "fixture python allowed" ok || check "fixture python allowed (got $rc)" bad

# Case 29 (old_tool_name_fails): a skill that names a deleted Python tool fails.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
printf '%s\n' 'Run lib/reach.py.' >> "$tmp/py/skills/reach/SKILL.md"
out="$(run_in "$tmp/py")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ python: skills/reach/SKILL.md:' <<<"$out" \
  && check "old tool name fails" ok || check "old tool name fails (got $rc)" bad

# Case 30 (unquoted_shim_fails): a skill that runs the shim without the quoted plugin root fails.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
printf '%s\n' 'Run bin/builder reach check.' >> "$tmp/py/skills/reach/SKILL.md"
out="$(run_in "$tmp/py")"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ guard: skills/reach/SKILL.md:[0-9]* runs the shim without the quoted plugin root' <<<"$out" \
  && check "unquoted shim fails" ok || check "unquoted shim fails (got $rc)" bad

# Case 31 (readme_shim_allowed): a README for people names the shim with "<plugin root>" without tripping the guard.
rm -rf "$tmp/py"; mkrepo "$tmp/clean" "$tmp/py"
printf '%s\n' '"<plugin root>/bin/builder" budget 19 1783504800' >> "$tmp/py/skills/budget/README.md"
out="$(run_in "$tmp/py")"; rc=$?
! grep -q 'runs the shim without' <<<"$out" && check "README plugin root allowed" ok || check "README plugin root allowed (got $rc)" bad

# Case 16: the run left the real repo's HEAD, refs and working tree as they were.
[ "$(snapshot)" = "$before" ] && check "real repo untouched by the test run" ok || check "real repo untouched by the test run" bad

exit "$fails"
