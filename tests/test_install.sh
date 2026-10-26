#!/usr/bin/env bash
# Test scripts/install.sh: it downloads, checks and installs the binary of one version. No Go, no network.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
if [ -x "$(command -v sha256sum 2>/dev/null)" ]; then sum() { sha256sum "$1"; }; else sum() { shasum -a 256 "$1"; }; fi

plugin="$tmp/plug in"
mkdir -p "$plugin/scripts" "$plugin/.claude-plugin"
cp "$root/scripts/install.sh" "$plugin/scripts/install.sh" 2>/dev/null || { echo "  ✗ scripts/install.sh is missing"; exit 1; }
printf '{\n  "name": "builder",\n  "version": "9.9.9"\n}\n' > "$plugin/.claude-plugin/plugin.json"

fakes="$tmp/fakes"; mkdir -p "$fakes"
printf '#!/bin/bash\ncase "$1" in -s) echo "${FAKE_OS:-Linux}";; -m) echo "${FAKE_ARCH:-x86_64}";; esac\n' > "$fakes/uname"
cat > "$fakes/gh" <<'GH'
#!/bin/bash
if [ "$1 $2" = "auth status" ]; then exit "${GH_AUTH:-1}"; fi
if [ "$1 $2" = "attestation verify" ]; then
  echo "gh $*" > "$GH_LOG"
  if [ "${GH_VERIFY:-0}" -ne 0 ]; then echo "fake attestation failure" >&2; fi
  exit "${GH_VERIFY:-0}"
fi
GH
chmod +x "$fakes/uname" "$fakes/gh"
export GH_LOG="$tmp/gh.log"

# A fake release for one target: the binary prints "v<claimed>", the .sha256 line is the real one.
mkrel() { # ver [claimed] [os-arch]
  local ver="$1" claimed="${2:-$1}" target="${3:-linux-amd64}" d="$tmp/rel/v$1"
  mkdir -p "$d"
  printf '#!/bin/bash\necho v%s\n' "$claimed" > "$d/builder-$target"
  (cd "$d" && sum "builder-$target" > "builder-$target.sha256")
}

bhome=""
fresh() { bhome="$tmp/b home"; rm -rf "$bhome" "$tmp/rel"; mkdir -p "$bhome/bin"; }
run() { # args...; sets out err rc
  out="$(env BUILDER_HOME="$bhome" BUILDER_RELEASE_URL="file://$tmp/rel" PATH="$fakes:$PATH" \
    bash "$plugin/scripts/install.sh" "$@" 2>"$tmp/err")"; rc=$?
  err="$(cat "$tmp/err")"
}
installed() { ls "$bhome/bin" | grep -E '^builder-[0-9]+\.[0-9]+\.[0-9]+$' | sed 's/^builder-//' | sort -t. -k1,1n -k2,2n -k3,3n | sed 's/^/builder-/' | tr '\n' ' '; }
mode() { stat -c %a "$1" 2>/dev/null || stat -f %Lp "$1"; }
nothing_left() { [ -z "$(ls -A "$bhome/bin")" ]; }

echo "test_installs_plugin_version"
fresh; mkrel 9.9.9; run
bin="$bhome/bin/builder-9.9.9"
[ "$rc" -eq 0 ] && [ "$out" = "installed builder 9.9.9 at $bin (checksum ok, provenance not checked: gh is missing or signed out. Check it: gh attestation verify \"$bin\" --repo Lutherwaves/builder)" ] \
  && check "installs the plugin version and says provenance was not checked" ok || check "installs the plugin version (got $rc: $out / $err)" bad
[ -x "$bin" ] && [ "$(mode "$bin")" = 755 ] && [ "$("$bin" version)" = v9.9.9 ] \
  && check "the binary is mode 755 and runs" ok || check "the binary is mode 755 and runs" bad
[[ "$(cat "$bhome/install.answer")" =~ ^yes\ [0-9]+$ ]] && check "install.answer is 'yes <epoch>'" ok || check "install.answer is 'yes <epoch>'" bad
[ "$(printf '%s\n' "$out" | wc -l | tr -d ' ')" = 1 ] && check "exactly one stdout line" ok || check "exactly one stdout line" bad

echo "test_version_flag_with_v"
fresh; mkrel 9.8.0; run --version v9.8.0
[ "$rc" -eq 0 ] && [ -x "$bhome/bin/builder-9.8.0" ] && check "--version v9.8.0 installs builder-9.8.0" ok || check "--version v9.8.0 installs builder-9.8.0 (got $rc: $out / $err)" bad
fresh; mkrel 9.8.0; run --version 9.8.0
[ "$rc" -eq 0 ] && [ -x "$bhome/bin/builder-9.8.0" ] && check "--version 9.8.0 installs builder-9.8.0" ok || check "--version 9.8.0 installs builder-9.8.0 (got $rc: $out / $err)" bad

echo "test_bad_version"
fresh; run --version 'x;1'
[ "$rc" -eq 2 ] && [ "$err" = 'builder: "x;1" is not a version like 3.0.0' ] && check "a bad version exits 2" ok || check "a bad version exits 2 (got $rc: $err)" bad

fresh; run --version ''
[ "$rc" -eq 2 ] && [ "$err" = 'builder: "" is not a version like 3.0.0' ] && check "an empty --version exits 2" ok || check "an empty --version exits 2 (got $rc: $err)" bad

echo "test_backslash_in_home"
fresh; mkrel 9.9.9; bhome="$tmp/back\\slash home"; rm -rf "$bhome"; run
[ "$rc" -eq 0 ] && [ -x "$bhome/bin/builder-9.9.9" ] && check "a BUILDER_HOME with a backslash installs" ok || check "a BUILDER_HOME with a backslash installs (got $rc: $out / $err)" bad

echo "test_unsupported_target"
fresh; mkrel 9.9.9; FAKE_OS=FreeBSD run
[ "$rc" -eq 1 ] && [ "$err" = "builder: no binary for FreeBSD/x86_64. builder ships for Linux and macOS on amd64 and arm64." ] \
  && check "an unsupported OS exits 1 with the line" ok || check "an unsupported OS exits 1 with the line (got $rc: $err)" bad
export FAKE_OS=FreeBSD; run; unset FAKE_OS
[ "$rc" -eq 1 ] && [ "$err" = "builder: no binary for FreeBSD/x86_64. builder ships for Linux and macOS on amd64 and arm64." ] \
  && check "FreeBSD/x86_64 exits 1 with the line" ok || check "FreeBSD/x86_64 exits 1 with the line (got $rc: $err)" bad

echo "test_targets_map"
fresh; mkrel 9.9.9 9.9.9 darwin-arm64; export FAKE_OS=Darwin FAKE_ARCH=arm64; run; unset FAKE_OS FAKE_ARCH
[ "$rc" -eq 0 ] && check "Darwin/arm64 fetches builder-darwin-arm64" ok || check "Darwin/arm64 fetches builder-darwin-arm64 (got $rc: $err)" bad
fresh; mkrel 9.9.9 9.9.9 linux-arm64; export FAKE_ARCH=aarch64; run; unset FAKE_ARCH
[ "$rc" -eq 0 ] && check "aarch64 maps to arm64" ok || check "aarch64 maps to arm64 (got $rc: $err)" bad
fresh; mkrel 9.9.9 9.9.9 linux-amd64; export FAKE_ARCH=amd64; run; unset FAKE_ARCH
[ "$rc" -eq 0 ] && check "amd64 maps to amd64" ok || check "amd64 maps to amd64 (got $rc: $err)" bad

echo "test_missing_release"
fresh; run
[ "$rc" -eq 1 ] && [ "$err" = "builder: cannot download file://$tmp/rel/v9.9.9/builder-linux-amd64. Check the network and the version, then run again." ] \
  && nothing_left && check "a missing release exits 1 with the download line" ok || check "a missing release exits 1 with the download line (got $rc: $err)" bad

echo "test_checksum_mismatch"
fresh; mkrel 9.9.9
bad="$(printf '0%.0s' $(seq 64))"; printf '%s  builder-linux-amd64\n' "$bad" > "$tmp/rel/v9.9.9/builder-linux-amd64.sha256"
run
good="$(sum "$tmp/rel/v9.9.9/builder-linux-amd64")"; good="${good%% *}"
[ "$rc" -eq 1 ] && [ "$err" = "builder: checksum mismatch for builder-linux-amd64 v9.9.9: expected $bad, got $good. Nothing was installed." ] \
  && nothing_left && [ ! -e "$bhome/install.answer" ] && check "a bad checksum installs nothing" ok || check "a bad checksum installs nothing (got $rc: $err)" bad

echo "test_provenance_fails"
fresh; mkrel 9.9.9; GH_AUTH=0 GH_VERIFY=1 run
[[ $err == *"fake attestation failure"* ]] && [[ $err == *"builder: the provenance check failed for builder-linux-amd64 v9.9.9. Nothing was installed." ]] \
  && [ "$rc" -eq 1 ] && nothing_left && check "a failed provenance check installs nothing" ok || check "a failed provenance check installs nothing (got $rc: $err)" bad

echo "test_provenance_ok"
fresh; mkrel 9.9.9; GH_AUTH=0 run
[ "$rc" -eq 0 ] && [ "$out" = "installed builder 9.9.9 at $bhome/bin/builder-9.9.9 (checksum ok, provenance ok)" ] \
  && check "a passing provenance check is reported" ok || check "a passing provenance check is reported (got $rc: $out / $err)" bad
grep -q -- "--repo Lutherwaves/builder --signer-workflow Lutherwaves/builder/.github/workflows/publish.yml --source-ref refs/tags/v9.9.9" "$GH_LOG" \
  && check "gh gets the repo, signer workflow and tag" ok || check "gh gets the repo, signer workflow and tag ($(cat "$GH_LOG"))" bad

echo "test_wrong_version_inside"
fresh; mkrel 9.9.9 1.2.3; run
[ "$rc" -eq 1 ] && [ "$err" = 'builder: the downloaded binary says "v1.2.3", not v9.9.9. Nothing was installed.' ] && nothing_left \
  && check "a binary of another version installs nothing" ok || check "a binary of another version installs nothing (got $rc: $err)" bad

echo "test_keeps_three_newest"
fresh; for v in 1.0.0 1.9.0 1.10.0; do : > "$bhome/bin/builder-$v"; done; mkrel 2.0.0; run --version 2.0.0
[ "$rc" -eq 0 ] && [ "$(installed)" = "builder-1.9.0 builder-1.10.0 builder-2.0.0 " ] \
  && check "1.10.0 sorts above 1.9.0; 1.0.0 goes" ok || check "1.10.0 sorts above 1.9.0 (got $rc: $(installed) $err)" bad
fresh; for v in 1.0.0 1.1.0 1.2.0; do : > "$bhome/bin/builder-$v"; done; mkrel 2.0.0; run --version 2.0.0
[ "$(installed)" = "builder-1.1.0 builder-1.2.0 builder-2.0.0 " ] && check "three installed + a new one leaves the three newest" ok || check "three newest (got $(installed))" bad

echo "test_keeps_an_old_install"
fresh; for v in 3.0.0 3.1.0 3.2.0; do : > "$bhome/bin/builder-$v"; done; mkrel 2.9.0; run --version 2.9.0
[ "$rc" -eq 0 ] && [ "$(installed)" = "builder-2.9.0 builder-3.1.0 builder-3.2.0 " ] \
  && check "the installed version stays even when older" ok || check "the installed version stays (got $rc: $(installed) $err)" bad

echo "test_other_names_untouched"
fresh; for v in 1.0.0 1.1.0 1.2.0; do : > "$bhome/bin/builder-$v"; done; : > "$bhome/bin/builder-dev"; : > "$bhome/bin/builder-1.2.0-rc1"; : > "$bhome/bin/other"
mkrel 2.0.0; run --version 2.0.0
[ -e "$bhome/bin/builder-dev" ] && [ -e "$bhome/bin/builder-1.2.0-rc1" ] && [ -e "$bhome/bin/other" ] && [ ! -e "$bhome/bin/builder-1.0.0" ] \
  && check "the prune touches only builder-N.N.N" ok || check "the prune touches only builder-N.N.N" bad

echo "test_only_shasum"
if command -v shasum >/dev/null 2>&1; then
  fresh; mkrel 9.9.9
  lean="$tmp/lean"; mkdir -p "$lean"
  for t in bash env perl curl mktemp chmod mv rm mkdir sort date shasum ls cat head tr cut dirname; do
    p="$(command -v "$t" 2>/dev/null)" && [ -x "$p" ] && ln -sf "$p" "$lean/$t"
  done
  cp "$fakes/uname" "$fakes/gh" "$lean/"
  out="$(env BUILDER_HOME="$bhome" BUILDER_RELEASE_URL="file://$tmp/rel" PATH="$lean" bash "$plugin/scripts/install.sh" 2>"$tmp/err")"; rc=$?
  [ "$rc" -eq 0 ] && [ -x "$bhome/bin/builder-9.9.9" ] && check "installs with shasum and no sha256sum" ok || check "installs with shasum and no sha256sum (got $rc: $out / $(cat "$tmp/err"))" bad
else
  echo "  shasum: skipped (no shasum)"
fi

echo "test_help"
run -h
[ "$rc" -eq 0 ] && [[ $out == usage:* ]] && check "-h prints usage" ok || check "-h prints usage (got $rc: $out)" bad

exit "$fails"
