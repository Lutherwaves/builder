# Skill evals

A skill is written test-first. First run an agent without the skill and watch it fail. Then write the skill. Then run the agent with the skill.

## Fixture

One directory per skill: `tests/evals/<skill>/`.

- `prompt.md` is the task given to the agent.
- `expect.md` lists 3 to 6 bullet lines. Each bullet is a behaviour the with-skill run shows and the baseline run does not.
- `allow` is optional. Each line is a tool rule the agent may use without a prompt, e.g. `Bash(python3 -m unittest:*)`. Without it, the agent cannot run the fixture's tests. Blank lines and `#` lines are skipped.
- `repo/` is optional. It is the starting project. Without it, the agent starts in an empty git repository.

## Run

    bash tests/evals/run.sh --check          # validate fixtures, no API call
    bash tests/evals/run.sh <skill>          # run baseline and with-skill
    bash tests/evals/run.sh <skill> --dry-run  # print the two commands only

Each run uses a fresh temp copy of `repo/` as its working directory.

- Baseline: `claude -p` with your normal setup. The runner feeds `prompt.md` on stdin, so a prompt that starts with `-` is safe.
- With: the same command plus `--plugin-dir <repo root>`.

Both use `--output-format text`, `--permission-mode acceptEdits`, and `--disallowedTools "Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch"`, plus the commands that steer a session (`matchblox send`, `tmux send-keys` and the like) and the tools that schedule a job or register an MCP server (`CronCreate`, `CronDelete`, `RemoteTrigger`, `crontab`, `claude mcp`). The agent can edit files in the temp dir. It cannot push, reach the network, schedule a job or register a server through these tools. The flag names match `claude --help`. The runner does not use `--bare`, because `--bare` needs an API key and most people sign in with OAuth.

Both runs set `BUILDER_HOME` to a temp dir with an onboarded ledger in which every recommendation is answered "never". The hook then injects only the soul: no setup question and no recommendation reach the agent.

Because the baseline uses your normal setup, plugins you have installed also load in it. Disable plugins that overlap with the skill under test before you run.

## Drop-in parity

    bash tests/evals/parity.sh             # run the builder and shape fixtures in two setups
    bash tests/evals/parity.sh --dry-run   # print the builder-led settings and the commands only

This eval answers one question: does builder replace a hand-built setup and keep its results?

- Current: `claude -p` with your setup as it is.
- Builder-led: `CLAUDE_CONFIG_DIR` points to a separate folder that holds only what builder recommends, each one accepted.
  Builder loads with `--plugin-dir`. superpowers is enabled. `env` sets `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` to 85. The `codebase-memory-mcp` server comes from your `.claude.json`.
  The matchblox hooks and status line come from your settings. If your settings have none, the run has none, and a note tells you to run `matchblox setup` first.
  The folder holds a copy of your installed superpowers version and its marketplace, so both runs use the same version. It links your gstack clone and copies the skill folders that gstack setup made. It holds no other plugin or skill.

Both runs use the deny list and the PATH shims of `run.sh` (shared in `lib.sh`) and the same temp builder home.

Claude Code keeps its sign-in per config folder, and the eval never copies credentials.
A new folder is not signed in, so the eval stops before any run and says how to fix it:
set `BUILDER_PARITY_CONFIG_DIR` to a separate folder you keep, run `parity.sh` once to prepare it, run `CLAUDE_CONFIG_DIR=<that folder> claude auth login` once, then run `parity.sh` again.

The eval refuses (exit 2) a folder that is your real config dir, sits inside it, or holds it. It also refuses a folder that is not empty and that it did not create (no `.builder-parity` marker). It never deletes anything in such a folder.

The runner writes `<fixture>/current.md` and `<fixture>/builder.md` to `<eval out>/parity/<timestamp>/` and prints both after each `expect.md`.

## Transcripts

The runner writes `baseline.md` and `with.md` (stdout) and `baseline.err` and `with.err` (stderr) to `${BUILDER_EVAL_OUT:-${XDG_STATE_HOME:-$HOME/.local/state}/builder/evals}/<skill>/<timestamp>/`, your XDG state folder, `~/.local/state/builder/evals` by default. Set `BUILDER_EVAL_OUT` to write somewhere else. This is outside the repository. Never commit a transcript. The runner prints `expect.md` and both transcripts for you to compare.

A skill task is done when the with-skill run shows every `expect.md` bullet.
