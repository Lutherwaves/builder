# Skill evals

A skill is written test-first. First run an agent without the skill and watch it fail. Then write the skill. Then run the agent with the skill.

## Fixture

One directory per skill: `tests/evals/<skill>/`.

- `prompt.md` is the task given to the agent.
- `expect.md` lists 3 to 6 bullet lines. Each bullet is a behaviour the with-skill run shows and the baseline run does not.
- `repo/` is optional. It is the starting project. Without it, the agent starts in an empty git repository.

## Run

    bash tests/evals/run.sh --check          # validate fixtures, no API call
    bash tests/evals/run.sh <skill>          # run baseline and with-skill
    bash tests/evals/run.sh <skill> --dry-run  # print the two commands only

Each run uses a fresh temp copy of `repo/` as its working directory.

- Baseline: `claude -p "<prompt>"` with your normal setup.
- With: the same command plus `--plugin-dir <repo root>`.

Both use `--output-format text`, `--permission-mode acceptEdits`, and `--disallowedTools "Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch"`. The agent can edit files in the temp dir. It cannot push or reach the network through these tools. The flag names match `claude --help`. The runner does not use `--bare`, because `--bare` needs an API key and most people sign in with OAuth.

Because the baseline uses your normal setup, plugins you have installed also load in it. Disable plugins that overlap with the skill under test before you run.

## Transcripts

The runner writes `baseline.md` and `with.md` to `${BUILDER_EVAL_OUT:-$HOME/builder-design/evals}/<skill>/<timestamp>/`. This is outside the repository. Never commit a transcript. The runner prints `expect.md` and both transcripts for you to compare.

A skill task is done when the with-skill run shows every `expect.md` bullet.
