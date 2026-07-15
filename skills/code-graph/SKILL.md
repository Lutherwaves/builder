---
name: code-graph
description: Use when you want an AI coding agent to answer structural questions about a repo (where is X, what calls Y, architecture) without grep-reading files — sets up a local codebase-memory-mcp graph per repo, isolated per clone/worktree, and watches for stale indexes. Triggers: "index this repo", "code graph", "give the agent a map of the codebase", "stop re-reading files", "which repos need re-indexing".
---

# code-graph — codebase-memory graph, provisioned & kept fresh

## Overview

Wires a local [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp)
graph into a repo so agents query a structural map (functions, calls, routes,
imports) instead of reading files one at a time — the single biggest per-task
token lever alongside `ctx-wire`. One graph per repo path, so your parallel
clones and worktrees stay **isolated** — a query in one never returns another
branch's code. A companion watcher flags graphs that have gone stale.

**Everything runs locally. No API key, no data leaves the machine.** The binary
is a single audited MIT C build; the only outbound call it makes is a background
GitHub version check (harmless). Prefer building from source over the release
binary if you want to erase all binary trust.

## Prerequisites

- `codebase-memory-mcp` binary discoverable: on `PATH`, at
  `~/dev/codebase-memory-mcp/build/c/codebase-memory-mcp`, or via `CBM_BIN`.
  If absent, `setup.sh` prints the build command and stops — it never
  auto-compiles (it's a ~279MB C build; opt in explicitly).
- `git` (optional) — used only to label the current HEAD in reports. Staleness
  itself compares the graph to the working tree, so it works without git too.
- `jq` (optional) — for merge-safe edits to an existing `.mcp.json`.

## Setup flow (per repo)

1. **Provision + index the current repo:**
   ```bash
   bash "${CLAUDE_PLUGIN_ROOT}/skills/code-graph/setup.sh" "$PWD"
   ```
   This locates the binary, merges a `codebase-memory` server into the repo's
   `.mcp.json` (never clobbering other servers), indexes the tree in ~9s under a
   **path-derived** project name, and registers the repo in
   `~/.claude/code-graph/repos.txt` for the watcher.
2. **Never force `--name`.** The path-derived name is what keeps clones and
   worktrees isolated. Overriding it makes two checkouts share (and clobber) one
   graph — the exact bug to avoid.
3. **Approve the MCP server once** in a Claude Code session rooted at the repo,
   then ask structural questions (`search_graph`, `get_architecture`) instead of
   grepping. `.mcp.json` is untracked; add it to `.git/info/exclude` if you
   don't want it in `git status`.

## Keep-fresh watcher

A graph indexed on branch A returns stale answers after you switch branches or
land commits. Schedule the drift check so you're told which graphs to refresh:

1. **Guard against duplicates.** List existing scheduled jobs first. If a
   `22,52 * * * *` watcher running `code-graph/drift.sh` already exists, STOP —
   offer to replace it, never add a second.
2. **Schedule the watcher.** Use the prompt in `watcher-prompt.md` (bundled
   beside this file). Schedule it recurring at `22,52 * * * *` (offset from the
   `watch-limits` watcher's `7,37` so they don't collide). If the host has no
   cron tool, run it as a `/loop` instead.
3. The watcher reads `repos.txt`, runs `drift.sh` per repo, and reports STALE /
   NOT_INDEXED graphs with the exact ~9s re-index command. It **never
   auto-re-indexes** — it surfaces the picks; you pull the trigger.

## Honesty note (why there's no savings badge)

`ctx-wire` savings are *metered* — it emits a real `saved_tokens` counter.
codebase-memory savings are *counterfactual* (tokens you didn't spend because
the agent queried the graph instead of reading files); any figure is a **model**,
not a measurement. Report it as an estimate if asked — do NOT publish it as a
hard badge alongside the metered ctx-wire numbers.
