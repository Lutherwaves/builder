# Security Policy

## Supported Versions

Only the latest minor version receives security fixes.

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅        |

## Reporting a Vulnerability

Please report security issues privately via GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability).
Do not open a public issue for security reports.

**Include:** description, steps to reproduce, potential impact, and suggested
remediation if known.

**Response time:** we aim to acknowledge within 72 hours and provide a fix or
mitigation plan within 14 days for confirmed issues.

## Scope

This plugin runs inside Claude Code and, as part of its normal flow, executes
shell commands and schedules recurring agent prompts on the user's machine.
Security-relevant surfaces:

- `skills/*/SKILL.md` — instructions Claude Code treats as trusted content.
- `scripts/*.sh` and bundled tools (e.g. `burn-proj.py`) — run locally.
- Files written under `~/.claude/code-graph/` (the graph registry) and
  `~/.claude/intake/` (config and seen messages).
- Scheduled watcher prompts that read `matchblox usage --json`,
  `matchblox sessions --json`, and the graph registry.

The tools are **read-only with respect to your account**: they read local files
and process output. They make no network calls and store no secrets.

Out of scope: the Claude Code harness itself, `matchblox` upstream, and
user-authored content.
