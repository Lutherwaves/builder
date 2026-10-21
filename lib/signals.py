#!/usr/bin/env python3
"""Cheap, offline machine signals. Every external call is bounded by one shared budget."""
import json
import os
import shutil
import signal as _signal
import subprocess
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import ledger  # noqa: E402

WEEK_MIN = 10080
CANON_MARKERS = ("DESIGN.md", ".claude/rules", "docs/canon")


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
        elif p.poll() is not None:
            stdout, _ = p.communicate()  # finished: read it, however late we are
            out[name] = (p.returncode, stdout)
        else:
            try:
                os.killpg(p.pid, _signal.SIGKILL)
            except OSError:
                p.kill()
            try:
                p.communicate(timeout=0.2)
            except Exception:
                pass
            out[name] = None
    return out


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


def collect(cwd, timeout=0.5):
    home = Path.home()
    cmds = {}
    has = {n: shutil.which(n) is not None for n in ("matchblox", "gh", "pgrep", "git")}
    if has["matchblox"]:
        cmds["sessions"] = ["matchblox", "sessions", "--json"]
        cmds["usage"] = ["matchblox", "usage", "--json"]
    if has["gh"]:
        cmds["gh"] = ["gh", "auth", "token"]
    if has["pgrep"]:
        cmds["pgrep"] = ["pgrep", "-x", "claude"]
    if has["git"]:
        cmds["git"] = ["git", "rev-parse", "--show-toplevel"]
        cmds["gpgsign"] = ["git", "config", "--get", "commit.gpgsign"]
        cmds["signkey"] = ["git", "config", "--get", "user.signingkey"]
    res = _run_all(cmds, cwd, timeout) if cmds else {}

    try:
        plugins = (home / ".claude/plugins/installed_plugins.json").read_text()
    except OSError:
        plugins = ""
    try:
        channel = json.loads((ledger.home() / "reach.json").read_text()).get("channel")
        channel = channel if isinstance(channel, str) and channel else None
    except (OSError, ValueError, AttributeError):
        channel = None
    sessions = _json(res.get("sessions"))
    root = _git_root(cwd, res.get("git"))
    pg = res.get("pgrep")

    return {
        "superpowers": "superpowers@" in plugins,
        "gstack": (home / ".claude/skills/gstack").exists(),
        "gh": bool(res.get("gh")) and res["gh"][0] == 0,
        "signing": _stdout(res.get("gpgsign")) == "true" and bool(_stdout(res.get("signkey"))),
        "channel": channel,
        "matchblox": has["matchblox"],
        "sessions": len(sessions) if isinstance(sessions, list) else None,
        "weekly_pct": _weekly_pct(_json(res.get("usage"))),
        "sessions_proc": len(pg[1].split()) if pg and pg[0] == 0 else 0,
        "has_canon": any((root / m).exists() for m in CANON_MARKERS),
    }


if __name__ == "__main__":
    print(json.dumps(collect(os.getcwd())))
