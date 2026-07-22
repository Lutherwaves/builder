# builder — agent guide

builder is the applied side of **harness engineering**: the practice of
improving a fixed coding agent's output by shaping the two external levers it
depends on — **context** and **tools** — and curating the environment around
them. Each skill here is one intervention on those levers. This guide routes an
agent to the right skill and to the conventions for working on builder itself.

## Which skill for which job

Pick the skill whose lever matches the gap you observe:

| You need to… | Skill | Lever |
| --- | --- | --- |
| Let an agent answer "where is X / what calls Y / architecture" without grep-reading files | `code-graph` | Tools + just-in-time context |
| Adopt that graph-first workflow in a new repo end to end | `harness` | Both, as a loop |
| Know whether you'll blow your weekly usage limit, and which idle session to compact | `watch-limits` | Effectiveness / economics |
| Reconcile a task tracker against work already in flight and groom the rest | `intake` | Feedback into infrastructure |

## Working on builder

- Skills live in `skills/<name>/SKILL.md`; they are auto-discovered — no manifest
  to edit. Each must open with a `---` frontmatter block carrying `name:` and
  `description:` (see any existing skill).
- Run `bash scripts/validate.sh` before opening a PR. It checks manifest JSON,
  skill frontmatter, and Python compilation. CI runs it too.
- Keep skills self-contained and composable: a new application skill should call
  an existing capability skill, not re-implement it.
- Conventional Commits; release is automated via release-please.

## Doctrine and attribution

builder's framing derives from Ryan Lopopolo's **Harness Engineering** corpus
(<https://github.com/lopopolo/harness-engineering>), licensed CC BY 4.0. builder
adapts its vocabulary and applies its improve-harness loop; it copies no prose.
