#!/usr/bin/env bash
# setup.sh — install builder-tui: download the release binary matching this
# plugin's version and verify its checksum, or build from this checkout when
# Go is installed (or when BUILDER_TUI_BUILD=1). Idempotent.
#
# Env: BUILDER_BIN_DIR   install directory (default ~/.local/bin)
#      BUILDER_TUI_BUILD=1  always build from source
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
BIN_DIR="${BUILDER_BIN_DIR:-$HOME/.local/bin}"
DEST="$BIN_DIR/builder-tui"
REPO="Lutherwaves/builder"

[ "$(uname -s)" = Linux ] || { echo "error: builder-tui reads /proc and /sys and runs on Linux only." >&2; exit 2; }
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "error: no build for $(uname -m)" >&2; exit 2 ;;
esac
mkdir -p "$BIN_DIR"

build() {
  command -v go >/dev/null 2>&1 || { echo "error: no release binary and no Go to build one (https://go.dev/dl)." >&2; exit 2; }
  echo "→ building from $HERE"
  (cd "$HERE" && CGO_ENABLED=0 GOTOOLCHAIN=auto go build -trimpath -ldflags="-s -w -X main.version=source" -o "$DEST.new" .)
  mv "$DEST.new" "$DEST"
}

download() {
  version="$(jq -r .version "$ROOT/.claude-plugin/plugin.json" 2>/dev/null)" || return 1
  [ -n "$version" ] && [ "$version" != null ] || return 1
  # release-please tags carry the package name: builder-v1.2.3
  base="https://github.com/$REPO/releases/download/builder-v$version"
  asset="builder-tui-linux-$arch"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  echo "→ downloading $asset v$version"
  curl -fsSL -o "$tmp/$asset" "$base/$asset" || return 1
  curl -fsSL -o "$tmp/checksums.txt" "$base/builder-tui-checksums.txt" || return 1
  (cd "$tmp" && grep " $asset\$" checksums.txt | sha256sum -c --quiet -) || { echo "error: checksum mismatch for $asset" >&2; exit 3; }
  install -m 0755 "$tmp/$asset" "$DEST"
}

if [ "${BUILDER_TUI_BUILD:-}" = 1 ]; then
  build
elif ! download; then
  echo "→ no release binary for this version; building instead"
  build
fi

echo "✓ installed $DEST ($("$DEST" version))"
case ":$PATH:" in *":$BIN_DIR:"*) ;; *) echo "  note: $BIN_DIR is not on PATH" ;; esac
echo "  run:    tmux new-window -d -n cockpit builder-tui"
echo "  goals:  ~/.config/builder/tui.toml (see $HERE/config.example.toml)"
