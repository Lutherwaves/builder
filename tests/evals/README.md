# Skill evals

A skill is written test-first. First run an agent without the skill and watch it fail. Then write the skill. Then run the agent with the skill.

## Fixture

One directory per skill: `tests/evals/<skill>/`.

- `prompt.md` is the task given to the agent.
- `expect.md` lists 3 to 6 bullet lines. Each bullet is a behaviour the with-skill run shows and the baseline run does not.
- `allow` is optional. Each line is a tool rule the agent may use without a prompt, e.g. `Bash(go test:*)`. Without it, the agent cannot run the fixture's tests. Blank lines and `#` lines are skipped.
- `results/` is not a fixture: it holds the graded counts per harness.
- `repo/` is optional. It is the starting project. Without it, the agent starts in an empty git repository.

## Run

    bash tests/evals/run.sh --check          # validate fixtures, no API call
    bash tests/evals/run.sh <skill>          # run baseline and with-skill
    bash tests/evals/run.sh <skill> --dry-run  # print the two commands only
    bash tests/evals/run.sh --harness codex <skill>   # same fixture on another harness

Each run uses a fresh temp copy of `repo/` as its working directory.

- Baseline: `claude -p` with your normal setup. The runner feeds `prompt.md` on stdin, so a prompt that starts with `-` is safe.
- With: the same command plus `--plugin-dir <repo root>`.

Both use `--output-format text`, `--permission-mode acceptEdits`, and `--disallowedTools "Bash(git push:*)" "Bash(gh:*)" "WebFetch" "WebSearch"`, plus the commands that steer a session (`matchblox send`, `tmux send-keys` and the like) and the tools that schedule a job or register an MCP server (`CronCreate`, `CronDelete`, `RemoteTrigger`, `crontab`, `claude mcp`). The agent can edit files in the temp dir. It cannot push, reach the network, schedule a job or register a server through these tools. The flag names match `claude --help`. The runner does not use `--bare`, because `--bare` needs an API key and most people sign in with OAuth.

Both runs set `BUILDER_HOME` to a temp dir, so no state of your own builder home reaches the agent.

Because the baseline uses your normal setup, plugins you have installed also load in it. Disable plugins that overlap with the skill under test before you run.

## One suite on every harness

`--harness claude|codex|cursor|opencode` picks the runner. The default is `claude`. Every harness runs the same fixtures and the same `expect.md` bullets, once without builder and once with it. The prompt goes in on stdin. Each run uses a fresh work dir and the temp builder home above.

| Harness | Command | Builder in the with run | Isolation |
| --- | --- | --- | --- |
| claude | `claude -p` | `--plugin-dir <repo root>` | deny list |
| codex | `codex exec -C . -s workspace-write -c approval_policy="never" --ephemeral -` | `codex plugin marketplace add <repo root>`, then `codex plugin add builder@builder`, in a temp `CODEX_HOME` | sandbox: writes only in the work dir, no network |
| cursor | `agent -p --force --trust --sandbox enabled` | `.cursor/hooks.json` and copies of the skill folders in `.agents/skills/<name>`, in the work dir (the manual install path; copies, so the agent never writes into your checkout) | `.cursor/cli.json` deny rules in the work dir, Cursor sandbox |
| opencode | `opencode run --auto` | `"plugin": ["<repo root>"]` in a temp `OPENCODE_CONFIG_DIR` | `permission` deny rules in the temp `opencode.json` |

`--dry-run` prints each command and the env it sets. It never prints a secret.
Codex needs your sign-in. The temp `CODEX_HOME` holds `auth.json` as a symlink to the real file. The runner never copies or prints it.
Cursor keeps its sign-in in its real config dir, so the runner uses that dir and writes nothing to it.
opencode keeps its sign-in in its data dir, not in the config dir, so the temp config dir does not hide it.

The `allow` file holds Claude Code tool rules. The other harnesses ignore it.

### Which guard is weaker

Claude Code denies tools by name (see above). The other harnesses cannot, or the rules are not proven here. So every non-Claude run also puts shims first on `PATH`: they block `gh`, `git push`, `matchblox` writes and `tmux`. A shim guards the program. It does not guard a call by absolute path.

- codex: no deny list for single commands. The sandbox is the guard. It stops network and writes outside the work dir. A shim blocks `gh` and `git push`. `git -c alias.p=push p` passes the `git` shim.
- cursor: the `cli.json` deny rules and the sandbox are best effort and not yet proven by a live run. The shims are the guard. Cursor keeps its chat history in its real config dir, and the runner does not remove it.
- opencode: the `permission` deny rules name `git push`, `gh`, `crontab`, `webfetch` and `websearch`. They are not yet proven by a live run. The shims back them up. opencode keeps its sessions in its data dir, and the runner does not remove them.
- All three: `CronCreate`, `RemoteTrigger` and `claude mcp` have no counterpart, so nothing blocks a harness-specific scheduler or MCP registration.

### Verdict and support table

After a run, the runner writes `verdict.json` next to the transcripts. It holds the harness, its version, the date, the fixture name and one entry per `expect.md` bullet: `{"text", "met": null, "quote": null}`. It holds no transcript and no path. The grader sets `met` to `true` or `false` and may add a `quote`.

    go run ./cmd/support-table record <verdict.json>   # keep counts in tests/evals/results/<harness>.json
    go run ./cmd/support-table render                  # print the README table
    go run ./cmd/support-table check README.md         # fail when the README table differs

`record` refuses a verdict with a `met` still `null`. It stores counts only. `render` shows a harness only when it has a results file. A harness is `supported` only when every fixture under `tests/evals/` is graded and meets every bullet. Otherwise it is `experimental: <failed fixture met/total>; <n> not graded`. With no results file, the table is the line "No harness has passed its evals yet." `scripts/validate.sh` runs `check`.

## Transcripts

The runner writes `baseline.md` and `with.md` (stdout) and `baseline.err` and `with.err` (stderr) to `${BUILDER_EVAL_OUT:-${XDG_STATE_HOME:-$HOME/.local/state}/builder/evals}/<skill>/<timestamp>/`, your XDG state folder, `~/.local/state/builder/evals` by default. Set `BUILDER_EVAL_OUT` to write somewhere else. This is outside the repository. Never commit a transcript. The runner prints `expect.md` and both transcripts for you to compare.

A skill task is done when the with-skill run shows every `expect.md` bullet.
