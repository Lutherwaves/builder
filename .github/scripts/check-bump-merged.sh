#!/usr/bin/env bash
# Passes when the plugin version equals the newest vX.Y.Z tag. When it does
# not, the bump pull request of that release has not merged, and a release
# now would skip it.
#
# Usage: <tag names, one per line> | check-bump-merged.sh <plugin X.Y.Z>
set -euo pipefail

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: <tags> | $0 <plugin version>" >&2
  exit 1
fi
plugin="$1"

newest="$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sed 's/^v//' | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)" || true
if [ -z "$newest" ]; then
  echo "no vX.Y.Z tag: push the seed tag (RELEASING.md, \"Before the first release\")" >&2
  exit 1
fi
if [ "$plugin" != "$newest" ]; then
  echo "the plugin is at $plugin, but the newest release is v$newest: the bump PR for v$newest has not merged. Merge it, then this release runs (RELEASING.md)" >&2
  exit 1
fi
