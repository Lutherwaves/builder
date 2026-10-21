---
name: shape
description: Use when a request arrives as a feature name, a component or a ticket title ("we need CSV export", "add SSO", "build a dashboard"), when you write an issue, spec or brief for someone else to build, or when a request has no stated person, moment, non-goals or done-when.
---

# shape — work starts from the person and the moment

A request names a thing. The work is a change for one person at one moment.
`shape` turns the request into one document that a builder can act on with no follow-up question.
It owns only the shape of that document. Other skills do the asking and the planning.

## Where shape sits

| Step | Skill | What it owns |
|---|---|---|
| Find the person, the moment, the why | superpowers `brainstorming` | The questions |
| Write the issue or spec | builder `shape` | The document: `issue-template.md` |
| Plan the build | superpowers `writing-plans` | Files, tasks, tests |

Do not repeat brainstorming's questions here. Do not write tasks, file names or function signatures here.
If brainstorming is not installed, ask its questions yourself: one at a time, each with your recommendation.
The builder skill sets when to ask and when to act. Read its "Questions" and "When nobody answers" rules.

## The document

Copy `issue-template.md`. Its sections, in order: Who, and the moment · Why it matters · The experience · Non-goals · Anti-goals · Rulings · Left to the builder · Calls to check · Done when.
Write the title after the moment. Fill the rest in this order, because each section feeds the next:

1. **Who, and the moment.** One person, by role. Their moment: when, where, what they hold, what they try to do.
   Find the evidence before you ask. Read the README, the tickets, the support notes, the code comments.
   Example: a ticket says the coordinator "re-types the rota into the group chat every Sunday night". That is the moment.
2. **Title.** The outcome, in the person's words: who can now do what.
   "A volunteer coordinator posts next week's rota in one step", not "rota export".
   Squint test: if the title fits any request in any product ("Improve export"), it is too vague. Rewrite it.
3. **Why it matters.** The cost today, in the person's units. Say "measured" or "estimated" for each number.
4. **The experience.** What the person sees and does, from their side of the screen.
   Write each way it goes wrong, and the words the person sees, with the fix.
   Formats, columns and endpoints are the builder's decisions. Put them in "Left to the builder", unless the person's work depends on them.
5. **Non-goals.** Things a builder could add and must not.
6. **Anti-goals.** Outcomes that make the work a failure even when every test passes.
   A non-goal limits scope. An anti-goal names a harm.
   Examples: "the rota shows a shift that the volunteer declined"; "phone numbers reach the group chat"; "the page freezes for a big rota".
7. **Rulings.** Each decision already made: date, decision, reason, and who decided.
   A decision you made alone says "proposed, not confirmed".
8. **Left to the builder.** Decisions you hand over on purpose. This stops the issue from becoming a plan.
9. **Calls to check.** Each open question, with your recommendation, its reason and the cost if wrong.
   A question with no recommendation is a stall. Delete it or answer it.
10. **Done when.** Checks a person can run and see. "Each volunteer opens the chat and sees their own shifts, and no declined shift."
    "Tests pass" and "code reviewed" are not done-when lines. Every change needs those.

Delete a section that does not apply. Never write "n/a", "TBD" or "none".

## Where it goes

1. Find where the repository keeps specs or issues: `docs/specs/`, `docs/rfcs/`, `docs/issues/`, an ADR folder, or a README line.
   Follow its file name rule.
2. If the repository has no such place, ask once, with your recommendation: "I recommend `docs/specs/YYYY-MM-DD-<slug>.md`. Keep it there?"
   If nobody can answer, use your recommendation and list it under Calls.
3. A ticket tracker is outside the repository. Write the draft to a local file. Post it only after the person says yes.

## How long

One screen is the target. Each line must change what a builder does.
Cut a line if the builder would do the same thing without it.
A long issue that prescribes the build is a plan. Move that content to `writing-plans`.

## When nobody answers

The person says they cannot answer, or the run is unattended.
Then do not wait. Fill every section from the evidence, mark each guess "proposed, not confirmed", and put it under Calls to check.
The issue is still the deliverable. Do not build unless the person asked for the build.

## After the document

End with the builder report from the builder skill. Its For line is the issue's title and moment.
Then offer the next step in one line: "Plan it with `writing-plans`, or change a ruling first?"

## Common mistakes

| Mistake | Fix |
|---|---|
| Title names a component: "Rota export", "Add ExportService" | Title the outcome in the person's words: who can now do what. |
| "Users" as the person | One role at one moment. "A nurse at the start of a night shift", not "users". |
| Anti-goals copied from non-goals | A non-goal is scope. An anti-goal is a harm that can happen even when the code works. |
| "Open questions" with no answers | Each question carries a recommendation, a reason and the cost if wrong. |
| Sections that say "n/a" or "TBD" | Delete the section. |
| The issue specifies columns, endpoints, libraries and test lists | Keep what the person sees. Move the rest to "Left to the builder" or to the plan. |
| Done-when says "tests pass" | Write a check the person can run and see. |
| Decisions written as facts with no date or owner | Each ruling has a date, a reason and who decided. |
| A real person's name in the document | Name the role. The person who decides is "the owner". |
| Posting the issue to a tracker to "finish" | Write the file. Post only after a yes. |

## Self-check

Before you hand the document over:

- [ ] The title is an outcome in the person's words, specific enough that it does not fit any request in any product.
- [ ] "Who, and the moment" names one role and one moment, with the evidence source.
- [ ] Anti-goals name harms, not scope cuts.
- [ ] Every open question has a recommendation, a reason and a cost if wrong.
- [ ] Every ruling has a date, a reason and who decided.
- [ ] No section is empty, "n/a" or "TBD".
- [ ] Each done-when line is a check a person can run and see.
- [ ] The file is where the repository keeps specs, or I asked once where to put it.
- [ ] Nothing went to a tracker without a yes.
- [ ] The reply ends with the builder report, then a one-line offer of the next step.
