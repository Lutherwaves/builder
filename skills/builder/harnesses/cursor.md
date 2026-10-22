<!-- hook -->
On Cursor, builder skills load from the plugin and from .agents/skills.
The full action map for this harness is in skills/builder/harnesses/cursor.md.
<!-- /hook -->

| Action | cursor |
|---|---|
| load a skill (and its syntax) | load by its name; Cursor loads builder skills from the plugin and from `.agents/skills` |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | not verified: do the work in this session, or ask the person to start another agent |
| message a peer session | not verified: write the message to a file and give its path to the person |
| send the person a notification | not available: use the OS notification step |
| schedule a recurring check | not verified: ask the person to run the check on a timer, or name the cron line and stop |
| see usage limits | `matchblox usage` (matchblox 0.14.0 or newer, macOS, with `[limits] cursor_usage = true` in matchblox's config) shows a monthly window; otherwise not available: say so and stop |
| compact the context | not verified: write the durable record, then ask the person to start a fresh chat |
| make a shareable page | not verified: write a local file and give its path |
| fetch a web page | not verified: use `curl` in the shell |
| track a todo list | not verified: keep the list in your reply or in a file |
| edit a file | not verified: use the edit tool this harness gives you |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
