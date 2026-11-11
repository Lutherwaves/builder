# builder framework — shape, plan, build, review and ship in one pack, with security in each stage

Issue: none, by the owner's ruling of 2026-11-10. Date: 2026-11-10. Status: draft, amended by the engineering review of 2026-11-10 (decisions D1 to D8 below). The owner has not yet approved the document as a whole.

## Intent

A person installs builder alone and gets one framework from a request to shipped work: shape → plan → build → review → ship.
Security is a step of each stage. It is not a separate pass that someone must remember.

Today builder routes the plan and the build to superpowers, and the browser, QA and shipping to gstack. The README says "Install all three".
The person must hold three packs in their head, and a security pass runs only when someone asks for it.
The owner's direction (2026-11-10): "builder simplified to have its own builder flow + the security side of things embedded in it".
And, on the first draft of this document: embed "the essence of gstack + superpowers + cloudflare security skill + existing builder mentality into a new framework for building things".

Three sources feed this work, next to builder's own thirteen tenets and its report. builder takes their ideas and copies no prose.

- obra/superpowers (MIT): the spec → plan → development loop.
- garrytan/gstack (MIT): the plan challenged from several roles, the product used as the person uses it, shipping as one workflow, the look back.
- cloudflare/security-audit-skill (MIT, commit c1c8a8c): the audit method.

## Goals

1. **One framework, five stages.** Each stage hands one artifact to the next, and each stage has one owner skill.

   | Stage | Skill | Takes | Hands over | Gate before the next stage |
   |---|---|---|---|---|
   | Shape | `shape` (changed) | A request | The spec (`issue-template.md`) | The person approves the spec |
   | Plan | `plan` (new) | The approved spec | The plan: tasks, files, tests | Every done-when line of the spec maps to a task |
   | Build | `build` (new) | The plan | Commits, one task at a time | Every test passes; the output is quoted |
   | Review | `review` (changed) | The change | The review, with its security pass and its use of the running product | No open Critical or Important finding |
   | Ship | `ship` (new) | The reviewed change | A pull request, then the merged and checked release | The person's yes before anything leaves the machine |

   After a ship, `canon` takes what broke or was corrected and turns it into a rule with its incident and its check.

2. **The flow has a size rule.** A small change skips the spec document and the plan.
   Small means: the request says everything that matters, and one task covers it.
   A small change still gets a failing test first, a review of its diff and the builder report.
   If the work grows, the agent stops, says so and steps up to the full flow.

3. **`shape` asks its own questions.** It no longer waits for another pack to ask them.
   - It reads the project before it asks. It never asks what the repository already answers.
   - It asks one question at a time, each with a recommendation.
   - It challenges the request before it shapes it: who wants this, what do they do today, and what is the narrowest version that helps them this week.
   - Before it writes the spec, it shows two or three ways to build the work, with trade-offs and its pick. The person picks.
   - The spec gains one line that is always present, **Surfaces**. It names each sensitive surface the work touches:
     authentication, money, uploads, webhooks, secrets, CI, an agent's tools, personal data.
     For each surface, an anti-goal names the harm. When the work touches none, the line says which surfaces were checked.
     This line is the one exception to "delete a section that does not apply".

4. **`plan` writes a plan that a builder with no context can run.**
   - A file map first: each file to create or change, and its one job.
   - Tasks that each end in a passing test and one commit. Each task lists its files, the failing test to write first, the exact command and its expected result, and a done-when line.
   - No placeholder. "Add error handling" is not a step. The plan shows the test and names the behaviour.
   - A self-check: each done-when line and each anti-goal of the spec maps to a task. The plan names the task.
   - Security: a task that touches a surface from the spec carries `Surface: <name>` and a hostile-input test for that surface's anti-goal.
   - A challenge from three roles before the gate. The founder asks: is this the right scope, and what do we cut. The engineer asks: where does it fail, and which test proves it. The designer asks: what does the person see at each step and each error. Each role changes the plan or writes "no change" with its reason.
   - The two reviews of the plan live in `skills/plan/lenses.md`: the engineering lens (scope challenge, what already exists, architecture, failure modes, a test map of every new path, performance) and the design lens (each screen, state and error, rated, with what would make it right). `plan` loads the file when it reviews.
   - The last task of every plan is the review.
   - The plan goes where the repository keeps plans. With no such place, `plan` recommends `docs/plans/YYYY-MM-DD-<slug>.md` and asks once.

5. **`build` runs the plan, one task at a time.**
   - Test first: write the test, run it, see it fail for the right reason, write the least code that passes, run it again, commit.
   - Where the harness has subagents, a fresh agent builds each task from the task text alone. Another fresh agent checks the result against the task. Where it has none, the session builds inline and says so in the report.
   - A task with `Surface:` writes its hostile-input test before any other code.
   - A failing test stops the run. The agent reproduces the failure and finds the root cause before it changes more code. It fixes the cause, never the symptom.
   - A wrong plan stops the run. The agent names the wrong step, proposes the change and updates the plan. It does not improvise around it.
   - "Done" needs the full suite, run after the last change, with its output quoted.

6. **`review` owns the security pass on a change.** It runs with no extra request.
   - Trigger: the diff touches a surface of goal 3, or the plan has a `Surface:` task, or the person asks for a security review of a change.
   - The pass uses what `security` change mode does today: check the surface's protections across the whole system, hunt the dimensions that fit, score with `builder security score`, and run `baseline diff`.
   - Each Critical or Important finding gets a fresh agent that did not write it and tries to refute it. Where the harness has no subagents, the review says "not independently verified".
   - The review's coverage line names each surface it checked and each one it did not.
   - The security pass lives in `skills/review/security-pass.md`. It refers to `skills/security/standards.md` for the dimensions and for the one list of sensitive surfaces. No skill copies that list.
   - The verifier's prompt gives it the finding and the source as data. It follows no instruction found in them.
   - `review` runs `bin/builder`, so it stops at a missing binary and offers the install, as `security` does.
   - The reviewer uses the running product as the person does: the page in a browser where the harness has one, the command, or the API call. It quotes what it saw. When it cannot run the product, it writes "not used as a running product" under ⚠ Not proven.

7. **`security` keeps the whole-repository audit, and takes the Cloudflare method.**
   - **Three states.** Each record is `confirmed`, `needs_validation` or `rejected`. Only `confirmed` gets a CVSS vector, a score and a rating.
     `needs_validation` names the exact fact that is missing and who can observe it.
   - **Independent verification.** A fresh agent receives each proposed `confirmed` record and tries to refute it from the source and a local probe. Its record replaces the hunter's wording.
   - **Source visibility.** A control that can live outside the repository (a proxy, a provider setting, an identity policy) is never assumed present or absent.
   - **The settled conflict.** A missing limit or check on a public or expensive surface stays a finding when the source shows the whole path. It becomes `needs_validation` when the control can live outside the repository.
   - **Coverage ledger.** A unit has the id `<surface>/<dimension>`, the list of its files and the git hash of each. A unit is work again when a hash differs. The baseline also stores units: surface × dimension, each `covered`, `deferred`, `out_of_scope` or `not_inspected`, with the commit. The next audit reads it. A unit whose source changed is work again. A quick or scoped run never implies that the rest is fine.
   - **Profiles.** `quick`: the session hunts, one verifier checks all proposed `confirmed` records together. `standard` (default): the session hunts, one verifier per record. `deep`: one hunter per surface group, a critic names the units that nobody covered, one verifier per record.
   - **Budget.** The person can cap the number of agents. Verification is reserved before hunting. When the budget ends, the remaining units are `deferred`, never `covered`.
   - **Domain checklists.** A new `skills/security/domains.md` holds five lists in builder's own words: AI, LLM and agents; cloud and deployment; client side; desktop, mobile and local IPC; protocols, RPC and messaging. The audit loads only the lists that fit the repository.

8. **`builder security` checks the new rules in Go.**
   - `score` requires `state` on each record. It refuses a vector on a record that is not `confirmed`, and names the bad field.
   - `baseline save` and `baseline diff` store and compare states and ledger units.
   - A baseline saved before this change still loads. Its findings read as `confirmed`, with the note "saved before states existed".

9. **`ship` takes a reviewed change to a checked release.**
   - It syncs with the base branch, runs the full suite, and refuses to go on with an open Critical or Important finding.
   - It writes the changelog entry by what the person can now do, and makes signed Conventional Commits.
   - It shows the diff and the pull request text. A push, a pull request and a merge each wait for the person's yes, or for a standing rule the person gave.
   - After the merge it checks the released thing where the person meets it, and quotes the result.
   - Each failure stops the stage and names the fix: a suite that fails after the sync, a push that the remote refuses, a release check that fails. `ship` never forces a push and never retries a merge by itself.
   - A `needs_validation` record on a surface that the change touches blocks `ship` until the person answers it. An unattended run stops before `ship` and lists each open record with the exact words to reply.
   - It ends with the look back: what broke, what the person corrected, and which of these `canon` turns into a rule.

10. **builder states the framework instead of a routing table.**
   - `skills/builder/SKILL.md` and `soul-compact.md` name the five stages and the size rule. The "Settlements" stay as builder's own rules, with no pack named.
   - The routing table goes. No row sends work to another pack.
   - "Show a visual as a shareable page" stays as a standing rule.
   - When another installed pack also offers a spec, plan, build, review or ship flow, the builder framework owns the work. One flow per piece of work.
   - The `superpowers` and `gstack` recommendations and their signals are deleted. A ledger that holds an old answer for either still loads.
   - README, AGENTS.md (thirteen skills, the three new table rows, the three attributions), the README and the licence notes (the three sources of the ideas). The README's "Why builder" section says that builder stands alone, the manifests' descriptions and CHANGELOG `[Unreleased]` say the same.

## Non-goals

- A debugging skill, a worktree skill, a visual companion, a design-system skill, a deploy or canary skill, or a browser of builder's own. `review` uses the browser that the harness has.
- A copy of the Cloudflare text, its JSON schema or its Node validators.
- An OS-level sandbox for probes. builder keeps the throwaway worktree and the in-process probe. The report's "What I ran" says so.
- Hunter waves with several critic rounds. Only the `deep` profile has one critic pass.
- A change to the hook's time or network rules, to `reach`, `fleet`, `intake`, `budget`, `harness` or `canon`.

## Anti-goals

- A one-line fix is pushed through a spec and a plan.
- The Surfaces line says "none" and nobody looked.
- A finding that no second agent checked carries a severity and reads as proven.
- Two flows run on one piece of work, and the person answers the same question twice.
- A skill passes 150 lines, or one rule lives in two skills.

## Constraints

- A skill has 60 to 150 lines, the two required sections, and ASD-STE100 prose.
- A skill change starts with an eval: the fixture fails without the skill first. New fixtures: `tests/evals/plan`, `tests/evals/build`, `tests/evals/ship`, `tests/evals/flow-gates` (an attended prompt: the agent asks one question and stops, writes no code and no plan). Changed fixtures: `shape`, `review`, `security-audit`, and `security` (renamed, see call 6).
- The skills work on all four harnesses. A step that needs subagents has a stated fallback.
- The binary is Go. The hook makes no network call.
- The repository is public. No private name enters a file, a commit or a pull request.
- The release bot owns the versions and the released CHANGELOG sections.

## Approaches

- **A. One skill per stage, security in each stage. Picked by the owner on 2026-11-10, then widened by the owner to include the gstack essence and the `ship` stage.** Thirteen skills, one path. Each stage has room for its rules and its own eval.
- **B. The flow as one section of the `builder` skill.** The smallest change. Rejected: `builder` is at its 150-line cap, so each stage would get a few lines and no eval.
- **C. One new `flow` skill for all stages.** One file to load. Rejected: it repeats what `shape` and `review` own.

## Calls to check

These are my calls, not the owner's. Each has a reason and the cost if it is wrong.

1. **Order: security first, then the flow, then ship and the cleanup.** One pull request and one release carry all three (owner's ruling D5). The milestones are the order of the commits.
   M1: goals 7 and 8. M2: goals 1 to 6. M3: goals 9 and 10. Reason: M1 has no dependency, and M2's review pass uses M1's states. Cost if wrong: a later start on the flow.
2. **The `superpowers` and `gstack` recommendations are deleted**, not kept as optional. Reason: builder no longer needs them, and an unused recommendation is noise. Cost if wrong: a person who wants both packs gets no hint.
3. **`review` owns change mode; `security` owns the audit.** The phrase "security review this PR" moves to the `review` description. Reason: one owner per job. Cost if wrong: a person who types `/builder:security` on a diff gets pointed to `review`.
4. **This repository's `docs/superpowers/` folder becomes `docs/specs/` and `docs/plans/` in M3**, with the excludes in `scripts/validate.sh`. Reason: the folder name is the old dependency. Cost if wrong: old links to these files break.
5. **The fresh verifier runs on Critical and Important findings in `review`, and on every proposed `confirmed` record in an audit.** Reason: cost follows stakes. Cost if wrong: an unverified Minor finding in a review.

6. **Change mode moves in M2, not M1.** The change-mode text leaves `security` and enters `review` in M2. Until then `security` keeps both modes. The fixture `tests/evals/security` (a change review) becomes `tests/evals/review-security`, and `tests/evals/security-audit` stays the audit fixture. Reason: one rule in one skill, and M1 stays small. Cost if wrong: one milestone where `security` still holds change mode.
7. **Security results map onto the review's severities.** A `confirmed` record with a CVSS rating of Critical or High is Critical. Medium is Important. Low is Minor. A `needs_validation` record has no severity. On a surface that the change touches it blocks `ship` (owner's ruling D7). On any other surface it becomes a ▶ You item that names who can observe the missing fact. A finding that the verifier refutes goes under "Declined to judge" with the verifier's reason. Cost if wrong: a Medium finding blocks a merge that the owner would let pass.
8. **A failed review gate goes back to `build`, twice at most.** `build` fixes each Critical and Important finding test-first, with the review's named test, and `review` runs again. After two rounds with an open finding, the flow stops and hands the review to the person. `review` itself still posts nothing and changes no code. Cost if wrong: a third round that would have closed it.
9. **The parity eval follows the recommendations.** Its builder-led setup is "builder and what builder recommends". With call 2, that setup has no superpowers and no gstack. M3 removes the code in `cmd/parity-config` and `tests/test_evals.sh` that enables them. The parity eval also runs the `plan`, `build`, `review`, `ship` and `flow-gates` fixtures once on Claude Code before the release (owner's ruling D8).

## Unattended runs

The rule of the `builder` skill, "When nobody answers", holds for every gate.
When the person says they cannot answer, or the run is headless, the agent takes its own recommendations.
Correction (2026-11-11): a one-shot or headless run alone is not unattended. The rule applies when the person says they cannot answer, or the run is plainly scheduled or overnight (decision D3, the flow-gates fixture).
It marks the spec "proposed, not confirmed", lists each call under Calls, and goes on to plan and build only when the request asked for the build.
The eval fixtures run this way. The `flow-gates` fixture proves that the agent stops at the first gate. What happens after the person answers, the two-round review loop and the release check are proven by an attended run by the owner, not by a fixture.

## Paid eval runs

The owner's ruling (2026-11-10): every fixture is graded on all four harnesses. Claude Code and Codex run first, then Cursor and opencode.
All fixtures run before the one pull request merges: about 72 runs (9 new or changed fixtures × 2 runs × 4 harnesses, estimated). The baseline run has superpowers and gstack turned off, as `tests/evals/README.md` requires for overlapping packs. A run that cannot start on a harness is reported with its error, never as a pass.

## Left to the builder

- The exact JSON field names for states and ledger units.
- How each harness file names "start a fresh agent", and the fallback words.
- Which lines move out of `review` and `shape` to make room under 150.
- The fixtures' sample repositories and the `expect.md` bullets.
- The commit type of each milestone.

## Done when

- A person with only builder installed says "add CSV export" in a sample repository. They answer the questions, approve a spec, and get a plan, commits with tests first, a review and a pull request draft, with no other pack.
- The same person says "fix this typo". No spec and no plan appear.
- A change that adds an upload endpoint gets a security pass in its review, and nobody asked for one.
- An audit report shows the three states. No `needs_validation` record has a score. `builder security score` refuses one that has.
- A second audit of the same commit names the units that the first one deferred.
- `bash scripts/validate.sh` passes, and `rg -i 'superpowers|gstack'` over the whole repository finds only the attributions, the changelog and the old specs.
- A person who updates to the new release gets the framework with no setup step.
- Correction (2026-11-11): the search also finds the tests that prove the removal (the old-ledger test, the parity tests, the README credits test), this spec, its plan and the one-line credit in `skills/plan/lenses.md`. The line is met when no file tells a person or an agent to use either pack.

## Engineering review record

Target: this document, as drafted on 2026-11-10. Reviewer: `/plan-eng-review`. The text above this heading is the reviewed plan.

Owner's amendment during the review (2026-11-10): "I keep often doing plan eng review + plan design review from gstack, superpowers from obra, review from gstack / superpowers, builder must consume all of this and compress it meaningfully into builder skill". Accepted scope: `plan` carries an engineering review and a design review of the plan; `review` carries the essence of both packs' code reviews.

Scope record: feature answers: none proposed; structure: A, Original arrangement (D1); accepted scope: 3 new skills `plan`, `build`, `ship`, reference files `plan/lenses.md`, `review/security-pass.md`, `security/domains.md`, 13 skills, 3 new fixtures; pending remedies: R1, R2, R3, R4.

## Decision ledger

### R1: what builder does when another installed pack also runs a spec, plan or build flow
Finding: 2, P1, confidence 8/10, this document goal 10 ("the builder framework owns the work. One flow per piece of work."), reviewer plan-eng-review.
Plan baseline: one rule in the `builder` skill; the `superpowers` and `gstack` recommendations and their signals are deleted. Nothing approved yet.
Runtime evidence: in the review session both hooks loaded, and the other pack's hook text marks its brainstorm as mandatory. Whether an agent then runs both flows is unknown.
Comparison grid:

| Choice | Current | A | B | C |
|---|---|---|---|---|
| R1 overlap handling | rule in `builder` skill only; signals deleted | signals kept and flipped into one "overlap" recommendation, plus the rule in the hook soul, plus the parity eval's current-setup run as the proof | rule in the hook soul and the skill; signals deleted | as designed |
| R2 gate tests | pending | pending | pending | pending |
| R3 default agent cap | pending | pending | pending | pending |
| R4 number of pull requests | pending | pending | pending | pending |

Question D2:
D2 — What should builder do when superpowers or gstack is also installed?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: After this change builder asks its own questions and writes its own plan. If another pack with the same job is installed, both can start, and the person answers the same question twice. The design has one sentence against this and nothing that checks it.
Stakes if we pick wrong: people who keep both packs get a slower, doubled flow and blame builder.
Recommendation: A because it reuses the signal code that exists and tells the person once, with a way to say never.
Completeness: A=9/10, B=6/10, C=3/10
Pros / cons:
A) Overlap recommendation (recommended)
  ✅ The person hears once that two packs do the same job, and picks yes, later or never
  ✅ Reuses the detection code in internal/signals instead of deleting it
  ❌ Keeps two recommendation entries and their tests alive in the catalog
B) Rule in the hook soul only
  ✅ The precedence rule reaches every session through the same channel as the other pack's hook
  ✅ Deletes the two recommendations and their signal code, so less to maintain
  ❌ No check and no word to the person; a doubled flow stays silent
C) Keep the design as written
  ✅ No extra work beyond the sentence that the design already has in the skill
  ✅ Smallest diff in the cleanup milestone and nothing new to test
  ❌ The rule loads only when the skill loads, after the other flow may have started
Net: a few kept lines of signal code buy a person who knows why they are asked twice.
Header: Overlap
Options:
A) Overlap recommendation (recommended)
Keep the superpowers and gstack signals and turn them into one recommendation: "this pack also runs a spec, plan or build flow; builder owns it; turn the other off to stop double questions". Put the precedence rule in the hook soul. Prove it with the parity eval's current-setup run. (human: ~3h / CC: ~15 min)
B) Rule in the hook soul only
Put the precedence rule in soul-compact.md and the builder skill. Delete both recommendations and their signals as designed. No message to the person. (human: ~1h / CC: ~5 min)
C) Keep the design as written
One rule in the builder skill. Both recommendations and signals deleted. No further work.

State: approved
Actual answer: C, Keep the design as written, with the owner's words (D2, 2026-11-10): "I find superpowers + gstack to be too noisy to recommend, lets extract their essence, note this in the README license, and continue with design as written".
Accepted scope: one rule in the `builder` skill; both recommendations and their signals are deleted; the README and the licence notes name superpowers, gstack and the Cloudflare skill as the sources of the ideas.
History: none

### R2: how the framework's gates are tested
Finding: 3, P1, confidence 8/10, this document "Unattended runs" ("The first done-when line is proven by an attended run by the owner, not by a fixture."), reviewer plan-eng-review.
Plan baseline: every fixture runs unattended; the gates (spec approval, pick of approach, yes before a push) are proven only by one attended run by the owner. Nothing approved yet.
Runtime evidence: `tests/evals/README.md` says each run is one shot; existing prompts say the person cannot answer. No fixture today shows an agent that stops at a gate.
Comparison grid:

| Choice | Current | A | B |
|---|---|---|---|
| R1 overlap handling | approved: C (D2) | approved: C (D2) | approved: C (D2) |
| R2 gate tests | owner's attended run only | one new fixture `flow-gates` with an attended prompt; its bullets: asks one question with a recommendation and stops, writes no code, writes no plan before the spec is approved | owner's attended run only |
| R3 default agent cap | pending | pending | pending |
| R4 number of pull requests | pending | pending | pending |

Question D3:
D3 — How do we prove that the framework stops at its gates?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: The framework must stop and wait for you at three points: before it writes a plan, before it picks an approach, and before anything leaves the machine. Every automatic test today tells the agent "nobody can answer", so no test ever sees it stop. A later edit to a skill could remove a gate and no test would fail.
Stakes if we pick wrong: a release where the agent builds straight through without asking, found by a user and not by a test.
Recommendation: A because one fixture guards the behaviour that the whole framework rests on.
Completeness: A=9/10, B=5/10
Pros / cons:
A) Add a `flow-gates` fixture (recommended)
  ✅ A test fails when an agent builds without the person's approval of the spec
  ✅ Runs on all four harnesses, so a harness that ignores the gate shows in the table
  ❌ One more fixture: about 8 more paid runs, and one more to keep graded
B) Owner's attended run only
  ✅ No new fixture and no extra paid eval runs on any of the four harnesses
  ✅ You see the real attended experience yourself before each release
  ❌ Nothing fails when a later skill edit removes a gate between your manual runs
Net: eight paid runs against a silent loss of the framework's main promise.
Header: Gate tests
Options:
A) Add a flow-gates fixture (recommended)
One new fixture, `tests/evals/flow-gates`, with a prompt that does not say "I cannot answer". Its bullets: the agent asks one question with a recommendation and stops, writes no code, and writes no plan before the spec is approved. Graded on all four harnesses. (human: ~2h / CC: ~10 min, plus about 8 paid runs)
B) Owner's attended run only
Keep the design: the gates are proven by one attended run by the owner before each release. No new fixture.

State: approved
Actual answer: A, Add a flow-gates fixture (D3, 2026-11-10).
Accepted scope: one new fixture `tests/evals/flow-gates` with an attended prompt; bullets: asks one question with a recommendation and stops, writes no code, writes no plan before the spec is approved; graded on all four harnesses.
History: none

### R3: default cap on the number of agents in an audit
Finding: 5, P2, confidence 7/10, this document goal 7 ("The person can cap the number of agents." and "`standard` (default): the session hunts, one verifier per record."), reviewer plan-eng-review.
Plan baseline: no cap unless the person sets one. Nothing approved yet.
Runtime evidence: none; the audit profiles do not exist yet. The number of records in a real audit is unknown.
Comparison grid:

| Choice | Current | A | B |
|---|---|---|---|
| R1 overlap handling | approved: C (D2) | approved: C (D2) | approved: C (D2) |
| R2 gate tests | approved: A (D3) | approved: A (D3) | approved: A (D3) |
| R3 default agent cap | none | quick 2, standard 8, deep 20 agents; the report names the cap and each deferred unit; the person can raise it | none |
| R4 number of pull requests | pending | pending | pending |

Question D4:
D4 — Should an audit have a default limit on the agents it starts?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: In an audit, a second agent checks each finding. A repository with forty findings would start forty agents, and each one spends from your usage limit. The design lets you set a limit but has none by default.
Stakes if we pick wrong: one audit on a large repository eats a large part of a weekly usage limit with no warning.
Recommendation: A because a stated default protects the usage limit and the report still says what was left unchecked.
Completeness: A=9/10, B=6/10
Pros / cons:
A) Default caps per profile (recommended)
  ✅ An audit cannot start more than 2, 8 or 20 agents unless you raise the limit
  ✅ The report names each unit that was deferred, so nothing reads as checked
  ❌ A large audit needs a second run or a raised cap to verify every finding
B) No default cap
  ✅ Every proposed finding gets its verifier in one run with no setting to learn
  ✅ Less text in the security skill, which is close to its 150-line cap
  ❌ The cost of an audit is unknown until it ends, and can be very large
Net: a bounded, stated cost against full verification in one run.
Header: Agent cap
Options:
A) Default caps per profile (recommended)
quick: 2 agents. standard: 8. deep: 20. Verification is reserved first. The report names the cap and each deferred unit. The person can raise the cap in the request. (human: ~1h / CC: ~5 min)
B) No default cap
Keep the design: no cap unless the person sets one.

State: approved
Actual answer: B, No default cap (D4, 2026-11-10).
Accepted scope: none; the design stands: no cap unless the person sets one.
History: none

### R4: how many pull requests and releases carry the work
Finding: 7, P2, confidence 8/10, this document call 1 ("Three plans under this one spec."), reviewer plan-eng-review.
Plan baseline: three milestones M1, M2, M3; the document does not say how they merge. Nothing approved yet.
Runtime evidence: `AGENTS.md` and commit 31a13ab say that a merge to `main` releases. So a milestone merged alone to `main` is a release of its own. After M2 alone, a release would hold the new flow while the README still says "Install all three".
Comparison grid:

| Choice | Current | A | B | C |
|---|---|---|---|---|
| R1 overlap handling | approved: C (D2) | approved: C (D2) | approved: C (D2) | approved: C (D2) |
| R2 gate tests | approved: A (D3) | approved: A (D3) | approved: A (D3) | approved: A (D3) |
| R3 default agent cap | approved: B, none (D4) | approved: B, none (D4) | approved: B, none (D4) | approved: B, none (D4) |
| R4 number of pull requests | unspecified | three pull requests into one branch `feat/framework`, then one merge to `main`: one release | one pull request with all three milestones: one release | three pull requests to `main`: three releases |

Question D5:
D5 — How should the three milestones reach users?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: The work has three parts: the security method, the flow, then ship and the cleanup. In this repository every merge to main publishes a release. If the parts merge to main one by one, users get a release in the middle where builder has its own flow but still tells them to install the other packs.
Stakes if we pick wrong: users install a half-finished framework, or you review one very large pull request in one sitting.
Recommendation: A because each part gets its own review and its own eval runs, and users get one whole release.
Note: options differ in kind, not coverage — no completeness score.
Pros / cons:
A) Three PRs into one branch, one release (recommended)
  ✅ Each milestone is reviewed and graded alone, in a diff of a readable size
  ✅ Users get the whole framework in one version, with no half state between
  ❌ Four merges in total, and the branch must stay in sync with main while it lives
B) One PR, one release
  ✅ One review, one merge and one set of paid eval runs for the whole change
  ✅ No long-lived branch to keep in sync with the main branch
  ❌ One very large diff: three new skills, four changed skills, Go and tests together
C) Three PRs to main, three releases
  ✅ Each part reaches users as soon as it is ready, starting with the security method
  ✅ Plain branches from main with no integration branch to manage
  ❌ The release after the second part holds the new flow and the old "install all three" advice
Net: one more branch to manage buys small reviews and a single whole release.
Header: Releases
Options:
A) Three PRs into one branch, one release (recommended)
M1, M2 and M3 each merge into `feat/framework` after their own review and eval runs. One final merge to `main` releases the whole framework. (four merges)
B) One PR, one release
All three milestones land in one pull request to `main`. One review, one release.
C) Three PRs to main, three releases
Each milestone merges to `main` and is released alone. The middle release still recommends the other packs.

State: approved
Actual answer: B, One PR, one release (D5, 2026-11-10).
Accepted scope: all three milestones land in one pull request to `main`; one review, one release. The milestones stay as the order of work and of commits inside that pull request.
History: none

### R5: the regression contract for behaviour that exists today
Finding: 8, P1, confidence 8/10, this document goal 8 ("`score` requires `state` on each record") and goal 10 (deleted recommendations), reviewer plan-eng-review.
Plan baseline: the document names two rules: an old baseline still loads, and an old ledger answer still loads. Nothing approved yet.
Runtime evidence: `internal/security/security_test.go` has 27 tests and `cmd/builder/security_test.go` has 5; none knows a `state`. `internal/ledger/ledger_test.go:285` proves an old ledger with a retired answer loads.
Comparison grid:

| Choice | Current | A | B |
|---|---|---|---|
| R1 overlap handling | approved: C (D2) | approved: C (D2) | approved: C (D2) |
| R2 gate tests | approved: A (D3) | approved: A (D3) | approved: A (D3) |
| R3 default agent cap | approved: B, none (D4) | approved: B, none (D4) | approved: B, none (D4) |
| R4 number of pull requests | approved: B, one (D5) | approved: B, one (D5) | approved: B, one (D5) |
| R5 regression contract | two rules named, no tests named | five behaviours kept, each with a named test; one intended break | the two Go rules only, each with a named test; one intended break |

Question D6:
D6 — Which of today's behaviours must provably keep working after this change?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: This change rewrites how security findings are scored and removes two recommendations. People already have saved audit baselines and saved answers on their machines, and some rely on the security review of a diff. We must name what keeps working and write a test for each, and name the one thing that breaks on purpose.
Stakes if we pick wrong: a person updates and their saved audit history fails to load, or the diff security review quietly gets weaker.
Recommendation: A because the three extra checks reuse fixtures that exist and guard what users already rely on.
Completeness: A=10/10, B=6/10
Pros / cons:
A) Full contract, five behaviours (recommended)
  ✅ A saved baseline and a saved ledger from the old version both load, each with a Go test
  ✅ The diff security review keeps its old eval bullets after it moves into `review`
  ❌ Three more assertions to keep green: the moved fixture, the hook soul and the typo path
B) Go rules only, two behaviours
  ✅ The two data files on a person's machine are guarded by Go tests that run in seconds
  ✅ Nothing new to grade in the paid evals beyond what is already planned
  ❌ The diff security review can lose a behaviour in the move and no test fails
Both: the one intended break is that `builder security score` refuses a findings file with no `state`.
Net: three cheap assertions against a silent loss in the move from `security` to `review`.
Header: Regression
Options:
A) Full contract (recommended)
Keep, each with a named test: (1) a baseline saved by 3.0.0 loads and its findings read as confirmed; (2) a ledger with an old superpowers or gstack answer loads; (3) every bullet of today's tests/evals/security/expect.md still passes in the renamed review-security fixture; (4) the hook still injects the soul (tests/test_hook.sh); (5) the existing CVSS, CWE and baseline diff tests pass with `state: confirmed` added. Intended break: `score` refuses a file with no `state`. (human: ~3h / CC: ~15 min)
B) Go rules only
Keep, each with a Go test: (1) a baseline saved by 3.0.0 loads and its findings read as confirmed; (2) a ledger with an old superpowers or gstack answer loads. Intended break: `score` refuses a file with no `state`. (human: ~1h / CC: ~5 min)

State: approved
Actual answer: A, Full contract (D6, 2026-11-10).
Accepted scope: five behaviours kept, each with a named test: (1) a baseline saved by 3.0.0 loads and its findings read as confirmed; (2) a ledger with an old superpowers or gstack answer loads; (3) every bullet of today's `tests/evals/security/expect.md` still passes in the renamed `review-security` fixture; (4) the hook still injects the soul; (5) the existing CVSS, CWE and baseline diff tests pass with `state: confirmed` added. Intended break: `score` refuses a file with no `state`.
History: none

### R6: whether a `needs_validation` record can pass the review gate
Finding: O1, Critical, confidence 8/10, this document call 7 ("A `needs_validation` record has no severity and does not block the gate.") with goal 7 ("It becomes `needs_validation` when the control can live outside the repository."), reviewer in-host fallback reviewer (Codex unavailable).
Plan baseline: `needs_validation` never blocks; it becomes a ▶ You item. Nothing approved yet.
Runtime evidence: none; the gate does not exist yet. The reviewer's point is a static read: most limits and checks can live outside a repository, so an agent can choose the state that passes.
Comparison grid:

| Choice | Current | A | B | C | D |
|---|---|---|---|---|---|
| R1 to R5 | approved (D2 to D6) | unchanged | unchanged | unchanged | unchanged |
| R6 needs_validation at the gate | never blocks | blocks `ship` when it is on a surface the change touches, until the person answers; an unattended run stops before `ship` and lists it | never blocks | undecided; one bounded check of how often the state appears in the two security fixtures; gate rule stays pending | never blocks for now; proposal stays open |
| R7 comparison with the three-pack setup | pending | pending | pending | pending | pending |

Question D7:
D7 — Can a security finding that still needs a check pass the review and ship?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: A finding gets the state "needs validation" when the protection might live outside the code, for example in a proxy. In the design such a finding has no severity and does not stop the review. But almost any missing limit might live outside the code. So an agent that wants to finish can label a real hole "needs validation" and ship it. In an unattended run nobody reads the note.
Stakes if we pick wrong: the "security in every stage" promise has a door that opens by choosing a label, and a real hole ships.
Recommendation: A because a finding on a surface that the change itself touches must be answered by a person before it leaves the machine.
Note: options differ in kind, not coverage — no completeness score.
Pros / cons:
A) Apply this change (recommended)
  ✅ A finding on a touched surface cannot ship until a person says the outside control exists
  ✅ An unattended run stops before ship and lists the open items with the words to reply
  ❌ More stops for the person on changes that touch auth, uploads, webhooks or money
B) Keep this row's current value
  ✅ No extra stop: the review passes and the open check is listed for the person
  ✅ Matches the Cloudflare rule that an unverified record carries no severity
  ❌ A label that the agent picks decides whether a real hole blocks the ship
C) Investigate before choosing
  ✅ One bounded check: count how often this state appears in the two security fixtures
  ✅ The decision then rests on a measured rate, not on a reading of the design
  ❌ The gate rule stays open, and the fixtures are small and may not show the problem
D) Defer this proposed change only
  ✅ The design ships as written and this proposal stays open in the review record
  ✅ No change to the review or ship skills beyond what is already agreed
  ❌ The bypass exists in the first release of the framework
Net: more stops on sensitive changes against a gate that a label can open.
Header: Gate bypass
Options:
A) Apply this change (recommended)
A `needs_validation` record on a surface that the change touches blocks `ship` until the person answers it. It still has no severity. An unattended run stops before `ship` and lists each open record with the exact words to reply. Records on untouched surfaces stay non-blocking. (human: ~2h / CC: ~10 min)
B) Keep this row's current value
`needs_validation` never blocks. It becomes a ▶ You item.
C) Investigate before choosing
Count how often `needs_validation` appears when the two security fixtures run. Change nothing until then. The gate rule stays pending.
D) Defer this proposed change only
Ship the design as written. This proposal stays open in the review record.

State: approved
Actual answer: A, Apply this change (D7, 2026-11-10).
Accepted scope: a `needs_validation` record on a surface that the change touches blocks `ship` until the person answers it; it has no severity; an unattended run stops before `ship` and lists each open record with the exact words to reply; records on untouched surfaces stay non-blocking.
History: call 7 of the draft said that `needs_validation` never blocks the gate.

### R7: a measured comparison of builder alone with the three-pack setup
Finding: O9, Medium, confidence 7/10, this document call 9 ("that setup has no superpowers and no gstack") and done-when 1, reviewer in-host fallback reviewer (Codex unavailable).
Plan baseline: no comparison; the parity eval's builder-led arm loses both packs and its fixture list stays `builder shape`. Nothing approved yet.
Runtime evidence: `tests/evals/parity.sh:4-7` runs the `builder` and `shape` fixtures in two arms: the person's current setup, and builder with what it recommends. Whether builder alone meets as many bullets as the three packs is unknown.
Comparison grid:

| Choice | Current | A | B | C | D |
|---|---|---|---|---|---|
| R1 to R6 | approved (D2 to D7) | unchanged | unchanged | unchanged | unchanged |
| R7 comparison | none | `parity.sh` also runs `plan`, `build`, `review`, `ship` and `flow-gates`; one run on Claude Code before the release; the result goes in the pull request; builder alone must meet at least as many bullets as the current setup, or the owner decides | none | undecided; one dry run to count the cost; nothing else changes | none for now; proposal stays open |

Question D8:
D8 — Do we measure builder alone against your current three-pack setup before the release?
Project/branch/task: builder on main, reviewing the builder framework design.
ELI10: Today you get plan reviews, test-first building and a browser from two other packs. The new builder does these jobs itself in a few hundred lines. It may do them worse, and nothing in the design would show that. The repository already has a script that runs the same tasks in two setups and compares them.
Stakes if we pick wrong: a release that makes your own daily workflow weaker, found only after you have removed the other packs.
Recommendation: A because the script exists and one run answers the one question that decides whether to ship.
Note: options differ in kind, not coverage — no completeness score.
Pros / cons:
A) Apply this change (recommended)
  ✅ One run shows, bullet by bullet, where builder alone is weaker than the three packs
  ✅ Reuses tests/evals/parity.sh, which already runs two setups side by side
  ❌ About 10 more paid runs on Claude Code, and a result that may send work back
B) Keep this row's current value
  ✅ No extra paid runs and no new step before the release of the framework
  ✅ Each fixture still proves that its skill beats a session with no packs at all
  ❌ Nothing shows whether you lose quality when you drop the other two packs
C) Investigate before choosing
  ✅ A dry run of the script prints the commands and lets us count the cost first
  ✅ Costs nothing and changes nothing in the plan until you have seen the number
  ❌ The question of quality stays open until a real run happens
D) Defer this proposed change only
  ✅ The framework ships first and the comparison can run on the released version
  ✅ Keeps the pull request smaller by leaving parity.sh as it is today
  ❌ You learn about a weaker stage after users already have the release
Net: ten paid runs against shipping a framework that may be weaker than what it replaces.
Header: Comparison
Options:
A) Apply this change (recommended)
`tests/evals/parity.sh` also runs the `plan`, `build`, `review`, `ship` and `flow-gates` fixtures. One run on Claude Code before the release, current setup against builder alone. The result goes in the pull request. If builder alone meets fewer bullets, the owner decides. (human: ~2h / CC: ~10 min, plus about 10 paid runs)
B) Keep this row's current value
No comparison. The parity eval keeps its two fixtures.
C) Investigate before choosing
Run `parity.sh --dry-run` with the longer fixture list and count the runs. Change nothing else. The comparison stays pending.
D) Defer this proposed change only
Ship first. This proposal stays open in the review record.

State: approved
Actual answer: A, Apply this change (D8, 2026-11-10).
Accepted scope: `tests/evals/parity.sh` also runs the `plan`, `build`, `review`, `ship` and `flow-gates` fixtures; one run on Claude Code before the release, current setup against builder alone; the result goes in the pull request; if builder alone meets fewer bullets, the owner decides.
History: call 9 of the draft only removed the two packs from the builder-led setup.

Approval readiness: PASS. Checked: R1 (D2), R2 (D3), R3 (D4), R4 (D5), R5 (D6), R6 (D7), R7 (D8), structure (D1).

## Review output

### NOT in scope
- A default cap on audit agents: the owner chose none (D4).
- A recommendation that warns about an overlapping pack: the owner chose the design as written (D2).
- A browser, a deploy skill or a design-system skill of builder's own: non-goals of the design.
- A Codex outside review of this plan: the Codex sandbox could not start in this container.

### What already exists
- `builder security score` and `baseline save|diff` (`cmd/builder/security.go`): extended with states and units, not rebuilt.
- `internal/ledger` already loads a ledger with a retired answer (`ledger_test.go:285`): the test is extended, the code stays.
- `tests/evals/parity.sh` already compares two setups: its fixture list grows.
- `skills/security/standards.md` already owns the standards map: it also becomes the one owner of the surfaces list.
- `skills/builder/harnesses/*.md` already map "start a subagent": one row "use a browser" is added to each.

### Diagram

```
request ─▶ shape ──spec──▶ plan ──plan──▶ build ──commits──▶ review ──▶ ship ──▶ release
            │ gate:          │ gate:        │ gate:            │ gate:      │ gate:
            │ person         │ each done-   │ suite green,     │ no open    │ person's yes
            │ approves       │ when maps    │ output quoted    │ Critical/  │ per push,
            │ the spec       │ to a task    │                  │ Important  │ PR, merge
            │                │              │                  │            │
            │ Surfaces line  │ Surface: tag │ hostile-input    │ security   │ blocked by an open
            │                │ + two lenses │ test first       │ pass +     │ needs_validation on
            │                │              │                  │ verifier   │ a touched surface
            │                                ▲                  │
            │                                └── fix, at most twice ──┘
            └─ small change: no spec, no plan ─▶ build ─▶ review ─▶ report
after ship: canon takes what broke.   audit (no diff): security, with states, verifier and ledger.
```

### Failure modes
| New path | Realistic failure | Test | Error handling | What the person sees |
|---|---|---|---|---|
| `score` with states | an old findings file with no state | Go test (R5) | named-field error | the field to fix |
| Baseline load | a baseline saved by 3.0.0 | Go test (R5) | reads as confirmed, with a note | history kept |
| `build` with no subagents | Cursor has no verified subagent | `build` fixture on cursor | inline fallback, said in the report | "built inline" |
| `review` security pass | binary missing | `tests/test_validate.sh` guard | stop, offer the install | the install command |
| `review` gate | a real hole labelled `needs_validation` | `review-security` fixture | blocks `ship` on a touched surface (D7) | the words to reply |
| `ship` | suite fails after the sync; push refused | `ship` fixture | stop, name the fix | the failing command |
| Two packs installed | both flows start | none | one rule in the `builder` skill | double questions: a known risk, accepted in D2 |

Critical gaps (no test, no handling, silent): 0. The "two packs" row has a rule but no test; the owner accepted it in D2.

### Worktree parallelization strategy
| Step | Modules touched | Depends on |
|---|---|---|
| M1 security method | `internal/security`, `cmd/builder`, `skills/security`, `tests/evals/security-audit` | — |
| M3a cleanup of the other packs | `lib`, `internal/signals`, `internal/ledger`, `cmd/parity-config`, `tests` | — |
| M2 flow skills | `skills/shape`, `skills/plan`, `skills/build`, `skills/review`, `tests/evals` | M1 |
| M3b ship and the framework text | `skills/ship`, `skills/builder`, README, AGENTS.md, CHANGELOG | M2, M3a |

Parallel lanes: Lane A: M1 → M2. Lane B: M3a. Then M3b.
Execution order: launch A and B. Merge both into the one branch. Then M3b.
Conflict flags: `tests/test_evals.sh` and `tests/evals/` are touched by lanes A and B; sequence those edits.

## Implementation Tasks
Synthesized from this review's findings. Each task derives from a specific
finding above. Run with Claude Code or Codex; checkbox as you ship.

- [ ] **T1 (P1, human: ~1 day / CC: ~30min)** — internal/security — require `state`, refuse a vector on a record that is not confirmed, load a 3.0.0 baseline as confirmed
  - Surfaced by: Test review, code-path gaps; R5 (D6)
  - Files: internal/security/security.go, internal/security/security_test.go, cmd/builder/security.go, cmd/builder/security_test.go
  - Verify: `go test ./internal/security/ ./cmd/builder/`
- [ ] **T2 (P1, human: ~1 day / CC: ~30min)** — internal/security — ledger units with id `<surface>/<dimension>`, file list and git hashes; a changed hash makes the unit work again
  - Surfaced by: Outside voice finding 8
  - Files: internal/security/security.go, internal/security/security_test.go
  - Verify: `go test ./internal/security/`
- [ ] **T3 (P1, human: ~3h / CC: ~15min)** — skills/review, skills/ship — an open `needs_validation` record on a touched surface blocks `ship`
  - Surfaced by: Outside voice finding 1; R6 (D7)
  - Files: skills/review/SKILL.md, skills/review/security-pass.md, skills/ship/SKILL.md, tests/evals/review-security, tests/evals/ship
  - Verify: `bash tests/evals/run.sh review-security`, `bash tests/evals/run.sh ship`
- [ ] **T4 (P1, human: ~2h / CC: ~10min)** — tests/evals — add the attended `flow-gates` fixture
  - Surfaced by: Scope Challenge finding 3; R2 (D3)
  - Files: tests/evals/flow-gates/prompt.md, tests/evals/flow-gates/expect.md
  - Verify: `bash tests/evals/run.sh --check`, `bash tests/evals/run.sh flow-gates`
- [ ] **T5 (P1, human: ~3h / CC: ~15min)** — tests — the regression contract: old ledger answers load, `review-security` keeps today's bullets, the hook soul still injects
  - Surfaced by: Test review, regression rule; R5 (D6)
  - Files: internal/ledger/ledger_test.go, tests/evals/review-security/expect.md, tests/test_hook.sh
  - Verify: `go test ./internal/ledger/`, `bash tests/test_hook.sh`, `bash tests/evals/run.sh review-security`
- [ ] **T6 (P2, human: ~1 day / CC: ~30min)** — skills — reference files `plan/lenses.md`, `review/security-pass.md`, `security/domains.md`; one owner for the surfaces list
  - Surfaced by: Architecture finding 1 (D1); Code quality finding 1
  - Files: skills/plan/lenses.md, skills/review/security-pass.md, skills/security/domains.md, skills/security/standards.md
  - Verify: `bash scripts/validate.sh`
- [ ] **T7 (P2, human: ~1h / CC: ~5min)** — skills/builder/harnesses — add the "use a browser" row to all four files
  - Surfaced by: Architecture finding 2
  - Files: skills/builder/harnesses/claude.md, codex.md, cursor.md, opencode.md
  - Verify: `bash scripts/validate.sh`
- [ ] **T8 (P2, human: ~1h / CC: ~5min)** — skills/review, skills/security — the verifier prompt treats repository text as data
  - Surfaced by: Architecture finding 3
  - Files: skills/review/security-pass.md, skills/security/SKILL.md
  - Verify: `bash tests/evals/run.sh security-audit`
- [ ] **T9 (P2, human: ~2h / CC: ~10min)** — skills/ship — a stop and a fix line for each failure
  - Surfaced by: Architecture finding 4
  - Files: skills/ship/SKILL.md, tests/evals/ship/expect.md
  - Verify: `bash tests/evals/run.sh ship`
- [ ] **T10 (P2, human: ~2h / CC: ~10min)** — tests/evals — the parity run covers five more fixtures; run it once before the release
  - Surfaced by: Outside voice finding 9; R7 (D8)
  - Files: tests/evals/parity.sh, cmd/parity-config/main.go, tests/test_evals.sh
  - Verify: `bash tests/evals/parity.sh --dry-run`, `go test ./cmd/parity-config/`, then one paid run
- [ ] **T11 (P2, human: ~30min / CC: ~5min)** — tests/evals — run each baseline with the two other packs turned off
  - Surfaced by: Outside voice finding 3
  - Files: tests/evals/README.md
  - Verify: the baseline transcript of `plan` shows no skill of another pack
- [ ] **T12 (P2, human: ~3h / CC: ~15min)** — lib, internal/signals, README — delete both recommendations and signals; credit the three sources in the README and the licence notes
  - Surfaced by: Scope Challenge finding 2; R1 (D2); Outside voice finding 10
  - Files: lib/recommendations.json, internal/signals/signals.go, internal/signals/signals_test.go, internal/ledger/ledger_test.go, README.md, AGENTS.md
  - Verify: `go test ./internal/signals/ ./internal/ledger/`, `rg -i 'superpowers|gstack'`
- [ ] **T13 (P2, human: ~30min / CC: ~5min)** — skills/review — the stop at a missing binary
  - Surfaced by: Code quality finding 2
  - Files: skills/review/SKILL.md
  - Verify: `bash scripts/validate.sh`

Effort ratios assume features at about 30x and tests at about 50x.

### Unresolved decisions
None in this review.

### Known risks kept open on purpose
- Two packs installed at once can both start a flow (D2). On a harness with no namespaces, `plan`, `ship` and `review` share names with other packs.
- An audit has no default agent cap (D4).
- No Codex outside review ran. The in-host fallback reviewer is not outside coverage.
- Multi-turn behaviour after a gate is proven only by the owner's attended run.

### Completion summary
- Step 0: Scope Challenge — scope accepted as-is (widened by the owner during the review)
- Reality rows: 7/7 applicable rows emitted (`PLAN_REALITY: phase=eng rows=7/7 missing=none unverified=0: verdict=complete`), 0 unverified premises
- Architecture Review: 4 issues found
- Code Quality Review: 2 issues found
- Test Review: diagram produced, 18 gaps identified
- Performance Review: 0 issues found
- NOT in scope: written
- What already exists: written
- TODOS.md updates: 0 items proposed to user
- Failure modes: 0 critical gaps flagged
- Unresolved decisions: 0 in this review
- Outside voice: codex unavailable (sandbox could not start); in-host fallback reviewer completed, 10 findings, 2 decided by the owner, 8 applied as corrections or recorded
- Parallelization: 2 lanes, 2 parallel / 2 sequential steps
- Lake Score: 1/4 answers picked a 10/10 option (D6); D2, D3 and D4 did not

### Suppressed findings
None.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Outside Review | codex, default-on after `/plan-eng-review` | Independent 2nd opinion | 1 | unavailable | sandbox could not start; in-host fallback: 10 findings |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | ISSUES OPEN | 24 issues, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | — |

- **OUTSIDE COVERAGE:** codex, plan-review phase, unavailable (bwrap loopback not permitted). The in-host fallback reviewer completed with 10 findings; it is not outside coverage.
- **VERDICT:** no review is CLEAR. Eng Review ran with every finding mapped to an approved decision or a task; eng review required to read CLEAR after the tasks land.

NO UNRESOLVED DECISIONS
