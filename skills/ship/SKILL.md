---
name: ship
description: Use when reviewed work must reach its users ("ship it", "open the PR", "merge and release", "land this"), on work that any tool or person built.
---

# ship — take reviewed work to a checked release

A merge is not a release. A release that nobody checked is a hope.
`ship` takes a reviewed change to the place where the person meets it, and proves that it arrived.
It is the fifth stage of the builder flow. It takes the review that the builder skill `review` wrote.
After the release, the builder skill `canon` turns what broke into a rule.
The builder skill sets how to ask and what to do when nobody answers. Read its "Questions" and "When nobody answers" rules.

## The rules

- Nothing leaves the machine without the person's yes, or a standing rule that the person gave for that exact write. A push, a pull request, a merge, a tag and a release are each one write.
- "Ship it" is not a yes for each write. It starts the stage. Each write still waits for its own yes.
- Never force a push. Never retry a merge or a push by yourself. A refusal stops the stage.
- Run before you claim. Every "passes", "merged" or "live" has a command and its output behind it.
- Collect every reason to stop, then stop once. Run each local check in steps 1 and 2, also after the first stop. The person then fixes everything in one round.

## 1. Check the gate

1. Find the review of this change. Read its "Most important first" list, its findings and the heading `Open checks that block ship`.
   With no review, stop. Hand the change to `review` first.
2. An open Critical or Important finding stops the stage. Name each one with its fix.
3. An open check on a surface that the change touches stops the stage. It has no severity, but it blocks until the person answers it.
   Copy each check with its missing fact, who can observe it, and the exact words to reply, as the review gives them.
4. In an unattended run, list each stop as a ▶ You item: "Reply `<exact words>` to <what the answer allows>."
5. A Minor finding does not stop the stage. List it in the pull request text.

## 2. Sync and test

1. Check that the working tree is clean. If it is not, stop. Name the changed files. Never stash, reset or discard them.
2. Bring the base branch up to date, then merge it into the branch. Never rebase a branch that is already pushed.
   A merge conflict stops the stage. Name each file in conflict. Do not resolve a conflict in code that the review did not see.
3. Run the full suite of the project after the merge, not only the tests of the change. Quote the summary line.
4. A test that fails after the sync stops the stage. Name the test, the line that fails, and the cause in the base branch's change.
   Name the fix. Do not write it here: new code goes back to `build`, and then to `review`.

## 3. Write it up

1. Write the changelog line under `## [Unreleased]`, or in the place where the repository keeps its changes.
   Write it by what the person can now do, not by what the code does. Example: "You can download your contacts as a CSV file." Not: "Add export_csv."
2. Follow the repository's own release rules first, for example in `CONTRIBUTING.md` or `RELEASING.md`. Never edit a version that a release bot owns.
3. Make commits as the "Commits" rules of the builder skill say: signed Conventional Commits, and in a public repository a grep of the staged diff for private names.
4. Write the pull request text: what the person can now do, the review's verdict, the suite's result after the sync, and each Minor finding.
5. When the stage stopped in step 1 or 2, still write the changelog line and the pull request text as drafts. Show them in the report. Commit nothing that the stop concerns.

## 4. Show, then send

1. Show the diff against the base branch, the changelog line and the pull request text.
2. Ask for each write on its own: the push, then the pull request, then the merge. Give the exact words to reply for each.
3. A standing rule counts only for the write that it names. A rule for a push is not a rule for a merge.
4. Before each write, check again that the branch, the remote and the target are the ones that the person approved.
5. After each write, quote its result: the pushed commit, the link to the pull request, the merge commit.

## 5. After the merge

1. Find where the person meets the change: the page, the package registry, the download, the app, the API.
2. Check the released thing there, as the person uses it. Use the "use a browser" action of your harness mapping for a page.
   Example: install the package version from the registry and run its new command.
3. Quote what you saw: the command or the page, and the result.
4. When you cannot reach it, write "release not checked where the person meets it" under ⚠ Not proven, with the reason.

## 6. When it fails

Each failure stops the stage and names the fix. Never retry by yourself.

- **The suite fails after the sync.** Name the test and its cause. The fix goes to `build`, then to `review`.
- **The remote refuses the push.** Quote the refusal. Name the cause: a protected branch, a moved remote, missing rights. Never force the push.
- **The release check fails.** Quote what the person sees. Name the revert or the fix, and ask before either runs.

## 7. Look back

1. List what broke during this ship: each stop, each failed check, each surprise.
2. List what the person corrected, in their words.
3. Hand each item that can happen again to `canon`. It turns the item into a rule with its scar and its guard.
4. Name the items that you hand over in the report, under Calls.

## Common mistakes

| Mistake | Fix |
|---|---|
| "Ship it" taken as a yes for the push, the pull request and the merge | Each write waits for its own yes. Ask for each one. |
| Stop at the first open check, with no sync and no test run | Run steps 1 and 2 in full. Report every reason to stop at once. |
| Only the tests of the change run after the sync | Run the full suite. A change in the base breaks tests the change never touched. |
| The failing test is fixed inside `ship` | Name the fix. New code goes to `build`, then `review`. |
| An open check on a touched surface is called "non-blocking" because it has no severity | It blocks `ship` until the person answers. Give the exact words to reply. |
| The changelog says "Add export_csv" | Write what the person can now do. |
| A refused push is forced, or a failed merge is retried | Quote the refusal. Name the fix. Stop. |
| "Merged, done" | Check the released thing where the person meets it, and quote it. |
| No look back | List what broke and what the person corrected. Hand each to `canon`. |

## Self-check

Before you report:

- [ ] I read the review. Each open Critical or Important finding, and each open check on a touched surface, is a stop in my report.
- [ ] Each open check has its missing fact, who can observe it, and the exact words to reply.
- [ ] I merged the base branch and ran the full suite after it. I quoted the result.
- [ ] Each failing test is named, with its cause and its fix. I did not write the fix here.
- [ ] I showed the changelog line, written by what the person can now do, and the pull request text.
- [ ] Each push, pull request and merge had its own yes or a standing rule for that exact write. With none, it is a ▶ You item.
- [ ] Nothing was forced. Nothing was retried by itself.
- [ ] After the merge, I checked the released thing where the person meets it and quoted it, or it is under ⚠ Not proven.
- [ ] The look back lists what broke and what the person corrected, and hands each one to `canon`.
