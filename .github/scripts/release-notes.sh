#!/usr/bin/env bash
# Prints the [Unreleased] section of the changelog: the notes of the release
# about to be made. Fails when it is empty, so no release ships without
# notes.
#
# Usage: release-notes.sh [changelog]   (default: CHANGELOG.md)
set -euo pipefail

file="${1:-$(dirname "$0")/../../CHANGELOG.md}"

notes="$(awk '
  $0 == "## [Unreleased]" { on = 1; next }
  on && /^## \[/ { exit }
  on && /^\[[^]]+\]: https?:/ { exit }
  on { print }
' "$file")"

# Trim the blank lines around it.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "$notes" ]; then
  echo "release-notes: nothing under [Unreleased] in $file; write the notes before a release" >&2
  exit 1
fi
printf '%s\n' "$notes"
