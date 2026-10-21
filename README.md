# builder

[![Release](https://github.com/Lutherwaves/builder/actions/workflows/release.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/release.yml)
[![Validate](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org)
[![Claude Code Plugin](https://img.shields.io/badge/Claude%20Code-plugin-8A2BE2)](https://claude.com/claude-code)

Operational tools for running AI coding agents at scale. Keep an eye on your
Claude subscription usage limits, session/context load, and token efficiency
while you work — without hand-rolling the math or babysitting a dashboard.

Built for [Claude Code](https://claude.com/claude-code).

## Install

```bash
# Step 1: Add the marketplace
/plugin marketplace add Lutherwaves/builder

# Step 2: Install the plugin
/plugin install builder@builder
```

## Skills

- `/builder:budget` — Read the weekly plan window from `matchblox usage --json`,
  project the rest of the week (activity-aware, not straight-line), and pick
  one idle session to compact. Optional watcher every ~30 min.
- `/builder:intake` — Grab & reconcile: pull today's captured tasks from a
  tracker (Todoist, or a GitHub Project's draft items), and optionally the
  messages you marked on your team channel; drop anything already in motion
  (live session, open PR, existing issue), and groom the raw leftovers.
  Annotates by default; creates issues only when you mark a task for it.
- `/builder:harness` — Give a repo a local
  [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) graph
  so agents answer structural questions (where is X, what calls Y,
  architecture) instead of grep-reading files: baseline → provision → route →
  freshness → verify. One graph per checkout, plus a watcher that names stale
  graphs. Local-only, no API key.

## Doctrine

builder is the applied side of **harness engineering** — improving a fixed
coding agent by shaping its two external levers, **context** and **tools**.
Each skill is one intervention on those levers, not a standalone gadget:

| Skill | What it shapes |
| --- | --- |
| `harness` | Routes context just in time and makes the codebase a legible tool — a queryable structural graph instead of grep-reading files, kept fresh, applied end to end: baseline → provision → route → freshness → verify. |
| `budget` | Optimizes for measured effectiveness — tokens and attention against your real usage limits. |
| `intake` | Turns feedback into infrastructure — captured work reconciled into durable issues. |

The framing derives from Ryan Lopopolo's
[Harness Engineering](https://github.com/lopopolo/harness-engineering) corpus
(CC BY 4.0). See [`AGENTS.md`](AGENTS.md) for skill-selection routing.

## What the budget watcher reports

1. **Weekly burn rate** — how much of your weekly limit you've used, and whether
   you're on pace to blow it before the (fixed) reset. Uses an **activity-aware**
   projection (discounts your sleep window + weekend) so a hot evening or weekday
   start doesn't trigger a false alarm.
2. **5-hour window** — the short window's use.
3. **Session load** — busy, idle and waiting sessions.
4. **Compaction pick** — the highest-context *idle* session.

## Prerequisites

- [`matchblox`](https://matchblox.sh) on PATH for the plan windows and the
  sessions (`curl -fsSL https://matchblox.sh | sh`). Without it, read `/usage`
  and pass the numbers to `burn-proj.py` by hand.

## Standalone use (no plugin)

Just want the projector? Copy `skills/budget/burn-proj.py` anywhere and:

```bash
python3 burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]
python3 burn-proj.py --from-matchblox [now_epoch]
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
  `budget` skill in this repo was written **test-first** with
  `superpowers:writing-skills`: baseline an agent *without* the skill, watch it
  fail, then write the skill to close exactly those failures.
- **[gstack](https://github.com/garrytan/gstack)** — Garry Tan's opinionated
  CEO / eng-manager / release-manager / QA skill set (`/ship`, `/review`, `/qa`)
  for planning, review, and release flows.

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

Doctrine and vocabulary derive from Ryan Lopopolo's
[Harness Engineering](https://github.com/lopopolo/harness-engineering),
licensed CC BY 4.0. builder adapts the concepts and copies no source prose.
