# builder — agent guide

builder is a plugin for coding agents: eight skills, one session-start hook, and a small Python library. It has adapters for Claude Code, Codex CLI, Cursor CLI and opencode.
This guide routes an agent that works on builder itself.

## Where each change goes

| You change… | Edit | Then run |
| --- | --- | --- |
| A skill's behaviour | `skills/<name>/SKILL.md` and its eval fixture `tests/evals/<name>/` | `bash tests/evals/run.sh --check`, then `bash tests/evals/run.sh <name>` |
| The soul the hook injects | `skills/builder/soul-compact.md` (keep it in step with `skills/builder/SKILL.md`) | `bash tests/test_hook.sh` |
| The SessionStart hook | `hooks/session-start`, `hooks/session_start.py` | `bash tests/test_hook.sh` |
| A recommendation or its signal | `lib/recommendations.json`, `lib/signals.py`, `lib/ledger.py` | `python3 -m unittest tests.test_signals tests.test_ledger` |
| A send rule for `reach` | `lib/reach.py`, `skills/reach/rules.default.json` | `python3 -m unittest tests.test_reach` |
| The budget forecast | `skills/budget/burn-proj.py` | `python3 -m unittest tests.test_burn_proj` |
| The graph drift check | `skills/harness/drift.sh` | `bash tests/test_drift.sh` |
| An issue or rule template | `skills/shape/issue-template.md`, `skills/canon/rule-template.md` | `python3 -m unittest tests.test_templates` |
| The parity eval | `tests/evals/parity.sh`, `tests/evals/lib.sh` | `bash tests/evals/parity.sh --dry-run` |
| Which harness runs builder, and where it keeps its files | `lib/harness.py` | `python3 -m unittest tests.test_harness` |
| What an action means on one harness (ask, schedule, usage, compact) | `skills/builder/harnesses/<harness>.md` | `bash scripts/validate.sh` |
| The manifests | `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `package.json` (description and keywords only; `version` stays equal in all four) | `bash scripts/validate.sh` |
| The opencode adapter | `.opencode/plugins/builder.js` | `node --test tests/test_opencode.mjs` |
| The support table in the README | `scripts/support-table.py` (never edit the block by hand) | `python3 -m unittest tests.test_support_table` |
| The eval runner of one harness | `tests/evals/lib.sh` (`eval_base_<harness>`, `eval_prepare_<harness>`) | `bash tests/test_evals.sh`, then `bash tests/evals/run.sh --harness <harness> <skill>` |

Before every commit, run `bash scripts/validate.sh`. It runs all of the checks above except the paid eval runs.

## Conventions

- A skill opens with frontmatter: `name`, and a `description` that starts "Use when" and lists its triggers.
- A skill has 60 to 150 lines and the sections `## Common mistakes` and `## Self-check`.
- Write in ASD-STE100: active voice, present tense, one idea per sentence, short sentences.
- A skill change starts with an eval: run the fixture without the skill, watch it fail, then write the skill.
- The hook uses Bash and the Python 3 standard library only. It makes no network call and finishes in under one second.
- Nothing writes outside the local machine without a send rule or the person's yes.
- The repository is public. Keep employer, client, product and private repository names out of files, commits and pull requests.
- Never commit an eval transcript. Transcripts live outside the repository.
- Commits follow Conventional Commits and are signed. See [CONTRIBUTING.md](CONTRIBUTING.md).
- Never edit `CHANGELOG.md` or the `version` in a manifest. release-please owns both.

## Attribution

The `harness` skill derives its framing from Ryan Lopopolo's **Harness Engineering** corpus
(<https://github.com/lopopolo/harness-engineering>), licensed CC BY 4.0.
builder adapts its vocabulary and applies its improve-harness loop. It copies no prose.
