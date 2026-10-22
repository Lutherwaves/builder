#!/usr/bin/env python3
"""Weekly-limit burn projection with sleep and weekend discounting.

Usage: burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]
       burn-proj.py --from-matchblox [now_epoch]

Prints naive (constant 7-day burn) and profile-aware projections.
Profile-aware discounts the hours you predictably don't burn — a sleep window
(02:00–09:00, x0.05) and the weekend (Sat x0.5, Sun x0.1) — so the forecast
isn't inflated by charging daytime rates to hours you're asleep or resting.
It also decays automatically as those low-usage hours accumulate.

--from-matchblox reads `matchblox usage --json` and projects the weekly window
(window_minutes 10080) of each account: one line per account. When matchblox
lists two or more accounts, each line ends with `account=<name>`. An account
that is not `measured`, or has a missing or malformed weekly window, is skipped
with its cause on stderr.
"""
import json
import shutil
import subprocess
import sys
import time
from datetime import datetime

WINDOW = 604800  # 7d in seconds
WEEKLY_MINUTES = 10080
HR = 3600
USAGE = ("usage: burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]\n"
         "       burn-proj.py --from-matchblox [now_epoch]")
INSTALL = "matchblox not found: install it with curl -fsSL https://matchblox.sh | sh"


def mult(ts):
    lt = time.localtime(ts)
    h, wd = lt.tm_hour, lt.tm_wday  # hour 0..23 ; Mon=0 .. Sun=6
    # Sleep window 02:00–09:00: near-zero usage any day.
    if 2 <= h < 9:
        return 0.05
    # Active window (09:00–02:00), weekend-discounted.
    if wd == 5:  # Sat
        return 0.5
    if wd == 6:  # Sun
        return 0.1
    return 1.0


def eff_hours(a, b):
    """Weekend-weighted hours between epochs a and b."""
    s, t = 0.0, a
    while t < b:
        s += mult(t) * min(HR, b - t) / HR
        t += HR
    return s


def project(used, reset, now):
    start = reset - WINDOW
    el_frac = (now - start) / WINDOW
    naive = used / el_frac if el_frac > 0 else float('nan')

    eff_el = eff_hours(start, now)
    eff_rem = eff_hours(now, reset)
    rate = used / eff_el if eff_el > 0 else 0.0  # % per normal-hour
    prof = used + rate * eff_rem

    reset_in = (reset - now) / 86400
    return (f"elapsed={el_frac*100:.1f}%  naive={naive:.0f}%  profile_aware={prof:.0f}%  "
            f"reset_in={reset_in:.2f}d")


def die(msg, code):
    print(msg, file=sys.stderr)
    sys.exit(code)


def from_matchblox(now):
    exe = shutil.which("matchblox")
    if not exe:
        die(INSTALL, 2)
    r = subprocess.run([exe, "usage", "--json"], capture_output=True, text=True)
    if r.returncode != 0:
        die(f"matchblox usage --json failed (exit {r.returncode}): {r.stderr.strip()}", 1)
    try:
        accounts = json.loads(r.stdout)
    except ValueError:
        die("matchblox usage --json printed no JSON. Run it by hand to see why.", 1)
    if not isinstance(accounts, list) or not all(isinstance(a, dict) for a in accounts):
        die("matchblox usage --json is not a list of accounts. Run it by hand to see why.", 1)
    lines = []
    for acc in accounts:
        name = acc.get("account") or acc.get("provider") or "?"
        line = account_line(acc, name, now)
        if line:
            lines.append((name, line))
    if not lines:
        die("no measured weekly window in matchblox usage: run a Claude Code session so matchblox reads the plan, then try again.", 1)
    for name, line in lines:
        # Label by the number of accounts, not lines: a skipped account must not hide the label.
        print(f"{line}  account={name}" if len(accounts) > 1 else line)


def account_line(acc, name, now):
    """The projection line of one account, or None with the cause on stderr."""
    def skip(cause):
        print(f"account {name}: {cause}", file=sys.stderr)
    state = acc.get("state", "measured")
    if state != "measured":
        return skip(f"state is {state}, not measured; no projection")
    windows = acc.get("windows")
    weekly = [w for w in windows if isinstance(w, dict) and w.get("window_minutes") == WEEKLY_MINUTES] \
        if isinstance(windows, list) else []
    if not weekly:
        return skip("no weekly window in matchblox usage")
    w = weekly[0]
    try:
        used = float(w["used_pct"])
        reset = int(datetime.fromisoformat(str(w["resets_at"]).replace("Z", "+00:00")).timestamp())
    except (KeyError, TypeError, ValueError):
        return skip(f"malformed weekly window (used_pct={w.get('used_pct')!r}, resets_at={w.get('resets_at')!r})")
    return project(used, reset, now)


def main():
    args = sys.argv[1:]
    if args and args[0] == "--from-matchblox":
        from_matchblox(int(args[1]) if len(args) > 1 else int(time.time()))
        return
    if len(args) < 2:
        die(USAGE, 2)
    used = float(args[0])
    reset = int(args[1])
    now = int(args[2]) if len(args) > 2 else int(time.time())
    print(project(used, reset, now))


if __name__ == "__main__":
    main()
