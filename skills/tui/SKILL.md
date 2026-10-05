---
name: tui
description: Use when you want one place to watch a machine that runs many AI coding agents — every agent session by tmux pane with its context use, machine load, detached busy loops, git worktrees, and a ranked list of what to do — or when an agent needs to read that state. Triggers: "open the cockpit", "what is loading the machine", "which session should I compact or clear", "find leaked processes", "which worktrees can I remove", "builder tui".
---

# builder tui — a cockpit for agent sessions, tmux, git and machine load

`builder-tui` is a single static Go binary (Linux, macOS, Windows) that a builder leaves open in
a tmux window. It answers, in one screen:

- **Sessions** — every agent process mapped pane → process → worktree (from
  the pane the agent records in its session file, else `TMUX_PANE` in its
  environment, so a re-parented process still finds its pane), busy/idle, context use and 30-minute burn, CPU of its whole process
  tree, and a suggestion: `compact`, `clear`, or nothing.
- **Machine** — CPU per core, load, PSI, memory and swap, package temperature,
  CPU power limits (alerting on drift), GPUs, containers and the CPU share of
  container groups you name (CI runners, say).
- **Procs** — the top processes by CPU attributed to their owner (pane,
  container or system), and **detached busy loops**: shells re-parented to init
  or a user service manager that keep burning CPU, with the pane that started
  them.
- **Git** — per checkout an agent uses: dirty paths, the default branch behind
  its remote, worktrees, which ones are merged, clean and unused, and how many
  cores the agents spend running git there.
- **Recs** — a ranked list, each with the evidence behind it and the exact
  command that resolves it.
- **History** — 24 hours of CPU, load, pressure, temperature, memory and swap
  from its own log (plus any CSV you point it at).

It is read-only by default. Every action prints its exact command first; `Enter`
runs a non-destructive one, and anything destructive (kill, remove a worktree,
type into a session) needs `x` and then a typed `y`.

## Install

```bash
bash "${CLAUDE_PLUGIN_ROOT}/skills/tui/setup.sh"
```

`setup.sh` downloads the release binary for this machine and checks its
SHA-256, or builds from the plugin checkout when Go is installed. It installs to
`~/.local/bin/builder-tui` (override with `BUILDER_BIN_DIR`). It runs on Linux
and macOS; on Windows, run it inside WSL, or download
`builder-tui-windows-amd64.exe` (or `-arm64.exe`) from the release page.

## Run

```bash
tmux new-window -d -n cockpit builder-tui
```

Keys: `1`–`6` or `tab` switch panels, `↑↓` select, `Enter` the panel's safe
action (jump to a pane, open a shell in a worktree), `x` the destructive one,
`r` rescans git, `q` quits.

It runs at `nice 19`. It reads cheap counters every 2 s, the process table every
10 s, `docker` and `nvidia-smi` every minute, and git every 5 minutes. It
re-checks merged worktrees at most every 2 h, because git refuses a dirty removal
anyway. The renderer is capped at 15 fps. On a workstation with ~1,100 processes,
25 agent sessions and ~190 worktrees it used 0.84% of one core over 10 minutes,
child commands included.

## For agents

```bash
builder-tui status          # JSON: sessions, machine, alerts, git, recommendations
builder-tui status --text   # the same as a short summary with exact commands
```

`status` reads the snapshot a running cockpit writes to
`~/.local/state/builder/tui.json` each sample; with no cockpit running it samples
on its own. Recommendations carry `primary` (safe) and `secondary` (destructive)
actions as argv steps. Ask the user before running a destructive step, and
check its `guards` right before each step: the same pid with the same
`start_ticks`, a worktree still clean and unused, a session still idle. The
cockpit does this itself and skips a step whose guard no longer holds.

## Configure

Machine goals live in a local file, never in a repository:
`~/.config/builder/tui.toml`. Every key is optional; see
[`config.example.toml`](config.example.toml) for all of them.

## Requirements

tmux for the Sessions panel. Optional: `docker` (containers), `nvidia-smi`
(GPUs), `gh` (merged pull requests, so squash merges count as merged).

| | Linux | macOS | Windows |
|---|---|---|---|
| Sessions, Git, Recs, History | yes | yes | in WSL (no native tmux) |
| CPU, memory, swap, network, disk, load, processes | yes | yes | yes |
| Pressure (PSI), power limits, temperature, container CPU | yes | no | no |

Linux reads `/proc` and `/sys` directly. macOS and Windows read through the
system's own APIs, which cost more per process scan, so the scan stays on its
10-second cadence. Under WSL the cockpit is the Linux build.
