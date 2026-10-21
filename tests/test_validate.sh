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

exit "$fails"
