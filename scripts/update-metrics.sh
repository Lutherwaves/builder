#!/usr/bin/env bash
# Refresh the anonymized ctx-wire badge endpoints under metrics/.
# Publishes ONLY aggregates (total tokens saved, total commands filtered,
# overall reduction %). No session, project, path, or account data.
#
# Usage: bash scripts/update-metrics.sh && git commit -am "chore: refresh metrics"
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p metrics

if ! command -v ctx-wire >/dev/null 2>&1; then
  echo "ctx-wire not found — skipping metrics refresh" >&2
  exit 0
fi

read -r saved commands pct < <(
  ctx-wire gain --json | jq -r '[.saved_tokens, .commands, .savings_pct] | @tsv'
)

# human-readable compaction: 16147021 -> 16.1M, 270232 -> 270K
human() {
  awk -v n="$1" 'BEGIN{
    if (n>=1e9) printf "%.1fB", n/1e9;
    else if (n>=1e6) printf "%.1fM", n/1e6;
    else if (n>=1e3) printf "%.0fK", n/1e3;
    else printf "%d", n;
  }'
}

endpoint() { # label message color -> shields endpoint JSON
  jq -n --arg l "$1" --arg m "$2" --arg c "$3" \
    '{schemaVersion:1, label:$l, message:$m, color:$c, cacheSeconds:3600}'
}

endpoint "ctx-wire saved" "$(human "$saved") tokens" blueviolet > metrics/ctxwire-saved.json
endpoint "commands filtered" "$(human "$commands")" blue        > metrics/ctxwire-commands.json
endpoint "output reduction" "$(printf '%.0f%%' "$pct")" brightgreen > metrics/ctxwire-reduction.json

echo "metrics refreshed:"
for f in metrics/*.json; do printf '  %s → %s\n' "$f" "$(jq -r .message "$f")"; done
