#!/usr/bin/env bash
# Test bin/builder: it runs the binary pinned to the plugin version, or says how to install it.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }

plugin="$tmp/plug in"
mkdir -p "$plugin/bin" "$plugin/.claude-plugin"
cp "$root/bin/builder" "$plugin/bin/builder" 2>/dev/null || { echo "  ✗ bin/builder is missing"; exit 1; }
printf '{\n  "name": "builder",\n  "version": "9.9.9"\n}\n' > "$plugin/.claude-plugin/plugin.json"
bhome="$tmp/b home"

# A fake binary that prints what it got.
fakebin() { mkdir -p "$(dirname "$1")"; printf '#!/bin/bash\necho "ran $0 root=$BUILDER_ROOT args=$*"\n' > "$1"; chmod +x "$1"; }

echo "test_builder_bin_wins"
fakebin "$tmp/dev/builder"
out="$(BUILDER_BIN="$tmp/dev/builder" BUILDER_HOME="$bhome" "$plugin/bin/builder" a "b c")"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "ran $tmp/dev/builder root=$plugin args=a b c" ] \
  && check "BUILDER_BIN runs and sees BUILDER_ROOT" ok || check "BUILDER_BIN runs and sees BUILDER_ROOT (got $rc: $out)" bad

echo "test_versioned_binary_runs"
fakebin "$bhome/bin/builder-9.9.9"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" hook --x)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "ran $bhome/bin/builder-9.9.9 root=$plugin args=hook --x" ] \
  && check "the binary of the plugin version runs" ok || check "the binary of the plugin version runs (got $rc: $out)" bad

echo "test_home_default"
fakebin "$tmp/home/.config/builder/bin/builder-9.9.9"
out="$(env -u BUILDER_BIN -u BUILDER_HOME HOME="$tmp/home" "$plugin/bin/builder" x)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "^ran $tmp/home/.config/builder/bin/builder-9.9.9 " <<<"$out" \
  && check "BUILDER_HOME defaults to ~/.config/builder" ok || check "BUILDER_HOME defaults to ~/.config/builder (got $rc: $out)" bad

echo "test_missing_binary_says_install"
rm -rf "$bhome"
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" x 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 3 ] && [ "$err" = "builder 9.9.9 is not installed. Install it: bash \"$plugin/scripts/install.sh\"" ] \
  && check "no binary: exit 3 and the install line" ok || check "no binary: exit 3 and the install line (got $rc: $err)" bad

echo "test_version_with_spaces"
printf '{"name": "builder",\n   "version" :  "1.2.3" ,\n "x": 1}\n' > "$plugin/.claude-plugin/plugin.json"
fakebin "$bhome/bin/builder-1.2.3"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" v)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "builder-1.2.3 " <<<"$out" && check "a version line with odd spaces reads" ok || check "a version line with odd spaces reads (got $rc: $out)" bad

echo "test_one_line_manifest"
printf '{"name":"builder","version":"4.5.6"}' > "$plugin/.claude-plugin/plugin.json"
fakebin "$bhome/bin/builder-4.5.6"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" v)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "builder-4.5.6 " <<<"$out" && check "a one-line plugin.json reads" ok || check "a one-line plugin.json reads (got $rc: $out)" bad

echo "test_no_version_fails"
printf '{"name": "builder"}\n' > "$plugin/.claude-plugin/plugin.json"
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" v 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 3 ] && [ "$err" = "builder: cannot read the version in $plugin/.claude-plugin/plugin.json" ] \
  && check "no version: exit 3 and the read error" ok || check "no version: exit 3 and the read error (got $rc: $err)" bad

echo "test_missing_manifest_fails"
rm -f "$plugin/.claude-plugin/plugin.json"
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" v 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 3 ] && [ "$err" = "builder: cannot read the version in $plugin/.claude-plugin/plugin.json" ] \
  && check "no manifest: exit 3 and the read error" ok || check "no manifest: exit 3 and the read error (got $rc: $err)" bad

echo "test_real_binary_through_the_shim"
if command -v go >/dev/null 2>&1; then
  (cd "$root" && go build -o "$tmp/real-builder" ./cmd/builder) \
    && out="$(BUILDER_BIN="$tmp/real-builder" "$root/bin/builder" version)" && [ "$out" = dev ] \
    && check "the Go binary answers through the shim" ok || check "the Go binary answers through the shim" bad
else
  echo "  go: skipped (no go)"
fi

exit "$fails"
