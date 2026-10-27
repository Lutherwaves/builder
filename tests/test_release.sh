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
out="$(bash "$tmp/bump/.github/scripts/bump-version.sh" 99.0.0 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && check "exits 0" ok || check "exits 0 (rc=$rc out=$out)" bad
for f in "${manifests[@]}"; do
  [ "$(jq -r .version "$tmp/bump/$f")" = 99.0.0 ] && check "$f version is 99.0.0" ok || check "$f version is 99.0.0" bad
  changed="$(diff "$tmp/orig/$f" "$tmp/bump/$f" | grep -E '^[<>]')"
  old="$(jq -r .version "$tmp/orig/$f")"
  [ "$changed" = "$(printf '< %s\n> %s' "  \"version\": \"$old\"," '  "version": "99.0.0",')" ] \
    && check "$f: only the version line changes" ok || check "$f: only the version line changes: $changed" bad
done

echo "bump_refuses_bad"
repo_copy "$tmp/bad"
for v in 3.0 v3.0.0 3.0.0- "" "3.0.0 x"; do
  out="$(bash "$tmp/bad/.github/scripts/bump-version.sh" "$v" 2>&1)"; rc=$?
  [ "$rc" -eq 1 ] && check "'$v' exits 1" ok || check "'$v' exits 1 (rc=$rc)" bad
done
same=ok
for f in "${manifests[@]}"; do cmp -s "$root/$f" "$tmp/bad/$f" || same=bad; done
check "a refused version changes no file" "$same"

echo "bump_takes_prerelease"
repo_copy "$tmp/rc"
out="$(bash "$tmp/rc/.github/scripts/bump-version.sh" 3.0.0-rc.1 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$(jq -r .version "$tmp/rc/.claude-plugin/plugin.json")" = 3.0.0-rc.1 ] \
  && check "3.0.0-rc.1 is a version, as in release.yml" ok || check "3.0.0-rc.1 is a version, as in release.yml (rc=$rc out=$out)" bad

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
bash "$scripts/check-next-version.sh" 10.0.0 9.0.0 >/dev/null 2>&1; rc=$?
[ "$rc" -eq 0 ] && check "10.0.0 after 9.0.0 exits 0" ok || check "10.0.0 after 9.0.0 exits 0 (rc=$rc)" bad
for args in "3.0 2.1.0" "3.0.0 v2.1.0" "3.0.0" "3.0.0 2.1.0 x"; do
  bash "$scripts/check-next-version.sh" $args >/dev/null 2>&1; rc=$?
  [ "$rc" -eq 1 ] && check "'$args' exits 1" ok || check "'$args' exits 1 (rc=$rc)" bad
done

echo "bump_merged_guard"
# A stub gh for check-bump-merged.sh: PUBLISHED lists the tags whose release is published and
# verified, PENDING the tags whose release is a draft or a pre-release. Any other tag has no release.
relstubs="$tmp/relstubs"; mkdir -p "$relstubs"
cat > "$relstubs/gh" <<'GH'
#!/bin/bash
[ "$1 $2 $4 $5 $6" = 'release view --json isDraft,isPrerelease -q' ] || exit 9
for t in ${PUBLISHED:-}; do [ "$t" = "$3" ] && { echo "false false"; exit 0; }; done
for t in ${PENDING:-}; do [ "$t" = "$3" ] && { echo "true true"; exit 0; }; done
echo "release not found" >&2; exit 1
GH
chmod +x "$relstubs/gh"
bump_merged() { out="$(printf '%s\n' "$tags" | env PATH="$relstubs:$PATH" bash "$scripts/check-bump-merged.sh" "$@" 2>&1)"; rc=$?; }
tags=$'builder-v2.1.0\nv2.1.0\nv3.0.0-rc.1\nv2.10.0\nv2.9.0'
PUBLISHED="v2.9.0 v2.10.0" bump_merged 2.10.0
[ "$rc" -eq 0 ] && check "plugin 2.10.0 with the published v2.10.0 exits 0" ok || check "plugin 2.10.0 with the published v2.10.0 exits 0 (rc=$rc out=$out)" bad
PUBLISHED="v2.9.0 v2.10.0" bump_merged 2.9.0
want='the plugin is at 2.9.0, but the newest release is v2.10.0: the bump PR for v2.10.0 has not merged. Merge it, then this release runs (RELEASING.md)'
[ "$rc" -eq 1 ] && [ "$out" = "$want" ] && check "a published v2.10.0 with its bump not merged exits 1 with the bump line" ok \
  || check "a published v2.10.0 with its bump not merged exits 1 with the bump line (rc=$rc out=$out)" bad
PUBLISHED="v2.9.0" PENDING="v2.10.0" bump_merged 2.9.0
[ "$rc" -eq 0 ] && check "an unpublished v2.10.0 is skipped: plugin 2.9.0, the newest published, exits 0" ok \
  || check "an unpublished v2.10.0 is skipped (rc=$rc out=$out)" bad
PUBLISHED="v2.9.0" bump_merged 2.9.0
[ "$rc" -eq 0 ] && check "a v2.10.0 tag with no release is skipped" ok || check "a v2.10.0 tag with no release is skipped (rc=$rc out=$out)" bad
PUBLISHED="v2.9.0" bump_merged 2.1.0
[ "$rc" -eq 1 ] && grep -qF 'the newest release is v2.9.0' <<<"$out" && check "plugin 2.1.0 behind the published v2.9.0 exits 1" ok || check "plugin 2.1.0 behind the published v2.9.0 exits 1 (rc=$rc out=$out)" bad
tags='v2.1.0'; bump_merged 2.1.0
[ "$rc" -eq 0 ] && check "the first release: the seed tag v2.1.0 has no release, plugin 2.1.0 exits 0" ok || check "the first release exits 0 (rc=$rc out=$out)" bad
tags=$'v2.1.0\nv3.0.0'; PENDING="v3.0.0" bump_merged 2.1.0
[ "$rc" -eq 0 ] && check "the first release that never published: plugin 2.1.0 exits 0" ok || check "the first release that never published (rc=$rc out=$out)" bad
tags='builder-v2.1.0'; bump_merged 2.1.0
[ "$rc" -eq 1 ] && grep -qF 'seed tag' <<<"$out" && check "no vX.Y.Z tag exits 1 and names the seed tag" ok || check "no vX.Y.Z tag exits 1 and names the seed tag (rc=$rc out=$out)" bad
bash "$scripts/check-bump-merged.sh" </dev/null >/dev/null 2>&1; rc=$?
[ "$rc" -eq 1 ] && check "no plugin version exits 1" ok || check "no plugin version exits 1 (rc=$rc)" bad

# A stub gh for push-tag.sh: the tag ref is TAG_REF ("<type> <sha>", or unset for none).
stubs="$tmp/stubs"; mkdir -p "$stubs"
cat > "$stubs/gh" <<'GH'
#!/bin/bash
echo "gh $*" >> "$GH_LOG"
case "$*" in
  "api repos/o/r/git/ref/tags/v3.0.0 -q .object.type + \" \" + .object.sha")
    [ -n "${TAG_REF:-}" ] || exit 1; echo "$TAG_REF" ;;
  "api repos/o/r/git/tags/"*" -q .object.sha") echo "${TAG_TARGET:-}" ;;
  "api repos/o/r/git/refs -f ref=refs/tags/v3.0.0 -f sha="*) exit "${CREATE_RC:-0}" ;;
  *) exit 9 ;;
esac
GH
chmod +x "$stubs/gh"
push_tag() { : > "$tmp/gh.log"; out="$(env PATH="$stubs:$PATH" GH_LOG="$tmp/gh.log" GH_REPO=o/r bash "$scripts/push-tag.sh" "$@" 3.0.0 abc123 2>&1)"; rc=$?; }

echo "push_tag_creates"
push_tag
[ "$rc" -eq 0 ] && grep -qF 'gh api repos/o/r/git/refs -f ref=refs/tags/v3.0.0 -f sha=abc123' "$tmp/gh.log" \
  && check "no tag: pushes v3.0.0 at the verified commit" ok || check "no tag: pushes v3.0.0 at the verified commit (rc=$rc out=$out $(cat "$tmp/gh.log"))" bad
CREATE_RC=1 push_tag
[ "$rc" -eq 1 ] && grep -qF 'pushing tag v3.0.0 failed: RELEASE_TOKEN may have expired (RELEASING.md)' <<<"$out" \
  && check "a failed push exits 1 and names the token" ok || check "a failed push exits 1 and names the token (rc=$rc out=$out)" bad

echo "push_tag_same_commit"
TAG_REF="commit abc123" push_tag
[ "$rc" -eq 0 ] && ! grep -qF 'git/refs -f' "$tmp/gh.log" && check "a lightweight tag at the verified commit passes, no push" ok || check "a lightweight tag at the verified commit passes (rc=$rc out=$out)" bad
TAG_REF="tag t999" TAG_TARGET=abc123 push_tag
[ "$rc" -eq 0 ] && grep -qF 'gh api repos/o/r/git/tags/t999 -q .object.sha' "$tmp/gh.log" && ! grep -qF 'git/refs -f' "$tmp/gh.log" \
  && check "an annotated tag of the verified commit passes" ok || check "an annotated tag of the verified commit passes (rc=$rc out=$out)" bad

echo "push_tag_other_commit"
want='tag v3.0.0 exists at def456, not at the verified abc123: delete it or release the next version (RELEASING.md)'
TAG_REF="commit def456" push_tag
[ "$rc" -eq 1 ] && [ "$out" = "$want" ] && check "a lightweight tag elsewhere exits 1 with the line" ok || check "a lightweight tag elsewhere exits 1 with the line (rc=$rc out=$out)" bad
TAG_REF="tag t999" TAG_TARGET=def456 push_tag
[ "$rc" -eq 1 ] && [ "$out" = "$want" ] && check "an annotated tag of another commit exits 1 with the line" ok || check "an annotated tag of another commit exits 1 with the line (rc=$rc out=$out)" bad

echo "push_tag_check_only"
TAG_REF="commit def456" push_tag --check
[ "$rc" -eq 1 ] && [ "$out" = "$want" ] && ! grep -qF 'git/refs -f' "$tmp/gh.log" \
  && check "--check refuses a tag elsewhere" ok || check "--check refuses a tag elsewhere (rc=$rc out=$out)" bad
push_tag --check
[ "$rc" -eq 0 ] && ! grep -qF 'git/refs -f' "$tmp/gh.log" && check "--check with no tag pushes nothing" ok || check "--check with no tag pushes nothing (rc=$rc out=$out $(cat "$tmp/gh.log"))" bad
TAG_REF="commit abc123" push_tag --check
[ "$rc" -eq 0 ] && check "--check passes a tag at the verified commit" ok || check "--check passes a tag at the verified commit (rc=$rc out=$out)" bad

echo "promote_first_release_links"
changelog "$tmp/first.md" "- a new thing"
grep -v '^\[Unreleased\]: ' "$tmp/first.md" > "$tmp/first.tmp" && mv "$tmp/first.tmp" "$tmp/first.md"
out="$(bash "$scripts/promote-changelog.sh" 3.0.0 2026-11-01 "$tmp/first.md" 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$(tail -n 2 "$tmp/first.md")" = "$(printf '%s\n' "[Unreleased]: $url/compare/v3.0.0...HEAD" "[3.0.0]: $url/releases/tag/v3.0.0")" ] \
  && check "no [Unreleased] link: both links, the release one a /releases/tag/ link" ok || check "no [Unreleased] link: both links (rc=$rc out=$out $(tail -n 2 "$tmp/first.md"))" bad

echo "promote_refuses_bad"
changelog "$tmp/keep.md" "- a new thing"; cp "$tmp/keep.md" "$tmp/keep.orig"
for args in "3.0 2026-11-01" "3.0.0 2026-1-01" "3.0.0 yesterday"; do
  bash "$scripts/promote-changelog.sh" $args "$tmp/keep.md" >/dev/null 2>&1; rc=$?
  [ "$rc" -eq 1 ] && cmp -s "$tmp/keep.md" "$tmp/keep.orig" && check "'$args' exits 1 and changes nothing" ok || check "'$args' exits 1 and changes nothing (rc=$rc)" bad
done
grep -vxF '## [Unreleased]' "$tmp/keep.orig" > "$tmp/nounrel.md"; cp "$tmp/nounrel.md" "$tmp/nounrel.orig"
bash "$scripts/promote-changelog.sh" 3.0.0 2026-11-01 "$tmp/nounrel.md" >/dev/null 2>&1; rc=$?
[ "$rc" -eq 1 ] && cmp -s "$tmp/nounrel.md" "$tmp/nounrel.orig" && check "no '## [Unreleased]' exits 1 and changes nothing" ok || check "no '## [Unreleased]' exits 1 and changes nothing (rc=$rc)" bad

echo "bot_subject_passes"
title="$(grep -o 'title="docs: v${VERSION} is released"' "$root/.github/workflows/release.yml")"
[ -n "$title" ] && check "release.yml titles the bot pull request docs: v<V> is released" ok || check "release.yml titles the bot pull request docs: v<V> is released" bad
printf '%s\n' "docs: v3.0.0 is released" | bash "$root/scripts/check-subjects.sh" >/dev/null; rc=$?
[ "$rc" -eq 0 ] && check "docs: v3.0.0 is released is a Conventional Commit" ok || check "docs: v3.0.0 is released is a Conventional Commit (rc=$rc)" bad
grep -qF 'gh pr update-branch --rebase "$branch"' "$root/.github/workflows/release.yml" \
  && check "the up-to-date loop rebases, so no merge commit fails the subject check" ok \
  || check "the up-to-date loop rebases, so no merge commit fails the subject check" bad

echo "bot_pr_waits_for_publish"
wf="$root/.github/workflows/release.yml"
wait_line="$(grep -nF 'gh release view "v${VERSION}" --json isDraft,isPrerelease' "$wf" | head -1 | cut -d: -f1)"
merge_line="$(grep -nF 'gh pr merge "$branch" --squash --auto' "$wf" | head -1 | cut -d: -f1)"
[ -n "$wait_line" ] && [ -n "$merge_line" ] && [ "$wait_line" -lt "$merge_line" ] \
  && check "the bot pull request waits for the published release before auto-merge" ok \
  || check "the bot pull request waits for the published release before auto-merge (wait=$wait_line merge=$merge_line)" bad
grep -qF '[ "$state" = "false false" ]' "$wf" \
  && check "the wait needs isDraft and isPrerelease both false" ok \
  || check "the wait needs isDraft and isPrerelease both false" bad

echo "release_fails_loudly"
promote="$(sed -n '/- name: Promote the changelog/,/- name: Keep the pull request/p' "$wf")"
uptodate="$(sed -n '/- name: Keep the pull request up to date/,$p' "$wf")"
grep -A1 -F 'is not published and verified after 60 min' <<<"$promote" | grep -qx '[[:space:]]*exit 1' \
  && check "a release still unpublished after 60 min fails the job" ok || check "a release still unpublished after 60 min fails the job" bad
merge_at="$(grep -nF 'gh pr merge "$branch" --squash --auto' <<<"$promote" | cut -d: -f1)"
out_at="$(grep -nF 'echo "merging=true" >> "$GITHUB_OUTPUT"' <<<"$promote" | cut -d: -f1)"
[ -n "$merge_at" ] && [ -n "$out_at" ] && [ "$merge_at" -lt "$out_at" ] \
  && check "merging=true is set only after auto-merge is on" ok || check "merging=true is set only after auto-merge is on (merge=$merge_at out=$out_at)" bad
grep -A1 -F 'gh pr merge "$branch" --squash --auto' <<<"$promote" | grep -q 'exit 1' \
  && check "a failed auto-merge fails the job" ok || check "a failed auto-merge fails the job" bad
grep -F 'CLOSED*)' <<<"$uptodate" | grep -q 'exit 1' && check "a closed pull request fails the job" ok || check "a closed pull request fails the job" bad
grep -A1 -F 'has not merged in 40 min' <<<"$uptodate" | grep -qx '[[:space:]]*exit 1' \
  && check "a pull request still open after 40 min fails the job" ok || check "a pull request still open after 40 min fails the job" bad
grep -qF '.github/scripts/check-bump-merged.sh' "$wf" && check "the release job checks that the previous bump merged" ok || check "the release job checks that the previous bump merged" bad
grep -qF '.github/scripts/push-tag.sh "$VERSION" "$SHA"' "$wf" && check "the release job pushes the tag through push-tag.sh" ok || check "the release job pushes the tag through push-tag.sh" bad
check_at="$(grep -nF '.github/scripts/push-tag.sh --check "$VERSION" "$SHA"' "$wf" | head -1 | cut -d: -f1)"
create_at="$(grep -nF 'gh release create "v${VERSION}"' "$wf" | head -1 | cut -d: -f1)"
[ -n "$check_at" ] && [ -n "$create_at" ] && [ "$check_at" -lt "$create_at" ] \
  && check "a tag elsewhere stops the job before a draft release exists" ok || check "a tag elsewhere stops the job before a draft release exists (check=$check_at create=$create_at)" bad

echo "publish_needs_main"
pub="$root/.github/workflows/publish.yml"
build="$(sed -n '/^  build:/,/^  attest:/p' "$pub")"
grep -qF 'git merge-base --is-ancestor "$GITHUB_SHA" origin/main' <<<"$build" \
  && grep -qF 'refusing to publish $TAG: its commit $GITHUB_SHA is not on main' <<<"$build" \
  && check "publish refuses a tag whose commit is not on main" ok || check "publish refuses a tag whose commit is not on main" bad
grep -A3 -F 'actions/checkout@' <<<"$build" | grep -qF 'fetch-depth: 0' && check "the build checkout has the history" ok || check "the build checkout has the history" bad
check_at="$(grep -nF 'merge-base --is-ancestor' <<<"$build" | cut -d: -f1)"
build_at="$(grep -nF 'go build -trimpath' <<<"$build" | cut -d: -f1)"
[ -n "$check_at" ] && [ -n "$build_at" ] && [ "$check_at" -lt "$build_at" ] && check "the main check runs before the build" ok || check "the main check runs before the build" bad
! grep -qF 'git fetch' <<<"$build" && check "the main check uses the origin/main of the full checkout, with no second fetch" ok || check "the main check uses the origin/main of the full checkout, with no second fetch" bad

exit "$fails"
