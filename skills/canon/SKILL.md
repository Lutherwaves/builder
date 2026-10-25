---
name: canon
description: Use when something broke in production or in review and the person says "never again", "again", or "this is the second time"; when the person corrects you the same way twice; when an incident, postmortem or revert lands; or when you start work in a repository that has no DESIGN, RULES or rules file and the agent files (CLAUDE.md, AGENTS.md) hold rules with no reasons.
---

# canon — a repository's rules carry their scars and their guards

A rule with no incident is taste. A rule with no guard is a wish.
`canon` grows the repository's own rules file. Each rule carries its scar (the incident that made it) and its guard (the check that fails when someone breaks it).
The builder skill's tenet 9 says why: "Rules are guards." This skill says how.

## What earns a rule

Write a rule only when one of these is true:

- **An incident.** Something broke for a person. You can name the date, the change and the harm.
- **A repeated correction.** The person corrected the same thing twice. Each correction is a scar. Name both.
- **A cost you can show.** A review found the same defect in two changes.

No scar, no new rule. Put a preference in a style guide or a linter config, not in the canon.

## The rule file: one home

1. Find the existing home: `RULES.md`, `DESIGN.md`, `docs/rules/`, an ADR folder, or a "Rules" section in `CONTRIBUTING.md`.
2. If there is none, create `RULES.md` at the repository root. Open it with one line: "Each rule names its incident and the check that enforces it."
3. Never start a second rules file next to an existing one. One fact has one owner.

Agent files (`CLAUDE.md`, `AGENTS.md`, `.cursor/rules`) hold one pointer line, not rule bodies:

```
- Rules, with the incident behind each and the check that enforces it: `RULES.md`. Read it before you change <the area>.
```

If the pointer line exists, do not add a second one.
If an agent file already holds rules, sort each line:

- **A rule with a scar.** Move it into the rules file with its incident.
- **A rule with no scar on record** (breaking it causes harm, but nobody wrote down when). Move it into the rules file with "Why: none recorded — ask the owner". Its Guard says what enforces it, or "none yet — add one".
- **A convention** (where files live, which command runs the tests). It stays in the agent file under a `## Conventions` heading.

List each move under Calls. Ask the owner for each missing scar under ▶ You.
Do not copy the rule into the README or the incident write-up. The incident write-up can link to the rule ID.

## Writing one rule

Copy the block in `rule-template.md`. Its lines, in order: Rule · Why · Guard · Supersedes. Fill them in this order:

1. **Why first.** Read the evidence: incident notes, the changelog, reverts, git log, the failing change itself.
   Quote the facts: dates, file names, what broke, for whom, for how long.
   If it happened twice, name both times. The second time is the reason the rule exists.
   Bad: "Caching is risky." Good: "2026-02-03: a change to `reports/cache.py` keyed the report cache by report ID only. For 40 minutes, one tenant saw another tenant's report."
2. **Rule.** One sentence that a reviewer can check against a diff.
   It names the mechanism, not the mood. Bad: "Be careful with caches." Good: "Every cache key includes the tenant ID."
   One sentence, no more. If the safe path takes several steps, write them in the guard's failure message. The person who breaks the rule reads them there, at the moment they need them.
3. **Guard.** The cheapest check that fails on the scar's own change. Prefer, in this order: a test in the suite, a CI or lint check, a check the release script runs.
   A review checklist line is not a guard. If nothing can enforce the rule yet, write "none yet — add one" and propose the check.
4. **Supersedes.** If this rule replaces another, link that rule's ID. Mark the old rule "superseded by <ID>" and keep it, because its scar is still history.

Give each rule a stable ID (`R1`, `R2`, …). The guard's failure message quotes the ID and the file.
Example: `R3 broken (RULES.md#r3): reports/cache.py builds a key with no tenant ID. See the 2026-02-03 incident.`

## Proving the guard

Break the rule on purpose. Something must fail.

1. Run the guard against the scar's own change. It must fail.
2. Run the suite on today's repository. It must pass.
3. If old, shipped changes break the rule, do not edit them. Pin them in the guard as known history, by name, with the incident. A new change still fails.
4. Quote both runs under ✓ Proven in the builder report.
5. Write the guard's limits in the rule's Guard line: what it does not catch, and where it does not run.
   Never claim "this can never happen again". Say what is now checked and what is not.

If the guard only protects production when CI runs it before deploy, say so. Changing CI settings on a hosting service is outside the repository. Draft the change and wait for a yes.

## Asking and acting

In an interactive session, show the rule sentence and the proposed guard. Ask one question with your recommendation: "I recommend this rule and this test. Add both?"
When nobody answers, write both, run the guard, and list the rule's wording and the file location under Calls.
End with the builder report from the builder skill.
Then offer the next step in one line: "Run this guard in the CI step before deploy, or write the next rule?"

## Common mistakes

| Mistake | Fix |
|---|---|
| The rule goes into AGENTS.md or CLAUDE.md as a paragraph | Put the rule in the rules file. The agent file gets one pointer line. |
| The same rule in three places: agent file, README, incident notes | One home. The others link to the rule ID. |
| "Be careful with X" | Name the mechanism a reviewer can check on a diff. |
| A Rule line of four sentences with a how-to | One sentence. The steps go in the guard's failure message. |
| Why says "this is risky" | Name each incident: date, change, harm. |
| A rule with no guard, and nobody says so | Write "none yet — add one" and propose the check. |
| The guard was never seen to fail | Run it on the scar's change. Quote the failure. |
| The guard fails on shipped history, so it gets switched off | Pin the shipped changes by name. New changes still fail. |
| "This can't happen again" | State what the guard checks and what it misses. |
| A rule for a preference with no scar | Use a linter or a style guide. The canon is for scars. |
| A scarless rule left in AGENTS.md, or deleted | Move it to the rules file with "Why: none recorded — ask the owner". List it under Calls. |
| Deleting a replaced rule | Mark it superseded and keep its scar. |

## Self-check

Before you report:

- [ ] There is one rules file, and the new rule is in it.
- [ ] The Rule line is one sentence that a reviewer can check on a diff.
- [ ] Why names each incident or correction with its date and its change.
- [ ] Guard names a test or check and its command, or says "none yet — add one".
- [ ] I ran the guard on the scar's change and saw it fail.
- [ ] I ran the suite on today's repository and saw it pass.
- [ ] The Guard line says what the check does not catch.
- [ ] Each agent file holds one pointer line and no rule body. Each rule that had no scar is in the rules file as "none recorded — ask the owner", and is under Calls.
- [ ] Supersedes is filled: a rule ID, or "none".
- [ ] Nothing outside the repository changed without a yes.
- [ ] The reply ends with the builder report, then a one-line offer of the next step.
