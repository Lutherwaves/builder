<!-- hook -->
On Codex, builder skills show as builder:<name>. Load a skill by its name.
The full action map for this harness is in skills/builder/harnesses/codex.md.
<!-- /hook -->

| Action | codex |
|---|---|
| load a skill (and its syntax) | load by its name; Codex shows builder skills as `builder:<name>` |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | `spawn_agent` (needs `[features] multi_agent = true` in `~/.codex/config.toml`) |
| compact the context | not verified: write the record file, then ask the person to compact in the Codex UI |
| clear the context | not verified: write the record file, then ask the person to start a new chat |
| make a shareable page | not verified: write a local file and give its path |
| use a browser | not verified: say 'not used as a running product' |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
