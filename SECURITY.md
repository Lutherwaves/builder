# Security Policy

## Supported Versions

Only the latest minor version receives security fixes.

| Version | Supported |
|---------|-----------|
| 1.x     | ✅        |
| < 1.0   | ❌        |

## Reporting a Vulnerability

Please report security issues privately via GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability).
Do not open a public issue for security reports.

**Include:** description, steps to reproduce, potential impact, and suggested
remediation if known.

**Response time:** we aim to acknowledge within 72 hours and provide a fix or
mitigation plan within 14 days for confirmed issues.

## Scope

This plugin runs inside Claude Code, Codex CLI, Cursor CLI and opencode. As part
of its normal flow, it runs shell commands and schedules recurring agent prompts on the user's machine.
Security-relevant surfaces:

- `hooks/` — the SessionStart hook. It runs at the start of every session and
  adds text to the session context. It runs local commands only (`git`,
  `gh auth token`, `ps`, `matchblox sessions --json`, `matchblox usage --json`)
  under one 0.5 s budget, and reads local files.
- The `builder` binary (`cmd/builder`, `internal/*`) and its shim `bin/builder`.
  The hook and the skills run it through the shim. It runs locally.
- `skills/*/SKILL.md` — instructions the harness treats as trusted content.
- `skills/*/*.sh` and `scripts/*.sh` — run locally.
- Files under `~/.config/builder/` (or `$BUILDER_HOME`): `ledger.json` (setup
  answers), `reach.json` (send rules and the team channel) and `reach-log.jsonl`.
  **`reach-log.jsonl` holds the full text of each message** that `reach` sent or
  skipped, with its recipient. It is written with mode 0600.
- Files under `$BUILDER_HOME/code-graph/` (the graph registry) and
  `$BUILDER_HOME/intake/` (config and seen messages). `$BUILDER_HOME` is
  `~/.config/builder` by default. Builder still reads the old
  `~/.claude/code-graph/` and `~/.claude/intake/` files when the new ones are
  missing. It never writes or deletes them.
- The adapters: the per-harness manifests, the hook files and the opencode plugin.
- The state directories of each harness that builder reads: the Claude Code
  directory (`~/.claude` or `CLAUDE_CONFIG_DIR`), the Codex directory
  (`~/.codex` or `CODEX_HOME`), the Cursor directory (`~/.cursor` or
  `CURSOR_CONFIG_DIR`), the opencode directory (`~/.config/opencode` or
  `OPENCODE_CONFIG_DIR`), and `$BUILDER_HOME`. From these it reads the MCP
  server and hook lists, and the project's `.mcp.json` and `.cursor/mcp.json`.
- Scheduled watcher prompts that read `matchblox usage --json`,
  `matchblox sessions --json`, and the graph registry.

Network:

- The hook makes no network call itself.
- A message leaves the machine only through `reach`, under its send rules or
  the person's yes, through the person's own chat connector, matchblox, or an
  OS notification.
- `intake` reads the person's trackers through `gh` and their MCP connectors.
  It writes to a tracker only under the person's own config rule
  (`annotate_in_tracker`), on a mark, or on the person's yes.
- The `codebase-memory-mcp` binary that `harness` sets up makes one version
  check against GitHub.
- A setup step that installs a tool (for example `gh auth login` or a
  `git clone`) runs only after the person's yes.

builder stores no secrets.

Out of scope: the four harnesses themselves, `matchblox` upstream, and
user-authored content.
