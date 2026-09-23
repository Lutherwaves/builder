---
name: intake
description: Use when you want to reconcile a task tracker against work-in-flight and groom the leftovers — pull today's captured tasks, drop anything already in motion (live session, open PR, existing issue), and turn the raw notes into groomed issues. Works with Todoist or a GitHub Project as the task source. Triggers: "grab and reconcile my tasks", "groom my todos", "intake", "which of my tasks aren't tracked yet", "reconcile todoist against github", "groom my project drafts".
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

Pick one with `source` in the config. Two ship:

- **`todoist`** (default): the Today view ∪ overdue. Use the Todoist MCP
  `find-tasks-by-date` for today, plus overdue. Read each task's comments too —
  the opt-in marker can live there.
- **`github-project`**: the **draft items** of one GitHub Project
  (`project.owner` + `project.number` in the config). A draft has no repo yet,
  which makes it GitHub's raw note-to-self: capture with
  `gh project item-create <n> --owner <owner> --title "…" [--body "…"]`.
  List drafts with GraphQL, which returns both ids in one call —
  `organization(login:)` (or `user(login:)`) → `projectV2(number:)` →
  `items { nodes { id content { ... on DraftIssue { id title body } } } }`,
  paginating `pageInfo`. The item `id` (`PVTI_…`) is what you convert; the
  content `id` (`DI_…`) is what you edit. Drafts have no comments, so the body
  is where markers and annotations live. Freshly created items can take a while
  to show in any listing; a draft missed this pass is picked up on the next.
- **Other sources:** implement the same contract with your own MCP (Linear, a
  notes app). The reconcile + groom core below never changes.

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

**Default → annotate (no issue created):** propose an issue *title*, a
one-paragraph *body*, and a suggested *repo + labels*. Stop there.

- `todoist`: post the proposal as a comment; add the label
  `intake:needs-grooming`.
- `github-project`: append the proposal to the draft body inside an
  `<!-- intake:needs-grooming -->` … `<!-- /intake -->` block with
  `gh project item-edit --id <DI_id> --title "<same title>" --body "…"` — the
  edit rejects a body-only call ("Title can't be blank"), so always resend the
  title. Keep the user's text above the block untouched.

**Escalate → create:** ONLY if the task carries an opt-in marker — a comment
containing `gh issue` (or `→gh`), **or** a description/body line
`intake: create`. Drafts can only use the body line.

- `todoist`: create the issue in the resolved repo, post the issue URL back as
  a Todoist comment, and swap the label to `intake:filed`.
- `github-project`: strip the `intake: create` line and any intake block from
  the body, then **convert the draft in place** with the GraphQL mutation
  `convertProjectV2DraftIssueItemToIssue(input: {itemId, repositoryId})` (the
  repo id comes from `gh repo view <repo> --json id`). Converting keeps the
  project item, so its fields (dates, status) carry over to the new issue.

**Idempotency:** the marker state is the memory. `todoist`: skip tasks labelled
`intake:filed`, and don't re-post to `intake:needs-grooming` tasks unless the
task changed. `github-project`: a converted draft is an issue and drops out of
the candidate list on its own; refresh an existing
`<!-- intake:needs-grooming -->` block only if the text above it changed.

## Setup

1. **Config:** copy `config.example.json` to `~/.claude/intake/config.json` and
   set `default_repo` (e.g. `blox-eng/blox`) and `source` (`todoist` or
   `github-project`; for the latter also `project.owner` + `project.number`).
   Tunable without touching the skill.
2. **Prerequisites:** `gh` authed (`gh auth status`) with access to the repos
   you reconcile against. `todoist` also needs the Todoist MCP connected;
   `github-project` needs the `project` scope (`gh auth refresh -h github.com
   -s project`). `cctop` on
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
| Building speculative source adapters | Two ship (Todoist, GitHub Project); document the contract for the rest |
| Converting a draft by creating a new issue and deleting the draft | Convert in place — a new issue loses the project item's dates and status |
