<!-- Canonical graph-first routing snippet. Inject into a target repo's
     CLAUDE.md or AGENTS.md verbatim. Source of truth: builder/skills/harness. -->

**Structural questions → query the graph first.** For "where is X", "what calls
Y", or "what's the architecture", use the `codebase-memory` graph tools
(`search_graph`, `query_graph`, `trace_path`, `get_architecture`,
`get_code_snippet`) before grep/glob/Explore. If this checkout has no graph,
provision one first with `/builder:code-graph` (~9s, local, no API key), then
query. Graphs are per-checkout; a drift watcher flags stale ones.
