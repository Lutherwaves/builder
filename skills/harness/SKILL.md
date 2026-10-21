---
name: harness
description: Use when agents grep and read file after file to answer "where is X?", "what calls Y?" or "what is the architecture?", when the person asks to "harness this repo", "index this repo", "set up a code graph" or "make agents stop scanning", or when they ask which repositories need a re-index.
---

# harness — give agents a map of the repository, and keep it fresh

An agent with no map reads file after file. Each read costs tokens and attention.
`harness` gives one repository a local code graph, routes every session to it, and keeps it fresh.
It applies the improve-harness loop: baseline → provision → route → freshness → verify.

Doctrine source: Ryan Lopopolo's [Harness Engineering](https://github.com/lopopolo/harness-engineering) (CC BY 4.0).
builder adapts its concepts and copies no source prose.

The graph is [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp). It runs on this machine. It needs no API key. No code leaves the machine.
The binary makes one outbound call: a version check against GitHub. To trust no release binary, build it from source.

## 1. Baseline

Ask one structural question in a fresh session at the repository root. Example: "What calls `charge_card`?"
Write down what the agent does: the number of files it reads, the greps, the time.
This is the cost that the next steps must lower. Without a baseline, step 5 proves nothing.
A repository that an agent reads in one pass gains little from a graph. Say so in the reply, then go on: the person asked for the harness.

## 2. Provision

```
bash "${CLAUDE_PLUGIN_ROOT}/skills/harness/setup.sh" "$PWD"
```

If you cannot run it (no permission, unattended run), give the person this exact command, and say why it has no `--name`.
Then do steps 3 and 4: they do not need the graph. End with step 5 as a handover: the exact verify question and its pass rule.
Never index with the MCP `index_repository` tool instead. It skips the `.mcp.json` wiring and the registry, so the watcher never sees the graph.

`setup.sh` does four things, and you can run it again safely:

1. It finds the binary: `CBM_BIN`, then `PATH`, then `~/dev/codebase-memory-mcp/build/c/`, then `~/.local/bin/`.
   No binary? It prints the build commands and exits 2. It never compiles for you: the build is large. The person opts in.
2. It adds a `codebase-memory` server to the repository's `.mcp.json`. It keeps every other server in the file. It merges with `jq` or `python3`. With neither, it stops and changes nothing.
3. It indexes the tree, under a project name made from the absolute path.
4. It registers the repository in `$BUILDER_HOME/code-graph/repos.txt` (default `~/.config/builder/code-graph/repos.txt`), with the HEAD at index time.
   It still reads the old `~/.claude/code-graph/repos.txt` once when the new file is missing. It writes only the new file.

**Never pass `--name`.** The path-based name keeps each clone and each worktree on its own graph.
A forced name makes two checkouts share one graph. Then a query in one checkout returns code from the other branch.

After the run, the person approves the `codebase-memory` server once, in a Claude Code session at the repository root.
`.mcp.json` holds an absolute path to the binary. Add it to `.git/info/exclude`, not to a commit.

## 3. Route

Add this snippet, as it is, to the repository's `CLAUDE.md` or `AGENTS.md`:

```
**Structural questions: ask the graph first.** For "where is X?", "what calls Y?" or "what is the
architecture?", use the `codebase-memory` graph tools (`search_graph`, `query_graph`, `trace_path`,
`get_architecture`, `get_code_snippet`) before grep, glob or Explore. If this checkout has no graph,
provision one first with the `harness` skill (local, no API key), then ask.
```

Every later session, in every checkout, then asks the graph first. A checkout with no graph provisions one first, so the snippet is safe before step 2 runs.
This is the durable part. Commit it in the repository, with a Conventional Commit.

## 4. Freshness

A graph built on branch A gives old answers after a branch switch or new commits.
`drift.sh <repo-path>` prints one line: `VERDICT=<FRESH|STALE|NOT_INDEXED|NO_BINARY|BAD_PATH> …`.

| Verdict | Means | Fix |
|---|---|---|
| `FRESH` | The graph matches the tree | Nothing |
| `STALE reason=commits` | HEAD moved after the index | Re-index |
| `STALE reason=edits` | A tracked file changed after the index | Re-index |
| `NOT_INDEXED` | Registered, but no graph is ready | Run step 2 again |
| `NO_BINARY` | The binary is gone | Install it, then run step 2 again |

Re-index: `<binary> cli index_repository --repo-path <path> --mode moderate`. It is incremental.

**Never re-index on your own.** Give the command. The person runs it, or says yes.
A re-index in the middle of another session's task changes the answers under it.

To watch every registered repository, schedule `watcher-prompt.md` (beside this file) when the person asks:

1. List the scheduled jobs first. A job that runs `drift.sh` already exists? Stop. Offer to replace it. Never add a second.
2. Schedule it at `22,52 * * * *`. This stays clear of the `budget` watcher at `7,37`. With no cron tool, schedule a check every 30 minutes (see your harness mapping).
3. Most scheduled jobs live only as long as the session. Tell the person to schedule it again after a restart.

## 5. Verify

Open a fresh session at the repository root. Ask the baseline question again.
Pass: the agent called a graph tool (`search_graph`, `query_graph`, `trace_path`, `get_architecture`) and gave a real answer.
A run that never touched the graph proves nothing. Find out why (server not approved, snippet not read), fix it, and ask again.
Report the baseline cost and the new cost side by side.

## Honesty about savings

A token counter measures tokens spent. Nothing measures the tokens the graph saved: those reads never happened.
Any savings figure is a model, not a measurement. Say "estimated" and show how you got it.
Never put it on a badge or next to measured numbers.

## After the work

End with the builder report from the builder skill. Its Evidence line holds the baseline and the verify result.
Offer other formats in one line.

## Common mistakes

| Mistake | Fix |
|---|---|
| `--name` to get a short project name | Never. The path-based name isolates each checkout. |
| Building the binary because it is missing | Give the build commands. The person opts in. |
| Re-indexing every STALE repository at once | Give the command for each. The person decides. |
| A second drift watcher | List the jobs first. Replace, never stack. |
| "Verified" after a run that only grepped | The verify run must call a graph tool. Retry until it does, or report why not. |
| No baseline, so no before and after | Ask the structural question before step 2. Write down its cost. |
| "The graph saves 60% of tokens" as a fact | Savings are counterfactual. Say "estimated" and show the model. |
| `.mcp.json` committed with an absolute binary path | Add it to `.git/info/exclude`. Commit only the routing snippet. |
| `index_repository` through MCP when `setup.sh` is blocked | Give the `setup.sh` command and why it has no `--name`. Do steps 3 and 4. Hand over step 5. |

## Self-check

Before you finish:

- [ ] I recorded a baseline before I provisioned.
- [ ] `setup.sh` ran with no `--name`, and I quoted its project line.
- [ ] If I could not run setup.sh, I gave the command, the reason for no `--name`, and the verify question to ask after it.
- [ ] I compiled nothing and re-indexed nothing without the person's yes.
- [ ] The routing snippet is in `CLAUDE.md` or `AGENTS.md`.
- [ ] I listed the scheduled jobs before I scheduled a drift watcher, and added no second one.
- [ ] The verify run called a graph tool, and I compared it with the baseline.
- [ ] Each savings figure says "estimated".
- [ ] The reply ends with the builder report and a one-line offer of other formats.
