#!/usr/bin/env bash
# Test the release scripts in .github/scripts/ on temp copies. Needs jq, no git and no Go.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
scripts="$root/.github/scripts"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
url="https://github.com/Lutherwaves/builder"
manifests=(.claude-plugin/plugin.json .codex-plugin/plugin.json .cursor-plugin/plugin.json package.json)

# changelog <file> <unreleased body>: a changelog with one released section and the seed link.
changelog() {
  {
    echo "# Changelog"
    echo
    echo "## [Unreleased]"
    echo
    [ -n "$2" ] && printf '%s\n\n' "$2"
    echo "## [2.1.0](https://example.com) (2026-10-26)"
    echo
    echo "* an old feature"
    echo
    echo "[Unreleased]: $url/compare/v2.1.0...HEAD"
  } > "$1"
}

echo "release_notes_prints_unreleased"
changelog "$tmp/notes.md" $'### Features\n\n- a new thing'
out="$(bash "$scripts/release-notes.sh" "$tmp/notes.md" 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = $'### Features\n\n- a new thing' ] \
  && check "prints the [Unreleased] body, trimmed" ok || check "prints the [Unreleased] body, trimmed (rc=$rc out=$out)" bad

echo "release_notes_fails_empty"
changelog "$tmp/empty.md" ""
out="$(bash "$scripts/release-notes.sh" "$tmp/empty.md" 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && grep -qF 'nothing under [Unreleased]' <<<"$out" \
  && check "empty [Unreleased] exits 1" ok || check "empty [Unreleased] exits 1 (rc=$rc)" bad

echo "promote_moves_unreleased"
changelog "$tmp/promote.md" "- a new thing"
out="$(bash "$scripts/promote-changelog.sh" 3.0.0 2026-11-01 "$tmp/promote.md" 2>&1)"; rc=$?
want="$(printf '%s\n' "# Changelog" "" "## [Unreleased]" "" "## [3.0.0] - 2026-11-01" "" "- a new thing" "" \
  "## [2.1.0](https://example.com) (2026-10-26)" "" "* an old feature" "" \
  "[Unreleased]: $url/compare/v3.0.0...HEAD" "[3.0.0]: $url/compare/v2.1.0...v3.0.0")"
[ "$rc" -eq 0 ] && [ "$(<"$tmp/promote.md")" = "$want" ] \
  && check "3.0.0 under an empty [Unreleased], with both links" ok || check "3.0.0 under an empty [Unreleased], with both links (rc=$rc out=$out)" bad

echo "promote_twice_is_noop"
cp "$tmp/promote.md" "$tmp/promote.once"
out="$(bash "$scripts/promote-changelog.sh" 3.0.0 2026-11-01 "$tmp/promote.md" 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && cmp -s "$tmp/promote.md" "$tmp/promote.once" \
  && check "a second run changes nothing" ok || check "a second run changes nothing (rc=$rc out=$out)" bad

# repo_copy <dir>: the bump script and the four manifests, as in the repository.
repo_copy() {
  mkdir -p "$1/.github/scripts" "$1/.claude-plugin" "$1/.codex-plugin" "$1/.cursor-plugin"
  cp "$scripts/bump-version.sh" "$1/.github/scripts/"
  for f in "${manifests[@]}"; do cp "$root/$f" "$1/$f"; done
}

echo "bump_sets_four_files"
repo_copy "$tmp/bump"; repo_copy "$tmp/orig"
out="$(bash "$tmp/bump/.github/scripts/bump-version.sh" 3.0.0 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && check "exits 0" ok || check "exits 0 (rc=$rc out=$out)" bad
for f in "${manifests[@]}"; do
  [ "$(jq -r .version "$tmp/bump/$f")" = 3.0.0 ] && check "$f version is 3.0.0" ok || check "$f version is 3.0.0" bad
  changed="$(diff "$tmp/orig/$f" "$tmp/bump/$f" | grep -E '^[<>]')"
  old="$(jq -r .version "$tmp/orig/$f")"
  [ "$changed" = "$(printf '< %s\n> %s' "  \"version\": \"$old\"," '  "version": "3.0.0",')" ] \
    && check "$f: only the version line changes" ok || check "$f: only the version line changes: $changed" bad
done

echo "bump_refuses_bad"
repo_copy "$tmp/bad"
for v in 3.0 v3.0.0 3.0.0-rc.1 "" "3.0.0 x"; do
  out="$(bash "$tmp/bad/.github/scripts/bump-version.sh" "$v" 2>&1)"; rc=$?
  [ "$rc" -eq 1 ] && check "'$v' exits 1" ok || check "'$v' exits 1 (rc=$rc)" bad
done
same=ok
for f in "${manifests[@]}"; do cmp -s "$root/$f" "$tmp/bad/$f" || same=bad; done
check "a refused version changes no file" "$same"

echo "next_version_guard"
bash "$scripts/check-next-version.sh" 3.0.0 2.1.0 >/dev/null 2>&1; rc=$?
[ "$rc" -eq 0 ] && check "3.0.0 after 2.1.0 exits 0" ok || check "3.0.0 after 2.1.0 exits 0 (rc=$rc)" bad
out="$(bash "$scripts/check-next-version.sh" 0.1.0 2.1.0 2>&1)"; rc=$?
seed='go-semantic-release says 0.1.0, but the plugin is already 2.1.0. Push the tag v2.1.0 on its release commit (RELEASING.md, "Before the first release").'
[ "$rc" -eq 1 ] && [ "$out" = "$seed" ] && check "0.1.0 after 2.1.0 exits 1 with the seed line" ok || check "0.1.0 after 2.1.0 exits 1 with the seed line (rc=$rc out=$out)" bad
bash "$scripts/check-next-version.sh" 2.10.0 2.9.0 >/dev/null 2>&1; rc=$?
[ "$rc" -eq 0 ] && check "2.10.0 after 2.9.0 exits 0" ok || check "2.10.0 after 2.9.0 exits 0 (rc=$rc)" bad
bash "$scripts/check-next-version.sh" 2.1.0 2.1.0 >/dev/null 2>&1; rc=$?
[ "$rc" -eq 1 ] && check "2.1.0 after 2.1.0 exits 1" ok || check "2.1.0 after 2.1.0 exits 1 (rc=$rc)" bad

echo "bot_subject_passes"
title="$(grep -o 'title="docs: v${VERSION} is released"' "$root/.github/workflows/release.yml")"
[ -n "$title" ] && check "release.yml titles the bot pull request docs: v<V> is released" ok || check "release.yml titles the bot pull request docs: v<V> is released" bad
printf '%s\n' "docs: v3.0.0 is released" | bash "$root/scripts/check-subjects.sh" >/dev/null; rc=$?
[ "$rc" -eq 0 ] && check "docs: v3.0.0 is released is a Conventional Commit" ok || check "docs: v3.0.0 is released is a Conventional Commit (rc=$rc)" bad
grep -qF 'gh pr update-branch --rebase "$branch"' "$root/.github/workflows/release.yml" \
  && check "the up-to-date loop rebases, so no merge commit fails the subject check" ok \
  || check "the up-to-date loop rebases, so no merge commit fails the subject check" bad

exit "$fails"
