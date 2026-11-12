---
name: spec
description: Use when a request arrives as a feature name, a component or a ticket title ("we need CSV export", "add SSO", "build a dashboard", "take it from here"), when you write an issue, spec or brief for someone else to build, or when a request has no stated person, moment, non-goals or done-when.
---

# spec — work starts from the person and the moment

A request names a thing. The work is a change for one person at one moment.
`spec` is the first stage of the builder flow. It turns the request into one spec that a builder can act on with no follow-up question.
`spec` asks the questions, writes the spec and stops at the gate. `builder:plan` takes the approved spec.
The builder skill sets how to ask and what to do when nobody answers. Read its "Questions" and "When nobody answers" rules.

A small change skips `spec`. The size rule of the builder skill defines a small change, and says when it steps up to `spec`.

## Look before you ask

Read the README, the tickets, the support notes, the specs folder and the code that the request touches.
Never ask what they answer. Say what you found before you ask: "The rota lives in `rota.py`. The coordinator re-types it into the group chat each Sunday."

## Challenge the request

Before you write the spec, answer three questions from the evidence:
- Who wants this, by role?
- What do they do today instead?
- What is the narrowest version that helps them this week?
Ask the person only the part that the evidence does not answer.

## Ask

Ask one question at a time. Each question carries your recommendation and its reason.
After a question, stop and wait for the answer. Ask the next question in the next turn.
Never ask a question that the repository answers.

## Show the ways

Before you write the spec, show two or three ways to build the work.
For each way, give its trade-off in one line. Then give your pick and its reason.
The person picks. Record the pick under Rulings: the ways, the pick and who picked.

## The spec

Copy `issue-template.md`. Its sections, in order: Who, and the moment · Why it matters · The experience · Non-goals · Anti-goals · Surfaces · Rulings · Left to the builder · Calls to check · Done when.
Write the title after the moment. Fill the rest in this order, because each section feeds the next:

1. **Who, and the moment.** One person, by role. Their moment: when, where, what they hold, what they try to do. Name the evidence source.
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
   Examples: "the rota shows a shift that the volunteer declined"; "the page freezes for a big rota".
7. **Surfaces.** Take the list from `skills/security/standards.md`, section "Sensitive surfaces". Never copy the list into the spec.
   Name each surface that the work touches, with the harm to avoid. Each touched surface gets its own anti-goal.
   Example: a rota with phone numbers touches personal data. Its anti-goal: "phone numbers reach the group chat".
   When the work touches no surface, write "No sensitive surface touched. Checked: <the surfaces you checked>."
8. **Rulings.** Each decision already made: date, decision, reason, and who decided, by role.
9. **Left to the builder.** Decisions you hand over on purpose. This stops the spec from becoming a plan.
10. **Calls to check.** Each open question, with your recommendation, its reason and the cost if wrong.
    A question with no recommendation is a stall. Delete it or answer it.
11. **Done when.** Checks a person can run and see. "Each volunteer opens the chat and sees their own shifts, and no declined shift."
    "Tests pass" and "code reviewed" are not done-when lines. Every change needs those.

Delete a section that does not apply. Never write "n/a", "TBD" or "none".
Surfaces is the one exception: it is always present.

## Where it goes

1. Find where the repository keeps specs or issues: `docs/specs/`, `docs/rfcs/`, `docs/issues/`, an ADR folder, or a README line.
   Follow its file name rule.
2. If the repository has no such place, ask once, with your recommendation: "I recommend `docs/specs/YYYY-MM-DD-<slug>.md`. Keep it there?"
   If nobody can answer, use your recommendation and list it under Calls.
3. A ticket tracker is outside the repository. Write the draft to a local file. Post it only after the person says yes.

One screen is the target. Each line must change what a builder does.
A long spec that prescribes the build is a plan. Move that content to `plan`.

## The gate

The person approves the spec. Nothing is planned or built before that approval.
On approval, add one ruling to Rulings: `<YYYY-MM-DD>: spec approved. Decided by: the owner.` Name the approver by role, never by name. `plan` reads this line.
Write no plan, no task list and no product code while you write the spec.
Until the person approves, each turn ends with one question, or with the spec and a request for approval.

The "When nobody answers" rule of the builder skill says when a run is unattended, and what to do in doubt.
In an unattended run, do not wait. Take your recommendations, and pick the way to build it yourself.
The ruling still lists each way and its trade-off.
Mark each decision you made alone "proposed, not confirmed", and list it under Calls to check. Write no approval line.
Load `builder:plan` only when the request asked for the build. Otherwise the spec is the deliverable.

## After the spec

End with the builder report from the builder skill. Its first line names the spec by the person's new ability, and links it.
While the gate is open, the report's ▶ You item is the one open question or the approval. Ask nothing else.
Then ask for the approval in one line: "Approve it, and I plan it. Or change a ruling first?"
Next: when the person approves, add the approval ruling and load `builder:plan` in the same turn. Do not wait for the person to name it.

## Common mistakes

| Mistake | Fix |
|---|---|
| Asking where the code lives, or what the README already says | Read first. Say what you found. Ask only what the evidence does not answer. |
| Three questions in one message | Ask one question, with your recommendation. Stop. |
| A plan, a task list or code before the person approves the spec | Stop at the gate. `builder:plan` starts after the approval. |
| One way to build it, presented as the only way | Show two or three ways, with trade-offs and a pick. The person picks. |
| No Surfaces line, or "Surfaces: n/a" | Always write it. Name the touched surfaces, or the surfaces you checked. |
| Title names a component: "Rota export", "Add ExportService" | Title the outcome in the person's words: who can now do what. |
| "Users" as the person | One role at one moment. "A nurse at the start of a night shift", not "users". |
| Anti-goals copied from non-goals | A non-goal is scope. An anti-goal is a harm that can happen even when the code works. |
| The spec specifies columns, endpoints, libraries and test lists | Keep what the person sees. Move the rest to "Left to the builder" or to the plan. |
| Done-when says "tests pass" | Write a check the person can run and see. |
| A decision with no date, no reason or a real person's name | Each ruling has a date and a reason. The decider is a role, such as "the owner". |
| An approval in the chat only | Add the approval ruling to the spec. `plan` reads the spec, not the chat. |
| Posting the spec to a tracker to "finish" | Write the file. Post only after a yes. |

## Self-check

Before you hand the spec over:

- [ ] I said what the repository told me, and I asked nothing it answers.
- [ ] Each message asked at most one question, with my recommendation.
- [ ] I showed two or three ways with trade-offs and a pick, and the pick is a ruling.
- [ ] The title is an outcome in the person's words, specific enough that it does not fit any request in any product.
- [ ] "Who, and the moment" names one role and one moment, with the evidence source.
- [ ] Anti-goals name harms, not scope cuts.
- [ ] The Surfaces line is present. Each touched surface has an anti-goal.
- [ ] Every ruling has a date, a reason and who decided, by role. My own calls say "proposed, not confirmed".
- [ ] Every open question has a recommendation, a reason and a cost if wrong.
- [ ] No section is empty, "n/a" or "TBD".
- [ ] Each done-when line is a check a person can run and see.
- [ ] I wrote no plan, no task list and no code before the approval.
- [ ] An approved spec has the ruling `<YYYY-MM-DD>: spec approved. Decided by: the owner.` An unattended spec has none.
- [ ] Nothing went to a tracker without a yes.
- [ ] The reply ends with the builder report, then the one-line request for approval. On approval I load `builder:plan`.
