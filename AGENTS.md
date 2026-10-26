# builder — agent guide

builder is a plugin for coding agents: ten skills, one session-start hook, and one Go binary, `builder`. The skills and the hook run it through `bin/builder`. builder has adapters for Claude Code, Codex CLI, Cursor CLI and opencode.
This guide routes an agent that works on builder itself.

## Where each change goes

| You change… | Edit | Then run |
| --- | --- | --- |
| A skill's behaviour | `skills/<name>/SKILL.md` and its eval fixture `tests/evals/<name>/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh <name>` |
| The soul the hook injects | `skills/builder/soul-compact.md` (keep it in step with `skills/builder/SKILL.md`) | `bash tests/test_hook.sh` |
| The SessionStart hook | `hooks/session-start`, `internal/hook` | `bash tests/test_hook.sh`, `go test ./internal/hook/` |
| A recommendation or its signal | `lib/recommendations.json`, `internal/signals`, `internal/ledger` | `go test ./internal/signals/ ./internal/ledger/` |
| The security score, CWE checks and audit baseline | `internal/security`, `skills/security/standards.md` | `go test ./internal/security/` |
| A send rule for `reach` | `internal/reach`, `skills/reach/rules.default.json` | `go test ./internal/reach/` |
| The budget forecast | `internal/budget` | `go test ./internal/budget/` |
| The graph drift check | `skills/harness/drift.sh` | `bash tests/test_drift.sh` |
| An issue or rule template | `skills/shape/issue-template.md`, `skills/canon/rule-template.md` | `go test ./tests/` |
| The parity eval | `tests/evals/parity.sh`, `tests/evals/lib.sh`, `cmd/parity-config` | `bash tests/evals/parity.sh --dry-run`, `go test ./cmd/parity-config/` |
| Which harness runs builder, and where it keeps its files | `internal/harness` | `go test ./internal/harness/` |
| What an action means on one harness (ask, schedule, usage, compact) | `skills/builder/harnesses/<harness>.md` | `bash scripts/validate.sh` |
| The manifests | `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `package.json` (description and keywords only; `version` stays equal in all four) | `bash scripts/validate.sh` |
| The opencode adapter | `.opencode/plugins/builder.js` | `node --test tests/test_opencode.mjs` |
| The support table in the README | `cmd/support-table` (never edit the block by hand) | `go test ./cmd/support-table/` |
| The shim that finds the binary | `bin/builder` | `bash tests/test_shim.sh` |
| The eval runner of one harness | `tests/evals/lib.sh` (`eval_base_<harness>`, `eval_prepare_<harness>`) | `bash tests/test_evals.sh`, then `bash tests/evals/run.sh --harness <harness> <skill>` |

Before every commit, run `bash scripts/validate.sh`. It runs all of the checks above except the paid eval runs.

## Conventions

- A skill opens with frontmatter: `name`, and a `description` that starts "Use when" and lists its triggers.
- A skill has 60 to 150 lines and the sections `## Common mistakes` and `## Self-check`.
- Write in ASD-STE100: active voice, present tense, one idea per sentence, short sentences.
- A skill change starts with an eval: run the fixture without the skill, watch it fail, then write the skill.
- The hook uses Bash and the `builder` binary only. It makes no network call and finishes in under one second.
- Nothing writes outside the local machine without a send rule or the person's yes.
- The repository is public. Keep employer, client, product and private repository names out of files, commits and pull requests.
- Never commit an eval transcript. Transcripts live outside the repository.
- Commits follow Conventional Commits and are signed. See [CONTRIBUTING.md](CONTRIBUTING.md).
- Never edit `CHANGELOG.md` or the `version` in a manifest. release-please owns both.

## Attribution

The `harness` skill derives its framing from Ryan Lopopolo's **Harness Engineering** corpus
(<https://github.com/lopopolo/harness-engineering>), licensed CC BY 4.0.
builder adapts its vocabulary and applies its improve-harness loop. It copies no prose.
