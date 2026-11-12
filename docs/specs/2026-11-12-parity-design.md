# A deslop pass, a debug skill, and the same flow on every harness

Status: approved. Stacked on `feat/chain`.

## Why

A comparison of builder with the two projects it learned from showed four gaps.
builder had principles against slop and no step that removes it.
It had one line on debugging and no method.
Its stage chain named skills in a form that only Claude Code and Codex show.
It had no adapter for Grok Build, and on Codex its session start printed the Claude Code action map.

## The person and the moment

A person builds with a coding agent on one of five harnesses: Claude Code, Codex CLI, Cursor CLI, Grok Build or opencode.
The person wants the same flow on each one, a change with nothing in it that the task did not need, and a bug fixed at its cause.

## Goals

1. `build` removes what its own change did not need, before the last test run, and says what it removed.
2. `review` lists such lines under "Slop". Slop is not a finding and does not touch the gate.
3. A failure with no known cause goes to a `debug` skill: reproduce, root cause in one sentence, a failing test, a fix at the cause, a stop after three fixes that did not hold.
4. On each of the five harnesses, the session start prints the action map of that harness, and the map says how to load a skill that a handoff names as `builder:<name>`.
5. A skill that runs the helper binary says where the binary is when no session-start hook ran.

## Non-goals

- No sixth stage. `debug` is outside the flow, as `security` is.
- No eval runner for Grok Build, and no row for it in the support table. Both wait for a machine that has Grok Build.
- No paid eval run. The owner ruled that out on 2026-11-12.
- No qualified skill names on Cursor CLI, Grok Build or opencode. Those harnesses name a plugin's skill by its plain name.

## Anti-goals

- The deslop pass must not remove a test, a guard on outside input, or a line outside the change.
- `review` must not report slop as a finding, and must not drop a real finding as slop.
- `debug` must not make a failure go away by a skipped test, a loosened assertion, a retry or a caught error.

## Surfaces

- Agent instructions. The session-start text decides which tool names an agent uses. Harm to avoid: an agent on one harness that gets the tool names of another, and stalls or loads a skill of the same name from another pack.

## Approaches

1. An eighth and a ninth skill, `debug` and `deslop`. Rejected for `deslop`: it is a step of building and of reviewing, not a job of its own.
2. **Picked.** One new skill, `debug`. One reference file, `skills/build/deslop.md`, that `build` runs and `review` reads. One owner for the list.
3. Harness detection by a manifest and a hook file per harness. Rejected for Codex and Grok Build: both read builder's Claude Code manifest, and both set their own variable for a plugin hook (`PLUGIN_ROOT`, `GROK_PLUGIN_ROOT`), so the one hook can tell them apart.

## What changes

- `skills/build/deslop.md` (new): the pass, the list, and what is never slop.
- `skills/build/SKILL.md`: section 4 loads `builder:debug`; section 8 starts with the deslop pass and removes the slop that the review lists.
- `skills/review/SKILL.md` and `skills/review/bar.md`: the "Slop" heading and its rule.
- `skills/debug/SKILL.md` and `tests/evals/debug/` (new).
- `skills/builder/SKILL.md` and `skills/builder/soul-compact.md`: they name `builder:debug`.
- `internal/harness`: `grok` is a harness; `auto` reads the markers in an order that tells Grok Build, Claude Code and Codex apart.
- `skills/builder/harnesses/grok.md` (new); the Cursor, Grok Build and opencode maps say what `builder:<name>` means.
- `skills/security/SKILL.md` and `skills/review/security-pass.md`: where the binary is with no hook.
- `tests/chain_test.go`: four tests for the rules above.
- `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `docs/install.md`, `CHANGELOG.md`, the manifests.

## Rulings

- 2026-11-12: add the deslop pass to `build` and `review`, add debugging, and close the chaining and harness gaps. Decided by: the owner.
- 2026-11-12: Grok is one of the harnesses builder supports. Decided by: the owner.
- 2026-11-12: `debug` is a skill outside the five stages. Proposed, not confirmed. Cost if wrong: a person expects a sixth stage in the flow table.
- 2026-11-12: slop does not block the gate of `review`. Proposed, not confirmed. Cost if wrong: a change with slop ships when `build` did not remove it.
- 2026-11-12: Codex is detected by `PLUGIN_ROOT` and Grok Build by `GROK_PLUGIN_ROOT`. Proposed, not confirmed. The Codex and xAI documents name both variables; no session on either harness has shown them. Cost if wrong: that harness gets the Claude Code action map, as before this change.
- 2026-11-12: the Grok Build action map marks six of seven actions "not verified". Proposed, not confirmed. Cost if wrong: an agent on Grok Build asks in plain text where a better tool exists.

## Done when

1. `go test ./...`, `bash scripts/validate.sh` and `bash tests/test_validate.sh` pass, and each SKILL.md has 60 to 150 lines.
2. With `PLUGIN_ROOT` set, the hook prints the Codex map. With `GROK_PLUGIN_ROOT` set, it prints the Grok Build map. With neither, in Claude Code, it prints the Claude Code map.
3. The tests fail when a harness map loses the `builder:<name>` line, when `build` no longer loads `deslop.md` or `builder:debug`, and when `review` loses the "Slop" heading.
4. `bash tests/evals/run.sh --check` accepts the `debug` fixture, and its sample test fails before any fix.
