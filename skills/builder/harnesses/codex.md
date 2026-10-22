<!-- hook -->
On Codex, builder skills show as builder:<name>. Load a skill by its name.
The full action map for this harness is in skills/builder/harnesses/codex.md.
<!-- /hook -->

| Action | codex |
|---|---|
| load a skill (and its syntax) | load by its name; Codex shows builder skills as `builder:<name>` |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | `spawn_agent` (needs `[features] multi_agent = true` in `~/.codex/config.toml`) |
| message a peer session | `followup_task` to a child agent; a peer session in another window: not verified, ask the person to relay the message |
| send the person a notification | not available: use the OS notification step |
| schedule a recurring check | not verified: ask the person to run the check on a timer, or name the cron line and stop |
| see usage limits | `/status` (manual); `matchblox usage --json` measures plan limits for Codex |
| compact the context | not verified: write the durable record, then ask the person to compact in the Codex UI |
| make a shareable page | not verified: write a local file and give its path |
| fetch a web page | not verified: use `curl` in the shell |
| track a todo list | `update_plan` |
| edit a file | `apply_patch` |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
