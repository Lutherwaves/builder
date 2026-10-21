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
# (for example $HOME, whose .claude.json is real). Refuse a cfg whose settings.json parity did not write.
realpath_() { python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$1"; }
refuse() {
  echo "✗ parity refuses CLAUDE_CONFIG_DIR=$cfg: $1" >&2
  echo "  Pick a separate, empty folder for BUILDER_PARITY_CONFIG_DIR. Your real config dir is $src." >&2
  exit 2
}
r_cfg="$(realpath_ "$cfg")"; r_src="$(realpath_ "$src")"
case "$r_cfg/" in "$r_src/"*) refuse "it is your real config dir or sits inside it." ;; esac
case "$r_src/" in "$r_cfg/"*) refuse "it holds your real config dir." ;; esac
[ "$r_cfg/.claude.json" = "$(realpath_ "$src_state")" ] && refuse "its .claude.json is your real one."
if [ -e "$cfg/settings.json" ] && [ ! -e "$cfg/.builder-parity" ]; then
  refuse "it holds a settings.json that parity did not write."
fi

# The builder-led config dir. Parity owns it (marker .builder-parity) and rebuilds it on each run:
# - plugins: a copy of the installed superpowers version and its marketplace, nothing else;
# - skills: a link to the gstack clone, and a copy of the skill folders gstack setup made (.gstack-owned);
# - settings.json and .claude.json: only what builder recommends.
# No path in it leads back into the real config, except the gstack clone, which Claude Code does not manage.
# It never copies credentials or account data.
make_config() {
  mkdir -p "$cfg" && : > "$cfg/.builder-parity" || return 1
  python3 - "$src" "$src_state" "$cfg" <<'PY'
import json, os, shutil, sys
from pathlib import Path
src, src_state, cfg = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])

def load(p):
    try:
        d = json.loads(Path(p).read_text())
    except (OSError, ValueError):
        return {}
    return d if isinstance(d, dict) else {}

def clear(p):  # remove a link without following it, or a folder parity owns
    if p.is_symlink() or p.is_file():
        p.unlink()
    elif p.is_dir():
        shutil.rmtree(p)

def copy_in(path):  # copy one folder under the person's plugins folder to the same place under cfg
    try:
        rel = Path(path).relative_to(src / "plugins")
    except ValueError:
        print("note: %s is outside your plugins folder; parity does not copy it" % path, file=sys.stderr)
        return
    if Path(path).is_dir():
        shutil.copytree(path, cfg / "plugins" / rel)

def relink(obj):  # paths into the person's plugins folder now point to cfg's copies
    return json.loads(json.dumps(obj).replace(json.dumps(str(src / "plugins"))[1:-1], json.dumps(str(cfg / "plugins"))[1:-1]))

for d in ("plugins", "skills"):
    clear(cfg / d)
(cfg / "plugins").mkdir()
installed = load(src / "plugins/installed_plugins.json")
sp = {k: v for k, v in (installed.get("plugins") or {}).items() if k.startswith("superpowers@")}
if not sp:
    print("note: superpowers is not installed; the builder-led run has no superpowers", file=sys.stderr)
for entries in sp.values():
    for e in entries if isinstance(entries, list) else []:
        if isinstance(e, dict) and e.get("installPath"):
            copy_in(e["installPath"])
(cfg / "plugins/installed_plugins.json").write_text(json.dumps(relink(dict(installed, plugins=sp)), indent=2))
markets = {k.split("@", 1)[1] for k in sp}
known = {k: v for k, v in load(src / "plugins/known_marketplaces.json").items() if k in markets}
for v in known.values():
    if isinstance(v, dict) and v.get("installLocation"):
        copy_in(v["installLocation"])
(cfg / "plugins/known_marketplaces.json").write_text(json.dumps(relink(known), indent=2))

gstack = src / "skills/gstack"
if gstack.is_dir():
    (cfg / "skills").mkdir()
    os.symlink(gstack, cfg / "skills/gstack")
    for d in sorted((src / "skills").iterdir()):
        if d.name != "gstack" and (d / ".gstack-owned").is_file():
            shutil.copytree(d, cfg / "skills" / d.name, symlinks=True)
else:
    print("note: gstack is not installed; the builder-led run has no gstack", file=sys.stderr)

# The matchblox hooks and status line that `matchblox setup` wrote to the person's settings.
settings = load(src / "settings.json")
hooks = {}
for event, groups in (settings.get("hooks") or {}).items():
    for g in groups if isinstance(groups, list) else []:
        keep = [h for h in (g.get("hooks") or []) if isinstance(h, dict) and "matchblox hook" in str(h.get("command", ""))]
        if keep:
            hooks.setdefault(event, []).append(dict(g, hooks=keep))
status = settings.get("statusLine") if "matchblox hook" in str((settings.get("statusLine") or {}).get("command", "")) else None
if not hooks:
    print("note: your settings have no matchblox hooks, so the builder-led run has none. Run `matchblox setup` first for a fair run.", file=sys.stderr)
new = {"enabledPlugins": {k: True for k in sp}, "env": {"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"}}
if hooks:
    new["hooks"] = hooks
if status:
    new["statusLine"] = status
(cfg / "settings.json").write_text(json.dumps(new, indent=2))

# The code graph server, merged into cfg's own .claude.json so a kept sign-in survives.
server = (load(src_state).get("mcpServers") or {}).get("codebase-memory-mcp")
if server is None:
    print("note: codebase-memory-mcp is not set up; the builder-led run has no code graph", file=sys.stderr)
else:
    state = load(cfg / ".claude.json")
    state.setdefault("mcpServers", {})["codebase-memory-mcp"] = server
    (cfg / ".claude.json").write_text(json.dumps(state, indent=2))
PY
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
if ! python3 -c 'import json,sys; sys.exit(0 if json.loads(sys.argv[1]).get("loggedIn") is True else 1)' "$auth" 2>/dev/null; then
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
