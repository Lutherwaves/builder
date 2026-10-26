# builder

[![Release](https://github.com/Lutherwaves/builder/actions/workflows/release.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/release.yml)
[![Validate](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org)

builder makes your coding agent answer to you. Every piece of work ends with a short
report: is it done, what you have to do, what isn't proven yet, what left your machine,
and every call the agent made on its own, with the reason. Nothing leaves the machine
(no push, no PR, no message) without a send rule or your yes.

It's a plugin for Claude Code, Codex CLI, Cursor CLI and opencode. Skills and one
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

superpowers and gstack make the agent better at a job: planning, debugging, a browser,
QA. builder's report is the same whichever skill ran: every finished piece of work ends
with what's done, what's yours to do, and what isn't proven. builder also covers ground
neither pack does: messages to people, rules that remember their incident, many sessions
at once, and your usage limit.

| | builder | superpowers | gstack | your own CLAUDE.md |
|---|---|---|---|---|
| Starts from the person and the non-goals | every piece of work | in brainstorming | in `/spec`, `/office-hours` | if you wrote it |
| Each solo call recorded with its reason | every report, under Calls | while executing a plan | durable decisions, in a decision log | if you wrote it |
| Unproven work marked as unproven | every report | when verification runs | in QA and review | if you wrote it |
| Messages to people: send rules, duplicate check, 6 an hour | `reach` | no | no | no |
| A rule carries its incident and a check that fails | `canon` | no | no | no |
| Many sessions, one lane each, hand-offs as briefs | `fleet` | no | no | no |
| Weekly usage-limit forecast | `budget` | no | no | no |

builder doesn't replace either pack. It routes to them: planning, TDD and debugging go to
superpowers; the browser, QA and shipping go to gstack. Install all three.

## Install

builder needs Bash and Python 3, on macOS or Linux.

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

In your first session the agent does your work first, then offers one setup step per
reply: superpowers, gstack, a signed-in `gh`, signed commits. Answer yes, not now (quiet
for 7 days) or never. Answers live in `~/.config/builder/ledger.json`; set `BUILDER_HOME`
to move them. After setup, a session shows at most one recommendation, and only when
something calls for it, such as your weekly usage passing 80%.

### Harness support

A harness counts as supported once every eval fixture passes on it.

<!-- support:start -->
| Harness | Version | Eval date | Status |
| --- | --- | --- | --- |
| Claude Code | 2.1.308 (Claude Code) | 2026-10-22 | experimental: canon 5/6; fleet 3/4; harness 4/6; reach 3/4; shape 2/6; 5 not graded |
<!-- support:end -->

## Skills

| Skill | What you get |
| --- | --- |
| `builder` | The thirteen tenets, routing to superpowers and gstack, and the report. |
| `shape` | A feature name becomes an issue: the person, the moment, anti-goals, open questions with a recommendation. |
| `canon` | A repo rule with its incident and a check that fails when someone breaks it. |
| `reach` | Messages to teammates wait for your yes, with a duplicate check and 6 sends an hour. |
| `fleet` | Many sessions side by side, one lane each; work outside a lane goes to its owner. |
| `intake` | Today's captured tasks reconciled against work in motion; nothing filed without your yes. |
| `budget` | A forecast of your weekly usage limit, and which idle session to compact. |
| `harness` | A local code graph that answers "what calls this?" in one query. |
| `review` | A review that proves each harm with evidence, a fix and a test. Posts nothing without your yes. |
| `security` | A full security stance: every layer, standards-mapped findings with CVSS scores, and what changed since last time. |

Ask for a skill by name, or just describe the job. The agent loads the right one.

## Works with

builder runs on its own and says once when a partner is missing.

- [superpowers](https://github.com/obra/superpowers): planning, TDD, debugging, review.
- [gstack](https://github.com/garrytan/gstack): a real browser, QA, security and ship workflows.
- [gh](https://cli.github.com): `intake` reads open PRs and issues through it.
- [matchblox](https://matchblox.sh): `budget` reads your plan windows and `fleet` your live sessions from it ([docs](https://docs.matchblox.com)).
- Your team channel's MCP server: `reach` sends through it. For Slack, the claude.ai Slack connector or Slack's own MCP server.
- [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp): `harness` sets it up; it runs locally with no API key.

## Contributing

Every change lands through a pull request that passes `bash scripts/validate.sh`. See
[CONTRIBUTING.md](CONTRIBUTING.md), and [SECURITY.md](SECURITY.md) to report a vulnerability.

## License

MIT. The `harness` skill takes its framing from Ryan Lopopolo's
[Harness Engineering](https://github.com/lopopolo/harness-engineering) (CC BY 4.0);
builder adapts the ideas and copies no prose.
