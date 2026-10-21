# Shared by run.sh and parity.sh. Source it: it defines a deny list and functions, and runs nothing.
# The functions read these globals: fx (the fixture dir), tmp (the temp root), tmp_slug (set by eval_init).

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
eval_shims() {
  local real_matchblox
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
