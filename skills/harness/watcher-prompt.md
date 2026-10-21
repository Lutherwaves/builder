# Stale-graph watcher — prompt template

Put this prompt into a recurring agent job (`/loop 30m`, or a cron prompt at `22,52 * * * *`).
It needs one run of `setup.sh` first, so the binary exists and `~/.claude/code-graph/repos.txt` lists the repositories.
The `harness` skill (`builder:harness`) is the full spec. This prompt drives one pass.

---

Run one freshness pass. Only read. Never re-index.

Read `~/.claude/code-graph/repos.txt`. Each line is `<abs-path>\t<project>\t<indexed-sha>`.
If the file is missing or empty, say "No repository is registered. Run /builder:harness in a repository first." and stop.

For each line, run:

    bash "${CLAUDE_PLUGIN_ROOT}/skills/harness/drift.sh" <abs-path>

Each call prints one `VERDICT=…` line. Then report:

1. **Stale.** Each `STALE` repository: the project, and `reason=` (`commits`: HEAD moved after the index; `edits`: a tracked file changed after the index).
   A stale graph gives old answers. Give the re-index command for each:
   `<binary> cli index_repository --repo-path <path> --mode moderate`
2. **Not indexed, no binary.** Each `NOT_INDEXED` or `NO_BINARY` line, with its one-command fix:
   run `setup.sh` again, or install the binary.
3. **Fresh.** Only a count: "N of M fresh".

Do not re-index. A re-index in the middle of another session's task changes the answers under it.
The person runs the command, or says yes. Keep the reply to a status readout.
When all graphs are fresh, say so in one line.
