---
name: intake
description: Use when you want to reconcile a task tracker against work-in-flight and groom the leftovers — pull today's captured tasks, drop anything already in motion (live session, open PR, existing issue), and turn the raw notes into groomed issues. Triggers: "grab and reconcile my tasks", "groom my todos", "intake", "which of my tasks aren't tracked yet", "reconcile todoist against github".
---

# intake — grab & reconcile

## Overview

A recurring **intake** pass: pull the tasks you captured today, filter out
everything already being worked, and groom what's left. It patrols the seam
where a tracker owns *when* and an issue tracker owns *what* — your raw
notes-to-self are the tasks that live in the former but not yet the latter.

**Two hard guardrails — this skill exists to enforce them:**

1. **Never create an issue without an explicit per-task marker.** Default is
   annotate-only. A GitHub write happens ONLY when the task carries an opt-in
   marker (see Groom). No marker → no write, ever.
2. **Never flag a task as "raw" while any signal says it's in motion.** Fall
   through to grooming ONLY when every reconciler comes back empty. When in
   doubt, treat it as tracked and skip it — a missed groom is cheap, a
   duplicate issue is not.

## The four units

Behind one interface, cheapest check first, short-circuit on the first hit.

### 1. Source adapter — `list_candidates()`

Returns normalized tasks: `{id, title, description, comments[], url}`.

- **Todoist adapter (shipped):** the Today view ∪ overdue. Use the Todoist MCP
  `find-tasks-by-date` for today, plus overdue. Read each task's comments too —
  the opt-in marker can live there.
- **Other sources:** implement the same contract with your own MCP (Linear, GH
  Projects, a notes app). The reconcile + groom core below never changes. Only
  Todoist ships; the contract is the extension point.

### 2. Repo resolver

Per task, pick the GitHub repo to reconcile against:

- Task text/description names a repo (`owner/name`) or a GitHub URL → use it.
- Otherwise → the `default_repo` from `~/.claude/intake/config.json`.

### 3. Reconcilers — is this task already in motion?

Run in order; **any** hit means tracked → skip the task:

| Order | Signal | How |
|-------|--------|-----|
| 1 | Explicit ref | Task text/comments contain `#123` or an issue/PR URL |
| 2 | Live session | A `cctop --json` session whose branch/project/prompt keyword-overlaps the task |
| 3 | Open PR | `gh pr list --repo <repo> --search "<keywords>"` returns a plausible match |
| 4 | Open/recent issue | `gh issue list --repo <repo> --state all --search "<keywords>"` returns a plausible match |

Matching is **fuzzy and a judgment call — you decide, not a regex.** Bias
conservative: a weak keyword overlap on an open PR still counts as "in motion."
Only genuinely-unmatched tasks proceed to grooming.

### 4. Groomer — raw tasks only

**Default → annotate (no GitHub write):** post a Todoist comment with a proposed
issue *title*, a one-paragraph *body*, and a suggested *repo + labels*; add the
label `intake:needs-grooming`. Stop there.

**Escalate → create (GitHub write):** ONLY if the task carries an opt-in marker —
a comment containing `gh issue` (or `→gh`), **or** a description line
`intake: create`. Then: create the issue in the resolved repo, post the issue
URL back as a Todoist comment, and swap the label to `intake:filed`.

**Idempotency:** the `intake:*` label + the back-link comment are the memory. On
every run, skip any task already labelled `intake:filed`, and don't re-post a
grooming proposal to a task already labelled `intake:needs-grooming` (refresh it
only if the task changed).

## Setup

1. **Config:** copy `config.example.json` to `~/.claude/intake/config.json` and
   set `default_repo` (e.g. `blox-eng/blox`). Tunable without touching the skill.
2. **Prerequisites:** the source MCP connected (Todoist), and `gh` authed
   (`gh auth status`) with access to the repos you reconcile against. `cctop` on
   PATH enables the live-session reconciler; skip signal 2 if absent.
3. **Schedule (optional):** for a recurring pass, drop `intake-prompt.md` into a
   cron/`/loop` job (suggested 4×/day in your active window). Guard duplicates:
   list jobs first, replace — never stack a second intake loop.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Creating an issue because the task "clearly needs one" | No marker → annotate only. The marker is the user's consent. |
| Flagging a task raw on a fuzzy PR near-miss | Weak match still counts as in-motion. Skip it. |
| Re-grooming an already-labelled task | `intake:filed` → skip; `intake:needs-grooming` → refresh only if changed |
| Hardcoding a repo | Resolve per task; fall back to `default_repo` in config |
| Building speculative source adapters | Ship Todoist; document the contract for the rest |
