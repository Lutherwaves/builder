---
name: watch-limits
description: Use when you want a recurring monitor of your Claude subscription usage limits — weekly/5h burn rate, whether you'll blow the weekly cap before it resets, which idle session to compact, and ctx-wire token savings. Triggers: "am I going to hit my limit", "watch my usage", "burn rate", "which session to compact".
---

# watch-limits — recurring usage-limit watcher

## Overview

Installs an activity-aware burn-rate projector and schedules a recurring status
watcher. The watcher reports, every ~30 min: weekly + 5h limit burn, a
compaction pick, and `ctx-wire` savings. **Do not hand-roll the projection or
the schedule — this skill is the canonical setup; reinventing it produces
false alarms and duplicate jobs.**

## Prerequisites

- `cctop` on PATH (`cctop --json` works) — per-session live stats.
- `~/.claude/cctop/usage.json` present and fresh (<1h). Written by a status-line
  tap (`cctop --capture-usage`); if missing, tell the user to set up that tap
  first — the watcher has no weekly data without it.
- `ctx-wire` (optional) for section 4. Skip that section if absent.

## Setup flow

1. **Install the projector:**
   ```bash
   mkdir -p ~/.claude/cctop
   cp "${CLAUDE_PLUGIN_ROOT}/skills/watch-limits/burn-proj.py" ~/.claude/cctop/burn-proj.py
   ```
2. **Guard against duplicates.** List existing scheduled jobs first. If a
   `7,37 * * * *` watcher reading `usage.json` already exists, STOP — offer to
   replace it (delete the old one), never add a second. Duplicate watchers
   double the noise and can clobber each other.
3. **Schedule the watcher.** Use the prompt in `watcher-prompt.md` (bundled
   beside this file). Schedule it recurring at `7,37 * * * *` (off the :00/:30
   marks on purpose). If the host has no cron tool, run it as a `/loop` instead.
4. **Seed the ctx-wire delta baseline** so the first tick shows a real interval:
   ```bash
   printf '{"saved_tokens":%s,"ts":%s}\n' \
     "$(ctx-wire gain --json | jq .saved_tokens)" "$(date +%s)" \
     > ~/.claude/cctop/ctxwire-last.json 2>/dev/null || true
   ```

## The projection (why not straight-line)

The 7-day window is **FIXED**, not rolling: it hard-resets at
`seven_day.resets_at`; window start = `resets_at − 604800`. `burn-proj.py`
prints two numbers:

- **naive** = `used / elapsed_fraction` — worst case, assumes you burn every day
  at your average-so-far rate. Self-heals as elapsed grows.
- **profile_aware** = discounts the hours you predictably don't burn — a sleep
  window (02:00–09:00 ×0.05) and the weekend (Sat ×0.5, Sun ×0.1). The real
  forecast — a hot evening or weekday start does NOT mean you'll blow the cap if
  the hours ahead are predictably light (asleep or resting).

**Lead with profile_aware; flag only when it exceeds 100%.** Reporting a naive
evening/mid-week spike as "WILL BLOW" is the false alarm this skill exists to
prevent — the projection should bend down as the clock nears your sleep window.
Tune the sleep hours and weekend multipliers in `burn-proj.py`'s `mult()` to
your own cadence (default active window 09:00–02:00).

## Caveats

- Scheduled jobs are **session-only** on most hosts (die when the agent exits,
  auto-expire in ~7 days). Re-run this skill after a restart. `durable:true` is
  often ignored — verify rather than assume.
- Cloud schedules can't see local `cctop`/`ctx-wire`/`usage.json`, so this
  watcher only works in a session with the binaries present.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Redesigning the burn math inline | Use `burn-proj.py` — sleep/weekend discounting is the point |
| Scheduling a 2nd watcher | Check existing jobs first; replace, don't duplicate |
| Reporting naive % as the verdict | Lead with profile_aware; naive is the ceiling |
| Sorting sessions on `contextTokens` raw | Treat null as 0 |
| Compacting a busy session | Only `idle`/`waiting` are safe targets |
