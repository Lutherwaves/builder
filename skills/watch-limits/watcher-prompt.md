# Recurring status watcher — prompt template

Drop this into a recurring agent job (Claude Code `/loop`, a cron prompt, etc.).
It assumes `cctop`, `ctx-wire`, and `~/.claude/cctop/burn-proj.py` (installed by
the `watch-limits` skill) are on the host, and that a status-line tap has
written `~/.claude/cctop/usage.json` (see cctop's usage-limits docs). Runtime
state lives in `~/.claude/cctop/` and is not committed.

---

Run `cctop --json` and read `~/.claude/cctop/usage.json`. Then produce a concise report:

1. WEEKLY BURN RATE: From usage.json's rate_limits, report seven_day used_percentage and time-until-reset. Then run `python3 ~/.claude/cctop/burn-proj.py <seven_day.used_percentage> <seven_day.resets_at> $(date +%s)` — it prints elapsed%, naive projection (constant 7-day burn), and profile_aware projection (discounts predictably-light hours: a sleep window 02:00–09:00 ×0.05 and weekend Sat×0.5/Sun×0.1). Report BOTH projections; lead with profile_aware as the real forecast and treat naive as the worst-case ceiling. Flag only if profile_aware >100%. The 7d window is FIXED (window start = resets_at − 604800); a quiet day pauses accumulation but does not roll off old usage, and the naive number self-heals as elapsed grows. Also report the five_hour window %. If usage.json is missing or stale (>1h old vs `date +%s`), say so explicitly.

2. AGENT/SESSION LOAD: Count active cctop sessions and how many are 'busy' vs idle/waiting/shell, plus total subagents running. Note that more concurrent busy sessions/subagents = faster weekly burn.

3. COMPACTION RECOMMENDATION: List the top 3 sessions by contextTokens (show sessionId short, project, model, ctx, state). Recommend which ONE to compact — prefer idle/waiting sessions with the highest contextTokens (compacting a busy session mid-task is disruptive). Give the exact context size and a one-line rationale.

4. CTX-WIRE SAVINGS: Run `ctx-wire gain --json`. Report cumulative saved_tokens and savings_pct. Then compute the DELTA since the last tick: read `~/.claude/cctop/ctxwire-last.json` if present (holds `{"saved_tokens":N,"ts":EPOCH}`); report tokens saved and elapsed minutes since then (and a rough saved-tokens/hour rate). Then overwrite that file with the current saved_tokens and `date +%s`. If the marker is missing, say "first tick — no delta yet" and just create it.

Keep it tight — a scannable status readout, not prose.
