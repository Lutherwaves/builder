<!-- hook -->
On Claude Code, load a builder skill as /builder:<name> or with the Skill tool.
The full action map for this harness is in skills/builder/harnesses/claude.md.
<!-- /hook -->

| Action | claude |
|---|---|
| load a skill (and its syntax) | `Skill` tool, or `/builder:<name>`; name a skill as `builder:<name>` |
| ask the person to choose | `AskUserQuestion` |
| start a subagent | `Agent` tool |
| message a peer session | `SendMessage` |
| send the person a notification | `PushNotification` |
| schedule a recurring check | `/loop <interval> <prompt>`; cron tools `CronCreate`, `CronDelete`; `ScheduleWakeup` for one wake-up |
| see usage limits | `/usage`; or `matchblox usage --json` |
| compact the context | `/compact` |
| make a shareable page | `Artifact` tool |
| fetch a web page | `WebFetch` |
| track a todo list | `TodoWrite` |
| edit a file | `Edit`; `Write` for a new file |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
