# Installing builder

builder needs Bash and Python 3, on macOS or Linux. Its session hook makes no network
call and finishes in under a second.

## Claude Code

```bash
/plugin marketplace add Lutherwaves/builder
/plugin install builder@builder
```

Call a skill as `/builder:shape`.

## Codex CLI

```bash
codex plugin marketplace add Lutherwaves/builder
codex plugin add builder@builder
```

Codex runs a plugin's hooks only after you trust them, and builder doesn't work around
that. Until you trust the hook, load the `builder` skill by name. Until Codex detection
lands (issue #17), builder's session start on Codex uses the Claude Code setup steps.

## Cursor CLI

In your project, create `.cursor/hooks.json`. Use the absolute path of your builder
checkout:

```json
{"version":1,"hooks":{"sessionStart":[{"command":"<path to builder>/hooks/session-start --harness cursor"}]}}
```

Keep only these keys: one invalid key disables every hook in the file. Then copy
builder's skill folders into `.agents/skills/` in the same project.

For the Cursor editor's plugin flow, use `.cursor-plugin/plugin.json` instead.

Without the hook, add this line to your project's `AGENTS.md`, which Cursor CLI reads.
builder never writes `AGENTS.md` itself.

```
Use the builder skill for the soul of this work: for whom, what it is not, each call with its reason, proof, and what left the machine.
```

## opencode

Use opencode 1.19 or newer: 1.1.x crashes on plugins with non-function exports. Add
builder to the `plugin` list in `opencode.json`. Point it at a local checkout:

```json
{"plugin": ["/path/to/builder"]}
```

Or use the git form that opencode documents:

```json
{"plugin": ["builder@git+https://github.com/Lutherwaves/builder.git"]}
```
