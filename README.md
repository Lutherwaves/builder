# builder

Small operational tools for running AI coding agents at scale — keeping an eye
on usage limits, session/context load, and token efficiency while you work.

Built for [Claude Code](https://claude.com/claude-code) + [`cctop`](https://github.com/) +
[`ctx-wire`](https://github.com/), but the pieces are generic.

## Tools

### `cctop/burn-proj.py` — weekly-limit burn projection

Projects where your rolling subscription usage lands at the window reset, with
**weekend load discounting** so a hot weekday start doesn't produce a false alarm.

```bash
python3 cctop/burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]
# elapsed=15.0%  naive=127%  profile_aware=102%  reset_in=5.95d
```

- **naive** — straight-line `used / elapsed_fraction`. The worst case: assumes
  you burn at the same average rate 7 days a week. Self-heals as elapsed grows.
- **profile_aware** — discounts the *remaining* weekend hours (Sat ×0.5,
  Sun ×0.1) against the per-normal-hour rate you've actually set. A truer
  mid-week forecast; tune the multipliers in `mult()` to your own cadence.

The window is treated as **fixed** (start = `resets_at − 604800`), matching how
subscription weekly limits reset rather than rolling continuously.

### `cctop/watcher-prompt.md` — recurring status watcher

A drop-in prompt for a recurring agent job (Claude Code `/loop`, cron, etc.) that
every ~30 min reports: weekly + 5h burn (via `burn-proj.py`), session/context
load, a compaction recommendation, and `ctx-wire` token savings with a
per-interval delta. Sanitized — no environment-specific paths.

## License

MIT
