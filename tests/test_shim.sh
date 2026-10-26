#!/usr/bin/env bash
# Test bin/builder: it runs the binary pinned to the plugin version, or says how to install it.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
# pwd -P: on macOS /var is a symlink to /private/var, and the shim resolves its root.
tmp="$(cd "$(mktemp -d)" && pwd -P)"; trap 'rm -rf "$tmp"' EXIT
fails=0
# Each case sets BUILDER_BIN itself; a caller's BUILDER_BIN would skip the version lookup.
unset BUILDER_BIN
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

echo "test_broken_path"
printf '{\n  "name": "builder",\n  "version": "9.9.9"\n}\n' > "$plugin/.claude-plugin/plugin.json"
fakebin "$bhome/bin/builder-9.9.9"
out="$(PATH=/nonexistent BUILDER_HOME="$bhome" /bin/bash "$plugin/bin/builder" version)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "ran $bhome/bin/builder-9.9.9 root=$plugin args=version" ] \
  && check "a broken PATH still runs the binary" ok || check "a broken PATH still runs the binary (got $rc: $out)" bad

echo "test_one_line_manifest_with_more_keys"
printf '{"name":"b","version":"9.9.9","description":"x"}' > "$plugin/.claude-plugin/plugin.json"
out="$(PATH=/nonexistent BUILDER_HOME="$bhome" /bin/bash "$plugin/bin/builder" v)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "builder-9.9.9 " <<<"$out" \
  && check "a one-line manifest with a key after the version reads" ok || check "a one-line manifest with a key after the version reads (got $rc: $out)" bad

echo "test_shim_run_as_a_bare_name"
out="$(cd "$plugin/bin" && PATH=/nonexistent BUILDER_HOME="$bhome" /bin/bash builder version)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "ran $bhome/bin/builder-9.9.9 root=$plugin args=version" ] \
  && check "a bare-name run finds the root" ok || check "a bare-name run finds the root (got $rc: $out)" bad

printf '{\n  "name": "builder",\n  "version": "9.9.9"\n}\n' > "$plugin/.claude-plugin/plugin.json"
usage="usage: builder install-answer later|never. To say yes, run: bash \"$plugin/scripts/install.sh\""

echo "test_install_answer_later"
rm -rf "$bhome"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" install-answer later)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = 'builder: saved "not now". The install offer comes back in 7 days.' ] \
  && [[ "$(cat "$bhome/install.answer")" =~ ^later\ [0-9]+$ ]] \
  && check "later: exit 0, the line, and 'later <epoch>' in the file" ok || check "later: exit 0, the line, and 'later <epoch>' in the file (got $rc: $out)" bad

echo "test_install_answer_never"
rm -rf "$bhome"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" install-answer never)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "builder: saved \"never\". To turn the binary on, run: bash \"$plugin/scripts/install.sh\"" ] \
  && [[ "$(cat "$bhome/install.answer")" =~ ^never\ [0-9]+$ ]] \
  && check "never: exit 0, the line, and 'never <epoch>' in the file" ok || check "never: exit 0, the line, and 'never <epoch>' in the file (got $rc: $out)" bad

echo "test_install_answer_bad"
rm -rf "$bhome"
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" install-answer yes 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 2 ] && [ "$err" = "$usage" ] && [ ! -e "$bhome/install.answer" ] \
  && check "a bad verdict: exit 2, the usage line, no file" ok || check "a bad verdict: exit 2, the usage line, no file (got $rc: $err)" bad
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" install-answer 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 2 ] && [ "$err" = "$usage" ] \
  && check "no verdict: exit 2 and the usage line" ok || check "no verdict: exit 2 and the usage line (got $rc: $err)" bad

echo "test_install_answer_unwritable"
printf 'x' > "$tmp/afile"
err="$(env -u BUILDER_BIN BUILDER_HOME="$tmp/afile/h" "$plugin/bin/builder" install-answer never 2>&1)"; rc=$?
[ "$rc" -eq 1 ] && [ "$err" = "builder: cannot write $tmp/afile/h/install.answer. Check that $tmp/afile/h is writable, then run again." ] \
  && check "an unwritable BUILDER_HOME: exit 1, the fix line, nothing said saved" ok || check "an unwritable BUILDER_HOME: exit 1, the fix line, nothing said saved (got $rc: $err)" bad

echo "test_install_answer_broken_path"
rm -rf "$bhome"
out="$(PATH=/nonexistent BUILDER_HOME="$bhome" /bin/bash "$plugin/bin/builder" install-answer later)"; rc=$?
[[ "$(cat "$bhome/install.answer")" =~ ^later\ [0-9]+$ ]] && [ "$rc" -eq 0 ] \
  && check "a broken PATH still saves 'later <epoch>'" ok || check "a broken PATH still saves 'later <epoch>' (got $rc: $out)" bad

echo "test_install_answer_beats_builder_bin"
rm -rf "$bhome"
fakebin "$tmp/dev/builder"
out="$(BUILDER_BIN="$tmp/dev/builder" BUILDER_HOME="$bhome" "$plugin/bin/builder" install-answer later)"; rc=$?
[ "$rc" -eq 0 ] && [[ $out != ran* ]] && [ -f "$bhome/install.answer" ] \
  && check "install-answer runs in the shim, not in BUILDER_BIN" ok || check "install-answer runs in the shim, not in BUILDER_BIN (got $rc: $out)" bad

echo "test_never_says_off"
rm -rf "$bhome"; mkdir -p "$bhome"; echo "never 1" > "$bhome/install.answer"
err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" x 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 3 ] && [ "$err" = "builder 9.9.9 is off: the person said never. Turn it on: bash \"$plugin/scripts/install.sh\"" ] \
  && check "never and no binary: exit 3 and the off line" ok || check "never and no binary: exit 3 and the off line (got $rc: $err)" bad
err="$(PATH=/nonexistent BUILDER_HOME="$bhome" /bin/bash "$plugin/bin/builder" x 2>&1 >/dev/null)"; rc=$?
[ "$rc" -eq 3 ] && [[ $err == *"is off: the person said never"* ]] \
  && check "the off line needs no PATH" ok || check "the off line needs no PATH (got $rc: $err)" bad

echo "test_later_keeps_install_line"
for ans in "later 1" "yes 1" "" "garbage"; do
  rm -rf "$bhome"; mkdir -p "$bhome"; printf '%s\n' "$ans" > "$bhome/install.answer"
  err="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" x 2>&1 >/dev/null)"; rc=$?
  [ "$rc" -eq 3 ] && [ "$err" = "builder 9.9.9 is not installed. Install it: bash \"$plugin/scripts/install.sh\"" ] \
    && check "answer '$ans' and no binary: today's install line" ok || check "answer '$ans' and no binary: today's install line (got $rc: $err)" bad
done

echo "test_never_does_not_block_an_installed_binary"
rm -rf "$bhome"; mkdir -p "$bhome"; echo "never 1" > "$bhome/install.answer"
fakebin "$bhome/bin/builder-9.9.9"
out="$(env -u BUILDER_BIN BUILDER_HOME="$bhome" "$plugin/bin/builder" v)"; rc=$?
[ "$rc" -eq 0 ] && grep -q "builder-9.9.9 " <<<"$out" \
  && check "an installed binary runs despite never" ok || check "an installed binary runs despite never (got $rc: $out)" bad

echo "test_real_binary_through_the_shim"
if command -v go >/dev/null 2>&1; then
  (cd "$root" && go build -o "$tmp/real-builder" ./cmd/builder) \
    && out="$(BUILDER_BIN="$tmp/real-builder" "$root/bin/builder" version)" && [ "$out" = dev ] \
    && check "the Go binary answers through the shim" ok || check "the Go binary answers through the shim" bad
else
  echo "  go: skipped (no go)"
fi

exit "$fails"
