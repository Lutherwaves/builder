#!/usr/bin/env python3
"""SessionStart hook: print the builder soul and at most one next step, in the shape the harness reads."""
import argparse
import json
import os
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "lib"))
import harness as harness_lib  # noqa: E402
import ledger  # noqa: E402
import signals  # noqa: E402

LEDGER_PY = '%s/lib/ledger.py' % ROOT
ANSWER = 'python3 "%s" answer %%s yes|later|never' % LEDGER_PY
ONBOARDED = 'python3 "%s" onboarded' % LEDGER_PY
NEXT = 'python3 "%s" next' % LEDGER_PY


def block(item):
    return "builder recommends: %s — %s. To do it: %s. After your answer, run: %s" % (
        item["say"], item["why"].rstrip("."), item["how"].rstrip("."), ANSWER % item["id"])


def onboarding_text(item):
    lines = [
        "builder onboarding: this person is new to builder.",
        "Answer the person's request first. Do the work before anything else.",
        "At the end of that reply, introduce builder with these lines, word for word.",
        "This is required, even when the request is one line. The reply is not complete without the intro.",
        "  builder: you decide. I propose, build, and end each piece of work with a report.",
        "  Nothing leaves this machine without a send rule or your yes.",
    ]
    if item:
        lines += [
            "  I offer a few setup steps, one at a time.",
            "Then offer this first setup step and ask: yes, not now, or never.",
            block(item).replace("builder recommends:", "Step:", 1),
            "Offer one step per reply. After each answer, offer the next due step in a later reply.",
            "To find the next due step, run: %s (it prints the step as JSON, or null when no step is due)." % NEXT,
            "Never invent a step. Offer only the step that this command prints.",
        ]
    lines.append("When all setup steps are answered, run: %s" % ONBOARDED)
    return "\n".join(lines)


def mapping_block(harness):
    try:
        text = (ROOT / "skills" / "builder" / "harnesses" / ("%s.md" % harness)).read_text()
        return text.split("<!-- hook -->", 1)[1].split("<!-- /hook -->", 1)[0].strip()
    except (OSError, IndexError):
        return ""


def build_body(harness):
    soul = (ROOT / "skills" / "builder" / "soul-compact.md").read_text().rstrip("\n")
    try:
        led = ledger.load(ledger.home() / "ledger.json")
        catalog = json.loads((ROOT / "lib" / "recommendations.json").read_text())
        sig = signals.collect(os.getcwd(), 0.5, harness=harness if harness in harness_lib.HARNESSES else None)
        now = time.time()
        if not led["onboarded"]:
            item = ledger.next_recommendation(led, ledger.due_catalog(led, catalog), sig, now, onboarding=True)
            return soul + "\n\n" + onboarding_text(item)
        item = ledger.next_recommendation(led, catalog, sig, now)
        if item:
            return (soul + "\n\nAnswer the person's request first. Put this as one short block at the end of your reply. Never put it before the work.\n"
                    + block(item))
    except Exception:
        pass
    return soul


def build(harness):
    text = build_body(harness) + "\n\nbuilder root: %s. Where a builder skill says ${CLAUDE_PLUGIN_ROOT}, use this path." % ROOT
    mapping = mapping_block(harness)
    return text + "\n\n" + mapping if mapping else text


def shape(harness, text):
    if harness == "cursor":
        return json.dumps({"additional_context": text})
    return json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": text}})


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument("--harness", default="auto")
    parser.add_argument("--format", choices=("json", "text"))
    args, _ = parser.parse_known_args(argv)
    harness = harness_lib.detect(os.environ, args.harness)
    fmt = args.format or ("text" if harness == "opencode" else "json")
    text = build(harness)
    print(text if fmt == "text" else shape(harness, text))


if __name__ == "__main__":
    main(sys.argv[1:])
