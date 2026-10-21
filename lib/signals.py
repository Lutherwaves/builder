#!/usr/bin/env python3
"""Cheap, offline machine signals. Every external call is bounded by one shared budget."""
import json
import os
import re
import shutil
import signal as _signal
import subprocess
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import harness as harnesses  # noqa: E402
import ledger  # noqa: E402

try:
    import tomllib
except ImportError:  # Python < 3.11: a line scan reads the few tables builder needs
    tomllib = None

WEEK_MIN = 10080
CANON_MARKERS = ("RULES.md", "DESIGN.md", "docs/rules")  # the homes the canon skill finds or creates
AUTOCOMPACT = "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"
GRAPH_SERVERS = ("codebase-memory-mcp", "codebase-memory")  # user-wide name; the name harness writes to .mcp.json


def _run_all(cmds, cwd, timeout):
    """Start every command at once; return {name: stdout or None} within ~timeout in total."""
    deadline = time.monotonic() + timeout
    procs = {}
    for name, cmd in cmds.items():
        try:
            procs[name] = subprocess.Popen(
                cmd, cwd=cwd, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL, text=True, start_new_session=True)
        except (OSError, ValueError):
            procs[name] = None
    live = [p for p in procs.values() if p is not None]
    while live and time.monotonic() < deadline:
        live = [p for p in live if p.poll() is None]
        if live:
            time.sleep(0.005)
    out = {}
    for name, p in procs.items():
        if p is None:
            out[name] = None
            continue
        if p.poll() is not None:
            try:
                # Finished: read it within the budget. A child it left behind can hold stdout open.
                stdout, _ = p.communicate(timeout=max(0.05, deadline - time.monotonic()))
                out[name] = (p.returncode, stdout)
                continue
            except subprocess.TimeoutExpired:
                pass
        _kill(p)
        out[name] = None
    return out


def _kill(p):
    """Kill the process group (the tool and any child it left), then reap it without waiting long."""
    try:
        os.killpg(p.pid, _signal.SIGKILL)
    except OSError:
        try:
            p.kill()
        except OSError:
            pass
    try:
        p.communicate(timeout=0.2)
    except Exception:
        pass


def _json(res):
    if not res or res[0] != 0:
        return None
    try:
        return json.loads(res[1])
    except ValueError:
        return None


def _stdout(res):
    return res[1].strip() if res and res[0] == 0 else ""


def _weekly_pct(usage):
    best = None
    if not isinstance(usage, list):
        return None
    for acct in usage:
        for w in (acct.get("windows") or []) if isinstance(acct, dict) else []:
            if isinstance(w, dict) and w.get("window_minutes") == WEEK_MIN and isinstance(w.get("used_pct"), (int, float)):
                best = w["used_pct"] if best is None else max(best, w["used_pct"])
    return best


def _git_root(cwd, res):
    if res and res[0] == 0 and res[1].strip():
        return Path(res[1].strip())
    p = Path(cwd).resolve()
    for d in (p, *p.parents):
        if (d / ".git").exists():
            return d
    return p


def _read_json(path):
    try:
        data = json.loads(Path(path).read_text())
    except (OSError, ValueError):
        return {}
    return data if isinstance(data, dict) else {}


def _hook_commands(settings):
    """Command strings from {"hooks": {event: [{"hooks": [{"command": ...}]}]}} (Claude Code's shape)."""
    hooks = settings.get("hooks")
    for groups in (hooks.values() if isinstance(hooks, dict) else ()):
        for group in (groups if isinstance(groups, list) else ()):
            entries = group.get("hooks") if isinstance(group, dict) else None
            for h in (entries if isinstance(entries, list) else ()):
                if isinstance(h, dict) and isinstance(h.get("command"), str):
                    yield h["command"]


def _flat_hook_commands(settings):
    """Command strings from {"hooks": {event: [{"command": ...}]}} (Cursor's shape)."""
    hooks = settings.get("hooks")
    for entries in (hooks.values() if isinstance(hooks, dict) else ()):
        for h in (entries if isinstance(entries, list) else ()):
            if isinstance(h, dict) and isinstance(h.get("command"), str):
                yield h["command"]


_TOML_MCP = re.compile(r'^\[mcp_servers\.("?)([^\]."]+)\1\]', re.M)
_TOML_SUPERPOWERS = re.compile(r'^\[plugins\."superpowers@', re.M)


def _toml(path):
    """(parsed dict or None, raw text). None means the text did not parse: scan it with a regex."""
    try:
        text = Path(path).read_text()
    except OSError:
        return {}, ""
    if tomllib is not None:
        try:
            return tomllib.loads(text), text
        except ValueError:
            pass
    return None, text


def _server_names(path, key):
    """MCP server names in one config file (JSON object under `key`, or TOML [key.<name>] tables)."""
    if Path(path).suffix != ".toml":
        servers = _read_json(path).get(key)
        return set(servers) if isinstance(servers, dict) else set()
    data, text = _toml(path)
    if data is None:
        return {m.group(2) for m in _TOML_MCP.finditer(text)}
    servers = data.get(key)
    return set(servers) if isinstance(servers, dict) else set()


def _mcp_servers(spec, harness, dirs):
    """Server names: the harness's user-level files, its repo-level files in each dir, and (Claude) local scope."""
    names = set()
    for path, key in spec["mcp"]:
        names |= _server_names(path, key)
        if harness == "claude":
            state = _read_json(path)
            projects = state.get("projects") if isinstance(state.get("projects"), dict) else {}
            for d in dirs:
                scope = projects.get(str(d))
                servers = scope.get(key) if isinstance(scope, dict) else None
                names.update(servers if isinstance(servers, dict) else ())
    for d in dirs:
        for rel, key in spec["project_mcp"]:
            names |= _server_names(d / rel, key)
    return names


def _superpowers(harness, spec):
    record = spec["plugins_record"]
    if record is None:
        return None
    if harness == "codex":
        data, text = _toml(record)
        if data is None:
            return bool(_TOML_SUPERPOWERS.search(text))
        plugins = data.get("plugins")
        return isinstance(plugins, dict) and any(str(k).startswith("superpowers@") for k in plugins)
    if harness == "opencode":
        entries = _read_json(record).get("plugin")
        return isinstance(entries, list) and any(isinstance(e, str) and "superpowers" in e for e in entries)
    try:
        return "superpowers@" in Path(record).read_text()
    except OSError:
        return False


def _gstack(harness, spec):
    skills = spec["config_dir"] / "skills"
    if harness == "claude":
        return (skills / "gstack").exists()
    return any(skills.glob("gstack-*"))


def _matchblox_hooks(harness, spec):
    if harness == "claude":
        return any("matchblox hook" in c for c in _hook_commands(_read_json(spec["settings"])))
    if harness == "cursor":
        return any("matchblox hook --cursor" in c for c in _flat_hook_commands(_read_json(spec["settings"])))
    return None


def _autocompact(harness, spec):
    if harness != "claude":
        return None
    env = _read_json(spec["settings"]).get("env")
    return (isinstance(env, dict) and AUTOCOMPACT in env) or AUTOCOMPACT in os.environ


def _process_cmds():
    """One pgrep per harness process match; -a (Linux) or -l (macOS) prints each pid with its args."""
    show = "-l" if sys.platform == "darwin" else "-a"
    return {
        "pgrep:%d" % i: ["pgrep", show, "-" + mode, pattern]
        for i, (mode, pattern, _) in enumerate(harnesses.all_process_match())
    }


def _count_processes(res):
    total = 0
    for i, (_, _, exclude) in enumerate(harnesses.all_process_match()):
        out = res.get("pgrep:%d" % i)
        if not out or out[0] != 0:
            continue
        total += sum(1 for line in out[1].splitlines() if line.strip() and not (exclude and exclude in line))
    return total


def collect(cwd, timeout=0.5, harness=None):
    env = os.environ
    home = Path.home()
    harness = harness or harnesses.detect(env)
    if harness not in harnesses.HARNESSES:
        harness = "claude"  # an unknown harness keeps the Claude Code view
    spec = harnesses.paths(harness, env, home)
    cmds = {}
    has = {n: shutil.which(n) is not None for n in ("matchblox", "gh", "pgrep", "git")}
    if has["matchblox"]:
        cmds["sessions"] = ["matchblox", "sessions", "--json"]
        cmds["usage"] = ["matchblox", "usage", "--json"]
    if has["gh"]:
        cmds["gh"] = ["gh", "auth", "token"]
    if has["pgrep"]:
        cmds.update(_process_cmds())
    if has["git"]:
        cmds["git"] = ["git", "rev-parse", "--show-toplevel"]
        cmds["gpgsign"] = ["git", "config", "--get", "commit.gpgsign"]
        cmds["signkey"] = ["git", "config", "--get", "user.signingkey"]
    res = _run_all(cmds, cwd, timeout) if cmds else {}

    try:
        channel = json.loads((ledger.home() / "reach.json").read_text()).get("channel")
        channel = channel if isinstance(channel, str) and channel else None
    except (OSError, ValueError, AttributeError):
        channel = None
    sessions = _json(res.get("sessions"))
    root = _git_root(cwd, res.get("git"))
    servers = _mcp_servers(spec, harness, {root, Path(cwd).resolve()})

    return {
        "superpowers": _superpowers(harness, spec),
        "gstack": _gstack(harness, spec),
        "gh": bool(res.get("gh")) and res["gh"][0] == 0,
        "signing": _stdout(res.get("gpgsign")) == "true" and bool(_stdout(res.get("signkey"))),
        "channel": channel,
        "matchblox": has["matchblox"],
        "matchblox_hooks": _matchblox_hooks(harness, spec) if has["matchblox"] else False,
        "code_graph": any(n in servers for n in GRAPH_SERVERS),
        "autocompact": _autocompact(harness, spec),
        "sessions": len(sessions) if isinstance(sessions, list) else None,
        "weekly_pct": _weekly_pct(_json(res.get("usage"))),
        "sessions_proc": _count_processes(res),
        "has_canon": any((root / m).exists() for m in CANON_MARKERS),
    }


if __name__ == "__main__":
    print(json.dumps(collect(os.getcwd())))
