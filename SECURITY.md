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
of its normal flow, it runs shell commands on the user's machine.
Security-relevant surfaces:

- `hooks/` — the SessionStart hook. It runs at the start of every session and
  adds text to the session context. It reads local files only.
- The `builder` binary (`cmd/builder`, `internal/*`) and its shim `bin/builder`.
  The hook and the skills run it through the shim. It runs locally.
- `skills/*/SKILL.md` — instructions the harness treats as trusted content.
- `scripts/*.sh` — run locally.
- Files under `$BUILDER_HOME` (`~/.config/builder` by default): the helper
  binary, the install answer and the audit baselines.
- The adapters: the per-harness manifests, the hook files and the opencode plugin.

Network:

- The hook makes no network call itself.
- A message leaves the machine only with the person's yes in the session.

builder stores no secrets.

Out of scope: the four harnesses themselves and user-authored content.
