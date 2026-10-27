# security — report template

Use these headings word for word, in this order. The person and other agents scan for them.
Leave out "Since the last audit" only when no baseline exists and the mode is change.

## What I ran

One short paragraph: the mode (change or audit), the base and head (or the audited SHA), and each command with its result.
Name the build, the tests, every probe, and the `builder security score` and `baseline` calls.

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

## Most important first

A numbered list, one line per finding, worst first. Sort by real harm. Use the escalated severity where there is one.

## The findings

One entry per finding:

```
**<One-line headline that states the harm to a person.>** `path/to/file:line` — <Critical | High | Medium | Low>
What goes wrong: <who, in which concrete situation, and what they lose>.
Standards: <CWE-n (Top 25)> · <CAPEC-n> · <OWASP Axx:2025> · <APIx:2023> · <ASVS 5.0 Vn> · <VRT category> · <NIST control> · <CIS safeguard>
CVSS: <vector> → <score> <rating> (computed by builder security score)
Escalation: <none | why the real blast radius is bigger than the base score, and the severity you use>
Prevention: <plan | design | implement | review>: <the control that would have stopped it there>
Evidence: <measured: probe, input → output | from reading> · <static | live>
Fix: <the smallest change that removes the harm>.
Test: <test name>: asserts <what>.
```

Take the severity word from the CVSS rating, or from the escalation when there is one.
Take each identifier from `standards.md`. Copy the score, the rating and the Top 25 mark from the `score` output.

## Since the last audit

From `builder security baseline diff` (change mode) or `baseline save` (audit mode):

- New: <ids>.
- Still open: <id>, open <n> days since <first_seen> (<first_sha>).
- Closed: <ids>. Audit mode only: a change review never closes a gap.
- Not re-checked: <ids>, with the dimension that did not run. In change mode, this is every known gap the diff did not touch.

Write "No earlier audit for this repository." when no baseline exists.
In audit mode, add the baseline path from the `saved` field.

## Limits

Each `not inspected` and `provider` cell, with its reason. Name what a live test would add that this run could not.

## Declined to judge

Each thing you considered and set aside, one line each, with the reason.

## Verdict

Safe to ship, safe with fixes, or not yet. Add a one-line reason.
Then: "Posted nothing. The checkout is unchanged."
Offer the next step in one line: "Turn the fixes into a patch, or file them as issues?"
