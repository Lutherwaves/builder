# Intake pass — prompt template

Drop this into a recurring job (`/loop`, cron, etc.). It assumes the `intake`
skill's contract: a source MCP (Todoist) is connected, `gh` is authed, `cctop`
is on PATH, and `~/.claude/intake/config.json` holds `default_repo`. The skill
(`builder:intake`) is the canonical spec — this prompt just drives one pass.

---

Run one **intake** pass. Read `~/.claude/intake/config.json` for `default_repo`.

1. **Grab.** List today's candidate tasks: Todoist Today view ∪ overdue
   (`find-tasks-by-date`). For each, capture title, description, comments, url.
   Read comments — the opt-in marker can be there.

2. **Reconcile.** For each task, resolve its repo (a repo/URL named in the task,
   else `default_repo`), then check signals cheapest-first and STOP at the first
   hit — the task is "in motion", skip it:
   a. Explicit `#123` or issue/PR URL in the text/comments.
   b. A live `cctop --json` session whose branch/project/prompt overlaps it.
   c. `gh pr list --repo <repo> --search "<keywords>"` plausible match.
   d. `gh issue list --repo <repo> --state all --search "<keywords>"` match.
   Matching is fuzzy — your judgment. Bias conservative: a weak match still
   counts as tracked. Only genuinely-unmatched tasks continue.

3. **Groom the raw ones.**
   - **Default → annotate only, NO GitHub write:** post a Todoist comment with a
     proposed issue title + one-paragraph body + suggested repo/labels; add label
     `intake:needs-grooming`.
   - **Create the issue ONLY if the task carries an opt-in marker** — a comment
     containing `gh issue` / `→gh`, or a description line `intake: create`. Then
     create the issue in the resolved repo, comment the issue URL back on the
     task, and set the label to `intake:filed`.
   - Skip tasks already labelled `intake:filed`. Don't re-post to
     `intake:needs-grooming` tasks unless the task changed.

4. **Report tight.** One scannable block: N candidates → M in-motion (with the
   signal that matched each) → K raw. For the raw ones, list what you annotated,
   and call out any you created an issue for (with URL). If you were about to
   create an issue but there was no marker, say so — do NOT create it.

Never create a GitHub issue without the explicit marker. Never flag a task raw
while any reconciler fired.
