#!/usr/bin/env bash
# Read commit subjects or titles on stdin, one per line. Print each one that is not a
# Conventional Commit and exit 1. Merge subjects from GitHub and git pass.
set -u
re='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9,#-]+\))?!?: .+'
rc=0
while IFS= read -r s || [ -n "$s" ]; do
  [ -z "$s" ] && continue
  case "$s" in "Merge "*) continue ;; esac
  if ! [[ "$s" =~ $re ]]; then printf '%s\n' "$s"; rc=1; fi
done
exit "$rc"
