#!/usr/bin/env bash
# Install the builder binary of one version: download, check, then move it into place.
set -euo pipefail

repo=Lutherwaves/builder
usage='usage: install.sh [--version vX.Y.Z|X.Y.Z]'
die() { echo "builder: $1" >&2; exit "${2:-1}"; }

src="${BASH_SOURCE[0]}"
if [[ $src == */* ]]; then dir="${src%/*}"; else dir=.; fi
root="$(cd "${dir:-/}/.." && pwd -P)"

version=""
have_version=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version) [ $# -ge 2 ] || { echo "$usage" >&2; exit 2; }; version="$2"; have_version=1; shift 2 ;;
    -h|--help) echo "$usage"; exit 0 ;;
    *) echo "$usage" >&2; exit 2 ;;
  esac
done

if [ -z "$have_version" ]; then
  manifest="$root/.claude-plugin/plugin.json"
  {
    while IFS= read -r line || [[ -n $line ]]; do
      if [[ $line =~ \"version\"[[:space:]]*:[[:space:]]*\"([^\"]*)\" ]]; then
        version="${BASH_REMATCH[1]}"
        break
      fi
    done < "$manifest"
  } 2>/dev/null || true
  [ -n "$version" ] || die "cannot read the version in $manifest"
fi
version="${version#v}"
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || die "\"$version\" is not a version like 3.0.0" 2

uname_s="$(uname -s)"; uname_m="$(uname -m)"
case "$uname_s" in Linux) os=linux ;; Darwin) os=darwin ;; *) os="" ;; esac
case "$uname_m" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) arch="" ;; esac
if [ -z "$os" ] || [ -z "$arch" ]; then
  die "no binary for $uname_s/$uname_m. builder ships for Linux and macOS on amd64 and arm64."
fi

command -v curl >/dev/null 2>&1 || die "curl is required. Install it, then run again."

asset="builder-$os-$arch"
# A BUILDER_RELEASE_URL is a test or a mirror: it may be file://, and it is no yes of the person.
if [ -n "${BUILDER_RELEASE_URL:-}" ]; then
  base="$BUILDER_RELEASE_URL" protos='=https,file' source=" from $BUILDER_RELEASE_URL"
else
  base="https://github.com/$repo/releases/download" protos='=https' source=""
fi
url="$base/v$version/$asset"
fetch() { curl -fsSL --proto "$protos" --proto-redir '=https' --connect-timeout 10 --max-time 300 "$@" 2>/dev/null; }

builder_home="${BUILDER_HOME:-$HOME/.config/builder}"
bin_dir="$builder_home/bin"
mkdir -p "$bin_dir"
tmp="$(mktemp "$bin_dir/.install.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

fetch -o "$tmp" "$url" || die "cannot download $url. Check the network and the version, then run again."
sums="$(fetch "$url.sha256")" || die "cannot download $url.sha256. Check the network and the version, then run again."

if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum < "$tmp")"
elif command -v shasum >/dev/null 2>&1; then
  got="$(shasum -a 256 < "$tmp")"
else
  die "sha256sum or shasum is required. Install one, then run again."
fi
got="${got%% *}"
want="${sums%%[[:space:]]*}"
[ "$got" = "$want" ] || die "checksum mismatch for $asset v$version: expected $want, got $got. Nothing was installed."

final="$bin_dir/builder-$version"
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
  if ! gh attestation --help >/dev/null 2>&1; then
    provenance="provenance not checked: this gh has no attestation command (gh 2.49 or newer has it). Check it: gh attestation verify \"$final\" --repo $repo"
  elif ! verdict="$(gh attestation verify "$tmp" --repo "$repo" \
      --signer-workflow "$repo/.github/workflows/publish.yml" --source-ref "refs/tags/v$version" 2>&1)"; then
    echo "$verdict" >&2
    die "the provenance check failed for $asset v$version. Nothing was installed."
  else
    provenance="provenance ok"
  fi
else
  provenance="provenance not checked: gh is missing or signed out. Check it: gh attestation verify \"$final\" --repo $repo"
fi

chmod 0755 "$tmp"
ran="$("$tmp" version 2>/dev/null)" || ran=""
[ "$ran" = "v$version" ] || die "the downloaded binary says \"$ran\", not v$version. Nothing was installed."

mv -f "$tmp" "$final"

kept=1
others=""
for f in "$bin_dir"/builder-*; do
  name="${f##*/}"
  [[ $name =~ ^builder-([0-9]+\.[0-9]+\.[0-9]+)$ ]] || continue
  [ "${BASH_REMATCH[1]}" = "$version" ] || others="$others${BASH_REMATCH[1]}"$'\n'
done
if [ -n "$others" ]; then
  for v in $(printf '%s' "$others" | sort -t. -k1,1nr -k2,2nr -k3,3nr); do
    if [ "$kept" -lt 3 ]; then kept=$((kept + 1)); else rm -f "$bin_dir/builder-$v"; fi
  done
fi

if [ -z "$source" ]; then
  now="$(/bin/date +%s 2>/dev/null || date +%s)"
  { echo "yes $now" > "$builder_home/install.answer"; } 2>/dev/null \
    || die "cannot write $builder_home/install.answer. Check that $builder_home is writable, then run again."
fi

echo "installed builder $version at $final$source (checksum ok, $provenance)"
