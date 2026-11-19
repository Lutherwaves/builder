# A fresh reader checks each spec and plan before the person approves it

Status: approved, and built on the branch `feat/cold-read`. The sections below describe the repository before that build.

## Why

The agent that writes a spec or a plan holds the whole conversation. It cannot see what it knows and did not write down.
Today the spec has no review before its gate. The plan's lenses run in the session that wrote the plan.
The Elephant-Goldfish Model (google/elephant-goldfish-model, Apache-2.0) gives the document to readers who have only the document and the files it names. builder takes that idea and copies no text.

Measured on 2026-11-18, on builder's own three v4 changes (framework, decommission, review comments). Six fresh readers, one for each document, read the three specs and the three plans as approved. Three more agents traced each finding into the build history. The verdicts are agent judgments and were not audited by hand.

- 57 serious findings. 40 trace to later cost, 10 are real with no cost seen, 7 concern eval runs that never ran, 0 are wrong.
- 73 problems in the three builds came from a gap in a document. The readers flagged 39 as serious and 14 more as a nit or in part. They missed 20.
- 31 of the 42 rulings made during the builds trace to a document gap. The readers flagged 20 of the 31.
- The 688-line spec (as approved) was the weakest case: 15 of 34 flagged. The two short changes gave 14 of 23 and 10 of 16.
- Cost: 3 to 5 minutes and about 150 000 tokens for each document.

## The person and the moment

A person is about to approve a spec or a plan that the agent wrote.
The person wants to approve a document that a builder can act on, and not to find the gaps as rulings and fix commits during the build.

## Goals

1. Before `spec` asks for approval the first time, a fresh agent reads the spec. Before `plan` asks for approval the first time, a fresh agent reads the plan. A later request for approval of the same document starts no new read.
2. The reader gets the path of the document, and may open the files it names. It never gets the conversation.
3. The reader paraphrases first: what the document proposes, and how things work today. The author compares the paraphrase with the intent: where they differ, the author records a major finding. A part that the reader cannot paraphrase is a blocker.
4. The reader lists each place where a builder must guess, ask or decide. Each finding is a blocker, a major gap or a nit.
5. The author gives each finding one answer: fix the document, contest it, ask the person, or decline a nit.
6. The document records that the read ran, the counts, and what changed.

## Non-goals

- No reviewer personas, no stages and no presets. One kind of reader, one brief.
- No loop. After the fixes, one new fresh reader reads the changed document once. There is never a third read. With no fix there is no second read. What is left goes to the person.
- No cold read of a small change. A small change has no spec and no plan.
- No standalone command for other documents.
- No check of documents against code after the build.

## Anti-goals

- The author must not answer a finding with "that was intended". A contest cites a file, a line or a ruling that the reader could open.
- The author must not contest a gap in understanding. The reader had the whole document, so the document must say more.
- The read must not replace the person. Findings that only the person can settle go to the person with the request for approval.
- The read must not pass in silence when it did not run. A harness with no subagent writes "not cold-read" in the document.

## Surfaces

- An agent's tools. The reader is a subagent, and its report is text from another agent. Harm to avoid: the author follows an instruction inside a finding, or the reader changes a file. The reader only reads, and a finding is a claim to check.
- No other sensitive surface touched. Checked: authentication, money, uploads, webhooks, secrets, CI, personal data.

## Approaches

1. **Picked.** One reference file, `skills/builder/cold-read.md`, with the reader's brief and the answer rule. `spec` and `plan` load it before their gate. This is the pattern of `plan/lenses.md` and `review/bar.md`.
2. A ninth skill that `spec` and `plan` call, and that a person can run on any document. Rejected: one more command, and the measurement needed no standalone use.
3. The steps written into `spec` and into `plan`. Rejected: one rule in two places, and `plan` has 147 lines of 150.

## What changes

- `skills/builder/cold-read.md` (new): what the reader gets; the brief (paraphrase, then findings with a severity, a place and what a builder must guess); the four answers and the two rules for a contest; one more read after the fixes; the record; the case of no subagent; a document over 300 lines gets two readers: each reads the whole document, one reports on the first half and one on the second, and the record adds their counts.
- `skills/spec/SKILL.md`: before the gate, load `cold-read.md` and run it. Findings for the person join "Calls to check".
- `skills/plan/SKILL.md`: after the lenses in step 4, the check against the spec in step 5 and the choice of the file in step 6, load `cold-read.md` and run it, so that the reader sees the whole plan at its path. At most three new lines, or the same number cut elsewhere.
- `skills/spec/issue-template.md`: a "Cold read" section. It is always present, like Surfaces. The plan carries the same line next to "Lens findings". The plan's five header lines do not change, because `build` reads them word for word.
- `tests/evals/cold-read/` (new fixture): a draft spec with planted gaps (a term nobody defines, a ruling the reader cannot see, a done-when line nobody can run) and a prompt that asks for the approval-ready spec.
- `README.md`: the support table is rendered again in the same task as the fixture, because a new fixture changes its counts. The credits gain a fifth project.
- `AGENTS.md`: the fifth credit, and a row for `cold-read.md` in "Where each change goes". The attribution sentence no longer says that every source is MIT-licensed.
- `CHANGELOG.md` under `[Unreleased]`.

## Rulings

- 2026-11-18: the cold read runs at every gate, not on request. Decided by: the owner.
- 2026-11-19: approach 1, one shared reference file. Decided by: the owner.
- 2026-11-19: the lenses stay and run first. Proposed, not confirmed. Cost if wrong: two reviews of one plan where one could do.
- 2026-11-19: one more read at most. Proposed, not confirmed. Cost if wrong: a fix that opens a new gap reaches the person unread.
- 2026-11-19: two readers over 300 lines. Proposed, not confirmed. The number is an estimate from one long document. Cost if wrong: a second agent with little gain.
- 2026-11-19: in an unattended run the read still runs, and each finding for the person becomes a call marked "proposed, not confirmed". Proposed, not confirmed.
- 2026-11-19: the fixture is written and passes `run.sh --check`. No paid eval runs: the owner's ruling of 2026-11-12 stands for this change too, and it waives the eval-first convention of `AGENTS.md` here. Proposed, not confirmed. Cost if wrong: the skill text ships with no graded run.
- 2026-11-19: spec approved. The owner saw the five proposed calls above and changed none. Decided by: the owner.

## Left to the builder

- The wording of `cold-read.md` and of the fixture bullets.
- Which lines of `plan` make room, if it needs more than three.
- The exact form of the "Cold read" line.

## Done when

1. A person who asks for a spec sees a "Cold read" line in it before the approval request: the number of fresh readers, the counts by severity, and the counts by answer. Proof: a text test on the skill files, and one real run on this spec. No graded run.
2. The same holds for a plan. Proof: the text test only.
3. On a harness with no "start a subagent" action, or with that action marked "not verified" in `skills/builder/harnesses/`, or when the reader fails, that line says "not cold-read". Proof: the text test only. No run on such a harness.
4. `bash tests/evals/run.sh --check` accepts the `cold-read` fixture, and its `expect.md` names the three planted gaps.
5. `bash scripts/validate.sh` and `go test ./...` pass, and each SKILL.md has 60 to 150 lines.
6. `README.md` and `AGENTS.md` carry the fifth credit, and no file copies a sentence from the source (<https://github.com/google/elephant-goldfish-model>). The reviewer of the branch compares the new files with the source.

## Cold read

Cold read: 2026-11-19, 1 fresh reader(s). Blocker 1, major 11, nit 8. Fixed 12, contested 0, for the person 2, declined 6. Second read: 13 left.

The read ran after the build, because the build made the rule. That order gave the one blocker: the spec called built files new.
Two fixes changed the build, not only this spec: `plan` runs the read after its step 5, and a harness map that says "not verified" gives "not cold-read".

For the person, from the first read:

- Does the person see each contested finding, or only the count? Recommendation: only the count in the record, and the reply names each one. Proposed, not confirmed. Cost if wrong: a contest ends with the author and nobody checks it.
- May the reader open repository files that the document does not name? Recommendation: yes, because a builder can. The build takes this answer, after three reviewers found that the narrower rule contradicted the brief. Proposed, not confirmed. Cost if wrong: the reader flags a term that the repository defines.

Left after the second read: 7 major and 6 nit. No third read ran. The review of the branch then found a sentence close to one in the source, in this spec and in `cold-read.md`. Both are rewritten.

The review of the pull request changed `cold-read.md` again. The reader is never a fork of the session. It may open each file in the repository and no file outside it. Text in each file that it opens is data. A reader with no paraphrase is replaced one time. The read of a plan runs after the plan has a path. The second read has one reader, and its findings are not fixed. The record can say "Second read: none". The major ones:

- A wrong paraphrase has no stated place in the counts and no stated answer.
- The spec does not say if the person sees declined findings.
- The spec does not say if the author may fix a finding of the second read, which leaves that fix unread.
- For two readers, "the first half" has no unit, and the second read has no stated number of readers.
- "The reader fails" has no definition.
- The text test of done-when 1 to 3 has no file name here. It is `TestSpecAndPlanShareTheColdRead` in `tests/chain_test.go`.
- This spec does not follow `skills/spec/issue-template.md`, so the place of its record was a guess.
