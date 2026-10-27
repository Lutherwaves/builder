#!/usr/bin/env bash
# Passes when the plugin version equals the newest vX.Y.Z whose release is
# published and verified (not a draft, not a pre-release). When it does not,
# the bump pull request of that release has not merged, and a release now
# would skip it. A tag whose release never published is skipped, so a broken
# release does not hold back the fixed one.
#
# Usage: <tag names, one per line> | GH_REPO=<owner/repo> check-bump-merged.sh <plugin X.Y.Z>
set -euo pipefail

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: <tags> | $0 <plugin version>" >&2
  exit 1
fi
plugin="$1"

tags="$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sed 's/^v//' | sort -t. -k1,1nr -k2,2nr -k3,3nr)" || true
if [ -z "$tags" ]; then
  echo "no vX.Y.Z tag: push the seed tag (RELEASING.md, \"Before the first release\")" >&2
  exit 1
fi
for v in $tags; do
  state="$(gh release view "v$v" --json isDraft,isPrerelease -q '"\(.isDraft) \(.isPrerelease)"' 2>/dev/null)" || state=""
  [ "$state" = "false false" ] || continue
  if [ "$plugin" != "$v" ]; then
    echo "the plugin is at $plugin, but the newest release is v$v: the bump PR for v$v has not merged. Merge it, then this release runs (RELEASING.md)" >&2
    exit 1
  fi
  exit 0
done
