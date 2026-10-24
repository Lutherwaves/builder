---
name: security
description: Use when the person asks for a security review, a security audit, a threat model or the posture of a repository ("is this secure?", "security review this PR", "audit before launch", "are we safe from DoS?"), or when a change touches authentication, money, uploads, webhooks, secrets, CI or an agent's tools.
---

# security — the full stance in one run

A security review that checks only injection, auth and secrets is a partial review. Most reviews do not say that they are partial.
`security` covers every layer by default: OSI L1 to L7, memory and resource exhaustion, availability and denial of wallet, and the usual application set.
Each finding is mapped to the standards and gets a CVSS 3.1 score that code computes. An audit remembers what it found, so the next audit says what changed.

## The rules

- Prove the harm. Name the person, the situation and what they lose. Report only a concrete path to harm. A finding with no harm is a nit. Drop it.
- Never invent what code can compute. The score, the rating, the Top 25 mark and the baseline diff come from `security.py`.
- Probe only where nothing real can break. Use a throwaway worktree: `git worktree add <temp dir>/sec HEAD`, or `.security-wt` inside the folder when your tools reach only it.
  Prefer an in-process probe: a test file that calls the handler with the hostile input. A local server on `localhost` is allowed.
  A remote, shared or live system needs the person's explicit go each time. Remove the worktree at the end.
- Defensive only. Write no working exploit for a live target, and do not inflate a severity.
- Post nothing. A comment, an issue or an advisory leaves the machine only after the person's yes.

## 1. Choose the mode

- **Change mode**: there is a diff, a branch, a commit range or a PR. Review what the change adds or alters.
  When the change adds or widens a public, expensive or webhook surface, also check that surface's protection across the whole system.
  Example: "the new endpoint has no limit, and nothing in the system provides one".
- **Audit mode**: there is no diff, or the person asks for an audit. Review the whole repository and its infra config.
- When you are not sure, use change mode and offer the audit in the verdict.

## 2. Map the surface

1. Read the README, the deploy and infra files (Dockerfile, compose, Terraform, proxy config, CI workflows) and the route table.
2. List each public, expensive or webhook surface, and each CI workflow. Expensive means CPU, memory, disk, a paid API or someone's time.
3. For each surface, write its protections: auth, signature, rate or quota limit, size cap, timeout, concurrency budget. Write `none` where one is missing.
   In audit mode, this map goes in the report. A `none` on a public or expensive surface is a finding by itself.

## 3. Hunt, by dimension

Go through each dimension. `standards.md` has the classes, the OSI notes and the per-language checks.

1. **OSI L1–L2**: usually the provider's. Name the provider and move on. Do not force a finding.
2. **OSI L3–L4**: edge DDoS protection, open ports, connection and stream limits, slowloris, server timeouts.
3. **OSI L5–L6**: who ends TLS, certificate checks on every outbound call, weak crypto, cookie flags, parsers of hostile encodings.
4. **Memory and exhaustion**: unbounded reads, archive, XML and parser bombs, large allocations from an input size, recursion depth, pixel and entry caps, blocking work on an event loop.
5. **Availability and denial of wallet**: rate and quota limits on public and expensive endpoints, per-actor concurrency, fan-out to paid services, autoscale caps against real downstream quota.
6. **Authn and sessions; authz, tenancy and isolation**: every route has an owner check, and every query is scoped to its tenant.
7. **Injection and SSRF**: SQL, shell, template, eval, deserialization, prompt injection, outbound URLs built from input.
8. **Secrets and data**: committed keys, secrets in logs, URLs and errors, personal data to the wrong actor.
9. **Supply chain and CI**: workflows that run untrusted code with secrets, unpinned actions and images, known-vulnerable dependencies.
10. **Agent and tool boundaries**: tools with more power than their task, model output used as a command, untrusted text in a prompt.
11. **Data integrity**: unsigned webhooks, races, error paths that fail open, missing security logs.

When you suspect a gap, confirm it with a probe. Use the project's own examples as inputs: the README, the docs, the description.
Mark each finding "measured" (a probe ran: quote the input and the output) or "from reading". Never mark a guess as measured.

## 4. Score

1. Write the findings file: JSON with `mode`, `repo` (absolute path), `sha`, `dimensions` (the ones you covered) and `findings`.
   Each finding has `id` (`<dimension>/<CWE>/<file>/<symbol>`), `dimension`, `cwe`, `location`, `vector` (`CVSS:3.1/...`) and `title`.
   Put it outside the repository, for example in a temp dir.
2. Run `python3 "${CLAUDE_PLUGIN_ROOT}/lib/security.py" score <findings.json>`. It adds `score`, `rating` and `top25`, or names the bad field.
   Fix the field and run it again. Never write a score yourself.
3. Escalate when the base score understates the blast radius, for example a Medium DoS on the only payment path. Say why. Keep the computed score next to it.

## 5. Remember

- Audit mode: run `python3 "${CLAUDE_PLUGIN_ROOT}/lib/security.py" baseline save <findings.json>`. It prints new, still open, closed and not re-checked, and the path it saved.
- Change mode: run `baseline diff` instead. It saves nothing. Use it to mark a finding "already known since <date>".
- The baseline lives in builder's local state, never in the repository. Commit it only on the person's request, and only in a private repository.
- A corrupt baseline is an error. Report it with its path. Never treat it as empty.

## 6. Report

Use `report-template.md`: What I ran · Coverage · Most important first · the findings · Since the last audit · Limits · Declined to judge · Verdict.
Each `inspected` cell in the matrix is a place you looked. If you did not look, write `not inspected` and give the reason under Limits.
If you found nothing important, say so plainly. Do not pad the report.

## Common mistakes

| Mistake | Fix |
|---|---|
| "DoS and rate limiting are out of scope" | They are in scope. Check availability and denial of wallet on every public or expensive surface. |
| Reviewing only the diff for a missing control | A missing limiter lives in no diff. Check the surface system-wide, or run the audit. |
| A CVSS score written by hand | Run `security.py score`. Copy its output. |
| A CWE or CAPEC id from memory | Take it from `standards.md`. If no row fits, check the source and say so. |
| A matrix full of `inspected` | Mark only what you looked at. The rest is `not inspected`, with its reason. |
| A forced finding at L1 or L2 | Name the provider. Write `provider`. |
| A probe against a live or shared host | Probe in process, or on `localhost`. A remote target needs a go each time. |
| Saving the baseline from a change review | Only an audit saves. A change review runs `baseline diff`. |
| The baseline committed to a public repository | It is a list of open holes. Keep it in local state. |
| Posting an issue or advisory to "finish" | Write the report here. Post only after a yes. |

## Self-check

- [ ] The report names its mode and opens with what I ran.
- [ ] Every OSI layer and every dimension of step 3 has a line in the matrix or under Limits.
- [ ] Each public, expensive or webhook surface the run touched has its protections listed.
- [ ] Each finding has a harm headline, `file:line`, the standards line, a CVSS vector with a computed score, a prevention note, evidence, a fix and a test.
- [ ] Each "measured" quotes a probe, an input and an output.
- [ ] Audit: the baseline is saved and its path is named. Change: nothing was saved.
- [ ] I ran nothing in the checkout. The throwaway worktree is removed.
- [ ] Nothing reached the forge without the person's yes.
