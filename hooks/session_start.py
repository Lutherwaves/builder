#!/usr/bin/env python3
"""SessionStart hook: print the builder soul and at most one next step as JSON."""
import json
import os
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "lib"))
import ledger  # noqa: E402
import signals  # noqa: E402

LEDGER_PY = '%s/lib/ledger.py' % os.environ.get("CLAUDE_PLUGIN_ROOT", str(ROOT))
ANSWER = 'python3 "%s" answer %%s yes|later|never' % LEDGER_PY
ONBOARDED = 'python3 "%s" onboarded' % LEDGER_PY


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
        ]
    lines.append("When all setup steps are answered, run: %s" % ONBOARDED)
    return "\n".join(lines)


def build():
    soul = (ROOT / "skills" / "builder" / "soul-compact.md").read_text().rstrip("\n")
    try:
        led = ledger.load(ledger.home() / "ledger.json")
        catalog = json.loads((ROOT / "lib" / "recommendations.json").read_text())
        sig = signals.collect(os.getcwd(), 0.5)
        now = time.time()
        if not led["onboarded"]:
            item = ledger.next_recommendation(led, [i for i in catalog if i.get("onboarding")], sig, now)
            return soul + "\n\n" + onboarding_text(item)
        item = ledger.next_recommendation(led, catalog, sig, now)
        if item:
            return (soul + "\n\nAnswer the person's request first. Put this as one short block at the end of your reply. Never put it before the work.\n"
                    + block(item))
    except Exception:
        pass
    return soul


if __name__ == "__main__":
    print(json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": build()}}))
