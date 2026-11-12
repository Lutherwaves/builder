<!-- hook -->
On Grok Build, load a builder skill by its name, or as a slash command with that name.
A skill text that says `builder:<name>` means the skill `<name>` of builder. When two skills share that name, take the one under the builder root.
The full action map for this harness is in skills/builder/harnesses/grok.md.
<!-- /hook -->

| Action | grok |
|---|---|
| load a skill (and its syntax) | load by its name, without the `builder:` prefix, or type `/<name>` |
| ask the person to choose | not verified: ask in plain text, one question at a time, with your recommendation |
| start a subagent | not verified: do the work in this session, or ask the person to start another agent |
| compact the context | not verified: write the record file, then ask the person to start a fresh session |
| clear the context | not verified: write the record file, then ask the person to start a fresh session |
| make a shareable page | not verified: write a local file and give its path |
| use a browser | not verified: say 'not used as a running product' |

A cell that starts with `not verified:` says what to do instead; the tool name is not confirmed on this harness.
