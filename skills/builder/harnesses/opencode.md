<!-- hook -->
On opencode, load a builder skill with the native skill tool.
The full action map for this harness is in skills/builder/harnesses/opencode.md.
<!-- /hook -->

| Action | opencode |
|---|---|
| load a skill (and its syntax) | native `skill` tool |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | `task` with `subagent_type: "general"` (v1.x); `subagent` with `agent: "general"` (v2) |
| message a peer session | not verified: write the message to a file and give its path to the person |
| send the person a notification | not available: use the OS notification step |
| schedule a recurring check | not verified: ask the person to run the check on a timer, or name the cron line and stop |
| see usage limits | not available: say so and stop (matchblox does not measure opencode usage) |
| compact the context | not verified: write the durable record, then ask the person to start a fresh session |
| make a shareable page | not verified: write a local file and give its path |
| fetch a web page | `webfetch` |
| track a todo list | `todowrite` (v1.x); v2 has no todo tool: track the plan in a markdown file |
| edit a file | `apply_patch` (v1.x); `patch`, `write` or `edit` (v2) |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
