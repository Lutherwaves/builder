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

This plugin runs inside Claude Code. As part of its normal flow, it runs
shell commands and schedules recurring agent prompts on the user's machine.
Security-relevant surfaces:

- `hooks/` — the SessionStart hook. It runs at the start of every session and
  adds text to the session context. It runs local commands only (`git`,
  `gh auth token`, `pgrep`, `matchblox sessions --json`, `matchblox usage --json`)
  under one 0.5 s budget, and reads local files.
- `lib/` — `ledger.py`, `signals.py` and `reach.py`. The hook and the skills run them.
- `skills/*/SKILL.md` — instructions Claude Code treats as trusted content.
- `skills/*/*.sh`, `skills/*/*.py` and `scripts/*.sh` — run locally.
- Files under `~/.config/builder/` (or `$BUILDER_HOME`): `ledger.json` (setup
  answers), `reach.json` (send rules and the team channel) and `reach-log.jsonl`.
  **`reach-log.jsonl` holds the full text of each message** that `reach` sent or
  skipped, with its recipient. It is written with mode 0600.
- Files under `~/.claude/code-graph/` (the graph registry) and
  `~/.claude/intake/` (config and seen messages).
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

Out of scope: the Claude Code harness itself, `matchblox` upstream, and
user-authored content.
