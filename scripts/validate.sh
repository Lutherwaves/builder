#!/usr/bin/env bash
# Validate the builder plugin: manifests parse, skills have frontmatter and the
# skill shape, the Go code builds and passes its tests. Run locally before opening a PR; CI runs it too.
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

# Every harness manifest and package.json parse and carry the same name, version and description.
manifests=(.codex-plugin/plugin.json .cursor-plugin/plugin.json .agents/plugins/marketplace.json package.json hooks/hooks-cursor.json)
for f in "${manifests[@]}"; do
  if jq -e . "$f" >/dev/null 2>&1; then note "✓ $f"; else err "$f is not valid JSON"; fi
done
for f in .codex-plugin/plugin.json .cursor-plugin/plugin.json package.json; do
  jq -e . "$f" >/dev/null 2>&1 || continue
  for field in name version description; do
    want="$(jq -r ".$field" .claude-plugin/plugin.json)"; got="$(jq -r ".$field" "$f")"
    [ "$want" = "$got" ] || err "$f $field is \"$got\" (.claude-plugin/plugin.json has \"$want\")"
  done
done
# One invalid key disables every hook in a Cursor hooks file.
if jq -e . hooks/hooks-cursor.json >/dev/null 2>&1; then
  [ "$(jq -r '.version' hooks/hooks-cursor.json)" = "1" ] || err "hooks/hooks-cursor.json version is not 1"
  extra="$(jq -r 'keys - ["version","hooks"] | join(", ")' hooks/hooks-cursor.json)"
  [ -z "$extra" ] || err "hooks/hooks-cursor.json has a key besides version and hooks: $extra"
fi

echo "→ opencode adapter"
if command -v node >/dev/null 2>&1; then
  for js in index.js .opencode/plugins/builder.js; do
    if chk="$(node --check "$js" 2>&1)"; then note "✓ node --check $js"; else printf '%s\n' "$chk"; err "$js failed node --check"; fi
  done
  if oc_out="$(node --test tests/test_opencode.mjs 2>&1)"; then note "✓ opencode adapter tests"; else printf '%s\n' "$oc_out"; err "opencode adapter tests failed"; fi
else
  note "opencode adapter: skipped (no node)"
fi

echo "→ release config"
for f in release-please-config.json .release-please-manifest.json; do
  jq -e . "$f" >/dev/null 2>&1 && note "✓ $f" || err "$f is not valid JSON"
done

echo "→ commit subjects"
if git rev-parse --verify -q origin/main >/dev/null 2>&1; then
  if ! subjects="$(git log origin/main..HEAD --format=%s 2>&1)"; then
    err "git log origin/main..HEAD failed: $subjects"
  elif bad="$(bash scripts/check-subjects.sh <<<"$subjects")" || [ -z "$subjects" ]; then
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

echo "→ harness words in skills"
# Skills say the action; skills/builder/harnesses/ maps it to each harness's own tool.
tools='\b(PushNotification|SendMessage|AskUserQuestion|TodoWrite|ScheduleWakeup|CronCreate|CronDelete|Monitor|Artifact|WebFetch|apply_patch|update_plan)\b'
slash='(^|[[:space:](`])/(loop|usage|compact|clear|plugin|builder:[a-z-]+)\b'
guard_hits="$(grep -rnE --exclude-dir=harnesses -e "$tools" -e "$slash" skills || true)"
if [ -n "$guard_hits" ]; then
  while IFS= read -r h; do err "guard: $h"; done <<<"$guard_hits"
else
  note "✓ no harness tool or slash command names in skills"
fi

# A skill runs the shim through the quoted plugin root, so a root with a space still works.
# Only files agents read are guarded (SKILL.md, *-prompt.md, scripts); a README is for people and says "<plugin root>".
# A script beside a skill has no CLAUDE_PLUGIN_ROOT on every harness, so it runs "$PLUGIN_ROOT/bin/builder".
shim_hits="$(grep -rn 'bin/builder' --include=SKILL.md --include='*-prompt.md' --include='*.sh' skills \
  | sed -e 's|"${CLAUDE_PLUGIN_ROOT}/bin/builder"||g' -e '/^[^:]*\.sh:/s|"$PLUGIN_ROOT/bin/builder"||g' \
  | grep 'bin/builder' | cut -d: -f1,2 || true)"
if [ -n "$shim_hits" ]; then
  while IFS= read -r h; do err "guard: $h runs the shim without the quoted plugin root"; done <<<"$shim_hits"
else
  note "✓ every skill runs the shim through the quoted plugin root"
fi

# A string-valued `how` runs on every harness, so it names no slash command. A map-valued `how` is per harness.
if ! how_hits="$(jq -r '.[] | select(.how | type == "string") | select(.how | test("(^|[\\s(\\x60])/(loop|usage|compact|clear|plugin|builder:[a-z-]+)\\b"))
    | "lib/recommendations.json: \(.id) how names a slash command"' lib/recommendations.json 2>&1)"; then
  printf '%s\n' "$how_hits"; err "lib/recommendations.json could not be read"
elif [ -n "$how_hits" ]; then
  while IFS= read -r h; do err "guard: $h"; done <<<"$how_hits"
else
  note "✓ no slash commands in string-valued recommendation steps"
fi

echo "→ no Python"
# builder runs without Python. Only the sample projects of the eval fixtures may hold it.
py_guard() {
  local rc=0 out
  out="$("$@" 2>&1)" || rc=$?
  if [ "$rc" -eq 0 ] && [ -n "$out" ]; then
    while IFS= read -r h; do err "python: $h"; done < <(cut -d: -f1,2 <<<"$out")
  elif [ "$rc" -gt 1 ]; then
    err "python: $1 $2 failed (exit $rc)"
  else
    return 0
  fi
  return 1
}
py_guard git ls-files '*.py' ':!tests/evals/*/*' \
  && note "✓ no Python file"
py_guard git grep -n -I -w -e python3 -- . ':!tests/evals/*/*' ':!docs/superpowers' ':!CHANGELOG.md' ':!scripts/validate.sh' ':!tests/test_validate.sh' \
  && note "✓ no python3 command"
py_guard git grep -n -E 'lib/(harness|ledger|reach|security|signals)\.py|burn-proj\.py|session_start\.py|support-table\.py' -- . ':!docs/superpowers' ':!CHANGELOG.md' ':!scripts/validate.sh' ':!tests/test_validate.sh' ':!skills/budget/SKILL.md' ':!tests/templates_test.go' \
  && note "✓ no deleted Python tool named"

echo "→ go"
unset BUILDER_BIN
if ! command -v go >/dev/null 2>&1; then
  err "go is required: install the Go version in go.mod"
else
  gotmp="$(mktemp -d)"; trap 'rm -rf "$gotmp"' EXIT
  if bd_out="$(go build -o "$gotmp/builder" ./cmd/builder 2>&1)"; then
    note "✓ go build"; export BUILDER_BIN="$gotmp/builder"
    if fmt_out="$(gofmt -l . 2>&1)" && [ -z "$fmt_out" ]; then note "✓ gofmt"; else printf '%s\n' "$fmt_out"; err "gofmt: these files need gofmt -w"; fi
    if vet_out="$(go vet ./... 2>&1)"; then note "✓ go vet"; else printf '%s\n' "$vet_out"; err "go vet failed"; fi
    if gt_out="$(go test ./... 2>&1)"; then note "✓ go test"; else printf '%s\n' "$gt_out"; err "go tests failed"; fi
    if sh_out="$(bash tests/test_shim.sh 2>&1)"; then note "✓ shim"; else printf '%s\n' "$sh_out"; err "shim tests failed"; fi
    # Fixed inputs give a fixed output shape.
    out="$("$BUILDER_BIN" budget 19 1783504800 1782990571 2>&1)" || true
    if grep -qE 'elapsed=.* naive=.* profile_aware=.* reset_in=' <<<"$out"; then
      note "✓ builder budget smoke: $out"
    else
      err "builder budget output shape changed: $out"
    fi
  else
    printf '%s\n' "$bd_out"; err "go build failed"
    for s in gofmt "go vet" "go test" shim "builder budget smoke"; do note "$s: skipped (no binary)"; done
  fi
fi
echo "→ harness drift tests"
if [ -n "${BUILDER_BIN:-}" ]; then
  if dr_out="$(bash tests/test_drift.sh 2>&1)"; then note "✓ drift"; else printf '%s\n' "$dr_out"; err "drift tests failed"; fi
  if sp_out="$(bash tests/test_state_paths.sh 2>&1)"; then note "✓ state paths"; else printf '%s\n' "$sp_out"; err "state path tests failed"; fi
else
  note "drift: skipped (no binary)"; note "state paths: skipped (no binary)"
fi

echo "→ hook tests"
if [ -n "${BUILDER_BIN:-}" ]; then
  if hk_out="$(bash tests/test_hook.sh 2>&1)"; then note "✓ hook"; else printf '%s\n' "$hk_out"; err "hook tests failed"; fi
else
  note "hook: skipped (no binary)"
fi

echo "→ eval fixtures"
if [ -n "${BUILDER_BIN:-}" ]; then
  if ev_out="$(bash tests/test_evals.sh 2>&1)"; then note "✓ eval runner tests"; else printf '%s\n' "$ev_out"; err "eval runner tests failed"; fi
  if fx_out="$(bash tests/evals/run.sh --check 2>&1)"; then note "✓ $fx_out"; else printf '%s\n' "$fx_out"; err "eval fixtures invalid"; fi
else
  note "evals: skipped (no binary)"
fi
if st_out="$(go run ./cmd/support-table check README.md 2>&1)"; then note "✓ $st_out"; else printf '%s\n' "$st_out"; err "README support table is out of date"; fi

echo "→ hedges"
# User-facing docs state what works; support status lives in the generated table.
# "Not proven" is a report field, so it is not on this list.
hedges=0
for p in "was proven" "were proven" "only the local path" "is not verified" "is unverified" "does not claim" "no live"; do
  rc=0
  out="$(git grep -l -I -i -F -e "$p" -- README.md CONTRIBUTING.md 'docs/*.md' ':!docs/superpowers' 2>&1)" || rc=$?
  if [ "$rc" -eq 0 ]; then
    while IFS= read -r f; do err "hedge: $f: $p"; done <<<"$out"; hedges=1
  elif [ "$rc" -ne 1 ]; then
    err "hedge: git grep failed (exit $rc)"; hedges=1; break
  fi
done
[ "$hedges" -eq 0 ] && note "✓ no hedges in user-facing docs"

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
