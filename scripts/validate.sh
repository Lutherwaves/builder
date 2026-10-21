#!/usr/bin/env bash
# Validate the builder plugin: manifests parse, skills have frontmatter and the
# skill shape, Python tools compile. Run locally before opening a PR; CI runs it too.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
note() { printf '  %s\n' "$1"; }
err()  { printf '  ✗ %s\n' "$1"; fail=1; }

echo "→ plugin manifests"
for f in .claude-plugin/plugin.json .claude-plugin/marketplace.json; do
  if jq -e . "$f" >/dev/null 2>&1; then note "✓ $f"; else err "$f is not valid JSON"; fi
done
# plugin.json must carry name + version
jq -e '.name and .version' .claude-plugin/plugin.json >/dev/null 2>&1 \
  || err ".claude-plugin/plugin.json missing name/version"

echo "→ release config"
for f in release-please-config.json .release-please-manifest.json; do
  jq -e . "$f" >/dev/null 2>&1 && note "✓ $f" || err "$f is not valid JSON"
done

echo "→ commit subjects"
if git rev-parse --verify -q origin/main >/dev/null 2>&1; then
  if bad="$(git log origin/main..HEAD --format=%s | bash scripts/check-subjects.sh)"; then
    note "✓ every commit subject in origin/main..HEAD follows Conventional Commits"
  else
    while IFS= read -r s; do err "commit subject is not a Conventional Commit: $s"; done <<<"$bad"
  fi
else
  note "commit subjects: skipped (no origin/main)"
fi

echo "→ skills"
shopt -s nullglob
found_skill=0
for skill in skills/*/SKILL.md; do
  found_skill=1
  # frontmatter must be a leading --- fenced block containing name: and description:
  fm="$(awk 'NR==1&&$0!="---"{exit 1} NR==1{next} $0=="---"{exit} {print}' "$skill" 2>/dev/null || true)"
  if [ -z "$fm" ]; then err "$skill has no YAML frontmatter block"; continue; fi
  before="$fail"; fail=0
  grep -q '^name:'        <<<"$fm" || err "$skill frontmatter missing name:"
  grep -q '^description:'  <<<"$fm" || err "$skill frontmatter missing description:"
  # Skill shape: a script for agents, short enough to read, with its mistakes and its self-check.
  grep -q '^description: Use when' <<<"$fm" || err "$skill description does not start \"Use when\""
  lines="$(wc -l < "$skill" | tr -d ' ')"
  { [ "$lines" -ge 60 ] && [ "$lines" -le 150 ]; } || err "$skill has $lines lines (need 60 to 150)"
  for section in "## Common mistakes" "## Self-check"; do
    grep -qx "$section" "$skill" || err "$skill has no \"$section\""
  done
  [ "$fail" -eq 0 ] && note "✓ $skill ($lines lines)"
  [ "$before" -eq 0 ] || fail=1
done
[ "$found_skill" -eq 1 ] || err "no skills found under skills/*/SKILL.md"

echo "→ python tools"
for py in $(find . -name '*.py' -not -path './.git/*'); do
  python3 -m py_compile "$py" && note "✓ compiles: $py" || err "$py failed to compile"
done
# burn-proj smoke test: fixed inputs → deterministic output shape
out="$(python3 skills/budget/burn-proj.py 19 1783504800 1782990571)"
grep -qE 'elapsed=.* naive=.* profile_aware=.* reset_in=' <<<"$out" \
  || err "burn-proj.py output shape changed: $out"
note "✓ burn-proj smoke: $out"

echo "→ unit tests"
if ut_out="$(python3 -m unittest discover -s tests -p 'test_*.py' 2>&1)"; then note "✓ unittest"; else printf '%s\n' "$ut_out"; err "unit tests failed"; fi

echo "→ harness drift tests"
if dr_out="$(bash tests/test_drift.sh 2>&1)"; then note "✓ drift"; else printf '%s\n' "$dr_out"; err "drift tests failed"; fi

echo "→ hook tests"
if hk_out="$(bash tests/test_hook.sh 2>&1)"; then note "✓ hook"; else printf '%s\n' "$hk_out"; err "hook tests failed"; fi

echo "→ eval fixtures"
if ev_out="$(bash tests/test_evals.sh 2>&1)"; then note "✓ eval runner tests"; else printf '%s\n' "$ev_out"; err "eval runner tests failed"; fi
if fx_out="$(bash tests/evals/run.sh --check 2>&1)"; then note "✓ $fx_out"; else printf '%s\n' "$fx_out"; err "eval fixtures invalid"; fi

echo "→ scrub"
if [ -z "${BUILDER_DENYLIST:-}" ]; then
  note "scrub: skipped (no BUILDER_DENYLIST)"
elif [ ! -r "$BUILDER_DENYLIST" ]; then
  err "scrub: BUILDER_DENYLIST is set but not a readable file"
else
  # Private names must never appear in a tracked file. -F fixed strings, -i case-insensitive.
  # Blank lines are dropped: an empty -F pattern matches every line.
  pats="$(mktemp)"
  grep -v '^[[:space:]]*$' "$BUILDER_DENYLIST" > "$pats" || true
  if [ ! -s "$pats" ]; then
    err "scrub: BUILDER_DENYLIST has no terms"
  else
    rc=0
    out="$(git grep -I -i -n -F -f "$pats" 2>&1)" || rc=$?
    if [ "$rc" -eq 0 ]; then
      while IFS= read -r f; do err "scrub: $f"; done < <(cut -d: -f1 <<<"$out" | sort -u)
    elif [ "$rc" -eq 1 ]; then
      note "✓ no deny-listed names in tracked files"
    else
      err "scrub: git grep failed (exit $rc)"
    fi
  fi
  rm -f "$pats"
fi

echo
if [ "$fail" -eq 0 ]; then echo "✓ all checks passed"; else echo "✗ validation failed"; exit 1; fi
