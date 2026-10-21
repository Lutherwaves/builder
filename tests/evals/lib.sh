# Shared by run.sh and parity.sh. Source it: it defines a deny list and functions, and runs nothing.
# The functions read these globals: fx (the fixture dir), tmp (the temp root), tmp_slug (set by eval_init),
# root (the builder repository root, for the per-harness functions).

# acceptEdits lets the agent edit files in the temp dir; the deny list blocks push, network tools,
# the commands that steer the person's real agent sessions (matchblox, tmux), and anything that
# schedules a job or registers an MCP server outside the temp dir.
eval_deny=("Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch"
      "Bash(matchblox send:*)" "Bash(matchblox compact:*)" "Bash(matchblox hold:*)"
      "Bash(matchblox release:*)" "Bash(matchblox kill:*)" "Bash(matchblox new:*)"
      "Bash(matchblox rename:*)" "Bash(matchblox stoker:*)"
      "Bash(tmux send-keys:*)" "Bash(tmux kill-session:*)"
      "CronCreate" "CronDelete" "RemoteTrigger" "Bash(crontab:*)" "Bash(claude mcp:*)")

# Set base=(claude -p ...) for fixture $fx. An optional allow file adds tool rules (one per line),
# e.g. the fixture's test command. The prompt goes in on stdin, so a prompt that starts with '-'
# never parses as an option.
eval_base() {
  local line allow=()
  if [ -f "$fx/allow" ]; then
    while IFS= read -r line || [ -n "$line" ]; do
      case "$line" in ''|'#'*) continue ;; esac
      allow+=("$line")
    done < "$fx/allow"
  fi
  base=(claude -p --output-format text --permission-mode acceptEdits
        ${allow[@]+--allowedTools "${allow[@]}"}
        --disallowedTools "${eval_deny[@]}")
}

# One runner per harness. eval_base_<harness> sets base=(the command) and with_extra=(what the with run adds),
# eval_notes_<harness> prints the dry-run notes, and eval_prepare_<harness> <baseline|with> <work dir> makes the
# temp config and sets runenv=(VAR=value ...) for the run. The prompt always goes in on stdin.
eval_base_claude() { eval_base; with_extra=(--plugin-dir "$root"); }

# Codex: sandboxed to the work dir (no network), approvals off because there is nobody to answer.
eval_base_codex() {
  base=(codex exec -C . -s workspace-write -c 'approval_policy="never"' --ephemeral --color never -)
  with_extra=()
}
# Cursor: --force runs commands without a prompt; its own sandbox stays on.
eval_base_cursor() {
  base=(agent -p --force --trust --sandbox enabled --output-format text)
  with_extra=()
}
# opencode: --auto approves whatever the config does not deny.
eval_base_opencode() { base=(opencode run --auto); with_extra=(); }

eval_notes_claude() { :; }
eval_notes_codex() {
  echo "# env: CODEX_HOME=<temp dir per run>; its auth.json is a symlink to the real auth.json (never copied or printed)"
  echo "# with run, before it starts: CODEX_HOME=<that dir> codex plugin marketplace add $root"
  echo "#                             CODEX_HOME=<that dir> codex plugin add builder@builder"
  echo "# guard: -s workspace-write (writes only in the work dir, no network); PATH shims block gh, git push, matchblox writes and tmux"
}
eval_notes_cursor() {
  echo "# env: the real Cursor config dir stays in use and is not written to"
  echo "# with run, in the work dir: .cursor/hooks.json -> $root/hooks/session-start --harness cursor"
  echo "#                            .agents/skills/<name> -> $root/skills/<name> (symlinks)"
  echo "# guard: deny rules in the work dir's .cursor/cli.json, --sandbox enabled; PATH shims block gh, git push, matchblox writes and tmux"
}
eval_notes_opencode() {
  echo "# env: OPENCODE_CONFIG_DIR=<temp dir per run> holding opencode.json (permission deny rules for webfetch, websearch, git push, gh, crontab)"
  echo "# with run, opencode.json adds: \"plugin\": [\"$root\"]"
  echo "# guard: the deny rules above; PATH shims block gh, git push, matchblox writes and tmux"
}

eval_prepare_claude() { runenv=(); }

eval_prepare_codex() {
  local home="$tmp/codex-$1" real="${CODEX_HOME:-$HOME/.codex}"
  [ -f "$real/auth.json" ] || { echo "✗ codex: no auth.json in $real (run codex login first)" >&2; return 1; }
  mkdir -p "$home" && ln -s "$real/auth.json" "$home/auth.json" || return 1
  if [ "$1" = with ]; then
    CODEX_HOME="$home" codex plugin marketplace add "$root" >/dev/null \
      && CODEX_HOME="$home" codex plugin add builder@builder >/dev/null \
      || { echo "✗ codex: cannot install builder into $home" >&2; return 1; }
  fi
  runenv=(CODEX_HOME="$home")
}

eval_prepare_cursor() { # the deny rules are best effort; the PATH shims are the guard
  local s
  mkdir -p "$2/.cursor" || return 1
  printf '%s\n' '{"permissions": {"allow": [], "deny": ["Shell(gh)", "Shell(tmux)", "Shell(crontab)", "Shell(git push)"]}}' > "$2/.cursor/cli.json"
  if [ "$1" = with ]; then
    printf '{"version": 1, "hooks": {"sessionStart": [{"command": "%s/hooks/session-start --harness cursor"}]}}\n' "$root" > "$2/.cursor/hooks.json"
    mkdir -p "$2/.agents/skills" || return 1
    for s in "$root"/skills/*/; do ln -s "${s%/}" "$2/.agents/skills/$(basename "$s")" || return 1; done
  fi
  runenv=()
}

eval_prepare_opencode() {
  local cfg="$tmp/opencode-$1"
  mkdir -p "$cfg" || return 1
  python3 - "$cfg/opencode.json" "$1" "$root" <<'PY' || return 1
import json, sys
path, mode, root = sys.argv[1:]
conf = {"permission": {"webfetch": "deny", "websearch": "deny",
                       "bash": {"*": "allow", "git push*": "deny", "gh *": "deny", "crontab*": "deny"}}}
if mode == "with":
    conf = {"plugin": [root], **conf}
json.dump(conf, open(path, "w"), indent=2)
PY
  runenv=(OPENCODE_CONFIG_DIR="$cfg")
}

# The version a runner reports: its first --version line, or "unknown".
eval_version() { # $1 = the runner's command name
  local v
  v="$("$1" --version </dev/null 2>/dev/null | head -n 1 | tr -d '\r')"
  case "$v" in ''|*[!A-Za-z0-9._\ +\(\)-]*) v="unknown" ;; esac
  printf '%s' "${v:0:60}"
}

# $out/verdict.json: a stub for the grader. One bullet per expect.md line, met and quote still null.
# It holds no transcript and no local path.
eval_verdict() { # $1 harness, $2 version, $3 fixture name, $4 expect.md, $5 out dir
  python3 - "$1" "$2" "$3" "$4" "$5/verdict.json" <<'PY'
import datetime, json, re, sys
harness, version, fixture, expect, path = sys.argv[1:]
bullets = []
for line in open(expect, encoding="utf-8"):
    m = re.match(r"\s*[-*]\s+(\S.*)$", line)
    if m:
        bullets.append({"text": m.group(1).rstrip(), "met": None, "quote": None})
json.dump({"harness": harness, "harness_version": version, "date": datetime.date.today().isoformat(),
           "fixture": fixture, "bullets": bullets}, open(path, "w"), indent=2)
PY
}

# Claude Code names a session folder after its cwd, every char not [A-Za-z0-9] turned into "-".
slug() { printf '%s' "$1" | sed 's/[^A-Za-z0-9]/-/g'; }

eval_init() { tmp_slug="$(slug "$(cd "$tmp" && pwd -P)")"; }

workdir() { # $1 = name; a fresh copy of repo/ (or an empty dir), made a git repo with one commit
  mkdir -p "$tmp/$1"
  if [ -d "$fx/repo" ]; then cp -R "$fx/repo/." "$tmp/$1/"; fi
  git -C "$tmp/$1" init -q || return 1
  git -C "$tmp/$1" config user.name "Eval"
  git -C "$tmp/$1" config user.email "eval@example.invalid"
  git -C "$tmp/$1" config commit.gpgsign false
  git -C "$tmp/$1" add -A \
    && git -C "$tmp/$1" commit -q --allow-empty -m "initial fixture" || return 1
}

# Claude Code only: the other runners keep no session folder named after the work dir.
# Remove the session folder of one run: exactly the folder named like the work dir, nothing else.
forget_sessions() { # $1 = work dir
  local real s dir
  real="$(cd "$1" 2>/dev/null && pwd -P)" || return 0
  s="$(slug "$real")"
  case "$s" in "$tmp_slug"?*) ;; *) echo "note: not removing session folder for $1 (slug outside temp root)" >&2; return 0 ;; esac
  dir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/projects/$s"
  if [ -d "$dir" ] && [ ! -L "$dir" ]; then rm -rf "$dir"; fi
}

# PATH shims in $tmp/bin: the deny list matches command text, so it misses `matchblox --json send`,
# `env matchblox send` or `bash -c`. The shims stop those at the program. Read-only matchblox
# commands pass to the real one. Call it once; it prepends $tmp/bin to PATH.
# With the argument "strict" (a harness that cannot deny a tool by name) it also blocks gh and `git push`.
eval_shims() {
  local real_matchblox real_git
  real_matchblox="$(command -v matchblox 2>/dev/null || true)"
  mkdir -p "$tmp/bin"
  cat > "$tmp/bin/matchblox" <<SHIM
#!/usr/bin/env bash
real='$real_matchblox'
cmd=""
for a in "\$@"; do case "\$a" in -*) ;; *) cmd="\$a"; break ;; esac; done
case "\$cmd" in
  sessions|ships|usage|peek|status|version|help|"")
    if [ -z "\$cmd" ] && [ "\$#" -gt 0 ]; then
      for a in "\$@"; do case "\$a" in --help|-h) ;; --json) ;; *) echo "eval: matchblox \$a is blocked in evals" >&2; exit 3 ;; esac; done
    fi
    [ -n "\$real" ] && [ -x "\$real" ] || { echo "matchblox not installed" >&2; exit 127; }
    exec "\$real" "\$@" ;;
  *) echo "eval: matchblox \$cmd is blocked in evals" >&2; exit 3 ;;
esac
SHIM
  printf '#!/usr/bin/env bash\necho "eval: tmux is blocked in evals" >&2\nexit 3\n' > "$tmp/bin/tmux"
  chmod +x "$tmp/bin/matchblox" "$tmp/bin/tmux"
  if [ "${1:-}" = strict ]; then
    real_git="$(command -v git 2>/dev/null || true)"
    printf '#!/usr/bin/env bash\necho "eval: gh is blocked in evals" >&2\nexit 3\n' > "$tmp/bin/gh"
    cat > "$tmp/bin/git" <<SHIM
#!/usr/bin/env bash
# Blocks the push subcommand after git's own options. An alias for push still gets through.
real='$real_git'
skip=0
for a in "\$@"; do
  if [ "\$skip" -eq 1 ]; then skip=0; continue; fi
  case "\$a" in
    -C|-c|--git-dir|--work-tree|--namespace|--exec-path) skip=1 ;;
    -*) ;;
    push) echo "eval: git push is blocked in evals" >&2; exit 3 ;;
    *) break ;;
  esac
done
[ -n "\$real" ] && [ -x "\$real" ] || { echo "git not installed" >&2; exit 127; }
exec "\$real" "\$@"
SHIM
    chmod +x "$tmp/bin/gh" "$tmp/bin/git"
  fi
  export PATH="$tmp/bin:$PATH"
}

# A temp builder home: onboarded, every recommendation answered "never". The hook then
# injects only the soul, so no setup question pollutes the transcript.
eval_builder_home() { # $1 = the builder repository root
  export BUILDER_HOME="$tmp/builder-home"
  python3 "$1/lib/ledger.py" onboarded >/dev/null || return 1
  local id
  for id in $(python3 -c 'import json,sys; print(" ".join(i["id"] for i in json.load(open(sys.argv[1]))))' "$1/lib/recommendations.json"); do
    python3 "$1/lib/ledger.py" answer "$id" never >/dev/null || return 1
  done
}
