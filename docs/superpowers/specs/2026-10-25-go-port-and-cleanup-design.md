# builder 3: one Go binary, a README that sells, a clean repo

Date: 2026-10-25. Status: draft for approval.

## Intent

builder's code is about 1,300 lines of stdlib Python plus Bash glue. The owner wants
it in Go: one toolchain with matchblox (a static Go binary), a typed codebase, no
`python3` requirement, and a smaller runtime footprint. In the same pass the README
must say in thirty seconds what builder does that superpowers, gstack or a plain
`CLAUDE.md` does not, and the repo loses its leftovers.

Measured on the owner's machine (Linux x86-64, 2026-10-25): the session hook takes
0.10 s and 37 MB RSS; bare `python3 -c pass` takes 0.01 s and 10 MB. A Go hook is
estimated at under 0.02 s and under 10 MB. The win is real but small; the main
reasons are one language, types and no interpreter dependency.

## Two projects, in this order

1. **Cleanup** (small, one PR, lands first).
2. **Go port and release** (a project, one PR per milestone below).

The README is rewritten in project 1 and its install section is updated in project 2.

## Goals

- Project 1
  - The TUI leftovers are gone. First push tag `archive/tui` at `225ee37`. Then
    delete the local and remote branch `feat/tui` and the `builder-tui` worktree
    (owner's call, 2026-10-25). Main has no TUI code today.
  - Before this project starts, `feat/brief-bluf` (the new report format: Done, You,
    Not proven, Calls) gets a PR and merges. Project 1 builds on that. The README and
    the question format use the new report, not For/Non-goals/Calls/Evidence/Outside.
  - The team-channel step is off by default: `channel` in `lib/recommendations.json`
    gets `"onboarding": false` and no signal offers it. The `reach` skill asks for the
    channel the first time the person asks to message a teammate (its setup section
    already exists).
  - Questions follow a slim version of gstack's decision brief (below).
  - The README is rewritten (outline below).
  - No user-facing doc carries a hedge such as "only the local path was proven",
    "not verified", "this README does not claim". Support status lives in one place:
    the generated support table. A validate check greps README, CONTRIBUTING and
    `docs/` for these phrases and fails on a hit.
- Project 2
  - Every Python file in the repo is replaced by Go. No `python3` call remains in
    skills, hooks, the opencode plugin, scripts or tests.
  - `builder` ships like matchblox: `linux/darwin × amd64/arm64`, each with
    `.sha256`, a CycloneDX SBOM, and build provenance and SBOM attestations.
  - The plugin itself still installs from git through each harness's marketplace,
    exactly as today (the superpowers model).
  - Without the binary, the session still gets the tenets and a single install offer.

## Non-goals

- Rust. Go matches matchblox and cross-compiles in one line; nothing here needs more speed.
- Windows. builder supports macOS and Linux today; that does not change.
- Any change to what a skill does. This is a port, not a redesign.
- Any change to on-disk formats. The Go code reads and writes `ledger.json`, the reach
  log and rules, and the security baseline exactly as the Python does today. Existing
  users keep their answers.
- A website or `curl | sh` domain for builder. The installer ships inside the plugin.

## Anti-goals (a failure even if it works)

- The session hook ever makes a network call, or ever shows an error, when the binary
  is missing.
- A plugin version runs a binary of another version.
- A download that skips the checksum.
- A README that reads like a feature dump or that hedges.

## Question format (project 1)

Add to `skills/builder/SKILL.md` under **Questions**, replacing the current two lines.
Taken from gstack's decision brief, minus its D-numbers, completeness scores and logging:

- Ask through the harness's choice tool (`AskUserQuestion` on Claude Code; the mapping
  files name the tool per harness). Prose only when the tool is missing.
- One decision per question. It carries:
  - **What is at stake**: two or three plain sentences a newcomer follows, and what
    breaks if the pick is wrong.
  - **Recommendation**: the option and one reason. The recommended option comes first
    and its label ends with `(recommended)`.
  - **Each option**: at least one pro and one con, concrete.
  - **Net**: one line naming the real trade-off.
- A destructive or irreversible option needs the typed word, never "ok".
- In a non-interactive run, take the recommended option, never a destructive one, and
  list it under **Calls** in the report.

## README outline (project 1)

Plain, direct English in paragraphs. Short sentences, but not one sentence per line.
ASD-STE100 stays the rule for skills (agents read them); the README is for people.

1. **One paragraph**: builder makes a coding agent answer to a person. Every piece of
   work ends with a report: who it is for, what it is not, each call the agent made
   alone and why, the test run that proves it, and what left the machine.
2. **The report example** (kept, trimmed).
3. **Why builder**: a table, builder vs superpowers vs gstack vs a hand-written
   `CLAUDE.md`. Rows: names the person and the non-goals before building; records each
   solo decision with its reason; proof from the session, not a claim; outbound gate
   (send rules, duplicate check, 6 sends an hour); a rule carries its incident and a
   check; one lane per session across many sessions; usage-limit forecast. Then one
   line: builder routes to superpowers and gstack, so use them together.
4. **Install**: one block per harness, three lines each. The Cursor manual path and
   harness caveats move to `docs/install.md`.
5. **Skills**: the ten-row table, one short line each.
6. **Contributing** and **License**, as today.

Removed from the README: "What happens in your first session" (it becomes three
lines under Install), "Compare with your own setup" (moves to `tests/evals/README.md`)
and "Standalone use of the budget forecast" (moves to the `budget` skill folder).

## Go port (project 2)

### Layout

```
go.mod                      module github.com/Lutherwaves/builder
cmd/builder/                the shipped binary
cmd/support-table/          dev only, never released
internal/harness/  ledger/  signals/  reach/  security/  budget/  hook/  mcpmerge/
bin/builder                 committed Bash shim (below)
scripts/install.sh          the installer (below)
```

Standard library only, plus a TOML parser for Codex config (the Python already
handles TOML with a regex fallback; `github.com/BurntSushi/toml` or keep the regex).

### Command surface (one to one with today's Python)

| Today | Go |
| --- | --- |
| `hooks/session_start.py` | `builder hook --harness … --format …` |
| `lib/ledger.py answer/next/onboarded` | `builder ledger answer/next/onboarded` |
| `lib/reach.py check/log/notify/matchblox-notify/channel` | `builder reach …` (same flags) |
| `lib/security.py score/baseline save` | `builder security score/baseline save` |
| `skills/budget/burn-proj.py` | `builder budget` (same args and output line) |
| Python JSON merge in `skills/harness/setup.sh` | `builder mcp-merge` |
| `scripts/support-table.py` | `go run ./cmd/support-table render/check` |
| `lib/signals.py`, `lib/harness.py` | internal packages, no command |

Output text and exit codes stay byte-identical where a skill or test reads them.

### The shim and version pinning

`bin/builder` is a committed Bash script. Skills, the hook and the opencode plugin call
`"${PLUGIN_ROOT}/bin/builder" …`, never a bare `builder`, so PATH does not matter.

1. If `BUILDER_BIN` is set, exec it (contributors point it at `go build` output).
2. Read the plugin version from `.claude-plugin/plugin.json` (Bash only, no jq).
3. Exec `$BUILDER_HOME/bin/builder-<version>` if it exists. `BUILDER_HOME` defaults
   to `~/.config/builder`, as today.
4. Else print `builder <version> is not installed. Install it: bash "<root>/scripts/install.sh"`
   on stderr and exit 3.

Because the binary name carries the plugin's version, plugin and binary cannot drift.
A plugin update needs the matching binary, and the hook offers it (below).

### The installer

`scripts/install.sh [--version vX.Y.Z]`, default the plugin's version:

- Detects OS and arch; refuses anything outside the four targets with the reason.
- Downloads the binary and its `.sha256` from the GitHub release, checks the sum,
  refuses on mismatch.
- With `gh` signed in, runs `gh attestation verify` and refuses on failure; without
  `gh`, says the attestation was skipped and how to check it.
- Installs to `$BUILDER_HOME/bin/builder-<version>` (mode 0755), removes older
  versions, prints one line: what it installed and where.

### The hook without the binary

`hooks/session-start` (Bash) calls the shim. On exit 3 it prints `soul-compact.md`
plus one onboarding block: "builder's helper binary is not installed. It runs the
setup steps, send checks and forecasts. Install it now? yes, not now, or never." with
the install command. It makes no network call. The answer is stored by the binary
after install; before install, "not now" and "never" are written by the shim to
`$BUILDER_HOME/install.answer` so the offer respects the same 7-day and forever rules.
A skill that needs the binary and finds it missing stops and offers the install in one
line (owner's call, 2026-10-25: no manual fallback).

A "yes" holds across versions (owner's call, 2026-10-25). When the binary for the
current plugin version is missing and the stored answer is "yes", the hook tells the
agent to run the installer in its first reply and to report it under **Outside**. The
hook itself stays offline. "never" holds across versions too. A skill that then finds
no binary says once that the binary is off and how to turn it on.

Old binaries: the installer keeps the three newest versions and deletes the rest.
`BUILDER_HOME` is shared by all harnesses, and each harness can hold a different
plugin version, so deleting every older version would break the other harnesses.

### Release

Keep release-please for versions, changelog and manifest bumps (it works today).
Set it to create a **draft** release. Release-please runs with `GITHUB_TOKEN`, and a
release made with that token starts no other workflow. So the build, attest and
publish jobs (copied from matchblox's `publish.yml`) are chained inside `release.yml`
behind release-please's `release_created` output. No new secret is needed. They
build the four targets with `CGO_ENABLED=0 go build -trimpath
-ldflags "-s -w -X main.version=<tag>"`, write `.sha256` and SBOM, attest, upload,
then publish the release. Then a matrix job on Linux and macOS, amd64 and arm64, runs
`scripts/install.sh` against the published release and checks `builder version`.

### Tests and CI

- Each `tests/test_*.py` becomes Go tests in its package, keeping every case.
- The Bash tests stay. Their inline `python3` (timing, JSON checks) becomes `date +%s%N`,
  `jq` or a `builder` call. `tests/evals/lib.sh` and `parity.sh` likewise.
- `scripts/validate.sh`: `go vet`, `go test ./...`, `gofmt -l` (empty), the shim
  test, the hedge grep, and a check that no `python3` remains outside test fixtures
  (`tests/evals/*/repo/` are sample projects in Python and stay).
- CI sets up Go from `go.mod`'s version.
- The hook test keeps its under-1-second bound and adds a case with no binary.

### Milestones (one PR each, into `feat/go`)

Every marketplace entry points at the repo (`"source": "./"`), so a git install runs
main HEAD. Milestone 2 alone would leave main calling a binary that no release has.
So the three milestones merge into an integration branch `feat/go`. `feat/go` merges
to main once, and that merge is released at once as builder 3. After that, a change
that adds a binary command lands in the same release as the skill that calls it.
Until the release PR merges, main can still run the previous binary (accepted risk).

1. Go module, `internal/*` ported with tests, `builder` commands, the shim. Python
   still present; a test runs both and diffs output on the shared fixtures.
2. Skills, hook, opencode plugin and scripts switch to the shim. Python deleted.
3. Installer, publish workflow, hook install offer, README install section.

## Calls made without the owner (review these)

- One integration branch `feat/go`, released once as builder 3 (see Milestones).
- The installer keeps the three newest binaries.
- Publish jobs chained inside `release.yml`, so no new secret is needed.

- Go over Rust (owner agreed, 2026-10-25).
- Plugin from git, binary from releases (owner agreed, 2026-10-25).
- A missing binary stops a skill with an install offer (owner agreed, 2026-10-25).
- Keep release-please and add a matchblox-style publish job, instead of replacing
  release-please with matchblox's own release workflow. Reason: release-please already
  bumps four manifests; only the build-and-attest half is missing.
- Binary path is versioned under `BUILDER_HOME`, not on PATH. Reason: version pinning
  and no PATH dependency in hooks.
- README prose is plain English, not ASD-STE100. Reason: tenet 12 targets explanations
  agents act on; the README sells to people. Skills keep STE.

## Left to the builder

- Package boundaries inside `internal/` beyond the list above.
- Whether TOML parsing uses a library or keeps the regex fallback.
- Wording of the README, within the outline.
- Which leftover files to delete in project 1, provided nothing references them
  (a `rg` for the path returns nothing) and the PR lists each one.
