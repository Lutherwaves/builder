# builder

[![Release](https://github.com/Lutherwaves/builder/actions/workflows/release.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/release.yml)
[![Validate](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml/badge.svg)](https://github.com/Lutherwaves/builder/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org)

builder is a plugin for coding agents. Your agent ends each piece of work with a builder report.
The report tells you, in this order: is it done, what you must do, what is not proven, what left the machine, and what the agent decided alone.
Nothing leaves this machine without a send rule or your yes.

builder is the soul behind that report.
You decide. The agent proposes, builds inside the repository, and reports.
Correct is not enough: the work must fit one person at one moment.
So each piece of work starts from that person and the non-goals.
Each call the agent makes alone carries its reason.
The proof is a check from the session, and a check it did not run shows as not proven.

Example, invented for illustration:

> **▶ Needs you:** a customer who moved can now change their email at checkout, in one step.
>
> **▶ You (1):** push `feat/change-email` and open the PR. I ask below.
>
> **⚠ Not proven:** the confirmation email in a real inbox.
>
> **✓ Proven:** `python3 -m unittest` → 14 passed (example).
>
> **↗ Outside:** nothing sent.
>
> **Calls:** the old address gets a notice, because a stolen account changes the email first.

The marks have one meaning each: ✓ done or proven, ✗ failed, ▶ you act, ⚠ not proven, ↗ left this machine.
After the report, the agent asks each ▶ You item as a choice, with its recommendation first. On a harness with no choice tool, it lists the exact words to reply.

## Install

builder installs on four harnesses. Pick yours.
builder needs Python 3 and Bash, on macOS or Linux.
Its session hook makes no network call and finishes in under one second.

### Claude Code

```bash
/plugin marketplace add Lutherwaves/builder
/plugin install builder@builder
```

In Claude Code, call a skill as `/builder:shape`.

### Codex CLI

```bash
codex plugin marketplace add Lutherwaves/builder
codex plugin add builder@builder
```

Codex runs a plugin's hooks only after you trust them. builder does not work around this step.
Until you trust the hook, the `builder` skill still loads by name.
Until Codex detection lands (issue #17), builder's session start on Codex uses the Claude Code step list and mapping.

### Cursor CLI

Install builder for Cursor CLI by hand:
In your project, create `.cursor/hooks.json`:

```json
{"version":1,"hooks":{"sessionStart":[{"command":"<path to builder>/hooks/session-start --harness cursor"}]}}
```

`<path to builder>` must be an absolute path.

Keep only these keys. One invalid key disables every hook in the file.
Then copy builder's skill folders into `.agents/skills/` in the same project.

The Cursor editor installs builder from `.cursor-plugin/plugin.json`.

As a fallback, add one line to your project's `AGENTS.md`. Cursor CLI reads that file.
builder never writes `AGENTS.md` itself.

```
Use the builder skill for the soul of this work: for whom, what it is not, each call with its reason, proof, and what left the machine.
```

### opencode

Add builder to the `plugin` list in `opencode.json`. Use opencode 1.19 or newer: version 1.1.x crashes on plugins with non-function exports.
A local checkout works:

```json
{"plugin": ["/path/to/builder"]}
```

opencode also takes the git form: `"plugin": ["builder@git+https://github.com/Lutherwaves/builder.git"]`.

### Harness support

A harness is listed as supported only after every eval fixture passes on it.

<!-- support:start -->
| Harness | Version | Eval date | Status |
| --- | --- | --- | --- |
| Claude Code | 2.1.308 (Claude Code) | 2026-10-22 | experimental: canon 5/6; fleet 3/4; harness 4/6; reach 3/4; shape 2/6; 5 not graded |
<!-- support:end -->

## What happens in your first session

1. You ask for your work as usual. builder adds its thirteen tenets to the session, and the agent does your work first.
2. At the end of that reply, the agent introduces builder in at most three short lines. This happens even after a one-line request.
3. The agent offers one setup step and asks: yes, not now, or never.
4. Each later reply offers the next step, one at a time: superpowers, gstack, a signed-in `gh`, and signed commits. A step you already did is skipped.
5. "Not now" stays quiet for 7 days. "Never" stays quiet for good.
6. When you answer every step, onboarding ends.

After that, a session shows at most one recommendation, and only when a signal fires.
Examples: you run three or more sessions, your weekly usage reaches 80%, or no code graph is set up.
The first two signals need matchblox.
builder keeps your answers in `~/.config/builder/ledger.json`. Set `BUILDER_HOME` to move it.

## The ten skills

| Skill | What you get |
| --- | --- |
| `builder` | The soul: thirteen tenets, a routing table to superpowers and gstack, and the builder report at the end of each piece of work. |
| `shape` | A feature name becomes an issue: the person, the moment, anti-goals, and each open question with a recommendation. |
| `canon` | A repository rule carries its incident and a check that fails when someone breaks it. |
| `reach` | By default, a message to a teammate waits for your yes. Every send passes a duplicate check and a limit of 6 sends per hour. |
| `fleet` | Many sessions work side by side, one lane each. Work outside a lane goes to its owner as a brief. |
| `intake` | The day's captured tasks are reconciled against work in motion. Nothing becomes an issue without your marker or your yes. |
| `budget` | A forecast of the weekly usage limit, and one idle session to compact. Where the harness has no usage reading, it says so. |
| `harness` | A local code graph answers "what calls this?" in one query, and a watcher names stale graphs. |
| `review` | A review proves how a change hurts a person: what it ran, each harm with its evidence, fix and test, and a verdict. Nothing is posted without your yes. |
| `security` | One run gives the full stance: every OSI layer, memory and DoS included, each finding mapped to the standards with a computed CVSS score, and what changed since the last audit. Nothing is posted without your yes. |

Call a skill by name, for example the `shape` skill. The agent also loads a skill when its job comes up.

## Works with

builder runs on its own. If a partner is missing, builder uses a fallback and says so once.

- [superpowers](https://github.com/obra/superpowers): planning, test-driven development and review. builder routes to these skills.
- [gstack](https://github.com/garrytan/gstack): a real browser, QA, security checks and ship workflows.
- [gh](https://cli.github.com): `intake` checks open pull requests and issues through it.
- [matchblox](https://matchblox.sh): `budget` reads the plan windows from it, and `fleet` reads the live sessions. Run `matchblox setup` to add its hooks ([docs](https://docs.matchblox.com)).
- Your channel's MCP server: `reach` sends through it. For Slack, use a Slack connector or Slack's official MCP server.
- [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp): `harness` provisions it. It runs locally and needs no API key.

## Compare with your own setup

`tests/evals/parity.sh` runs two fixtures twice: once with your setup, and once with builder and its recommendations only.
The builder run accepts every recommendation.
It prints both transcripts next to the expected behaviour, so you can compare them.
It runs the agent four times, so it uses tokens from your plan.
`parity.sh` works on Claude Code only.
See [tests/evals/README.md](tests/evals/README.md).

## Standalone use of the budget forecast

`skills/budget/burn-proj.py` works without the plugin:

```bash
python3 burn-proj.py <used_pct> <resets_at_epoch> [now_epoch]
python3 burn-proj.py --from-matchblox [now_epoch]

python3 burn-proj.py 19 1783504800 1782990571
# elapsed=15.0%  naive=127%  profile_aware=100%  reset_in=5.95d
```

`--from-matchblox` needs matchblox on your PATH. Without it, read your agent's usage view and pass the numbers by hand.

- `naive` draws a straight line from the use so far. It is the worst case.
- `profile_aware` discounts the hours you do not work: 02:00 to 09:00, and the weekend. Tune them in `mult()`.

## Contributing

Every change lands through a pull request, and the `validate` check must pass.
Run the same check before you push:

```bash
bash scripts/validate.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the rules, and [SECURITY.md](SECURITY.md) to report a vulnerability.

## License

MIT.

The `harness` skill derives its framing from Ryan Lopopolo's
[Harness Engineering](https://github.com/lopopolo/harness-engineering), licensed CC BY 4.0.
builder adapts the concepts and copies no source prose.
