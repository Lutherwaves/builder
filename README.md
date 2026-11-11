# builder

[![Release](https://github.com/Lutherwaves/builder/actions/workflows/release.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/release.yml)
[![Validate](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org)

builder makes your coding agent answer to you. Every piece of work ends with a short
report: is it done, what you have to do, what isn't proven yet, what left your machine,
and every call the agent made on its own, with the reason. Nothing leaves the machine
(no push, no PR, no message) without your yes.

It's a plugin for Claude Code, Codex CLI, Cursor CLI and opencode. It takes a request
through shape, plan, build, review and ship, with security in every stage. Skills and one
session hook; no server, no account, no network call at session start.

> **▶ Needs you:** a customer who moved can now change their email at checkout, in one step.
>
> **▶ You (1):** push `feat/change-email` and open the PR. I ask below.
>
> **⚠ Not proven:** the confirmation email in a real inbox.
>
> **✓ Proven:** `go test ./...` → 14 passed (example).
>
> **↗ Outside:** nothing sent.
>
> **Calls:** the old address gets a notice, because a stolen account changes the email first.

The marks mean one thing each: ✓ done or proven, ✗ failed, ▶ you act, ⚠ not proven,
↗ left this machine. After the report the agent asks each ▶ You item as a choice, its
recommendation first.

## Why builder

builder is one framework from a request to a release. Five stages hand the work on,
and each stage stops at a gate before the next one starts.

| Stage | Skill | You get | Gate before the next stage |
| --- | --- | --- | --- |
| Shape | `shape` | A spec: the person, the moment, the non-goals and when it is done | You approve the spec |
| Plan | `plan` | Tasks that each start with a failing test and end in one commit | Each done-when line of the spec maps to a task |
| Build | `build` | The code, one task at a time, test first | Every test passes, with the output quoted |
| Review | `review` | Each way the change hurts a person, proven, with a fix and a test | No open Critical or Important finding |
| Ship | `ship` | A pull request, then a merged and checked release | Your yes before anything leaves the machine |

A small change skips the spec and the plan. It still gets a failing test or a named check,
a review and the report.

Security is a step of each stage, not a pass someone must remember. The spec names each
sensitive surface the work touches. A task on such a surface writes its hostile-input
test first. The review runs a security pass by itself, and `ship` stops on an open check.
`security` audits the whole repository and remembers what it covered.

| | builder | your own CLAUDE.md |
|---|---|---|
| Starts from the person and the non-goals | every piece of work | if you wrote it |
| Each solo call recorded with its reason | every report, under Calls | if you wrote it |
| Unproven work marked as unproven | every report | if you wrote it |
| Security in each stage, not on request | spec, plan, build, review and ship | if you wrote it |

## Install

builder runs on macOS and Linux. It needs Bash, and it offers to install one small helper binary in your first session.

**Claude Code**

```bash
/plugin marketplace add Lutherwaves/builder
/plugin install builder@builder
```

**Codex CLI.** Trust the plugin's hook when Codex asks.

```bash
codex plugin marketplace add Lutherwaves/builder
codex plugin add builder@builder
```

**Cursor CLI** and **opencode:** see [docs/install.md](docs/install.md).

The first session asks to install the helper binary. It runs the session-start hook and
the security score. Answer yes, not now (quiet for 7 days) or never. The installer
downloads the binary of your plugin's version from builder's GitHub release, checks the
checksum and, with a signed-in `gh`, the build provenance. It installs to
`~/.config/builder/bin`. To install it yourself, run
`bash <plugin root>/scripts/install.sh`.

### Harness support

A harness counts as supported once every eval fixture passes on it.

<!-- support:start -->
| Harness | Version | Eval date | Status |
| --- | --- | --- | --- |
| Claude Code | 2.1.308 (Claude Code) | 2026-10-22 | experimental: shape 2/6; 8 not graded |
<!-- support:end -->

## Skills

| Skill | What you get |
| --- | --- |
| `builder` | The thirteen tenets, the five stages and their gates, and the report. |
| `shape` | A request becomes a spec. It reads the repository, asks one question at a time with its recommendation, shows two or three ways to build it, and waits for your approval. |
| `plan` | An approved spec becomes tasks that each start with a failing test, reviewed as an engineer and as a designer. |
| `build` | The plan runs one task at a time, test first, one commit per task. It stops when a step is wrong and proposes the change. |
| `review` | A review that proves each harm with evidence, a fix and a test, with a security pass and the running product. Posts nothing without your yes. |
| `ship` | Reviewed work becomes a checked release. It stops on an open finding and waits for your yes before each push, pull request and merge. |
| `security` | A whole-repository audit: every layer, standards-mapped findings with CVSS scores, and what changed since last time. |

Ask for a skill by name, or just describe the job. The agent loads the right one.

## Works with

builder runs on its own. When the remote is on GitHub, `ship` checks that `gh` is present
and signed in. It also checks that commit signing is on. When a check fails, `ship` stops
and names the fix.

- [gh](https://cli.github.com): `ship` opens and merges the pull request through it.

## Contributing

Every change lands through a pull request that passes `bash scripts/validate.sh`. See
[CONTRIBUTING.md](CONTRIBUTING.md), and [SECURITY.md](SECURITY.md) to report a vulnerability.

## Credits

builder takes ideas, and no text, from three MIT-licensed projects:
[obra/superpowers](https://github.com/obra/superpowers) (the spec, plan and build loop),
[garrytan/gstack](https://github.com/garrytan/gstack) (the reviews of a plan, the use of the running product, shipping)
and [cloudflare/security-audit-skill](https://github.com/cloudflare/security-audit-skill) (the audit method).

## License

MIT. The ideas builder learned from other projects, and their licenses, are in [Credits](#credits).
