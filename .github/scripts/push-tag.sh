#!/usr/bin/env bash
# Pushes the tag v<version> on the verified commit. A tag that exists passes
# only when it names that commit, lightweight or annotated. --check refuses a
# tag elsewhere and pushes nothing.
#
# Usage: GH_REPO=<owner/repo> push-tag.sh [--check] <version> <sha>
set -euo pipefail

check_only=""
if [ "${1:-}" = --check ]; then check_only=1; shift; fi
if [ $# -ne 2 ] || [ -z "${GH_REPO:-}" ]; then
  echo "usage: GH_REPO=<owner/repo> $0 [--check] <version> <sha>" >&2
  exit 1
fi
version="$1"
sha="$2"

if ref="$(gh api "repos/$GH_REPO/git/ref/tags/v$version" -q '.object.type + " " + .object.sha' 2>/dev/null)"; then
  type="${ref%% *}"
  at="${ref#* }"
  if [ "$type" = tag ]; then
    at="$(gh api "repos/$GH_REPO/git/tags/$at" -q .object.sha)"
  fi
  if [ "$at" != "$sha" ]; then
    echo "tag v$version exists at $at, not at the verified $sha: delete it or release the next version (RELEASING.md)" >&2
    exit 1
  fi
  echo "tag v$version exists"
  exit 0
fi
[ -z "$check_only" ] || exit 0
gh api "repos/$GH_REPO/git/refs" -f ref="refs/tags/v$version" -f sha="$sha" >/dev/null ||
  { echo "pushing tag v$version failed: RELEASE_TOKEN may have expired (RELEASING.md)" >&2; exit 1; }
echo "pushed tag v$version at $sha"
