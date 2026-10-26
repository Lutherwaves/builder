#!/usr/bin/env bash
# Moves the [Unreleased] entries of the changelog under the released version,
# and adds its link. The entries do not change; [Unreleased] stays, empty.
# Running it again for the same version changes nothing.
#
# Usage: promote-changelog.sh <version> <date> [changelog]
#   version  without the v (0.1.0)
#   date     YYYY-MM-DD
set -euo pipefail

if [ $# -lt 2 ] || [ $# -gt 3 ]; then
  echo "usage: $0 <version> <date> [changelog]" >&2
  exit 1
fi
version="$1"
date="$2"
file="${3:-$(dirname "$0")/../../CHANGELOG.md}"
url="https://github.com/Lutherwaves/builder"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "promote-changelog: '$version' is not a version (like 0.1.0)" >&2
  exit 1
fi
if [[ ! "$date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
  echo "promote-changelog: '$date' is not YYYY-MM-DD" >&2
  exit 1
fi
if grep -qF "## [$version] - " "$file"; then
  echo "promote-changelog: $file already has $version"
  exit 0
fi
if ! grep -qxF "## [Unreleased]" "$file"; then
  echo "promote-changelog: no '## [Unreleased]' in $file" >&2
  exit 1
fi

# The previous release is the base of the [Unreleased] compare link. The
# first release has no link yet: it gets both, at the end of the file.
previous="$(sed -n 's|^\[Unreleased\]: .*/compare/v\(.*\)\.\.\.HEAD$|\1|p' "$file")"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
awk -v version="$version" -v date="$date" -v previous="$previous" -v url="$url" '
  !heading && $0 == "## [Unreleased]" {
    print; print ""; print "## [" version "] - " date
    heading = 1; next
  }
  !link && previous != "" && $0 ~ /^\[Unreleased\]: / {
    print "[Unreleased]: " url "/compare/v" version "...HEAD"
    print "[" version "]: " url "/compare/v" previous "...v" version
    link = 1; next
  }
  { print }
  END {
    if (previous == "") {
      print ""
      print "[Unreleased]: " url "/compare/v" version "...HEAD"
      print "[" version "]: " url "/releases/tag/v" version
    }
  }
' "$file" >"$tmp"
cat "$tmp" >"$file"
echo "promote-changelog: $version ($date)${previous:+, after $previous}"
