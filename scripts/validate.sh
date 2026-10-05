#!/usr/bin/env bash
# Validate the builder plugin: manifests parse, skills have frontmatter,
# Python tools compile, Go tools are formatted, vetted and tested. Run locally
# before opening a PR; CI runs it too.
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

echo "→ go tools"
for mod in $(find skills -name go.mod -not -path '*/testdata/*'); do
  dir="$(dirname "$mod")"
  if ! command -v go >/dev/null 2>&1; then err "$dir needs Go to validate (https://go.dev/dl)"; continue; fi
  unformatted="$(cd "$dir" && gofmt -l .)"
  [ -z "$unformatted" ] && note "✓ gofmt: $dir" || err "$dir: gofmt needed: $unformatted"
  (cd "$dir" && go vet ./...) && note "✓ vet: $dir" || err "$dir: go vet failed"
  (cd "$dir" && go test ./...) >/dev/null && note "✓ tests: $dir" || err "$dir: go test failed (run: cd $dir && go test ./...)"
done
# builder-tui smoke: the binary reads the fixture machine and the example config
if [ -f skills/tui/go.mod ] && command -v go >/dev/null 2>&1; then
  bin="$(mktemp)"
  (cd skills/tui && CGO_ENABLED=0 go build -o "$bin" .)
  out="$("$bin" status --text --fixtures skills/tui/testdata/machine --config skills/tui/config.example.toml)"
  grep -q 'sessions: 3 (busy 1, idle 2)' <<<"$out" && note "✓ builder-tui smoke" || err "builder-tui smoke output changed: $out"
  rm -f "$bin"
fi

echo
if [ "$fail" -eq 0 ]; then echo "✓ all checks passed"; else echo "✗ validation failed"; exit 1; fi
