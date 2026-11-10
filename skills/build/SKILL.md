---
name: build
description: Use when a plan exists and you write the code for it ("build it", "implement the plan", "go"), or when you make a small change with no plan.
---

# build — run the plan one task at a time, test first

A plan says what to build. `build` writes the code, one task at a time, and proves each task with a test.
`build` is the third stage of the builder flow. It takes the plan that the builder skill `plan` wrote.
It hands the change to the builder skill `review`.
The builder skill sets how to ask and what to do when nobody answers. Read its "Questions" and "When nobody answers" rules.

## The rules

- No code before its test. Write the test, run it and see it fail. Then write the code.
- One task at a time, in the plan's order. One finished task is one commit.
- The plan is the contract. When the plan is wrong, stop and change the plan. Never build around it.
- Show each result. Quote the failing run and the passing run, not "tests pass".
- Work only in the repository. A push, a pull request or a post waits for the person's yes.

## 1. Before the first task

1. Read the whole plan. Then read the spec that it names.
2. Find the test command in the plan. Run the full suite once. Quote the result. A suite that fails before you start is a finding: report it and stop.
3. Check that the working tree is clean. If it is not, stop. Name the changed files. Never stash, reset or discard them.
4. Check the branch. On the default branch, make a new branch for the plan, for example `build/orders-csv-export`.

## 2. One task at a time

For each task, in order:

1. Read the task. Check each file, function and command that it names. Open the file and find the name.
   A name that does not exist makes the plan wrong. Go to section 5.
2. Write the test of the task, as the plan gives it. Write no product code yet.
3. Run the test. See it fail for the right reason: the plan's "before" result, not a typo or an import error of your own. Quote the failing line.
4. Write the least code that makes the test pass. Add nothing that the task does not name.
5. Run the test, then the full suite. Quote the passing line.
6. Commit the task with the plan's Conventional Commit subject. One task, one commit. Never a WIP commit.

**A `Surface:` task.** A task with the line `Surface: <name>` touches a sensitive surface.
Write its hostile-input test first, before any other test or code of the task. Run it and see it fail.
Only then write the other tests and the code. Never weaken the hostile input to make the test pass.

## 3. Fresh agents

Where your harness has the "start a subagent" action, start one fresh agent per task.
Give it the task text, the spec path and the test command. Give it no other context.
It follows section 2 and section 5, and it reports the failing run, the passing run and the commit.
Then start a second fresh agent. It checks the result against the task: the diff, the done-when line and a new test run.
A gap that the checker finds goes back to a builder agent before the next task starts.

Where your harness has no such action, build each task inline in this session. Say so in the report.

## 4. When a test fails

A test that fails after your code stops the run. Change no more code until you know why.

1. Reproduce the failure with one command. Quote it.
2. Find the root cause. Read the error, the code and the data it uses. State the cause in one sentence.
3. Fix the cause, never the symptom. Never skip a test, loosen an assertion or catch the error to hide it.
4. Run the test, then the full suite.

## 5. When the plan is wrong

A step can name a function that does not exist, a file in the wrong place, or a test that cannot fail. Then the plan is wrong.

1. Stop at that step. Write no code for it.
2. Name the step and what is wrong: "Task 2, step 2 calls `store.load_rows`. `store.py` has no such function."
3. Propose the change, with its reason: "Use `store.rows`. It returns the same rows."
4. Update the plan file: change the step and add a line under "Changes during build" with the reason.
5. In an interactive run, ask the person to approve the change. When nobody answers, take your proposal, go on, and list it under Calls.

Never invent the missing piece in silence. A new function is a plan change too.
A change that alters the spec's scope or a done-when line is not yours to take. Stop and hand it to the person.
Commit the plan change with the task it changes.

## 6. The small change

A small change has no plan. Small means: the request says everything that matters, and one task covers it.

1. Write one test that fails without the change. Run it and quote the failure.
2. Make the change. Run the test, then the full suite. Quote the result.
3. Commit with a Conventional Commit.

If the work grows past one task, stop. Say so, and step up to the builder skills `shape` and `plan`.

## 7. Done

1. Run the full suite after the last change. Quote its output: the command and the summary line. A run before the last change does not count.
2. Hand the change to the builder skill `review`, with the plan, the spec and the test command.
3. When the review returns a Critical or an Important finding, fix each one test-first, with the test that the review names. Then ask for the review again.
4. Do this at most twice. After the second round with an open Critical or Important finding, stop. Hand the review to the person.

End with the builder report from the builder skill.
In it, list each task with its commit and who built it: a fresh agent, or this session inline.
List each plan change under Calls.

## Common mistakes

| Mistake | Fix |
|---|---|
| The code first, the test after it | Write the test, run it, quote the failure. Then the code. |
| A test that fails for the wrong reason | Check the failure against the plan's "before" result. |
| "Tests pass" with no output | Quote the command and its summary line. |
| One commit for the whole plan | One commit per task, with the plan's subject. |
| A missing function written in silence | Stop. Name the step, propose the change, update the plan. |
| The hostile-input test written last | In a `Surface:` task, the hostile-input test comes first. |
| A skipped test or a loosened assertion to go green | Find the root cause. Fix the cause. |
| The suite run before the last fix, quoted as done | Run the full suite after the last change. |
| A third review round | Stop after two. Hand the review to the person. |
| No word on who built each task | Say "fresh agent" or "this session" for each task. |

## Self-check

Before you say the build is done:

- [ ] I read the plan and its spec. The tree was clean. I built on a branch that is not the default one.
- [ ] For each task, I quoted a failing run before the code and a passing run after it.
- [ ] Each `Surface:` task got its hostile-input test before its other code.
- [ ] Each finished task has one Conventional Commit.
- [ ] Each wrong step is named, its change is proposed, and the plan file holds the change.
- [ ] No failing test is skipped or loosened. Each fix names its root cause.
- [ ] The full suite ran after the last change. Its output is quoted.
- [ ] The change went to `review`. Each Critical and Important finding is fixed, or handed to the person after two rounds.
- [ ] The report says, per task, whether a fresh agent or this session built it.
- [ ] Nothing left the machine without the person's yes.
