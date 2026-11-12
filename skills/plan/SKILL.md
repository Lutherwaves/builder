---
name: plan
description: Use when a spec or an issue is approved and the work has more than one task ("plan this", "break this down", "how do we build it?"), before any code.
---

# plan — tasks a builder with no context can run

A spec says what changes for whom. A plan says how to build it, one task at a time.
`plan` is the second stage of the builder flow. It takes the spec that `builder:spec` wrote and the person approved.
It hands the plan to `builder:build`. The builder of each task has the plan and the repository, and nothing else.
The builder skill sets how to ask and what to do when nobody answers. Read its "Questions" and "When nobody answers" rules.

A small change skips the plan. The size rule of the builder skill defines a small change.

## The rules

- Write no product code. The plan names the code. `build` writes it.
- Every step is concrete. "Add error handling", "write tests" and "handle edge cases" are not steps.
  Name the behaviour, the input and the test that proves it.
- Every code task ends in a passing test and one commit. The last task is the review.
- Every done-when line and every anti-goal of the spec has a task. The plan names the task.
- The plan is reviewed before the gate. A review that changes nothing still writes its reason.

## 1. Read the spec

1. Read the spec, then the code and the tests it touches. Say what you found: "`orders.month` already returns one shop's month."
2. Check that the spec is approved. Its Rulings section holds the line `<YYYY-MM-DD>: spec approved. Decided by: <role>.`, which `spec` adds on approval.
   A spec with no such line, or with an approval marked "proposed, not confirmed", is not approved. When the person approves it now, add the line first.
   If it is not approved, stop. Say so, and offer `builder:spec`.
   In an unattended run, go on only when the request asked for the build. Write `Spec approved: no` in the plan header and "spec not approved; planned anyway" under Calls.
3. Copy out each done-when line, each anti-goal and each line of Surfaces. Step 5 checks the plan against them.
4. Find the test command in the README, the CI file or the build file. Each task uses it.

## 2. The header and the file map

The plan opens with this header, word for word. `build` reads it and stops without it.

```
Spec: <path of the spec>
Spec approved: <YYYY-MM-DD> | no
Test command: <the command>
Plan approved: <YYYY-MM-DD> by <role> | no
Next: builder:build
```

The file map comes next, before any task.
List each file to create or change. Give each file one job, in one line.

```
- `export.py` (create) — turns one shop's month of orders into CSV text.
- `tests/test_export.py` (create) — the tests of `export.py`.
- `orders.py` (change) — no change to `month`; the export reads it.
```

A file with two jobs is two files, or one job is out of scope. Prefer the files that already exist.

## 3. The tasks

Number the tasks and give each a name: "Task 3: the file leaves out customer email".
Each task has these parts, in this order:

- **Files:** each file the task creates or changes, from the file map.
- **Test first:** the test name, its input and the assertion. Show the test code when it is short.
- **Run:** the exact command, and the result before the code (fails, and how) and after it (passes).
- **Do:** the change, in plain steps. Name the function, its inputs and what it returns.
- **Done when:** one line that a reader can check. "`test_no_email_column` passes."
- **Commit:** the Conventional Commit subject.

Order the tasks so that each one builds on a passing state. One task changes one behaviour.

**Surfaces.** The spec's Surfaces line names each sensitive surface (see `skills/security/standards.md`, section "Sensitive surfaces").
A task that touches a named surface carries the line `Surface: <name>`.
It also carries a hostile-input test for that surface's anti-goal. Use the input an attacker or a mistake sends.
Example: the anti-goal is "a customer name runs as a formula". The test exports a customer named `=HYPERLINK("http://x","y")` and asserts that the cell starts with `'`.

## 4. Review the plan

Load `lenses.md`. Run the engineering lens on the whole plan.
Then run the design lens, when a person sees or types anything.
Write each finding in the plan, under "Lens findings". Each finding changes the plan, or says "no change, because <reason>".
A change from a finding is a new task, a changed task or a cut. Name it: "added Task 4", "cut `utils.py`".

## 5. Check against the spec

Write a table at the end of the plan: each done-when line, each anti-goal, and the task that proves it, by name.

| Spec line | Task |
|---|---|
| The file has no customer email | Task 3: the file leaves out customer email |

A line with no task is a gap. Add the task, or move the line to Calls with the reason.

## 6. Where it goes

1. Find where the repository keeps plans: `docs/plans/`, a plans folder, or a README line. Follow its file name rule.
2. With no such place, recommend `docs/plans/YYYY-MM-DD-<slug>.md` and ask once.
   If nobody can answer, use the recommendation and list it under Calls.
3. A ticket tracker is outside the repository. Write a local file. Post it only after the person says yes.

## The gate

The last task of every plan is the review of the whole change.
It names `builder:review`, the spec and the test command.
The gate to `build`: every done-when line maps to a task, the last task is the review, and in an attended run the person approves the plan.
On approval, write the date and the role into the header line `Plan approved:`. `build` reads the plan, not the chat.
In an unattended run, leave it at `no`, and load `builder:build` only when the request asked for the build. Otherwise the plan is the deliverable.
Build nothing while you plan.

## After the plan

End with the builder report from the builder skill. Its first line names the plan by the person's new ability, and links it.
Then ask for the approval in one line: "Approve it, and I build it. Or change a task first?"
Next: when the person approves, fill in `Plan approved:` and load `builder:build` in the same turn. Do not wait for the person to name it.

## Common mistakes

| Mistake | Fix |
|---|---|
| Tasks before the header and the file map | Open with the header, then the file map. Each file has one job. |
| An approval in the chat only | Write it into `Plan approved:`. `build` reads the plan, not the chat. |
| "Add error handling" or "write tests" as a step | Name the behaviour, the input, the test and its assertion. |
| A task with no failing test, or no expected result | Write the test first, the command, and what it prints before and after. |
| A task that touches personal data with only happy-path tests | Add `Surface: <name>` and a hostile-input test for the anti-goal. |
| A review that lists findings and leaves the plan as it was | Each finding changes the plan or says "no change, because …". |
| The design lens skipped for a change the person sees | Run it on every screen, message, file and error the person meets. |
| A done-when line with no task | Map each line by task name. Close each gap with a task. |
| The plan ends with the last code task | The last task is the review. |
| A spec with no approval ruling treated as approved | Look for `<YYYY-MM-DD>: spec approved. Decided by: <role>.` under Rulings. Without it, the spec is not approved. |
| Code written "to check the plan works" | The plan names code. `build` writes it. |
| Asking where to put the plan in an unattended run | Use `docs/plans/YYYY-MM-DD-<slug>.md` and list it under Calls. |

## Self-check

Before you hand the plan over:

- [ ] The spec holds the approval ruling, or the header says `Spec approved: no`.
- [ ] The plan opens with the five header lines, then the file map. Each file has one job.
- [ ] Each task has files, a failing test, the command with its result before and after, a done-when line and a commit.
- [ ] No step is a placeholder. Each names a behaviour and its test.
- [ ] Each task that touches a spec surface has `Surface: <name>` and a hostile-input test for its anti-goal.
- [ ] I ran the engineering lens, and the design lens when a person sees anything.
- [ ] Each finding changed the plan or says "no change, because …".
- [ ] Each done-when line and each anti-goal maps to a task by name.
- [ ] The last task is the review. In an unattended run, `build` starts only when the request asked for the build.
- [ ] I wrote no product code.
- [ ] The reply ends with the builder report, then the one-line request for approval. On approval I fill in `Plan approved:` and load `builder:build`.
