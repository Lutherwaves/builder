# Usage watcher — prompt template

Put this prompt into a recurring agent job (every 30 minutes, or a cron prompt at `7,37 * * * *`; see your harness mapping).
It needs `matchblox` on the machine that runs the job.
The `budget` skill (`builder:budget`) is the full spec. This prompt drives one pass.

---

Run one budget pass. Only read. Do not compact, stop or start a session.

1. **Weekly.** Run `matchblox usage --json`. For each account, find the window with `window_minutes` 10080.
   Report its `used_pct` (measured) and the time until `resets_at`.
   Then run `python3 "${CLAUDE_PLUGIN_ROOT}/skills/budget/burn-proj.py" --from-matchblox`.
   Report profile_aware first, as the forecast. Report naive as the ceiling. Both are estimated.
   Flag the account only when profile_aware is over 100%, or `used_pct` is 80 or more.
   If the script exits non-zero, quote its stderr line and stop this step. Never report a 0% you did not read.

2. **5-hour.** Report the `used_pct` of the window with `window_minutes` 300, per account.

3. **Load.** Run `matchblox sessions --json`. Count the sessions by `word`: `busy`, `idle`, `asks`.
   More busy sessions burn the week faster.

4. **Compaction pick.** Name the one `idle` session with the highest `context_pct`. Give its `context_pct` and one line of reason.
   Never pick a `busy` or `asks` session. Do not compact it. The person runs it through the `fleet` skill.

Keep the reply to a status readout of at most eight lines. When all is normal, say so in one line.
