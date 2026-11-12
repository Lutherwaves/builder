# build — remove what the change did not need

An agent adds more than the task asks for: a comment that says what the code says, a guard for a case that cannot happen, a helper with one caller.
Each such line costs the next reader time and hides the lines that matter.
`build` runs this pass over its own diff before the last test run. `review` uses the same list on any change.

## The pass

1. Read the whole diff of the work: `git diff <base>...HEAD`. Read each added line, not the summary.
2. For each added line or block, ask one question: which task, done-when line or test needs it?
   When the answer is "none", it is slop. Remove it.
3. Remove, do not rewrite. This pass makes the diff smaller. It adds no feature and no new abstraction.
4. Touch only lines that this work added. Slop that was there before is not yours to remove here.
5. Run the full suite after the pass. Commit the pass as one commit: `refactor: remove what the change did not need`.
6. Say the result in the report, in one line. Example: "Deslop: removed 2 restating comments and 1 unused option (14 lines)." or "Deslop: nothing to remove."

## The list

In code:

- A comment that restates the code, or that tells the story of the change ("added for the new flow", "fixed the bug here").
  A comment stays when it says why, and the reason is not visible in the code.
- A guard for a case that the callers, the types or the validation rule out. Read the callers before you remove it.
- A `try` that catches an error and hides it, or that throws it again unchanged.
- A helper, a wrapper, a class or an interface with one caller, when the caller is as clear without it.
- An option, a flag, a parameter or a config key that no task names.
- Code that repeats a helper that the project already has. Use the helper.
- Dead code: an unused import, variable or function, code in a comment, a debug print, a leftover file.
- A change of names, order or layout in lines that the task did not need to touch.
- Style that differs from the code next to it.

In tests:

- A test that asserts nothing, or that only asserts what its own mock returns.
- A test that copies the logic of the code it tests.
- Two tests that prove the same thing.

In prose (docs, messages, commit text, the report):

- A sentence that repeats the heading or the sentence before it.
- Praise and filler: "comprehensive", "robust", "seamless", "simply", "it is worth noting".
- A hedge with no fact behind it: "might", "should work", "in most cases".
- A summary file, a notes file or a plan file that nobody asked for.

## What is never slop

- A test that the plan names, and the hostile-input test of a `Surface:` task.
- A guard on input from outside the program: a request, a file, an environment variable, another user.
- An error message that says the fix.
- A line that a reviewer or the person asked for.

When you cannot tell whether a line is needed, keep it. Name it under Calls, with the question.
Never remove a line to make the diff look small.
