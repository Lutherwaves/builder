# security skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A new builder `security` skill gives the full security stance in one run: per-change and whole-system modes, every dimension of #21, standards-mapped findings with a code-computed CVSS score, and a local audit baseline that later runs diff against.

**Architecture:** `skills/security/SKILL.md` holds the method. Two reference files beside it hold the standards map and the report shape. `lib/security.py` (Python standard library only) owns everything the model must not invent: the CVSS 3.1 score, the CWE id check, the CWE Top 25 flag and the baseline save and diff. Two eval fixtures prove the skill test-first, one per mode.

**Tech Stack:** Markdown skills, Python 3 standard library, `unittest`, Bash eval runner (`tests/evals/run.sh`).

**Spec:** `docs/superpowers/specs/2026-10-24-security-skill-design.md`

## Global Constraints

- A skill has 60 to 150 lines, frontmatter `name` + a `description` that starts "Use when", and the sections `## Common mistakes` and `## Self-check` (`scripts/validate.sh:69-87`).
- Prose in ASD-STE100: active voice, present tense, one idea per sentence.
- Library code: Python 3 standard library only. Tests: `unittest`, in `tests/test_<name>.py`, importing from `lib/` via `sys.path.insert` (as `tests/test_ledger.py` does).
- State lives under `harness.builder_home(os.environ, Path.home())`. The baseline path is `<builder home>/security/<key>.json`.
- The repository is public. Keep employer, client, product and private repository names out of files, commits and the PR. Grep the staged diff before each commit.
- Commits: signed Conventional Commits, never `--no-verify`, no WIP commits. Never edit `CHANGELOG.md` or a manifest `version`.
- Never commit an eval transcript.
- A skill change starts with an eval: run the fixture without the skill, watch it fail, then write the skill.
- Pinned editions (spec, call 5): OWASP Top 10:2025, OWASP API Security Top 10 2023, ASVS 5.0, CWE Top 25 2025, NIST SP 800-53 Rev. 5, CIS Controls v8.1, Bugcrowd VRT (current).
- Before every commit: `bash scripts/validate.sh` passes.

## Review Focus

1. **A CVSS vector with metrics out of order, a duplicate metric, a missing metric or a `CVSS:3.0/` prefix.** Expect: order does not matter, and a duplicate, a missing metric or a wrong prefix is rejected with the bad part named. Pinned in Task 1.
2. **First audit on a repo (no baseline file) and a corrupt baseline file.** Expect: the first run reports every finding as `new`. A corrupt file is an error that names the path, never a silent empty baseline (tenet 7: never invent an empty list). Pinned in Task 2.
3. **A per-change run calls `baseline save`.** Expect: refused with exit 2 and "only an audit saves the baseline". Pinned in Task 2.
4. **The same repo reached through a symlink or a trailing slash.** Expect: the same baseline key. Pinned in Task 2.
5. **Two findings with the same id in one findings file.** Expect: rejected, naming the id, because the diff keys on it. Pinned in Task 2.

---

## The findings file (shared contract)

Tasks 1, 2 and 4 use this JSON shape. The agent writes it. The library enriches, checks and diffs it.

```json
{
  "mode": "audit",
  "repo": "/abs/path/to/repo",
  "sha": "<40-hex commit>",
  "dimensions": ["osi-l7", "memory", "availability"],
  "findings": [
    {
      "id": "memory/CWE-409/app/upload.py/extract",
      "dimension": "memory",
      "cwe": "CWE-409",
      "location": "app/upload.py:41",
      "vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
      "title": "One uploaded zip fills the disk and stops the service"
    }
  ]
}
```

- `mode` is `"change"` or `"audit"`.
- `dimension` and each `dimensions` entry come from `security.DIMENSIONS`:
  `("osi-l1-l2", "osi-l3-l4", "osi-l5-l6", "osi-l7", "memory", "availability", "authn", "authz", "injection", "ssrf", "secrets", "supply-chain", "agent-boundary", "data-integrity")`.
- `id` is `<dimension>/<CWE>/<file>/<symbol>`, so it survives line moves (spec: left to the builder, settled here).

---

### Task 1: The score, the CWE check and the Top 25 flag are computed, never asserted

**Files:**
- Create: `lib/security.py`
- Test: `tests/test_security.py`

**Interfaces:**
- Produces:
  - `parse_vector(vector: str) -> dict[str, str]`: raises `ValueError` whose message names the bad part.
  - `base_score(vector: str) -> float`: CVSS 3.1 base score, one decimal.
  - `rating(score: float) -> str`: `"None"` (0.0), `"Low"` (0.1–3.9), `"Medium"` (4.0–6.9), `"High"` (7.0–8.9), `"Critical"` (9.0–10.0).
  - `check_cwe(cwe: str) -> str`: returns the normalised `"CWE-<n>"`. Raises `ValueError` unless the input matches `^CWE-[1-9][0-9]*$`.
  - `TOP25_2025: tuple[str, ...]`, and `in_top25(cwe: str) -> bool`.
  - `DIMENSIONS: tuple[str, ...]` (the list above).
  - `enrich(doc: dict) -> dict`: validates the findings file (mode, each dimension, unique ids, CWE, vector). It adds `score`, `rating` and `top25` to each finding. It raises `ValueError` on the first problem.
  - CLI: `python3 lib/security.py cvss <vector>` prints `<score> <rating>` (e.g. `9.8 Critical`) and exits 0, or prints the error to stderr and exits 2.
  - CLI: `python3 lib/security.py score <findings.json>` prints the enriched JSON to stdout, or the error and exit 2.

- [ ] **Step 1: Write the failing tests** in `tests/test_security.py`, class `CvssTest`:
  - `test_known_scores`: each vector → its NVD score:
    - `AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H` → 9.8
    - `AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H` → 10.0
    - `AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H` → 7.5
    - `AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N` → 6.1
    - `AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H` → 7.8
    - `AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:L/A:N` → 6.4
    - `AV:P/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N` → 1.6

    Each also has the `CVSS:3.1/` prefix.
  - `test_no_impact_is_zero`: `C:N/I:N/A:N` → 0.0 and rating `"None"`.
  - `test_order_does_not_matter`: shuffled metrics give the same score.
  - `test_rejects` (with `subTest`): `CVSS:3.0/…`, a duplicate `AV`, a missing `A`, `AV:X`, an empty string. Each raises `ValueError`, and the message contains the bad token.
  - `test_rating_bounds`: 0.1 → Low, 3.9 → Low, 4.0 → Medium, 6.9 → Medium, 7.0 → High, 8.9 → High, 9.0 → Critical.
  - `CweTest.test_check_cwe`: `"CWE-79"` ok; `"cwe-79"`, `"79"`, `"CWE-079"`, `"CWE-"` raise.
  - `CweTest.test_top25`: `in_top25("CWE-79")` is True; `in_top25("CWE-409")` is False; `len(TOP25_2025) == 25`.
  - `EnrichTest`: one valid doc gains `score`/`rating`/`top25`. An unknown dimension, a bad mode, a duplicate id, a bad CWE and a bad vector each raise `ValueError`, naming the finding id.
  - `CliTest.test_cvss_cli`: `subprocess.run([sys.executable, LIB/"security.py", "cvss", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"])` → stdout `9.8 Critical\n`, exit 0. A bad vector → exit 2, stderr non-empty.

- [ ] **Step 2: Run the tests and watch them fail.**
  Run: `python3 -m unittest tests.test_security -v`
  Expected: `ModuleNotFoundError: No module named 'security'`.

- [ ] **Step 3: Implement `lib/security.py`.**
  - Score with the CVSS v3.1 specification §7 formulas and weights.
  - `PR` weights change under `S:C` (L 0.68, H 0.50).
  - Use the spec's Appendix A `Roundup` (integer arithmetic on `round(x * 100000)`), not `math.ceil(x*10)/10`. The float form gives wrong scores on some vectors.
  - `TOP25_2025`: copy the list in rank order from MITRE's published 2025 CWE Top 25 (https://cwe.mitre.org/top25/). Put the URL and the access date in a comment above it. Do not write the list from memory.
  - The module docstring follows `lib/ledger.py`: one line.

- [ ] **Step 4: Run the tests and watch them pass.**
  Run: `python3 -m unittest tests.test_security -v`
  Expected: all PASS.

- [ ] **Step 5: Commit.**
  `git add lib/security.py tests/test_security.py`
  `git commit -S -m "feat(security): compute CVSS 3.1 scores and check CWE ids in code"`

### Task 2: An audit remembers what it found, and the next audit says what changed

**Files:**
- Modify: `lib/security.py`
- Test: `tests/test_security.py`

**Interfaces:**
- Consumes: `enrich(doc)`, `DIMENSIONS` (Task 1); `harness.builder_home(env, home)` (`lib/harness.py:33`).
- Produces:
  - `baseline_path(repo: str, env: Mapping, home: Path) -> Path`: `<builder home>/security/<sha256(realpath(repo))[:16]>.json`.
  - `load_baseline(path: Path) -> dict | None`: returns `None` when the file is absent. Raises `ValueError("corrupt baseline: <path>")` on bad JSON or the wrong shape.
  - `diff(prior: dict | None, current: dict, today: datetime.date) -> dict`: returns `{"new": [...], "still_open": [...], "closed": [...], "not_rechecked": [...]}`.
    - Each entry is the finding plus `first_seen` (ISO date) and `first_sha`. `still_open` and `not_rechecked` also carry `days_open`.
    - Rules (spec, call 2):
      - an id in current but not prior → `new`, with `first_seen = today`;
      - in both → `still_open`, keeping the prior `first_seen`;
      - in prior, absent from current, and its `dimension` is in `current["dimensions"]` → `closed`;
      - in prior, absent from current, and its dimension did not run → `not_rechecked`, keeping its `first_seen`.
  - `next_baseline(prior, current, today) -> dict`: the doc to save. It holds `repo`, `sha`, `date`, `dimensions` and `open` (new + still_open + not_rechecked, each with `first_seen`/`first_sha`). Closed findings drop out.
  - CLI: `python3 lib/security.py baseline diff <findings.json> [--today YYYY-MM-DD]` enriches, loads the prior, and prints the diff as JSON. `baseline save` does the same, then writes `next_baseline` atomically (the `tempfile` + `os.replace` pattern of `lib/ledger.py:39-52`). `save` with `mode != "audit"` → stderr `only an audit saves the baseline`, exit 2.

- [ ] **Step 1: Write the failing tests**, class `BaselineTest`. Use a `tempfile.TemporaryDirectory` as `BUILDER_HOME` in an env dict.
  - `test_first_audit_all_new`: `diff(None, cur, d)`: every finding is in `new` with `first_seen == d.isoformat()`; the other lists are empty.
  - `test_still_open_keeps_first_seen`: prior first seen 2026-10-01, today 2026-10-24 → `still_open`, `days_open == 23`, `first_seen == "2026-10-01"`.
  - `test_closed_only_when_dimension_ran`: a prior `memory` finding absent from current. With `"memory"` in current dimensions → `closed`. Without it → `not_rechecked`, keeping `first_seen`.
  - `test_next_baseline_drops_closed`: `closed` ids are absent from `open`; `not_rechecked` ids are present.
  - `test_missing_file_is_none` and `test_corrupt_file_raises`: the message contains the path.
  - `test_key_is_stable`: `baseline_path` is the same for `repo`, `repo + "/"` and a symlink to `repo`.
  - `test_save_refuses_change_mode`: the CLI `baseline save` with `"mode": "change"` → exit 2, stderr contains `only an audit saves the baseline`, and no file is written.
  - `test_save_then_diff_round_trip`: save at `--today 2026-10-01`, then diff the same findings at `--today 2026-10-24` → all `still_open`, `days_open == 23`.

- [ ] **Step 2: Run the tests and watch them fail.**
  Run: `python3 -m unittest tests.test_security -v`
  Expected: the `BaselineTest` cases fail with `AttributeError` (no `diff`/`baseline_path`).

- [ ] **Step 3: Implement the interfaces above in `lib/security.py`.**

- [ ] **Step 4: Run the tests and watch them pass.**
  Run: `python3 -m unittest tests.test_security -v && bash scripts/validate.sh`
  Expected: all PASS; validate ends with no `✗`.

- [ ] **Step 5: Commit.**
  `git commit -S -m "feat(security): save an audit baseline and diff the next audit against it"`

### Task 3: Two eval fixtures fail without the skill

**Files:**
- Create: `tests/evals/security/{prompt.md,expect.md,allow}` and `tests/evals/security/repo/…`
- Create: `tests/evals/security-audit/{prompt.md,expect.md,allow}` and `tests/evals/security-audit/repo/…`

**Interfaces:**
- Consumes: the fixture format in `tests/evals/README.md`, and `tests/evals/review/` as the precedent (a `repo/pr/DESCRIPTION.md` + `repo/pr/*.diff` applied with `git apply`).

- [ ] **Step 1: Build the `security` (per-change) fixture.**
  - `repo/` is a small Python standard-library HTTP service (`http.server`), with its own `unittest` tests that pass.
  - `repo/pr/DESCRIPTION.md` claims "adds safe uploads and a summary proxy".
  - `repo/pr/feature.diff` adds the four seeded gaps of spec "Eval fixtures":
    1. `rfile.read(int(Content-Length))` with no cap;
    2. `zipfile.extractall` with no size or entry cap;
    3. a `/summarize` proxy to a paid API with no rate limit, and no limiter anywhere in `repo/`;
    4. an outbound call with `ssl._create_unverified_context()`.
  - `prompt.md`, in the voice of `tests/evals/review/prompt.md`: "Run a security review of the change in `pr/` … I can't answer questions."
  - `allow`: the review `allow` lines, plus `Bash(python3:*)`.

- [ ] **Step 2: Build the `security-audit` (whole-system) fixture.**
  - Same service style, no `pr/`.
  - Seeded gaps:
    - a public `/webhook` with no signature check;
    - an expensive `/search` with no limit;
    - `.github/workflows/ci.yml` using `pull_request_target` with a checkout of the PR head and a secret in `env`.
  - `prompt.md`: "Audit the security of this whole repository before we launch."

- [ ] **Step 3: Write each `expect.md`, 3 to 6 bullets.** Between the two files, cover:
  - an OWASP × OSI coverage matrix that marks each cell inspected or not, with L1/L2 named as the provider's;
  - each finding with CWE, CAPEC, an OWASP id and a CVSS 3.1 vector with its score;
  - an SDLC prevention note on each finding;
  - "measured" or "from reading" on each finding;
  - the absence-of-control finding (security: no limiter on `/summarize` anywhere; audit: the webhook with no signature check);
  - "posted nothing";
  - (audit only) the baseline saved, and its path named.

- [ ] **Step 4: Validate the fixtures.**
  Run: `bash tests/evals/run.sh --check`
  Expected: `fixtures ok`.

- [ ] **Step 5: Run the baseline and watch it fail.** These are paid model runs, so ask the person first.
  Run: `bash tests/evals/run.sh security` and `bash tests/evals/run.sh security-audit`
  Expected: the baseline transcripts miss most `expect.md` bullets (most likely: no matrix, no computed CVSS, no availability or memory findings). Record the met/total per fixture for the PR. Never commit the transcripts.

- [ ] **Step 6: Commit the fixtures.**
  `git commit -S -m "test(security): eval fixtures for a change review and a whole-system audit"`

### Task 4: The skill, its reference files and its place in builder

**Files:**
- Create: `skills/security/SKILL.md`, `skills/security/standards.md`, `skills/security/report-template.md`
- Modify:
  - `skills/builder/SKILL.md` (the routing row "Check security", line 96);
  - `README.md` (the skills heading "The nine skills" → "The ten skills", plus a table row after `review`);
  - `AGENTS.md` ("nine skills" → "ten skills", plus a "where each change goes" row: `lib/security.py` → `python3 -m unittest tests.test_security`);
  - `CONTRIBUTING.md` ("nine" → "ten");
  - `.claude-plugin/plugin.json` (add keyword `"security-review"` after `"code-review"`).

**Interfaces:**
- Consumes: the CLI of Tasks 1–2, called as `python3 "${CLAUDE_PLUGIN_ROOT}/lib/security.py" …` (as `skills/reach/SKILL.md:33` calls `reach.py`); the findings-file contract above.

- [ ] **Step 1: Write `standards.md`.**
  - Open with the pinned editions (Global Constraints).
  - Then one table, one row per vulnerability class the hunt covers, with the columns: class · dimension · CWE · CAPEC · OWASP Top 10:2025 · API Top 10 2023 · ASVS 5.0 chapter · VRT category · NIST 800-53 control · CIS v8.1 safeguard.
  - Then OSI scoping notes, L1–L7: who usually owns the layer, and what to check.
  - Then memory-safety and exhaustion checks per language: Python, Node/TS, Go, Rust, Java/JVM, C/C++.
  - Copy each identifier from its source. Never write one from memory.

- [ ] **Step 2: Write `report-template.md`.**
  - The headings in order: What I ran · Coverage · Most important first · the findings · Since the last audit (audit mode, or when a baseline exists) · Limits · Declined to judge · Verdict.
  - The coverage matrix: rows OWASP Top 10:2025 A01–A10, columns OSI L1–L7; a cell is `inspected`, `not inspected` or `provider`.
  - One finding entry, in the `review` entry format (`skills/review/SKILL.md:60-68`) plus the lines: Standards · CVSS (vector, score, rating, Top 25 from `score`) · Escalation · Prevention (phase + control) · Evidence (measured/from reading, static/live).

- [ ] **Step 3: Write `SKILL.md`, 60–150 lines.**
  - **Frontmatter `description`:** "Use when the person asks for a security review, a security audit, a threat model or the posture of a repo ("is this secure?", "security review this PR", "audit before launch", "are we safe from DoS?"), or when a change touches authentication, money, uploads, webhooks, secrets or CI."
  - **Sections:** the rules (prove the harm with `review`'s bar; probe only in a throwaway worktree and in process; a remote target needs a go each time; post nothing) · Choose the mode (diff → change; none or "audit" → audit; per call 1, a change also checks system-wide protection for any surface it adds or widens) · Hunt, by dimension, with a pointer to `standards.md` · Score (write the findings file, run `score`, never write a score yourself) · Remember (audit: `baseline save`; change: `baseline diff` read-only, to mark "already known") · Report (`report-template.md`) · Common mistakes · Self-check.

- [ ] **Step 4: Register the skill.**
  - Routing row: `| Check security, a threat model or posture | builder `security` (gstack `/cso` for a second opinion) | Say "no security pass" under Evidence |`.
  - The README row: "`security` | One run gives the full stance: every OSI layer, memory and DoS included, each finding mapped to the standards with a computed CVSS score, and what changed since the last audit. Nothing is posted without your yes."

- [ ] **Step 5: Validate.**
  Run: `bash scripts/validate.sh`
  Expected: no `✗` (the skill shape, the line count and the unit tests all pass).

- [ ] **Step 6: Run the with-skill evals.**
  Run: `bash tests/evals/run.sh security` and `bash tests/evals/run.sh security-audit`
  Expected: the with-skill run meets every `expect.md` bullet, and the baseline still misses them. Change the skill until it meets them; never weaken a bullet to pass.

- [ ] **Step 7: Check for private names, then commit.**
  Run: the private-name grep over `git diff --cached` (the list stays out of this public file)
  Expected: no output.
  `git commit -S -m "feat(security): a security skill gives the full stance in one run"`
  The commit body says "Closes #21".

### Task 5: Ship

- [ ] **Step 1:** Commit the spec and this plan with the code: `docs(security): design and plan`. It is one PR.
- [ ] **Step 2:** Show the person the diff summary and the eval met/total (baseline vs with), and ask before pushing.
- [ ] **Step 3:** On a yes:
  - `git push -u origin feat/security-skill`;
  - `gh pr create --repo Lutherwaves/builder`. The title is the person's outcome ("one security run gives the full stance: OSI, memory, DoS and standards-scored findings"). The body has the builder report sections and "Closes #21".
