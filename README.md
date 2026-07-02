# builder

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Operational tools for running AI coding agents at scale. Keep an eye on your
Claude subscription usage limits, session/context load, and token efficiency
while you work — without hand-rolling the math or babysitting a dashboard.

Built for [Claude Code](https://claude.com/claude-code) +
[`cctop`](https://github.com/) + [`ctx-wire`](https://github.com/).

## Install

```bash
# Step 1: Add the marketplace
/plugin marketplace add Lutherwaves/builder

# Step 2: Install the plugin
/plugin install builder@builder
```

## Skills

- `/builder:watch-limits` — Install the burn-rate projector and schedule a
  recurring usage-limit watcher (weekly + 5h burn, compaction pick, ctx-wire
  savings). Run it once and the monitor reports every ~30 min.

## What the watcher reports

1. **Weekly burn rate** — how much of your weekly limit you've used, and whether
   you're on pace to blow it before the (fixed) reset. Uses a **weekend-aware**
   projection so a hot weekday start doesn't trigger a false alarm.
2. **Session/context load** — busy vs idle sessions and total subagents.
3. **Compaction pick** — the highest-context *idle* session, safe to compact now.
4. **ctx-wire savings** — cumulative tokens saved + a per-interval delta.

## Prerequisites

- `cctop` on PATH, with its status-line tap writing `~/.claude/cctop/usage.json`
  (see cctop's usage-limits docs). The watcher has no weekly data without it.
- `ctx-wire` (optional) — enables the savings section.

## Standalone use (no plugin)

Just want the projector? Copy `skills/watch-limits/burn-proj.py` anywhere and:

```bash
python3 burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]
# elapsed=15.0%  naive=127%  profile_aware=102%  reset_in=5.95d
```

- **naive** — straight-line `used / elapsed_fraction`; the worst case, self-heals
  as time passes.
- **profile_aware** — discounts remaining weekend hours (Sat ×0.5, Sun ×0.1) for
  a truer mid-week forecast. Tune the multipliers in `mult()` to your cadence.

## License

MIT
