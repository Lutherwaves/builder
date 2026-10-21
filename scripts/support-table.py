#!/usr/bin/env python3
"""Build the README support table from graded eval results.

  support-table.py record <verdict.json>   fold a graded verdict into tests/evals/results/<harness>.json
  support-table.py render                  print the table
  support-table.py check <README.md>       exit 1 when the block between the markers differs from `render`

A results file keeps counts only: no transcript, no quote, no path.
"""
import argparse
import json
import re
import sys
from pathlib import Path

HARNESSES = {"claude": "Claude Code", "codex": "Codex CLI", "cursor": "Cursor CLI", "opencode": "opencode"}
NONE_NOTE = "No harness has passed its evals yet."
START, END = "<!-- support:start -->", "<!-- support:end -->"
VERSION = re.compile(r"^[A-Za-z0-9._ +()-]{1,60}$")
FIXTURE = re.compile(r"^[a-z0-9_-]+$")
DATE = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def fail(message):
    print(f"support-table: {message}", file=sys.stderr)
    sys.exit(1)


def count(verdict):
    for key, pattern in (("harness_version", VERSION), ("fixture", FIXTURE), ("date", DATE)):
        if not isinstance(verdict.get(key), str) or not pattern.match(verdict[key]):
            fail(f"{key} is missing or not a plain value")
    if verdict.get("harness") not in HARNESSES:
        fail(f"harness must be one of {', '.join(HARNESSES)}")
    bullets = verdict.get("bullets")
    if not isinstance(bullets, list) or not bullets:
        fail("bullets is empty")
    if any(not isinstance(b, dict) or not isinstance(b.get("met"), bool) for b in bullets):
        fail("every bullet needs met true or false: the verdict is not graded yet")
    return {"met": sum(b["met"] for b in bullets), "total": len(bullets)}


def record(results_dir, source):
    try:
        verdict = json.loads(Path(source).read_text(encoding="utf-8"))
    except (OSError, ValueError) as e:
        fail(f"cannot read {source}: {e}")
    counts = count(verdict)
    path = results_dir / f"{verdict['harness']}.json"
    saved = json.loads(path.read_text(encoding="utf-8")) if path.exists() else {"fixtures": {}}
    if saved.get("harness_version") not in (None, verdict["harness_version"]):
        saved["fixtures"] = {}  # a new version starts a new result
    saved["fixtures"][verdict["fixture"]] = counts
    saved.update(harness=verdict["harness"], harness_version=verdict["harness_version"],
                 date=max(verdict["date"], saved.get("date", "")))
    results_dir.mkdir(parents=True, exist_ok=True)
    ordered = {k: saved[k] for k in ("harness", "harness_version", "date", "fixtures")}
    path.write_text(json.dumps(ordered, indent=2) + "\n", encoding="utf-8")
    print(f"recorded {verdict['harness']} {verdict['fixture']}: {counts['met']}/{counts['total']}")


def render(results_dir):
    rows = []
    for harness, name in HARNESSES.items():
        path = results_dir / f"{harness}.json"
        if not path.exists():
            continue
        r = json.loads(path.read_text(encoding="utf-8"))
        short = [f"{f} {c['met']}/{c['total']}" for f, c in r["fixtures"].items() if c["met"] != c["total"]]
        status = "supported" if r["fixtures"] and not short else "experimental: " + (", ".join(short) or "no fixture graded")
        rows.append(f"| {name} | {r['harness_version']} | {r['date']} | {status} |")
    if not rows:
        return NONE_NOTE
    return "\n".join(["| Harness | Version | Eval date | Status |", "| --- | --- | --- | --- |", *rows])


def check(results_dir, readme):
    try:
        text = Path(readme).read_text(encoding="utf-8")
    except OSError as e:
        fail(f"cannot read {readme}: {e}")
    if START not in text or END not in text or text.index(START) > text.index(END):
        fail(f"{readme} has no {START} ... {END} block")
    block = text.split(START, 1)[1].split(END, 1)[0].strip()
    if block != render(results_dir):
        fail(f"the support table in {readme} is out of date: paste the output of `scripts/support-table.py render` between the markers")
    print("support table ok")


def main():
    default = Path(__file__).resolve().parent.parent / "tests" / "evals" / "results"
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--results-dir", type=Path, default=default)
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("record").add_argument("verdict")
    sub.add_parser("render")
    sub.add_parser("check").add_argument("readme")
    args = parser.parse_args()
    if args.cmd == "record":
        record(args.results_dir, args.verdict)
    elif args.cmd == "render":
        print(render(args.results_dir))
    else:
        check(args.results_dir, args.readme)


if __name__ == "__main__":
    main()
