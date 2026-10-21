#!/usr/bin/env bash
# Read commit subjects or titles on stdin, one per line. Print each one that is not a
# Conventional Commit and exit 1. An empty line fails too. No exceptions.
set -u
re='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9,#-]+\))?!?: .+'
rc=0
while IFS= read -r s || [ -n "$s" ]; do
  if ! [[ "$s" =~ $re ]]; then printf '%s\n' "${s:-(empty)}"; rc=1; fi
done
exit "$rc"
