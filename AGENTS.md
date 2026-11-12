# builder — agent guide

builder is a plugin for coding agents: eight skills, one session-start hook, and one Go binary, `builder`. The skills and the hook run it through `bin/builder`. builder has adapters for Claude Code, Codex CLI, Cursor CLI, Grok Build and opencode.
This guide routes an agent that works on builder itself.

## Where each change goes

| You change… | Edit | Then run |
| --- | --- | --- |
| A skill's behaviour | `skills/<name>/SKILL.md` and its eval fixture `tests/evals/<name>/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh <name>` |
| The plan stage | `skills/plan/SKILL.md` and its eval fixture `tests/evals/plan/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh plan` |
| The build stage | `skills/build/SKILL.md` and its eval fixture `tests/evals/build/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh build` |
| The ship stage | `skills/ship/SKILL.md` and its eval fixture `tests/evals/ship/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh ship` |
| The engineering and design lenses of a plan review | `skills/plan/lenses.md` | `bash tests/evals/run.sh plan` |
| The architecture lens of a plan review (API contract, boundaries, more than one customer, tools for agents) | `skills/plan/architecture.md` | `go test ./tests/ -run Architecture` |
| The bar a review finding must pass, and the last sweep | `skills/review/bar.md` | `bash tests/evals/run.sh review-noise` |
| How build answers review comments | `skills/build/comments.md` | `bash tests/evals/run.sh review-comments` |
| What build removes from its own diff, and what review lists as slop | `skills/build/deslop.md` | `go test ./tests/ -run Slop` |
| How a failure with no known cause is found and fixed | `skills/debug/SKILL.md` and its eval fixture `tests/evals/debug/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh debug` |
| The security pass of a review | `skills/review/security-pass.md` | `bash tests/evals/run.sh review-security` |
| A domain checklist of an audit (AI, cloud, client, desktop and mobile, protocol) | `skills/security/domains.md` | `bash tests/evals/run.sh security-audit` |
| How the stages chain (who loads whom, the plan header, the review path) | the handoff lines of each `skills/<stage>/SKILL.md` and `skills/builder/soul-compact.md` | `go test ./tests/ -run 'Stage|Soul|Handoff'` |
| The soul the hook injects | `skills/builder/soul-compact.md` (keep it in step with `skills/builder/SKILL.md`) | `bash tests/test_hook.sh` |
| The SessionStart hook | `hooks/session-start`, `internal/hook` | `bash tests/test_hook.sh`, `go test ./internal/hook/` |
| The security score, CWE checks and audit baseline | `internal/security`, `skills/security/standards.md` | `go test ./internal/security/` |
| The issue template | `skills/spec/issue-template.md` | `go test ./tests/` |
| Which harness runs builder, and where builder keeps its files | `internal/harness` | `go test ./internal/harness/` |
| What an action means on one harness (ask, subagent, browser, compact) | `skills/builder/harnesses/<harness>.md` | `bash scripts/validate.sh` |
| The manifests | `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `package.json` (description and keywords only; `version` stays equal in all four) | `bash scripts/validate.sh` |
| The opencode adapter | `.opencode/plugins/builder.js` | `node --test tests/test_opencode.mjs` |
| The support table in the README | `cmd/support-table` (never edit the block by hand) | `go test ./cmd/support-table/` |
| The shim that finds the binary | `bin/builder` | `bash tests/test_shim.sh` |
| The eval runner of one harness | `tests/evals/lib.sh` (`eval_base_<harness>`, `eval_prepare_<harness>`) | `bash tests/test_evals.sh`, then `bash tests/evals/run.sh --harness <harness> <skill>` |
| The release workflows and their scripts (see [RELEASING.md](RELEASING.md)) | `.github/workflows/release.yml`, `.github/workflows/publish.yml`, `.github/scripts/` | `bash tests/test_release.sh`, `actionlint .github/workflows/*.yml` |

Before every commit, run `bash scripts/validate.sh`. It runs all of the checks above except the paid eval runs.

## Conventions

- A skill opens with frontmatter: `name`, and a `description` that starts "Use when" and lists its triggers.
- A skill has 60 to 150 lines and the sections `## Common mistakes` and `## Self-check`.
- Write in ASD-STE100: active voice, present tense, one idea per sentence, short sentences.
- A skill change starts with an eval: run the fixture without the skill, watch it fail, then write the skill.
- The hook uses Bash and the `builder` binary only. It makes no network call and finishes in under one second.
- Nothing writes outside the local machine without the person's yes.
- The repository is public. Keep employer, client, product and private repository names out of files, commits and pull requests.
- Never commit an eval transcript. Transcripts live outside the repository.
- Commits follow Conventional Commits and are signed. See [CONTRIBUTING.md](CONTRIBUTING.md).
- Write your change under `## [Unreleased]` in `CHANGELOG.md`. Never edit a released section or a manifest `version`: the release bot owns both.

## Attribution

builder takes ideas, and no text, from four MIT-licensed projects:

- **obra/superpowers** (<https://github.com/obra/superpowers>): the spec, plan and build loop. `spec`, `plan` and `build` follow it.
- **garrytan/gstack** (<https://github.com/garrytan/gstack>): the reviews of a plan, the use of the running product, and shipping. `skills/plan/lenses.md`, `review` and `ship` follow it.
- **cloudflare/security-audit-skill** (<https://github.com/cloudflare/security-audit-skill>): the audit method. `security` and `skills/security/domains.md` follow it.
- **PostHog/posthog** (<https://github.com/PostHog/posthog>, MIT outside its `ee/` folder): the bar a review finding must pass, the last sweep, and the handling of review comments. `skills/review/bar.md` and `skills/build/comments.md` follow it.

Keep each credit when you change the file that carries it. Copy no prose from any of the four.
