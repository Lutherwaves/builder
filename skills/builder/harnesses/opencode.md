<!-- hook -->
On opencode, load a builder skill with the native skill tool.
The full action map for this harness is in skills/builder/harnesses/opencode.md.
<!-- /hook -->

| Action | opencode |
|---|---|
| load a skill (and its syntax) | native `skill` tool |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | `task` with `subagent_type: "general"` (v1.x); `subagent` with `agent: "general"` (v2) |
| compact the context | not verified: write the record file, then ask the person to start a fresh session |
| clear the context | not verified: write the record file, then ask the person to start a fresh session |
| make a shareable page | not verified: write a local file and give its path |
| use a browser | not verified: say 'not used as a running product' |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
