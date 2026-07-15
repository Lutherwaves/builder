# Stale-index watcher — prompt template

Drop this into a recurring agent job (Claude Code `/loop`, a cron prompt, etc.).
It assumes the `code-graph` skill has run at least once, so the
codebase-memory-mcp binary is discoverable and `~/.claude/code-graph/repos.txt`
lists the repos to watch. `drift.sh` lives beside this file at
`${CLAUDE_PLUGIN_ROOT}/skills/code-graph/drift.sh`. Runtime state lives in
`~/.claude/code-graph/` and is not committed.

---

Read `~/.claude/code-graph/repos.txt` (tab-separated: `<abs-path>\t<project>`,
one repo per line). If it is missing or empty, say "no repos registered — run
/builder:code-graph in a repo first" and stop. Otherwise, for EACH line's path
(the first field), run:

  bash ${CLAUDE_PLUGIN_ROOT}/skills/code-graph/drift.sh <path>

(drift.sh resolves the project name from the registry itself — just pass paths.)

Each call prints one `VERDICT=…` line. Produce a tight, scannable report:

1. STALE INDEXES: List every repo whose verdict is `STALE`, showing the short
   project slug and `reason=` (`commits` = HEAD moved since index;
   `uncommitted:N` = N working-tree files differ; or both). These return
   wrong/old answers until re-indexed. For each, give the exact fix (re-index is
   ~9s and incremental):
     <binary> cli index_repository --repo-path <path> --mode moderate
   Recommend re-indexing the STALE repos that have a LIVE Claude Code session
   first (cross-reference `cctop --json` session cwds if available) — a stale
   graph under an active session is the one actively misleading an agent.

2. NOT INDEXED / NO BINARY: List repos that report `NOT_INDEXED` (registered but
   never indexed, or the graph was deleted) or `NO_BINARY` (binary missing —
   the setup step must run again). One line each with the one-command fix.

3. FRESH: Just a count ("N of M fresh"). Do not enumerate them.

Never auto-re-index without the user saying so — indexing is cheap but not free,
and re-indexing a busy repo mid-task can churn. Surface the picks; let them
pull the trigger. Keep it to a status readout, not prose.
