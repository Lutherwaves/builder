#!/usr/bin/env bash
# Validate the builder plugin: manifests parse, skills have frontmatter,
# Python tools compile. Run locally before opening a PR; CI runs it too.
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

echo "→ skills"
shopt -s nullglob
found_skill=0
for skill in skills/*/SKILL.md; do
  found_skill=1
  # frontmatter must be a leading --- fenced block containing name: and description:
  fm="$(awk 'NR==1&&$0!="---"{exit 1} NR==1{next} $0=="---"{exit} {print}' "$skill" 2>/dev/null || true)"
  if [ -z "$fm" ]; then err "$skill has no YAML frontmatter block"; continue; fi
  grep -q '^name:'        <<<"$fm" || err "$skill frontmatter missing name:"
  grep -q '^description:'  <<<"$fm" || err "$skill frontmatter missing description:"
  [ "${fail}" -eq 0 ] && note "✓ $skill"
done
[ "$found_skill" -eq 1 ] || err "no skills found under skills/*/SKILL.md"

echo "→ python tools"
for py in $(find . -name '*.py' -not -path './.git/*'); do
  python3 -m py_compile "$py" && note "✓ compiles: $py" || err "$py failed to compile"
done
# burn-proj smoke test: fixed inputs → deterministic output shape
out="$(python3 skills/watch-limits/burn-proj.py 19 1783504800 1782990571)"
grep -qE 'elapsed=.* naive=.* profile_aware=.* reset_in=' <<<"$out" \
  || err "burn-proj.py output shape changed: $out"
note "✓ burn-proj smoke: $out"

echo
if [ "$fail" -eq 0 ]; then echo "✓ all checks passed"; else echo "✗ validation failed"; exit 1; fi
