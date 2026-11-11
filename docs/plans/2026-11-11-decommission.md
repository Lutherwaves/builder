# Decommission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** builder has seven skills, a binary with three commands, and a hook that prints the soul and asks nothing.

**Architecture:** Delete from the outside in. First free what stays from what goes (one shared function, the hook, `ship`). Then delete the skills in two groups with the Go code only they use. A guard test then keeps the removed names out. Every commit leaves the repository passing `bash scripts/validate.sh`.

**Tech Stack:** Go 1.x (`cmd/builder`, `internal/*`), Bash (hook, shim, tests, eval runner), markdown skills.

**Spec:** `docs/specs/2026-11-11-decommission-design.md`

## Global Constraints

- The seven skills that stay: `builder`, `shape`, `plan`, `build`, `review`, `ship`, `security`. The three commands that stay: `hook`, `security`, `version`.
- No behaviour change in the seven skills apart from the lines that name a removed thing.
- Never delete or rewrite a file on a person's machine. A `ledger.json` or a send-rule file from 3.0.0 stays and nothing reads it. The hook prints no error when one is present.
- "a send rule or the person's yes" becomes "the person's yes" in every file that stays.
- A skill has 60 to 150 lines, the sections `## Common mistakes` and `## Self-check`, and ASD-STE100 prose. It names actions, never a harness tool or a slash command.
- A skill change starts with an eval fixture. Paid eval runs happen only in Task 7.
- Run `bash scripts/validate.sh` before each commit. Commits are signed Conventional Commits. Never `--no-verify`.
- The repository is public: no employer, client, product or private repository name in any file or commit. Search the staged diff before each commit.
- Never edit a released `CHANGELOG.md` section or a manifest `version`. Never edit the README support table by hand: `go run ./cmd/support-table render README.md`.
- Never commit an eval transcript. No push and no pull request without the owner's yes.

## Review Focus

1. A `ledger.json` from 3.0.0 is in the builder home: the hook prints the soul, exit 0, no error text. (Task 2)
2. The harness name is unknown or its map file is missing: the hook prints the soul without the map, exit 0. (Task 2)
3. The binary is missing: the Bash fallback prints the soul and the reworded install offer, and names no removed feature. (Task 2)
4. `gh` is installed but not signed in, or signing is off while `gh` is fine: `ship` names the one failing check and its fix, not all three. (Task 3)
5. An eval run starts with an empty builder home: `tests/evals/run.sh --dry-run` and `tests/test_evals.sh` pass with no `ledger` command. (Task 2)

---

### Task 1: The audit saves its baseline without the ledger package

**Files:**
- Modify: `internal/cli/` (new file `save.go`, test `save_test.go`), `internal/ledger/ledger.go`, `cmd/builder/security.go`, `cmd/builder/reach.go`

**Interfaces:**
- Produces: `cli.SaveJSON(path string, v any, indent string) error`, the same behaviour as `ledger.SaveJSON` today (temp file in the same folder, then rename).

- [ ] `ledger.SaveJSON` has no test of its own today. Write `internal/cli/save_test.go`: `TestSaveJSONWritesTheValue` (the file holds the JSON, with the indent given, and no temp file stays in the folder) and `TestSaveJSONLeavesTheOldFileOnError` (a value that cannot be encoded, such as a channel, gives an error and the old file is unchanged). Run: `go test ./internal/cli/` → FAIL (undefined `SaveJSON`).
- [ ] Move the function body to `internal/cli/save.go`. `ledger.Save`, `cmd/builder/security.go` and `cmd/builder/reach.go` call `cli.SaveJSON`. Delete `ledger.SaveJSON`.
- [ ] Run: `go test ./...` → PASS. `bash scripts/validate.sh` → "all checks passed".
- [ ] Commit: `refactor: the atomic JSON save lives in the cli package`

### Task 2: The hook prints the soul and asks nothing

**Files:**
- Delete: `internal/ledger/`, `internal/signals/`, `lib/recommendations.json`, `cmd/builder/ledger.go`, `cmd/builder/ledger_test.go`, `tests/evals/parity.sh`, `cmd/parity-config/`
- Modify: `internal/hook/hook.go`, `internal/hook/hook_test.go`, `tests/test_hook.sh`, `hooks/session-start`, `bin/builder` (usage text only), `tests/test_offer.sh`, `tests/evals/lib.sh`, `tests/evals/run.sh`, `tests/test_evals.sh`, `tests/evals/golden-dry-run-claude.txt`, `tests/evals/README.md`, `scripts/validate.sh`, `internal/harness/harness.go` and its test, `README.md`, `AGENTS.md`, `CONTRIBUTING.md`

**Interfaces:**
- Produces: `hook.Build(root, name string, env map[string]string, home, cwd string, now float64) string` keeps its signature. Its result is: the soul, then the "builder root" line, then the harness mapping block. Nothing else.

- [ ] Rewrite `internal/hook/hook_test.go`. Keep the shape tests (`TestHookShapeCursor`, `TestHookShapeClaude`, `TestHookFailureFallsBackToSoul`). Replace the two onboarding tests with:
  - `TestHookPrintsSoulRootAndMap`: the output contains the first line of `soul-compact.md`, `builder root: `, and the hook block of `harnesses/claude.md`.
  - `TestHookOffersAndAsksNothing`: the output contains none of `builder recommends`, `onboarding`, `ledger`, `yes|later|never`.
  - `TestHookIgnoresAnOldLedger`: with a `ledger.json` holding `{"onboarded":false,"answers":{"fleet":"later"}}` in the builder home, the output equals the output without that file.
  - `TestHookUnknownHarnessGetsTheSoul`: name `"nope"` gives the soul and no error text.
- [ ] Run: `go test ./internal/hook/` → FAIL.
- [ ] Reduce `hook.body` to the soul. Delete `commands`, `block`, `onboardingText`. Delete the imports of `ledger` and `signals`.
- [ ] Rewrite `tests/test_hook.sh` to the same four facts, through `hooks/session-start` on each of the four harness names. Keep its timing check (under one second).
- [ ] `hooks/session-start` line 56 and `README.md` line 91: the offer text becomes "builder's helper binary is not installed. It runs the session-start hook and the security score. Install it now? yes, not now, or never." Update the exact string in `tests/test_offer.sh`.
- [ ] `tests/evals/lib.sh` `eval_builder_home`: keep the temp `BUILDER_HOME` and the binary build. Delete lines 212 to 214 (`ledger onboarded`, `ledger answer`). `run.sh` line 57 says `# env: BUILDER_HOME=<temp dir>`. Update the golden dry-run file and `tests/test_evals.sh` (the check at line 67 keeps its assertion that no "builder recommends" text reaches the agent; delete the parity cases).
- [ ] `scripts/validate.sh`: delete the catalog check (the `how_hits` block, lines 120 to 128). The guard above it, that a skill which runs the shim stops at exit 3, stays. Delete every line in `AGENTS.md`, `CONTRIBUTING.md` and `tests/evals/README.md` that names the catalog, the ledger, the signals or the parity eval.
- [ ] `internal/harness`: delete the fields and functions whose only readers were `ledger` and `signals`. Leave what `internal/mcpmerge` reads; Task 5 removes those.
- [ ] Delete the listed files. Run: `go build ./... && go test ./... && bash tests/test_hook.sh && bash tests/test_offer.sh && bash tests/test_evals.sh && bash tests/evals/run.sh --check && bash scripts/validate.sh` → all pass.
- [ ] Commit: `feat!: the session-start hook prints the soul and recommends nothing`

### Task 3: ship checks its own tools and hands nothing over

**Files:**
- Modify: `skills/ship/SKILL.md`, `tests/evals/ship/expect.md`, `tests/evals/ship/repo/README.md` (or the fixture's `allow` file), `skills/builder/SKILL.md` line 116

- [ ] Fixture first. Add one bullet to `tests/evals/ship/expect.md` (it has five; six is the limit): "Before any other step it checks that `gh` is present and signed in and that commit signing is on, and it names the one command that fixes the check that fails." State in the fixture's README that the eval runner blocks `gh` through the PATH, so the check fails in every run. Run: `bash tests/evals/run.sh --check` → "fixtures ok".
- [ ] `skills/ship/SKILL.md`, new first step of the procedure: run `gh --version`, `gh auth status` and `git config --get commit.gpgsign` as shell commands. On the first one that fails, stop, name that check, and name its fix (install `gh`; `gh auth login`; the signing setup of the repository's CONTRIBUTING file). GitHub only. Add one row to Common mistakes and one line to Self-check.
- [ ] Delete `canon` from `skills/ship/SKILL.md` (lines 11, 85, 86, 101, 117): the look back lists what broke and what the person corrected, in the report. It hands nothing over. Delete the `canon` pointer from `skills/builder/SKILL.md` line 116.
- [ ] Run: `wc -l skills/ship/SKILL.md skills/builder/SKILL.md` (each 60 to 150), `bash scripts/validate.sh` → pass.
- [ ] Commit: `feat(ship): ship checks the forge tool and commit signing itself`

### Task 4: reach and budget go, and every send needs the person's yes

**Files:**
- Delete: `skills/reach/`, `skills/budget/`, `tests/evals/reach/`, `tests/evals/budget/`, `internal/reach/`, `internal/budget/`, `cmd/builder/reach.go`, `cmd/builder/budget.go` and their tests
- Modify: `skills/builder/SKILL.md` (lines 62 and 145), `skills/builder/harnesses/*.md`, `tests/templates_test.go`, `scripts/validate.sh`, `tests/test_validate.sh`, `tests/evals/results/*.json`, `README.md`, `AGENTS.md`, `SECURITY.md`, `.github/ISSUE_TEMPLATE/bug_report.md`, `package.json` (keyword `usage-budget`), `.opencode/plugins/builder.js` if it names a removed row

- [ ] `tests/templates_test.go`: change the assertion on "a send rule or the person's yes" to "the person's yes" and add `TestNoSendRuleInTheBuilderSkill` (the string `send rule` is absent from `skills/builder/SKILL.md` and `soul-compact.md`). Delete the tests that read `skills/reach` or `skills/budget`. Run: `go test ./tests/` → FAIL.
- [ ] `skills/builder/SKILL.md` lines 62 and 145: "without the person's yes in this session". Stay at or under 150 lines.
- [ ] Harness files: delete the rows "send the person a notification", "schedule a recurring check" and "see usage limits" from all four. Update the check in `scripts/validate.sh` and the opencode adapter if they list the rows.
- [ ] `scripts/validate.sh` lines 164 to 172: the smoke test becomes `"$BUILDER_BIN" security score <a one-finding file in a temp dir>` and asserts the printed `rating`. Name it "builder security smoke".
- [ ] `tests/test_validate.sh`: every case that uses `budget`, `intake`, `harness`, `reach` or `fleet` as its target skill uses one of `shape`, `plan`, `build`, `review`, `ship` instead. Run it: `bash tests/test_validate.sh` → pass.
- [ ] Delete the listed files and the two fixtures' keys in `tests/evals/results/`. Delete each README, AGENTS, SECURITY and bug-template line that names them. Run `go run ./cmd/support-table render README.md`.
- [ ] Run: `go build ./... && go test ./... && bash scripts/validate.sh` → pass. `bin/builder` with no argument does not list `reach` or `budget`.
- [ ] Commit: `feat!: reach and budget leave builder`

### Task 5: fleet, canon, intake and harness go

**Files:**
- Delete: `skills/{fleet,canon,intake,harness}/`, `tests/evals/{fleet,canon,intake,harness}/`, `internal/mcpmerge/`, `cmd/builder/mcpmerge.go`, `tests/test_drift.sh`, `tests/test_state_paths.sh`
- Modify: `skills/builder/soul-compact.md` (line 31), `skills/builder/harnesses/*.md` (row "message a peer session"), `internal/harness/harness.go` and its test, `tests/templates_test.go`, `scripts/validate.sh` (lines 175 to 180), `.github/workflows/*.yml` if they run a deleted test, `tests/evals/results/*.json`, `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, the four manifests (description and keywords only; keyword `fleet`), `cmd/builder/main.go` (the package comment)

- [ ] `tests/templates_test.go`: delete `TestRuleTemplateMatchesCanonSkill` and every test that reads a deleted skill. In `TestSoulListsTheLinesInOrder` or a new `TestSoulNamesNoRemovedSkill`, assert that the soul names none of the six. Run: `go test ./tests/` → FAIL.
- [ ] Delete soul line 31 ("One role per session…"). Delete the harness row "message a peer session".
- [ ] `internal/harness`: delete `MCP`, `ProjectMCP`, `SkillsDirs`, `Settings`, `ProcessMatch`, `PluginRootVar`, `AllProcessMatch` when no non-test caller is left (`go vet` and a search prove it). Keep `Detect`, `Names` and `BuilderHome`.
- [ ] Delete the listed files and results keys. `AGENTS.md` says "seven skills" and loses each routing row of a deleted thing. Render the support table.
- [ ] Run: `ls skills` → seven folders. `bin/builder` with no argument → `usage: builder <hook|security|version> ...`. `go test ./... && bash scripts/validate.sh && bash tests/test_validate.sh` → pass.
- [ ] Commit: `feat!: fleet, canon, intake and harness leave builder`

### Task 6: A guard keeps the removed names out, and the changelog says what went

**Files:**
- Create: `tests/removed_test.go`
- Modify: `CHANGELOG.md` (`[Unreleased]`), `RELEASING.md` if it names a removed thing

- [ ] `tests/removed_test.go`, `TestNoFileTellsAnyoneToUseARemovedThing`: walk the tracked files outside `docs/`, `CHANGELOG.md` and this test. Fail on a match of ``\b(builder (ledger|reach|budget|mcp-merge)|skills/(budget|canon|fleet|harness|intake|reach)\b|`(budget|canon|fleet|intake|reach)` skill|send rule|recommendations\.json)``. Run: `go test ./tests/ -run Removed` → it passes, or it names a leftover line; fix each leftover.
- [ ] Prove the guard: add the line "use the `fleet` skill" to a scratch copy and see the test fail; do not commit the line.
- [ ] `CHANGELOG.md` under `[Unreleased]`, heading "Removed": the six skills, the commands `ledger`, `reach`, `budget` and `mcp-merge`, the recommendations and onboarding, the parity eval. One sentence: "Release 3.0.0 is the last release that has them." One sentence: "builder leaves the state files of an older version on your machine and reads none of them."
- [ ] Run: `go test ./... && bash scripts/validate.sh` → pass.
- [ ] Commit: `test: a guard keeps removed skills and commands out`

### Task 7: Prove it

The controller does this task with the owner. No subagent runs it.

- [ ] `bash tests/test_validate.sh` → pass. `go test ./...` → pass.
- [ ] Paid evals, in the clean config folder for Claude Code: `ship`, `builder` and `flow-gates` on Claude Code and Codex. Grade each. Compare with the grades of the framework pull request: no fixture meets fewer bullets. If one does, stop and ask the owner.
- [ ] Record the verdicts (`go run ./cmd/support-table record <verdict.json>`), render the table, validate, commit.
- [ ] The owner starts a new session with the binary installed, reads the hook output, and finds nothing to answer (done-when 6).
- [ ] Show the owner the diff summary and the pull request text. Push and open the pull request only on the owner's yes.
