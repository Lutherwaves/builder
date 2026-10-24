#!/usr/bin/env python3
"""What a security finding may not invent: its CVSS 3.1 score, its CWE id and its Top 25 flag."""
import argparse
import json
import math
import re
import sys
from pathlib import Path

PREFIX = "CVSS:3.1/"
METRICS = ("AV", "AC", "PR", "UI", "S", "C", "I", "A")
_VALUES = {
    "AV": {"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2},
    "AC": {"L": 0.77, "H": 0.44},
    "PR": {"N": 0.85, "L": 0.62, "H": 0.27},
    "UI": {"N": 0.85, "R": 0.62},
    "S": {"U": None, "C": None},
    "C": {"H": 0.56, "L": 0.22, "N": 0.0},
    "I": {"H": 0.56, "L": 0.22, "N": 0.0},
    "A": {"H": 0.56, "L": 0.22, "N": 0.0},
}
_PR_CHANGED = {"N": 0.85, "L": 0.68, "H": 0.5}

# MITRE 2025 CWE Top 25, in rank order. Source: https://cwe.mitre.org/top25/archive/2025/2025_cwe_top25.html
# (read 2026-10-24).
TOP25_2025 = (
    "CWE-79", "CWE-89", "CWE-352", "CWE-862", "CWE-787", "CWE-22", "CWE-416", "CWE-125",
    "CWE-78", "CWE-94", "CWE-120", "CWE-434", "CWE-476", "CWE-121", "CWE-502", "CWE-122",
    "CWE-863", "CWE-20", "CWE-284", "CWE-200", "CWE-306", "CWE-918", "CWE-77", "CWE-639",
    "CWE-770",
)

DIMENSIONS = (
    "osi-l1-l2", "osi-l3-l4", "osi-l5-l6", "osi-l7", "memory", "availability", "authn", "authz",
    "injection", "ssrf", "secrets", "supply-chain", "agent-boundary", "data-integrity",
)
MODES = ("change", "audit")


def parse_vector(vector):
    if not vector.startswith(PREFIX):
        raise ValueError("vector must start with %s, got %r" % (PREFIX, vector.split("/")[0]))
    metrics = {}
    for part in vector[len(PREFIX):].split("/"):
        key, _, value = part.partition(":")
        if key in metrics:
            raise ValueError("duplicate metric %s in %r" % (key, vector))
        if key not in _VALUES or value not in _VALUES[key]:
            raise ValueError("bad metric %s in %r" % (part, vector))
        metrics[key] = value
    missing = [m for m in METRICS if m not in metrics]
    if missing:
        raise ValueError("missing metric %s in %r" % (", ".join(missing), vector))
    return metrics


def _roundup(x):
    # CVSS v3.1 specification, Appendix A: avoids float error in a plain ceil(x * 10).
    n = round(x * 100000)
    return n / 100000.0 if n % 10000 == 0 else (math.floor(n / 10000) + 1) / 10.0


def base_score(vector):
    m = parse_vector(vector)
    changed = m["S"] == "C"
    iss = 1 - (1 - _VALUES["C"][m["C"]]) * (1 - _VALUES["I"][m["I"]]) * (1 - _VALUES["A"][m["A"]])
    impact = 7.52 * (iss - 0.029) - 3.25 * (iss - 0.02) ** 15 if changed else 6.42 * iss
    if impact <= 0:
        return 0.0
    pr = (_PR_CHANGED if changed else _VALUES["PR"])[m["PR"]]
    exploitability = 8.22 * _VALUES["AV"][m["AV"]] * _VALUES["AC"][m["AC"]] * pr * _VALUES["UI"][m["UI"]]
    total = impact + exploitability
    return _roundup(min(1.08 * total if changed else total, 10))


def rating(score):
    if score == 0:
        return "None"
    if score < 4.0:
        return "Low"
    if score < 7.0:
        return "Medium"
    if score < 9.0:
        return "High"
    return "Critical"


def check_cwe(cwe):
    if not isinstance(cwe, str) or not re.fullmatch(r"CWE-[1-9][0-9]*", cwe):
        raise ValueError("bad CWE id %r (want CWE-<number>)" % (cwe,))
    return cwe


def in_top25(cwe):
    return check_cwe(cwe) in TOP25_2025


def enrich(doc):
    if doc.get("mode") not in MODES:
        raise ValueError("mode must be one of %s, got %r" % (", ".join(MODES), doc.get("mode")))
    for dim in doc.get("dimensions", []):
        if dim not in DIMENSIONS:
            raise ValueError("unknown dimension %r in dimensions" % (dim,))
    seen = set()
    out = []
    for f in doc.get("findings", []):
        fid = f.get("id")
        try:
            if fid in seen:
                raise ValueError("duplicate id")
            seen.add(fid)
            if f.get("dimension") not in DIMENSIONS:
                raise ValueError("unknown dimension %r" % (f.get("dimension"),))
            score = base_score(f.get("vector", ""))
            top25 = in_top25(f.get("cwe"))
        except ValueError as e:
            raise ValueError("finding %s: %s" % (fid, e)) from None
        out.append(dict(f, score=score, rating=rating(score), top25=top25))
    return dict(doc, findings=out)


def _read(path):
    try:
        return json.loads(Path(path).read_text())
    except (OSError, ValueError) as e:
        raise ValueError("cannot read %s: %s" % (path, e)) from None


def main(argv=None):
    p = argparse.ArgumentParser(prog="security.py")
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("cvss").add_argument("vector")
    sub.add_parser("score").add_argument("findings")
    a = p.parse_args(argv)
    try:
        if a.cmd == "cvss":
            score = base_score(a.vector)
            print("%.1f %s" % (score, rating(score)))
        else:
            print(json.dumps(enrich(_read(a.findings)), indent=2))
    except ValueError as e:
        print("security: %s" % e, file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
