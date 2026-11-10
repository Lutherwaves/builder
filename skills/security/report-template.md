# security — report template

Use these headings word for word, in this order. The person and other agents scan for them.
Leave out "Since the last audit" only when no baseline exists and the mode is change.

## What I ran

One short paragraph: the mode (change or audit), the profile, the agent cap if the person set one, the base and head (or the audited SHA), and each command with its result.
Name the build, the tests, every probe, each verifier, and the `builder security score` and `baseline` calls.

## Coverage

Rows are the OWASP Top 10:2025 categories. Columns are the OSI layers. Each cell is one word:

- `inspected`: you looked, and the findings below are what you found.
- `not inspected`: you did not look. Give the reason under Limits.
- `provider`: the layer belongs to the provider. Name the provider under Limits.
- `n/a`: the category cannot occur at that layer.

| | L1–L2 | L3–L4 | L5–L6 | L7 |
|---|---|---|---|---|
| A01 Broken Access Control | | | | |
| A02 Security Misconfiguration | | | | |
| A03 Software Supply Chain Failures | | | | |
| A04 Cryptographic Failures | | | | |
| A05 Injection | | | | |
| A06 Insecure Design | | | | |
| A07 Authentication Failures | | | | |
| A08 Software or Data Integrity Failures | | | | |
| A09 Security Logging and Alerting Failures | | | | |
| A10 Mishandling of Exceptional Conditions | | | | |

Under the table, give one line for the dimensions with no OWASP row: memory, availability and denial of wallet, agent and tool boundaries.

In audit mode, add the surface map before the table: one row per public, expensive or webhook surface, and one row per CI workflow.
Each row gives the protections the surface has (auth, signature, rate limit, size cap, timeout). Write `none` where a protection is missing.

### Coverage units

One row per unit, from the findings file. A quick or scoped run lists only its units, and says that the rest is not covered by this run.

| Unit (`<surface>/<dimension>`) | Status | Reason |
|---|---|---|
| `webhook-payments/data-integrity` | covered | |
| `search/availability` | deferred | <why: the budget ended, the cap was reached, ...> |

Status is `covered`, `deferred`, `out_of_scope` or `not_inspected`. Every status except `covered` has a reason.

## Most important first

One row per record that is not rejected, worst first. Sort by real harm. Use the escalated severity where there is one.

| # | Headline | State | Severity |
|---|---|---|---|
| 1 | <harm to a person> | confirmed | <Critical, High, Medium or Low> |
| 2 | <harm to a person> | needs validation | — |

A `needs_validation` record has no severity. A rejected record goes under Declined to judge.

## The findings

One entry per confirmed record:

```
**<One-line headline that states the harm to a person.>** `path/to/file:line` — <Critical | High | Medium | Low>
State: confirmed
Verified by: <a second agent, and what it ran | not independently verified, and why>
What goes wrong: <who, in which concrete situation, and what they lose>.
Standards: <CWE-n (Top 25)> · <CAPEC-n> · <OWASP Axx:2025> · <APIx:2023> · <ASVS 5.0 Vn> · <VRT category> · <NIST control> · <CIS safeguard>
CVSS: <vector> → <score> <rating> (computed by builder security score)
Escalation: <none | why the real blast radius is bigger than the base score, and the severity you use>
Prevention: <plan | design | implement | review>: <the control that would have stopped it there>
Evidence: <measured: probe, input → output | from reading> · <static | live>
Fix: <the smallest change that removes the harm>.
Test: <test name>: asserts <what>.
```

One entry per `needs_validation` record. It has no CVSS line and no severity:

```
**<One-line headline that states the possible harm.>** `path/to/file:line`
State: needs validation
Missing: <the exact fact that the source does not show> · Who can observe it: <the person or the role, and where they look>
What goes wrong if the fact is no: <who, in which concrete situation, and what they lose>.
Fix if the fact is no: <the smallest change that removes the harm>.
```

Take the severity word from the CVSS rating, or from the escalation when there is one.
Take each identifier from `standards.md`. Copy the score, the rating and the Top 25 mark from the `score` output.

## Since the last audit

From `builder security baseline diff` (change mode) or `baseline save` (audit mode):

- New: <ids>.
- Still open: <id>, open <n> days since <first_seen> (<first_sha>).
- Closed: <ids>. Audit mode only: a change review never closes a gap.
- Not re-checked: <ids>, with the dimension that did not run. In change mode, this is every known gap the diff did not touch.
- Units carried: <ids> that the last audit deferred or did not inspect, and that this run did not name.
- Units changed: <ids> that the last audit covered, and whose files changed since.

Write "No earlier audit for this repository." when no baseline exists.
In audit mode, add the baseline path from the `saved` field.

## Limits

Each `not inspected` and `provider` cell, with its reason. Name what a live test would add that this run could not.

## Declined to judge

Each thing you considered and set aside, one line each, with the reason.
Each rejected record goes here, with the reason from the verifier or the source.

## Verdict

Safe to ship, safe with fixes, or not yet. Add a one-line reason.
Then: "Posted nothing. The checkout is unchanged."
Offer the next step in one line: "Turn the fixes into a patch, or file them as issues?"
