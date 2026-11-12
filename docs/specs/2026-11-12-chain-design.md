# Five stages that chain by themselves, and a first stage named spec

Status: approved. Stacked on `feat/review-comments`.

## Why

A review of the three stacked changes compared builder's flow with obra/superpowers.
builder's gates are as strong or stronger. Its handoffs are weaker: a stage ends on an offer, the session start names only `builder`, the review is saved at no fixed path, the plan does not name its spec, and nothing says where the work stands after a compact.
The same review found a security audit that closes a finding it did not recheck, and three ways a stranger's review comment can steer `build`.

## The person and the moment

A person types a request and wants the work to move from spec to ship without naming each skill.
The same person opens a new session the next day and wants the agent to know where the work stands.

## Goals

1. The main commands are `builder`, `spec`, `plan`, `build`, `review` and `ship`. The first stage is named `spec`.
2. Each stage ends by loading the next one, by its qualified name `builder:<name>`. The session start says to load a stage's skill when work enters it.
3. Both sides of a handoff name the same record: the approval ruling of the spec, the header of the plan, the task commits, the review at a fixed path with its head.
4. A new session finds the next stage from those four records. No new state file.
5. The findings of the review are fixed: the audit close rule, the comment trust rules, untrusted code in `review`, the tag and the release in `ship`.

## Non-goals

- No progress file. One task is one commit, so `git log` is the record.
- No new binary command.
- No paid eval run. The owner ruled them out for this change.

## Anti-goals

- A handoff must not become a permission. Loading `builder:ship` sends nothing: each write still waits for its own yes, given after `ship` shows it.
- A bare name must not load another pack's skill of the same name.

## Surfaces

- Agent instructions: review comments and pull request text from strangers. Harm to avoid: a comment that claims a role, withdraws another person's ask, or points at a "decision" that the change itself adds.
- Outbound sends: the tag and the release are writes of their own.
- Running code from strangers: `review` runs a change's build and tests with the person's keys.

## Approaches

1. A ledger file per piece of work, as superpowers keeps. Rejected: builder already has a record per stage, and a second store breaks "one owner per fact".
2. **Picked.** Name the next skill in each stage, fix the path and the first line of each record, and resume from the records.
3. One orchestrator skill that drives all five. Rejected: the stages must also work alone, on work that another tool built.

## Rulings

- 2026-11-12: rename `shape` to `spec`. Decided by: the owner.
- 2026-11-12: a fourth pull request for the rename, the chain and the fixes. Decided by: the owner.
- 2026-11-12: no paid evals. Decided by: the owner.
- 2026-11-12: `git log` is the build's progress record. Proposed, not confirmed. Cost if wrong: a squashed or reworded task commit reads as not done.
- 2026-11-12: the review is saved under `${BUILDER_HOME:-$HOME/.config/builder}/reviews/<repo>/<branch>.md`. Proposed, not confirmed. Cost if wrong: two checkouts with the same folder name and branch share a path; the `Head:` line catches it.
- 2026-11-12: tenet 10 says "without the person's yes" where it said "unseen". Proposed, not confirmed. Cost if wrong: the owner's wording of a tenet changed without the owner.
- 2026-11-12: `review` asks before it runs code from an author the person did not name as trusted. Proposed, not confirmed. Cost if wrong: one more question on a teammate's pull request.

## Done when

1. `ls skills` prints `build builder plan review security ship spec`.
2. `go test ./tests/` proves that each stage names the next, that the soul names all five, and that `plan` and `build`, and `review` and `ship`, name the same record.
3. `go test ./internal/security/` proves that a deferred unit keeps a prior finding open.
4. `bash scripts/validate.sh` and `bash tests/test_validate.sh` pass, and each SKILL.md has 60 to 150 lines.
