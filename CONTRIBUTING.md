# Contributing to builder

Thanks for your interest. builder is a plugin for coding agents: nine skills, one session-start hook,
and a small Python library. It runs on Claude Code, Codex CLI, Cursor CLI and opencode. [AGENTS.md](AGENTS.md) tells you which file to change for each job.

## Quick start

1. Fork and clone the repository.
2. Load your checkout in a session (see below).
3. A skill lives in `skills/<name>/SKILL.md`. Its tools sit beside it, for example `skills/budget/burn-proj.py`.

## Load your checkout

Pick your harness. In each case, builder loads from your working copy.

- **Claude Code:** `claude --plugin-dir .`
- **Codex CLI:** `codex plugin marketplace add .`, then `codex plugin add builder@builder`.
  Codex copies the plugin at install. After each change, add it again.
  Trust the plugin's hook when Codex asks.
- **Cursor CLI:** use the manual project path from the README. Point `.cursor/hooks.json` at `hooks/session-start --harness cursor` in your checkout, and copy `skills/*` into `.agents/skills/` of a test project.
- **opencode (1.19 or newer):** put the path of your checkout in the `plugin` list of `opencode.json`.

## Add a harness

A new harness needs six things. Do them in this order.

1. **Manifest.** Add the harness's plugin manifest, with the same description and keywords as the other manifests. Keep `version` equal.
2. **Mapping file.** Add `skills/builder/harnesses/<harness>.md`. It holds the `<!-- hook -->` block and one row per action. A tool you did not verify starts with `not verified:` and names a safe fallback.
3. **Detection marker.** Add the harness to `lib/harness.py`: its state directory, its plugin-root variable and its process match. Add an environment marker only after you saw a real session set it. Extend `tests/test_harness.py`.
4. **Eval runner.** Add `eval_base_<harness>`, `eval_notes_<harness>` and `eval_prepare_<harness>` to `tests/evals/lib.sh`, so that `bash tests/evals/run.sh --harness <harness> <skill>` runs every fixture. The runner writes nothing outside its temp dirs.
5. **Results.** Run the fixtures and commit the result files that `tests/evals/README.md` describes. Never commit a transcript.
6. **Support table.** Run `python3 scripts/support-table.py render` and paste the block it prints into the README. Never write a support claim by hand. A harness shows as supported only after every fixture passes on it.

Then add its install block to the README, and its state directory to the scope in `SECURITY.md`.

## Before you open a pull request

Run the validator. CI runs the same script, and it gates every merge.

```bash
bash scripts/validate.sh
```

It checks that:

- the manifests and the release config parse;
- every commit subject since `origin/main` follows Conventional Commits;
- every skill has the skill shape (see below);
- the Python tools compile, and `burn-proj.py` keeps its output shape;
- the unit tests, the hook tests, the drift tests and the eval runner tests pass;
- every eval fixture is valid;
- no tracked file holds a deny-listed name, when `BUILDER_DENYLIST` is set (see "Scrub guard").

## Skill conventions

- Frontmatter holds `name` and a `description`. The description starts "Use when" and lists triggers, not the workflow.
- A skill has 60 to 150 lines.
- A skill has the sections `## Common mistakes` and `## Self-check`.
- Write in ASD-STE100: active voice, present tense, one idea per sentence, short sentences.
- After an explanation, the skill offers other formats in one line.
- Refer to a bundled tool through `${CLAUDE_PLUGIN_ROOT}`, so the path works after install. The hook gives the agent that path on every harness.
- Add the skill to the table in the README.

## Evals

A skill is written test-first. Each skill has a fixture in `tests/evals/<name>/`:
`prompt.md` holds the task, and `expect.md` holds 3 to 6 behaviours that only the with-skill run shows.

```bash
bash tests/evals/run.sh --check     # validate the fixtures, no API call
bash tests/evals/run.sh <name>      # run without the skill, then with it
```

A skill change is done when the with-skill run shows every `expect.md` bullet.
Transcripts go outside the repository. Never commit one.
See [tests/evals/README.md](tests/evals/README.md).

## Scrub guard

The repository is public. No file, commit or pull request names an employer, a client,
a product or a private repository.

Keep your own deny-list outside the repository: one name per line.
Then run the validator with it:

```bash
BUILDER_DENYLIST=/path/outside/the/repo/denylist.txt bash scripts/validate.sh
```

The scrub prints each tracked file that holds a listed name, and the check fails.
Never copy the deny-list, or any name in it, into the repository.

## Commits

Every commit follows two rules.

1. **Conventional Commits.** The subject is `type(scope)?!?: outcome`, and the outcome says what a person can now do.
   Example: `feat: a new builder is walked through setup one step at a time`.
   `scripts/check-subjects.sh` enforces this rule in `validate.sh` and in the CI job, for every commit and the pull request title.
2. **Signed commits.** Sign every commit. Never use `--no-verify`. Never make a WIP commit.

The release pipeline ([release-please](https://github.com/googleapis/release-please)) reads the subjects:

- `feat: ...` adds a feature (a minor bump).
- `fix: ...` fixes a bug (a patch bump).
- `feat!: ...`, or `BREAKING CHANGE:` in the body, breaks a contract (a major bump).
- `docs:`, `refactor:`, `ci:` and `chore:` cause no bump.

## Pull requests

- One logical change per pull request.
- The title is a Conventional Commit. Pull requests are squash-merged, so the title becomes the commit.
- No machine-specific paths, tokens or personal names.
- Every pull request passes the `validate` check and a review. Nobody pushes to `main` directly.

## Release process

The release is automated.

1. Merge pull requests with Conventional Commit titles.
2. release-please opens a release pull request with the next version and the changelog.
3. Merging that pull request tags the version, cuts a GitHub release, and bumps the manifest versions.

Never edit `CHANGELOG.md` or the `version` in a manifest by hand.
