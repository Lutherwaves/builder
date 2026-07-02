# Contributing to builder

Thanks for your interest. `builder` is a Claude Code plugin of operational tools
for running AI coding agents — usage-limit projection, session/context load, and
token-efficiency watchers.

## Quick start

1. Fork + clone the repo.
2. Install the plugin locally for testing:
   ```bash
   /plugin marketplace add <your-fork-path-or-owner/repo>
   /plugin install builder@builder
   ```
3. Skills live under `skills/<skill-name>/SKILL.md` (Markdown + YAML frontmatter).
   Bundled tools sit beside the skill (e.g. `skills/watch-limits/burn-proj.py`).

## Before you open a PR

Run the validator — CI runs the same script and it gates merges:

```bash
bash scripts/validate.sh
```

It checks: manifests parse, skills have valid frontmatter, Python tools compile,
and `burn-proj.py`'s output shape is stable.

## Commit convention

We use [Conventional Commits](https://www.conventionalcommits.org/). The release
pipeline ([release-please](https://github.com/googleapis/release-please)) reads
commit/PR titles to decide version bumps and generate `CHANGELOG.md`.

- `feat: ...` — new feature (minor bump pre-1.0)
- `fix: ...` — bug fix (patch bump)
- `feat!: ...` or `BREAKING CHANGE:` in body — breaking change
- `docs: ...`, `refactor: ...`, `ci: ...`, `chore: ...` — no bump (docs/refactor/ci still appear in the changelog)

PRs are **squash-merged**, so the PR title becomes the commit that drives the release.

## Adding a new skill

1. Create `skills/<name>/SKILL.md` with frontmatter: `name` + a `description` that
   starts with "Use when…" and describes triggering conditions (not the workflow).
2. Keep bundled tools beside the skill; reference them via `${CLAUDE_PLUGIN_ROOT}`
   in setup steps so they resolve after install.
3. Run `bash scripts/validate.sh`.
4. Update the README **Skills** list.

## Pull requests

- One logical change per PR.
- PR title in conventional-commits format.
- No machine-specific paths, tokens, or personal identifiers.
- All PRs require the `validate` check to pass and go through review — direct
  pushes to `main` are blocked.

## Release process

Fully automated:
1. Merge PRs with conventional-commit titles.
2. release-please opens/updates a **release PR** with the next version + changelog.
3. Merging the release PR tags the version, cuts a GitHub release, and bumps
   `.claude-plugin/plugin.json`.

Maintainers: do not bump versions manually.
