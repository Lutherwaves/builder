# security — the full security stance in one run

Issue: #21. Date: 2026-10-24.

## Intent

A person runs one security command and gets the complete stance of their code, with no extra prompting.
Today an agent security review covers a familiar subset: injection, authn/z and secrets.
It explicitly drops availability, denial of service, resource exhaustion, memory safety and infrastructure posture.
It reviews only the diff, so it cannot see a control that is absent everywhere ("no rate limiter anywhere") or a posture gap.
Real, high-impact gaps go unreported, and the person does not know they were never inspected.

The model is Claude Code's built-in `/security-review`: high confidence, a concrete exploit path, a false-positive filter.
`security` keeps that bar and adds what that review excludes, a whole-system mode, a standards mapping, a computed score and a memory of the last audit.

## Goals

1. **Per-change mode** (the default when there is a diff, a branch or a PR). It guards against new regressions.
2. **Whole-system mode** (no diff, or the person asks for an audit). It maps every public, expensive and webhook surface to its protections and reports each surface with none. It also checks infra posture.
3. **Dimensions, all covered by default:**
   - OSI L1–L7, scoped honestly. L1/L2 are usually the provider's; say so and move on. L3/L4: volumetric, connection and stream exhaustion. L5/L6: TLS and crypto posture. L7: application logic, the bulk.
   - Memory safety and resource exhaustion, language-aware: unbounded reads, decompression, zip, XML and parser bombs, large allocations, event-loop blocking, pixel, entry and recursion caps.
   - Availability, DoS and denial of wallet: rate and quota limits on expensive and public endpoints, per-actor concurrency budgets, autoscale caps tied to real downstream quota, edge DDoS posture.
   - The usual set: authn and sessions, authz, tenancy and isolation, injection, SSRF, secrets and data handling, CI/CD and supply chain, agent and tool boundaries, data integrity.
4. **Each finding carries:**
   - CWE (flag the CWE Top 25), CAPEC, OWASP Top 10, OWASP API Security Top 10, OWASP ASVS, Bugcrowd VRT, NIST SP 800-53 / CIS.
   - A CVSS 3.1 vector and its base score. Code computes the score, never the model.
   - An escalation flag when the base score understates the real blast radius (for example, a "Medium" DoS on the only payment path).
   - A prevention note: the cheapest SDLC phase to have stopped it (plan, design, implement, review) and the specific control.
   - Evidence marked "measured" (a probe ran) or "from reading", and "static" or "live".
5. **The report:** an OWASP × OSI coverage matrix (what was inspected and what was not), the scored register, and the limits stated plainly.
6. **Memory:** an audit baseline in builder's local state. The next run diffs against it. New gaps surface, fixed gaps close, and each open gap has a dated exposure window.

## Non-goals

- No attack on a live or remote system. A remote target needs the person's explicit go, each time.
- No scanner integration (Semgrep, Trivy and so on) in this version. The skill may use a scanner the person already has; it installs nothing.
- No auto-fix. The skill proposes the smallest fix and the test that pins it, like `review`.
- No posting. A comment, issue or advisory leaves the machine only on a yes.
- No change to `review`'s method. `review` stays the general reviewer.

## Anti-goals (failure even if it works)

- A fabricated exploit, an invented CWE or an inflated severity.
- A matrix cell marked "inspected" that the run never looked at.
- An open-gap list committed to a public repository by default.
- A report so long that the one Critical drowns. Worst first, always.

## Constraints

- builder conventions (AGENTS.md): a skill has 60–150 lines, `## Common mistakes` and `## Self-check`, ASD-STE100 prose, and a "Use when" description.
- A skill change starts with an eval. Run the fixture without the skill, watch it fail, then write the skill.
- Library code uses the Python 3 standard library only, with unit tests under `tests/`.
- The repository is public. No private names in files, commits or the PR.
- State lives under `$BUILDER_HOME` (default `~/.config/builder`).

## Approaches

| | Approach | Trade-off |
|---|---|---|
| **A (chosen)** | One new `security` skill + two reference files + `lib/security.py` (CVSS and baseline) with unit tests | The most files, but the score and the baseline diff are enforced by tests, and the finding format exists once |
| B | A security lens inside `review` + a separate `audit` skill | `review` is 114 lines and would pass the cap. The standards map and finding format would live in two places |
| C | Prose only | The model computes CVSS in its head and keeps a freeform baseline. This breaks "no inflated severities", and the diff is fuzzy |

The person chose A on 2026-10-24: it is the only option where the rules that matter have a test.

## Decided

- **Name:** `security`. Location: `skills/security/`.
- **Files:**
  - `skills/security/SKILL.md`: the method for both modes, 60–150 lines.
  - `skills/security/standards.md`: the map from vulnerability class to CWE, CAPEC, OWASP Top 10, API Top 10, ASVS, VRT and NIST/CIS; the CWE Top 25 list; OSI scoping notes; memory-safety checks per language (Python, Node/TS, Go, Rust, Java/JVM, C/C++).
  - `skills/security/report-template.md`: the matrix, the register entry, the baseline-diff section and the limits section.
  - `lib/security.py`: `cvss` (vector → base score and rating, per the CVSS 3.1 specification) and `baseline` (`save`, `diff`).
  - `tests/test_security.py`: unit tests for both.
  - `tests/evals/security/`: the eval fixture.
- **Probing:** like `review`. The skill may start the app in a throwaway worktree and probe `localhost` only (for example, a 50 MB body or a zip bomb). Anything else needs an explicit go. Each finding says measured or from reading.
- **Baseline location:** `$BUILDER_HOME/security/<key>.json`, local to this machine and never committed by default. The person can opt in to commit it in a private repository.
- **Baseline content:** the repo, the audited SHA, the mode, the date, the dimensions that ran, and each finding (a stable id, class, CWE, location, score, status, first-seen date and SHA).
- **Diff:** `new`, `still open` (with days open since first seen) and `closed` (no longer found at the new SHA).
- **CLI:** `python3 lib/security.py cvss <vector>` prints the score and rating, and exits non-zero on an invalid vector. `python3 lib/security.py baseline diff|save <repo> <findings.json>` reads the agent's findings, prints the diff, and saves on `save`.
- **Routing:** the builder routing row "Check security" points to builder `security`, with gstack `/cso` as the fallback. README, AGENTS.md (the "where each change goes" table), CONTRIBUTING and `.claude-plugin/plugin.json` gain the skill the way #19 added `review`.
- **Eval fixtures, two, one per mode.** Both use a small web service written with the Python standard library.
  - `tests/evals/security/` covers per-change mode. The diff adds an upload endpoint and a paid-API proxy endpoint. Seeded gaps:
    1. An unbounded request-body read (memory exhaustion, CWE-770).
    2. A zip extraction with no size or entry cap (decompression bomb, CWE-409).
    3. The new paid proxy has no rate limit, and no limiter exists anywhere in the system (denial of wallet, CWE-770/799).
    4. A TLS context with verification off on an outbound call (CWE-295).
  - `tests/evals/security-audit/` covers whole-system mode. There is no diff. The prompt asks for an audit. Seeded gaps: a public webhook with no signature check, an expensive search endpoint with no limit, and a CI workflow that runs untrusted PR code with secrets.
  - Each `expect.md` has 3–6 bullets (the `run.sh --check` cap). Between them they assert the matrix, CWE and CVSS vectors with computed scores, the SDLC note, the measured-versus-reading marks, the absence-of-control finding, and "posted nothing".

## Calls after the builder check (2026-10-24)

A fresh reviewer read this design as its builder. These are its five questions and the call on each.

1. **Per-change mode sees an absence only where the change needs a control.** When a diff adds or widens a public, expensive or webhook surface, per-change mode checks that surface's protection system-wide ("this new endpoint has no limit, and nothing in the system provides one"). It does not sweep surfaces the diff never touched. That is whole-system mode's job. Reason: a missing limiter on the new endpoint is a regression the change introduces.
2. **Only whole-system mode saves the baseline.** Per-change mode reads it, to mark a finding "already known since <date>", and never writes it. There is one baseline per repo.
   `diff` marks a finding `closed` only when its dimension ran in the new audit and the finding is gone. When the dimension did not run, the finding is `not re-checked`, and its first-seen date stays.
   Reason: a narrow run must never close a gap it never looked at (the "inspected but never looked at" anti-goal).
3. **Memory is proved by unit tests, not by an eval.** `tests/test_security.py` covers save, `new`, `still open` with days open, `closed` and `not re-checked`. The eval runner stays unchanged (a fresh `BUILDER_HOME` for each run).
   Reason: changing the shared eval infrastructure is outside this scope.
4. **Probes run in process, so the fixtures pass on every harness.** A "measured" mark comes from a test file in the throwaway worktree that calls the handler directly, the way `review` probes. A real `localhost` server stays allowed where the harness permits it. The fixtures never need it.
   Reason: the codex sandbox blocks network, and a harness counts as supported only when every fixture passes.
5. **Pinned editions, and code checks the identifiers it can.** `standards.md` names its editions at the top: OWASP Top 10:2025, OWASP API Security Top 10 2023, ASVS 5.0, CWE Top 25 2025, NIST SP 800-53 Rev. 5, CIS Controls v8.1, Bugcrowd VRT (current). `lib/security.py` rejects a malformed CWE id and computes the Top 25 flag from a pinned list. The model never asserts the flag.
   Checking every CWE against the full catalogue is a non-goal, because that needs a large data file.

## Left to the builder

- The exact stable-id scheme for a finding (it must survive line moves; class + CWE + file + symbol is a sensible start).
- The repo key for the baseline file (the realpath of the repo root, hashed, is a sensible start).
- The order of the hunt list inside SKILL.md, and which detail moves to `standards.md` to stay under 150 lines.
- The fixture's code, as long as it seeds the four gaps above.
