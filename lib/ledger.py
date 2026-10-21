#!/usr/bin/env python3
"""Local memory of what builder recommended and what the person answered."""
import json
import os
import sys
import tempfile
import time
from pathlib import Path

COOLDOWN_S = 604800  # "not now" stays quiet for 7 days
VERDICTS = ("yes", "later", "never")


def home():
    env = os.environ.get("BUILDER_HOME")
    return Path(env) if env else Path.home() / ".config" / "builder"


def _empty():
    return {"answers": {}, "onboarded": False}


def load(path):
    try:
        data = json.loads(Path(path).read_text())
    except (OSError, ValueError):
        return _empty()
    if not isinstance(data, dict) or not isinstance(data.get("answers"), dict):
        return _empty()
    return {"answers": data["answers"], "onboarded": bool(data.get("onboarded", False))}


def save(path, data):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=str(path.parent), prefix=path.name + ".", suffix=".tmp")
    try:
        with os.fdopen(fd, "w") as f:
            json.dump(data, f)
        os.replace(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def _holds(when, signals):
    for key, expected in when.items():
        actual = signals.get(key)
        if actual is None and expected is not None:
            return False
        if isinstance(expected, dict):
            bound = expected.get(">=")
            if not isinstance(actual, (int, float)) or isinstance(actual, bool) or actual < bound:
                return False
        elif actual != expected:
            return False
    return True


def next_recommendation(ledger, catalog, signals, now, cooldown_s=COOLDOWN_S):
    answers = ledger.get("answers", {})
    for item in sorted(catalog, key=lambda i: i.get("order", 0)):
        a = answers.get(item["id"])
        if a:
            if a.get("verdict") in ("yes", "never"):
                continue
            if a.get("verdict") == "later" and now - a.get("at", 0) < cooldown_s:
                continue
        if _holds(item.get("when", {}), signals):
            return item
    return None


def _catalog():
    return json.loads((Path(__file__).resolve().parent / "recommendations.json").read_text())


def main(argv):
    path = home() / "ledger.json"
    if len(argv) == 4 and argv[1] == "answer" and argv[3] in VERDICTS:
        data = load(path)
        data["answers"][argv[2]] = {"verdict": argv[3], "at": time.time()}
        save(path, data)
        print("ok")
        return 0
    if len(argv) == 2 and argv[1] == "next":
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import signals
        sig = signals.collect(os.getcwd())
        print(json.dumps(next_recommendation(load(path), _catalog(), sig, time.time())))
        return 0
    print("usage: ledger.py answer <id> yes|later|never | ledger.py next", file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
