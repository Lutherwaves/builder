#!/usr/bin/env python3
"""Weekly-limit burn projection with weekend load discounting.

Usage: burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]

Prints naive (constant 7-day burn) and profile-aware projections.
Profile-aware discounts the hours you predictably don't burn — a sleep window
(02:00–09:00, x0.05) and the weekend (Sat x0.5, Sun x0.1) — so the forecast
isn't inflated by charging daytime rates to hours you're asleep or resting.
It also decays automatically as those low-usage hours accumulate.
"""
import sys, time

WINDOW = 604800  # 7d in seconds
HR = 3600

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

def main():
    used = float(sys.argv[1])
    reset = int(sys.argv[2])
    now = int(sys.argv[3]) if len(sys.argv) > 3 else int(time.time())
    start = reset - WINDOW

    el_frac = (now - start) / WINDOW
    naive = used / el_frac if el_frac > 0 else float('nan')

    eff_el = eff_hours(start, now)
    eff_rem = eff_hours(now, reset)
    rate = used / eff_el if eff_el > 0 else 0.0  # % per normal-hour
    prof = used + rate * eff_rem

    reset_in = (reset - now) / 86400
    print(f"elapsed={el_frac*100:.1f}%  naive={naive:.0f}%  profile_aware={prof:.0f}%  "
          f"reset_in={reset_in:.2f}d")

if __name__ == "__main__":
    main()
