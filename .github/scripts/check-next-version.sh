#!/usr/bin/env bash
# Passes when the next version is greater than the plugin's version. With no
# v* tag yet, go-semantic-release picks a first version below the plugin's:
# the seed tag must come first.
#
# Usage: check-next-version.sh <next> <current>   (both X.Y.Z)
set -euo pipefail

semver='^[0-9]+\.[0-9]+\.[0-9]+$'
if [ $# -ne 2 ] || [[ ! "$1" =~ $semver ]] || [[ ! "$2" =~ $semver ]]; then
  echo "usage: $0 <next X.Y.Z> <current X.Y.Z>" >&2
  exit 1
fi
next="$1"
current="$2"

IFS=. read -r -a n <<<"$next"
IFS=. read -r -a c <<<"$current"
for i in 0 1 2; do
  if ((10#${n[i]} > 10#${c[i]})); then exit 0; fi
  if ((10#${n[i]} < 10#${c[i]})); then break; fi
done
echo "go-semantic-release says $next, but the plugin is already $current. Push the tag v$current on its release commit (RELEASING.md, \"Before the first release\")." >&2
exit 1
