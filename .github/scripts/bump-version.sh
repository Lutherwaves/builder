#!/usr/bin/env bash
# Sets the plugin version in the four manifests. The key order and the
# 2-space indent stay, so only the version line changes.
#
# Usage: bump-version.sh <X.Y.Z>
set -euo pipefail

if [ $# -ne 1 ] || [[ ! "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: $0 <X.Y.Z> (like 3.0.0)" >&2
  exit 1
fi
version="$1"
cd "$(dirname "$0")/../.."

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
for f in .claude-plugin/plugin.json .codex-plugin/plugin.json .cursor-plugin/plugin.json package.json; do
  jq --indent 2 --arg v "$version" '.version = $v' "$f" >"$tmp"
  cat "$tmp" >"$f"
done
echo "bump-version: $version"
