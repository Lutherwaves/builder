# builder

[![Release](https://github.com/Lutherwaves/builder/actions/workflows/release.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/release.yml)
[![Validate](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org)
[![Claude Code Plugin](https://img.shields.io/badge/Claude%20Code-plugin-8A2BE2)](https://claude.com/claude-code)

[![ctx-wire saved](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/Lutherwaves/builder/main/metrics/ctxwire-saved.json)](#the-toolkit-that-builds-this)
[![commands filtered](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/Lutherwaves/builder/main/metrics/ctxwire-commands.json)](#the-toolkit-that-builds-this)
[![output reduction](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/Lutherwaves/builder/main/metrics/ctxwire-reduction.json)](#the-toolkit-that-builds-this)

Operational tools for running AI coding agents at scale. Keep an eye on your
Claude subscription usage limits, session/context load, and token efficiency
while you work — without hand-rolling the math or babysitting a dashboard.

Built for [Claude Code](https://claude.com/claude-code) +
[`cctop`](https://github.com/) + [`ctx-wire`](https://github.com/).

> The three live badges above are **anonymized aggregates** — total tokens
> `ctx-wire` has trimmed out of my agent transcripts, commands filtered, and the
> overall output-reduction ratio. No session, project, or account data. Refresh
> them with `bash scripts/update-metrics.sh` (see
> [The toolkit that builds this](#the-toolkit-that-builds-this)).

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
   you're on pace to blow it before the (fixed) reset. Uses an **activity-aware**
   projection (discounts your sleep window + weekend) so a hot evening or weekday
   start doesn't trigger a false alarm.
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
- **profile_aware** — discounts the hours you predictably don't burn: a sleep
  window (02:00–09:00 ×0.05) and the weekend (Sat ×0.5, Sun ×0.1), for a forecast
  that bends down as your quiet hours approach. Tune the sleep window and
  multipliers in `mult()` to your own cadence (default active window 09:00–02:00).

## The toolkit that builds this

`builder` is small on purpose — but it's built inside a heavily-extended Claude
Code setup that lets one person ship almost anything, fast. If you're pulling
this repo to hack on it (or just curious how it's made), here's the stack. None
of it is required to *use* the plugin — the install above is fully
self-contained — but it's what makes contributing frictionless.

- **[Claude Code](https://claude.com/claude-code) with auto-edit on.** Running in
  `acceptEdits` permission mode means the agent edits, runs the validator, and
  iterates without a prompt on every step — the flow that produced this repo
  end-to-end. Turn it on with `/permissions` → *Auto-accept edits* (or `⇧⇥`).
- **[superpowers](https://github.com/obra/superpowers)** (obra's skill
  marketplace) — process skills for real engineering discipline. The
  `watch-limits` skill in this repo was written **test-first** with
  `superpowers:writing-skills`: baseline an agent *without* the skill, watch it
  fail, then write the skill to close exactly those failures.
- **[gstack](https://github.com/Lutherwaves/gstack)** — Garry Tan's opinionated
  CEO / eng-manager / release-manager / QA skill set (`/ship`, `/review`, `/qa`)
  for planning, review, and release flows.
- **[`ctx-wire`](https://github.com/)** — filters noisy command output out of the
  agent's context. It's why the badges up top exist: those are the tokens it
  kept out of my transcripts. Lean context = cheaper, faster, longer sessions.
- **[`cctop`](https://github.com/)** — live per-session monitor. `builder`'s
  `watch-limits` watcher reads it to pick which idle session to compact.

Refresh the metrics badges (anonymized aggregates only) any time with:

```bash
bash scripts/update-metrics.sh
git commit -am "chore: refresh metrics"
```

## Contributing

PRs welcome. `main` is protected — every change lands via PR, the `validate`
check must pass, and CI cuts releases automatically from
[Conventional Commit](https://www.conventionalcommits.org/) titles. Run the same
gate locally before you push:

```bash
bash scripts/validate.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full flow, and
[SECURITY.md](SECURITY.md) for how to report vulnerabilities privately.

## License

MIT

