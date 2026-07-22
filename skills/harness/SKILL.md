---
name: harness
description: Use when you want to adopt the graph-first workflow in a repo end to end — index the codebase, route agents to the graph before grep, and keep it fresh. Applies harness engineering's improve-harness loop by composing the code-graph skill. Triggers: "harness this repo", "make agents stop scanning this codebase", "apply the doctrine here", "set up graph-first retrieval".
---

# harness — adopt graph-first retrieval in a repo

## Overview

Applies harness engineering's **improve-harness loop** to one repo so its agents
stop cold-scanning: baseline → provision → route → freshness → verify. This skill
composes existing builder skills; it does not re-implement indexing.

Doctrine source: Ryan Lopopolo's
[Harness Engineering](https://github.com/lopopolo/harness-engineering) (CC BY 4.0).

## The loop

1. **Baseline.** In a fresh trajectory, note how the agent answers a structural
   question in this repo today — does it grep and read N files, or query a graph?
   Record the observable cost. This is what the intervention must improve.

2. **Provision.** Index the repo with `/builder:code-graph` (see that skill).
   From the repo root: `bash "${CLAUDE_PLUGIN_ROOT}/skills/code-graph/setup.sh" "$PWD"`.
   The path-derived project name keeps clones/worktrees isolated — never force
   `--name`.

3. **Route.** Inject the canonical snippet from `routing-snippet.md` (beside this
   file) into the repo's checked-in `CLAUDE.md` or `AGENTS.md`, so every future
   session — in any checkout — knows to query the graph first and to
   self-provision when its checkout is unindexed. Commit this; it is the durable
   part of the intervention.

4. **Freshness.** Ensure the `code-graph` drift watcher covers the repo. It has
   its own dedupe guard — if a `22,52 * * * *` watcher already runs, do not add a
   second.

5. **Verify.** In a fresh session rooted at the repo, ask the same structural
   question from the baseline. Confirm the agent used a graph tool
   (`search_graph`/`query_graph`/`get_architecture`) and returned a real result,
   not a grep sweep. A rerun that never touched the graph proves nothing — retry
   until the intervention is actually exercised.

## Honesty note

codebase-memory savings are counterfactual (tokens you did not spend because the
agent queried the graph). Report any figure as an estimate, never as a metered
number.
