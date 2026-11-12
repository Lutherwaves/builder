# A builder with seven skills and one job

Status: approved. Stacked on `feat/framework` (the five-stage framework).

## Why

The owner wrote six skills and never used them: `budget`, `canon`, `fleet`, `harness`, `intake`, `reach`.
builder's tenet 4 says: delete what nobody uses.
After the framework change, builder has one job: shape, plan, build, review and ship, with security in every stage.
The six skills do other jobs. They cost every user context, install size and reading time, and they cost the maintainer tests and evals.

## The person and the moment

A person installs builder to build something with a coding agent.
After this change the person sees seven skills, and each one is a stage of that work or the frame around it.
The session start shows the soul and nothing else.

## Goals

1. builder has seven skills: `builder`, `shape`, `plan`, `build`, `review`, `ship`, `security`.
2. The binary has three commands: `hook`, `security`, `version`.
3. The session-start hook prints the soul, the action map of the harness and the line that names the builder root. It recommends nothing and asks nothing. One exception stays: when the binary is missing, the hook offers to install it.
4. `ship` finds a missing forge command line tool or missing commit signing itself, stops, and says the fix.
5. No file that stays tells a person or an agent to use a removed skill, command or rule.
6. A person who used a removed skill can find out, in one place, what went and which release still has it.

## Non-goals

- No replacement plugin for the removed skills. A person who wants one takes it from the git history.
- No change to the behaviour of the seven skills that stay, apart from the lines that name a removed skill.
- No migration of state files.

## Anti-goals

- The change must not weaken "nothing leaves this machine without the person's yes". Today that promise does not depend on `reach`: tenet 10 and the gates of `ship` carry it.
- The change must not delete a file on a person's machine. A ledger file or a send-rule file that an older version wrote stays where it is. Nothing reads it.
- The hook must not fail or print an error when it finds state from an older version.

## Surfaces

- Outbound sends. Removing `reach` removes the send rules, the only way to pre-approve a send. After the change, each send needs the person's yes in the session. Harm to avoid: a leftover phrase "a send rule" that an agent reads as permission.
- The release. This is a breaking release for every person who invokes a removed skill or command.

## Approaches

1. Skills only. Delete six skill folders, their fixtures and their README rows. The Go packages stay with no caller. Rejected: dead code on the first day.
2. Skills and everything only they use. Also delete `internal/reach`, `internal/budget`, `internal/mcpmerge`, seven of the nine recommendations and the wiring. The recommendation engine stays for `gh` and `signing`. Rejected: an engine with two entries.
3. All of that, plus the recommendation engine and onboarding. **Picked by the owner (2026-11-11).** builder ends as seven skills, a hook that prints the soul, and the security score.

## What goes

| Thing | Where | Why it can go |
| --- | --- | --- |
| Six skills | `skills/{budget,canon,fleet,harness,intake,reach}/` | Never used |
| Their fixtures | `tests/evals/{budget,canon,fleet,harness,intake,reach}/` and their entries in `tests/evals/results/` | Nothing to grade |
| Send rules | `internal/reach`, `cmd/builder/reach.go` | Only `reach` uses them |
| Usage forecast | `internal/budget`, `cmd/builder/budget.go` | Only `budget` uses it |
| MCP config merge | `internal/mcpmerge`, `cmd/builder/mcpmerge.go` | Only `skills/harness/setup.sh` calls it |
| Recommendations | `lib/recommendations.json`, `internal/signals`, `internal/ledger`, `cmd/builder/ledger.go` | Seven of nine entries serve removed skills |
| Onboarding | the first-run text in `internal/hook` | It lives in the ledger; the soul says the same in every session |
| Drift check | `tests/test_drift.sh` | It tests `skills/harness/drift.sh` |
| State path test | `tests/test_state_paths.sh` | It tests the repository list of `skills/harness/drift.sh` |
| Harness paths | the fields of `internal/harness` that only `mcpmerge`, `signals` and `ledger` read (`MCP`, `ProjectMCP`, `SkillsDirs`, `Settings`, `ProcessMatch`, `PluginRootVar`, `AllProcessMatch`) and their tests | No caller stays. `Detect` and `BuilderHome` stay |
| Keywords | `fleet` and `usage-budget` in `package.json` | They name removed skills |

## What changes in the files that stay

- `internal/hook`: `body` returns the soul and the harness mapping block. No catalog, no ledger, no signals. The "builder root" line stays: the skills need it.
- `internal/hook/hook_test.go` and `tests/test_hook.sh`: write them again for the new output. They assert the ledger and onboarding today. Add one case with a ledger file from 3.0.0 present.
- `cmd/builder/security.go`: it saves the audit baseline with `ledger.SaveJSON`. Move that function into `internal/security` before `internal/ledger` goes.
- The install offer (`hooks/session-start`, `bin/builder`, `scripts/install.sh`, `tests/test_offer.sh`): the offer stays. Its text says today that the binary "runs the setup steps, send checks and forecasts". New text: the binary runs the session-start hook and the security score. The fallback of the hook without the binary prints the soul only, as today.
- `internal/harness`: keep harness detection. Delete what only the ledger or the signals used.
- `skills/builder/soul-compact.md`: delete the rule "One role per session…". It points at `fleet`.
- `skills/builder/SKILL.md`: delete the pointer to `canon`. Replace "a send rule or the person's yes" with "the person's yes", in the tenet test and in the self-check. Tenet 9 (rules are guards) stays.
- `skills/ship/SKILL.md`: the look back still lists what broke and what the person corrected. It hands nothing over. New first check, which the agent runs as shell commands with no help from the binary: `gh` is present, `gh auth status` passes, and `git config commit.gpgsign` is true. When one fails, `ship` stops and names the command that fixes it. The check covers GitHub only, as the two recommendations do today. `ship` names `canon` in five places; all five go.
- `skills/builder/harnesses/*.md`: delete each action row that only a removed skill uses: schedule, usage, message a peer, notify. The compact row stays, because the report of `builder` uses it. `scripts/validate.sh` and the opencode adapter check these files; change the checks with them.
- `tests/evals/lib.sh`, `tests/evals/run.sh`, `tests/test_evals.sh`: they prepare an onboarded ledger today. They stop doing that.
- `tests/evals/results/claude.json`: delete the keys of the six fixtures.
- `tests/templates_test.go`: delete the tests that read `skills/{canon,reach,budget}`. Change the assertion on "a send rule or the person's yes" to the new wording.
- `scripts/validate.sh` and `tests/test_validate.sh`: validate runs `builder budget` as a smoke test and checks the catalog; use `builder security` for the smoke test. The cases of `test_validate.sh` use removed skills as their targets; give them targets among the seven that stay.
- `tests/evals/ship/`: one new bullet or fixture state for the missing-tool stop. The work dir is a temp copy, so the fixture hides `gh` through the PATH. Run the fixture without the change first and watch it fail.
- `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `SECURITY.md`, the four manifests, the bug report template, `scripts/validate.sh`, `tests/test_validate.sh`, the release workflow: delete each line that names a removed thing. Regenerate the support table; never edit it by hand.
- `CHANGELOG.md`, under `[Unreleased]`: one "Removed" entry that lists the six skills and the four commands, and says that release 3.0.0 is the last one that has them.

## Rulings

- 2026-11-11: depth 3 (delete the recommendation engine too). Decided by: the owner.
- 2026-11-11: delete onboarding, keep no intro line. Decided by: the owner.
- 2026-11-11: this is a second pull request, after the framework one. Decided by: the owner.
- 2026-11-11: spec approved. Decided by: the owner.
- 2026-11-11: delete the "one role per session" rule with `fleet`. Proposed, not confirmed. Cost if wrong: parallel sessions get no guidance from builder.
- 2026-11-11: "a send rule or the person's yes" becomes "the person's yes". Proposed, not confirmed. Cost if wrong: a person who relied on send rules is asked for each send.
- 2026-11-11: the look back in `ship` stays as a list and hands nothing over. Proposed, not confirmed. Cost if wrong: a repeated mistake is listed and never becomes a rule.
- 2026-11-11: leave old state files on the person's machine untouched. Proposed, not confirmed. Cost if wrong: a few stale files stay in the builder state folder.

- 2026-11-11: the offer to install the binary stays, with new text. Proposed, not confirmed. Cost if wrong: a new session without the binary still has one thing to answer.
- 2026-11-11: `ship` checks GitHub only. Proposed, not confirmed. Cost if wrong: a person on another forge gets a stop that does not fit.

## Open questions for the owner

Does the parity eval stay? `tests/evals/parity.sh` and `cmd/parity-config` compare builder alone with a person's own setup. The framework release uses them once, to prove that builder alone is not worse.
Recommendation: delete both in this pull request, after that one use. Cost if wrong: a later claim "builder alone is enough" has no eval behind it.


One release or two? The framework and this change are both breaking.
Recommendation: hold the framework release until this pull request merges, so that a person meets one breaking release.
This changes the earlier decision "one pull request, one release" to "two pull requests, one release".
Cost if the recommendation is wrong: the framework reaches its users some days later.

## Left to the builder

- The order of deletion, and how to split the commits. Each commit passes `bash scripts/validate.sh`.
- How `ship` detects the forge tool and commit signing on each harness.
- Whether `internal/cli` loses helpers that only the removed commands used.

## Done when

1. `ls skills` prints seven folders, and `builder` with no argument lists `hook`, `security` and `version` only.
2. The hook output on each of the four harnesses is the soul and the action map, and `bash tests/test_hook.sh` proves it, also with a ledger file from 3.0.0 present.
3. A search of the files that stay finds no instruction to use a removed skill, command or send rule. The changelog entry and the tests that prove the removal may name them.
4. `go test ./...` and `bash scripts/validate.sh` pass, and `bash tests/test_validate.sh` passes.
5. The `ship` fixture has a state with no forge tool, and the run with builder stops and names the fix. The `builder`, `ship` and `flow-gates` fixtures are run again on Claude Code and Codex, and meet no fewer bullets than before this change.
6. The owner reads the hook output of a new session, with the binary installed, and finds nothing to answer.
