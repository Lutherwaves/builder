---
name: review
description: Use when you are asked to review a pull request, a branch, a commit range, a patch or a diff ("review this PR", "is this ready to merge?", "look over my branch"), at any size, your own change or a teammate's.
---

# review — prove how a change hurts a person

A review that guesses is an opinion. A review that runs is evidence.
`review` is the method: find each way the change hurts a real person, prove it, and hand over the fix and the test that pins it.
superpowers `requesting-code-review` and gstack `/review` start a review flow. This skill is how the reviewer works inside it.

## The rules

- Prove the harm. Name the person, the situation and what they lose. A finding with no harm is a nit. Drop it.
- Run before you claim. Every "passes", "fails" or "leaks" has a command and its output behind it.
- Leave the checkout as it was. Apply the change and run probes only in a throwaway worktree or a temp dir.
  Example: `git worktree add <temp dir>/pr HEAD`, then `git apply` the patch there.
  If your tools reach only the working folder, add the worktree inside it (`git worktree add .review-wt HEAD`).
  Run probes as test files in the worktree, with the project's own test command. Remove the worktree at the end.
  A blocked command is not a reason to stop running. Try the in-folder worktree before you fall back to reading.
- Hunt only what no existing test catches. The suite already guards the rest.
- Post nothing. A comment on the forge, a push or an approval leaves the machine. It waits for the person's yes.

## 1. Before you read

1. Find the base and the head. Read the description, the linked issue and any spec or doc.
2. Write down every claim they make, one per line. You check each one in step 2.
3. In the throwaway location, run what the project runs: build, lint or vet, the full test suite.
   Run the race detector (or the language's equivalent) several times on the changed packages.
4. Record each command and its result. They open the review.

## 2. Hunt

Go through the diff by area. For each changed behaviour, ask these questions. Rank what you find, worst harm first.

1. **Real environments.** Where does it run that the tests never fake? Another OS or filesystem, a symlink or a network path, case-insensitive paths, a read-only folder, another user, a slow or dead dependency, a sleeping machine, real-world sizes.
2. **Harm.** What leaks, and to whom? What hangs, what is lost, what becomes permanent by mistake, what costs money or quota?
3. **Consent, credentials, irreversible actions.** Can a stray key, a retry, a race or a default trigger them?
   Can a secret reach a log, a file, a URL, an error message or another device? Can one identity use another's credentials?
4. **Success only on success.** Does "done", "saved", "sent" or "safe" ever show when the work failed, was partial, or never ran? Counts, status lines and hints count.
5. **The change's own claims.** Check the description, docs, comments and changelog against the code, sentence by sentence. Name each false claim.
6. **Seams.** Is a value cleaned in one place and compared raw in another: paths, case, trailing slashes, IDs, labels?
   Do keys match across layers (API, storage, UI, CLI)? Can old clients and old data read the new format?
7. **Concurrency.** Shared state, lock order, tasks that outlive their caller, two requests for the same thing, cleanup that removes another request's work.
8. **Hot paths.** Work per request, frame, keystroke or poll, at a realistic size.
9. **Timeouts and waits.** Add up budgets, grace periods and queued work against each caller's timeout.
10. **Silent skips.** Anything skipped or dropped must show where the person sees it.

When you suspect a bug, confirm it. Write a tiny probe or test in the throwaway location. Capture the input and the output.
Use the project's own examples as inputs: the README, the docs, the description. A bug that the README's own example triggers is a bug every reader hits.
Mark each finding with the exact words "confirmed by running" or "from reading". Never mark a guess as confirmed.

## 3. Write the review

Use these headings word for word, in this order: What I ran · Most important first · the findings · Claims checked · Declined to judge · Verdict.
The person and other agents scan for them. The verdict comes last, after the evidence, never at the top.

Open with two parts:

- **What I ran.** One short paragraph: each command and its result.
- **Most important first.** A numbered list, one line per finding, worst first.

Then one entry per finding:

```
**<One-line headline that states the harm to a person.>** `path/to/file:line` — <Critical | Important | Minor>
What goes wrong: <who, in which concrete situation; an example or a small input → output table>.
Evidence: confirmed by running: <command>, <input> → <output>. | from reading.
Fix: <the smallest change that removes the harm>.
Test: <test name> — asserts <what>.
```

The location is a real line number in the changed code (`app.py:58`), not a function name. Read it from the applied change.

Severity: **Critical** is data loss, a security hole, a leaked credential or a broken core path.
**Important** is wrong behaviour a person will hit, or code that breaks under normal conditions. **Minor** is an edge case, polish or cost.
Do not inflate a severity.

Close with three parts:

- **Claims checked.** Each claim from step 1: true, or false with the finding number.
- **Declined to judge.** Each thing you considered and set aside, one line each, with the reason.
- **Verdict.** Ready to merge, ready with fixes, or not yet. Add a one-line reason.

Then say what stayed local: "Posted nothing. The checkout is unchanged." Offer the next step in one line: "Post these as review comments, or turn the fixes into a patch?"
If you found nothing important, say so plainly. Do not pad the review.

## Common mistakes

| Mistake | Fix |
|---|---|
| "Tests pass, LGTM" | The suite guards only what it tests. Hunt what no test catches. |
| A finding with no person and no situation | Name who loses what, and when. Otherwise drop it. |
| "This could leak the token" with nothing run | Run a probe. Quote input → output, or mark it "from reading". |
| `git apply` in the person's checkout | Apply and probe in a throwaway worktree or temp dir. |
| The description is taken at its word | Check each claim against the code. A false claim is a finding. |
| A probe with an invented input | Use the README's and the description's own examples first. |
| Style nits and praise | Cut them. Report harm, its evidence and its fix. |
| A fix with no test | Name the test that fails today and passes after the fix. |
| Every finding marked Critical | Use the scale. Inflation hides the real Critical. |
| Posting the review or approving to "finish" | Write the review here. Post only after a yes. |

## Self-check

Before you hand the review over:

- [ ] It opens with what I ran and the results, then the most-important-first list.
- [ ] Each finding has a bold harm headline, `file:line` with a line number, a severity, evidence, a fix and a test.
- [ ] Each "confirmed by running" quotes a command, an input and an output.
- [ ] Each claim in the description is marked true or false.
- [ ] I ran nothing in the checkout. The throwaway worktree or temp dir is removed.
- [ ] No finding repeats what an existing test already covers.
- [ ] It ends with "Declined to judge", then the verdict with its reason. The verdict is not at the top.
- [ ] Nothing reached the forge without the person's yes.
