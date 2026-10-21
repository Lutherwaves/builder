#!/usr/bin/env bash
# Test the scrub step of scripts/validate.sh.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

term="secret""corp"; printf '%s\n' "$term" > "$tmp/deny.txt"

# Case 1: a deny-listed term in a tracked file fails the scrub.
cp -R "$root" "$tmp/dirty"
echo "$term" >> "$tmp/dirty/README.md"
git -C "$tmp/dirty" add -A >/dev/null
out="$(cd "$tmp/dirty" && BUILDER_DENYLIST="$tmp/deny.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && check "dirty repo exits 1" ok || check "dirty repo exits 1 (got $rc)" bad
grep -q '✗ scrub: README.md' <<<"$out" && check "output names README.md" ok || check "output names README.md" bad

# Case 2: the clean repo passes.
rm -rf "$tmp/clean"; cp -R "$root" "$tmp/clean"
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
rm -rf "$tmp/nogit"; cp -R "$tmp/clean" "$tmp/nogit"; rm -rf "$tmp/nogit/.git"
out="$(cd "$tmp/nogit" && GIT_CEILING_DIRECTORIES="$tmp" BUILDER_DENYLIST="$tmp/deny.txt" bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ scrub: git grep failed' <<<"$out" && check "git grep error fails" ok || check "git grep error fails" bad

# Case 8 (test_skill_without_self_check_fails): a skill with no "## Self-check" fails the shape check.
rm -rf "$tmp/shape"; cp -R "$tmp/clean" "$tmp/shape"
sed -i.bak 's/^## Self-check$/## Checks/' "$tmp/shape/skills/budget/SKILL.md"; rm -f "$tmp/shape/skills/budget/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/budget/SKILL.md has no "## Self-check"' <<<"$out" \
  && check "skill without self-check fails" ok || check "skill without self-check fails (got $rc)" bad

# Case 9: a description that does not start "Use when" fails.
rm -rf "$tmp/shape"; cp -R "$tmp/clean" "$tmp/shape"
sed -i.bak 's/^description: Use when/description: Handy when/' "$tmp/shape/skills/intake/SKILL.md"; rm -f "$tmp/shape/skills/intake/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/intake/SKILL.md description does not start "Use when"' <<<"$out" \
  && check "description without Use when fails" ok || check "description without Use when fails (got $rc)" bad

# Case 10: a skill outside 60 to 150 lines fails; one without "## Common mistakes" fails.
rm -rf "$tmp/shape"; cp -R "$tmp/clean" "$tmp/shape"
head -40 "$tmp/clean/skills/harness/SKILL.md" > "$tmp/shape/skills/harness/SKILL.md"
sed -i.bak 's/^## Common mistakes$/## Pitfalls/' "$tmp/shape/skills/reach/SKILL.md"; rm -f "$tmp/shape/skills/reach/SKILL.md.bak"
out="$(cd "$tmp/shape" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -q '✗ skills/harness/SKILL.md has 40 lines (need 60 to 150)' <<<"$out" \
  && grep -q '✗ skills/reach/SKILL.md has no "## Common mistakes"' <<<"$out" \
  && check "short skill and missing common mistakes fail" ok || check "short skill and missing common mistakes fail (got $rc)" bad

# Case 11: commit subjects in origin/main..HEAD must follow Conventional Commits.
commit_in() { git -C "$1" -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit -q --allow-empty -m "$2"; }
run_in() { (cd "$1" && env -u BUILDER_DENYLIST bash scripts/validate.sh 2>&1); }
rm -rf "$tmp/cc"; cp -R "$tmp/clean" "$tmp/cc"
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

exit "$fails"
