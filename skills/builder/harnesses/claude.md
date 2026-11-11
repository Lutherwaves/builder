<!-- hook -->
On Claude Code, load a builder skill as /builder:<name> or with the Skill tool.
The full action map for this harness is in skills/builder/harnesses/claude.md.
<!-- /hook -->

| Action | claude |
|---|---|
| load a skill (and its syntax) | `Skill` tool, or `/builder:<name>`; name a skill as `builder:<name>` |
| ask the person to choose | `AskUserQuestion` |
| start a subagent | `Agent` tool |
| compact the context | `/compact` |
| clear the context | `/clear` |
| make a shareable page | `Artifact` tool |
| use a browser | not verified: say 'not used as a running product' |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
