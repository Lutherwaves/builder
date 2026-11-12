# The two lenses of a plan review

Credit: these lenses follow the plan reviews of gstack (MIT) and the plan format of superpowers (MIT). No text is copied.

`plan` loads this file in step 4. Run the engineering lens on every plan.
Run the design lens when a person sees or types anything: a screen, a message, a file, a command or an error.
Write each finding under "Lens findings" in the plan. Each finding changes the plan, or says "no change, because <reason>".

## Engineering lens

### Scope

The founder's question: is this the right work, and what do we cut?

- For each part of the plan, name what already solves it: a function, a module, a library the project uses, the platform.
  Reuse it, or write why it does not fit.
- State the minimum that meets every done-when line. Cut each task that serves no spec line.
- Count the files and the new parts: modules, classes, tables, endpoints, dependencies.
  More than eight files or two new parts is a smell. Ask for fewer, and write what you cut.
- A new dependency needs a reason that the existing code cannot meet.

### Architecture

- Draw the boundaries: which module owns which data, and which calls which. Keep one owner per piece of data.
- Trace the data flow of each new path: input, each step, output. Name where the data is checked and where it leaves the system.
- For each new path, name one failure that happens in production, not in a test.
  Examples: the store returns zero rows; a field holds a comma, a quote or a newline; the caller is signed out; a second shop's id comes in the request.
- For each failure, name the task that handles it and the test that proves it. A failure with no task is a gap.

### Tests

- Trace every new path, and every way the person meets it: the happy path, each error, each empty state, each hostile input.
- Write a test map, one line per path:

  ```
  export of one month, refunds marked      — tested by Task 2
  customer name starts with "="            — tested by Task 4
  request for another shop's month         — gap
  ```

- Close each gap with a task, or with a line in an existing task. Then mark it "tested by <task>".
- A test proves behaviour the person sees. A test that only checks a mock is not on the map.

### Regression

- Name each existing behaviour that the plan puts at risk: a function it changes, a shared helper, an output format, a total.
- Name the test that keeps each one. If no test exists, add a task that writes it first, before the change.

### Performance

- For each new path, state the work per request or per loop, and its scale: "one pass over the month's orders, up to 5,000 rows".
- When the scale is not known, write "unknown". Name the cheapest check that finds it.
- A query or a read inside a loop over rows is a finding. Move it out, or write why the scale makes it safe.

## Design lens

The designer's question: what does the person see at each step and at each error?

### Every state

List each screen, message, file and command output that the person meets. For each one, write its states:

- **Empty:** nothing to show. What the person sees, and what they can do next.
- **Loading:** the work takes time. What shows, and for how long.
- **Error:** each way it fails. The words, and the fix the person can make.
- **Success:** the result, and how the person knows it worked.

A state that the plan does not cover is a finding. Add it to a task, with its test.

### Rate each one

Rate each screen or message from 0 to 10. Then write what makes it a 10.
Example: "Empty month: 6. It says 'No data'. A 10 names the month and says there is nothing to download."
A rating below 8 changes the plan, or says "no change, because <reason>".

### The words

- Each error message says what happened, in the person's words, and what to do next.
- Each error names no code, no stack trace and no internal id.
- Take the words from the spec's experience section when it gives them. Put them in the test assertion.

### One actor, one job

- Each screen serves one person, by role, doing one job.
- A screen that serves two roles, or asks for two jobs, is a finding. Split it, or cut the second job.
