#!/usr/bin/env bash
# Drop-in parity eval. See tests/evals/README.md.
#   parity.sh [--dry-run]
# Runs the builder and shape fixtures twice:
#   current  the person's setup as it is (no --plugin-dir)
#   builder  CLAUDE_CONFIG_DIR=<cfg>: only builder (--plugin-dir) and what builder recommends,
#            with every recommendation accepted. Same plugin and skill versions: no re-install.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
evals="${BUILDER_EVAL_DIR:-$here}"
fixtures=(builder shape)
dry=0
case "${1:-}" in
  --dry-run) dry=1 ;;
  "") ;;
  *) echo "usage: parity.sh [--dry-run]" >&2; exit 2 ;;
esac
for f in "${fixtures[@]}"; do
  for p in prompt.md expect.md; do
    [ -s "$evals/$f/$p" ] || { echo "✗ $f: $p is missing or empty (try run.sh --check)" >&2; exit 2; }
  done
done
. "$here/lib.sh"

src="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"   # the person's config dir
src_state="${CLAUDE_CONFIG_DIR:+$CLAUDE_CONFIG_DIR/.claude.json}"; src_state="${src_state:-$HOME/.claude.json}"
out="${BUILDER_EVAL_OUT:-${XDG_STATE_HOME:-$HOME/.local/state}/builder/evals}/parity/$(date +%Y%m%d-%H%M%S)"

tmp="$(mktemp -d)"
[ -n "$tmp" ] && [ -d "$tmp" ] || { echo "parity: cannot make a temp dir" >&2; exit 1; }
eval_init
# BUILDER_PARITY_CONFIG_DIR: a folder you keep, signed in once (see the auth error below). Default: a temp dir.
cfg="${BUILDER_PARITY_CONFIG_DIR:-$tmp/config}"
[ "$dry" -eq 1 ] && cfg="$tmp/config"   # a dry run writes only to the temp dir
cleanup() {
  local f
  for f in "${fixtures[@]}"; do
    forget_sessions "$tmp/$f-current"
    CLAUDE_CONFIG_DIR="$cfg" forget_sessions "$tmp/$f-builder"
  done
  rm -rf "$tmp"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# Never touch the real config. Refuse a cfg that is the real config dir, sits inside it, or holds it
# (for example $HOME, whose .claude.json is real). Refuse a cfg that parity did not create: it must
# not exist yet, be empty, or hold parity's marker. Parity never deletes anything in a foreign folder.
# realpath_ works as os.path.realpath: it resolves each part left to right, follows a symlink where it
# finds one, and keeps a missing part as written. ".." drops the part before it only after that part is resolved.
realpath_() {
  local rest="$1" out="" part t hops=0
  [[ $rest == /* ]] || rest="$PWD/$rest"
  while [ -n "$rest" ]; do
    part="${rest%%/*}"
    if [[ $rest == */* ]]; then rest="${rest#*/}"; else rest=""; fi
    case "$part" in
      ''|.) ;;
      ..) out="${out%/*}" ;;
      *)
        if [ -L "$out/$part" ] && [ "$hops" -lt 40 ]; then
          t="$(readlink "$out/$part")" || return 0
          hops=$((hops + 1))
          [[ $t == /* ]] && out=""
          rest="$t${rest:+/$rest}"
        else
          out="$out/$part"
        fi ;;
    esac
  done
  printf '%s\n' "${out:-/}"
}
refuse() {
  echo "✗ parity refuses CLAUDE_CONFIG_DIR=$cfg: $1" >&2
  echo "  Pick a separate, empty folder for BUILDER_PARITY_CONFIG_DIR. Your real config dir is $src." >&2
  exit 2
}
r_cfg="$(realpath_ "$cfg")"; r_src="$(realpath_ "$src")"
case "$r_cfg/" in "$r_src/"*) refuse "it is your real config dir or sits inside it." ;; esac
case "$r_src/" in "$r_cfg/"*) refuse "it holds your real config dir." ;; esac
[ "$r_cfg/.claude.json" = "$(realpath_ "$src_state")" ] && refuse "its .claude.json is your real one."
if [ -e "$cfg" ] || [ -L "$cfg" ]; then
  [ -d "$cfg" ] || refuse "it is not a folder."
  if [ ! -e "$cfg/.builder-parity" ] && [ -n "$(ls -A "$cfg" 2>/dev/null)" ]; then
    refuse "it is not empty, and parity did not create it (no .builder-parity marker)."
  fi
fi

# The builder-led config dir. Parity owns it (marker .builder-parity) and rebuilds it on each run:
# - plugins: a copy of the installed superpowers version and its marketplace, nothing else;
# - skills: a link to the gstack clone, and a copy of the skill folders gstack setup made (.gstack-owned);
# - settings.json and .claude.json: only what builder recommends.
# No path in it leads back into the real config, except the gstack clone, which Claude Code does not manage.
# It never copies credentials or account data.
make_config() {
  mkdir -p "$cfg" && : > "$cfg/.builder-parity" || return 1
  (cd "$root" && go build -o "$tmp/parity-config" ./cmd/parity-config) \
    && "$tmp/parity-config" "$src" "$src_state" "$cfg"
}

make_config || { echo "✗ parity: cannot write the builder-led config in $cfg" >&2; exit 1; }

if [ "$dry" -eq 1 ]; then
  echo "# transcripts: $out/<fixture>/{current,builder}.md (stderr: .err)"
  echo "# builder-led config dir: ${BUILDER_PARITY_CONFIG_DIR:-<temp dir>}"
  echo "# settings.json:"; cat "$cfg/settings.json"; echo
  for f in "${fixtures[@]}"; do
    fx="$evals/$f"; eval_base
    echo "# $f current (cwd: temp copy of $fx/repo)"; printf '%q ' "${base[@]}"; echo
    echo "# $f builder (cwd: temp copy of $fx/repo)"; printf 'CLAUDE_CONFIG_DIR=%q ' "$cfg"; printf '%q ' "${base[@]}" --plugin-dir "$root"; echo
  done
  exit 0
fi

# Claude Code keeps its sign-in per config dir. Check it before any model run.
auth="$(CLAUDE_CONFIG_DIR="$cfg" claude auth status --json 2>/dev/null)"   # stdout only: a warning on stderr is not JSON
if ! jq -e '.loggedIn == true' >/dev/null 2>&1 <<<"$auth"; then
  cat >&2 <<EOF
✗ parity: Claude Code is not signed in under CLAUDE_CONFIG_DIR=$cfg.
  Claude Code keeps its sign-in per config dir, so your normal sign-in does not carry over.
  This eval never copies credentials. No fixture ran.
  To fix: pick a separate folder you keep, let parity prepare it, sign in there once, and run again:
    export BUILDER_PARITY_CONFIG_DIR=\$HOME/.local/state/builder/parity-config
    bash tests/evals/parity.sh    # prepares the folder, then stops here
    CLAUDE_CONFIG_DIR=\$BUILDER_PARITY_CONFIG_DIR claude auth login
    bash tests/evals/parity.sh
  claude auth status said: $(tr -s '\n ' ' ' <<<"$auth" | cut -c1-200)
EOF
  exit 4
fi

eval_shims
eval_builder_home "$root" || exit 1
for f in "${fixtures[@]}"; do
  fx="$evals/$f"; eval_base
  mkdir -p "$out/$f" || exit 1
  workdir "$f-current" && workdir "$f-builder" || { echo "✗ cannot prepare the work dirs for $f" >&2; exit 1; }
  (cd "$tmp/$f-current" && "${base[@]}" < "$fx/prompt.md" > "$out/$f/current.md" 2> "$out/$f/current.err") \
    || echo "note: $f current run exited non-zero (see $out/$f/current.err)" >&2
  forget_sessions "$tmp/$f-current"
  (cd "$tmp/$f-builder" && CLAUDE_CONFIG_DIR="$cfg" "${base[@]}" --plugin-dir "$root" < "$fx/prompt.md" > "$out/$f/builder.md" 2> "$out/$f/builder.err") \
    || echo "note: $f builder run exited non-zero (see $out/$f/builder.err)" >&2
  CLAUDE_CONFIG_DIR="$cfg" forget_sessions "$tmp/$f-builder"
done

for f in "${fixtures[@]}"; do
  printf '\n######## %s ########\n' "$f"
  printf '\n===== Expected =====\n'; cat "$evals/$f/expect.md"
  printf '\n===== Current setup =====\n'; cat "$out/$f/current.md"
  printf '\n===== Builder-led setup =====\n'; cat "$out/$f/builder.md"
done
echo; echo "transcripts: $out"
