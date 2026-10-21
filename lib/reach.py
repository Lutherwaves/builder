#!/usr/bin/env python3
"""Send rules for messages a builder's agent writes to people. Check before a send, log after it."""
import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import ledger  # noqa: E402

RECIPIENTS = ("self", "teammate", "channel")
VERDICTS = ("send", "draft", "deny", "skip")
DEFAULTS = Path(__file__).resolve().parent.parent / "skills" / "reach" / "rules.default.json"


DEDUPE_S = 3600
PER_HOUR = 6


def _norm(text):
    """Case and spacing do not make a new message."""
    return " ".join(str(text or "").split()).casefold()


def _who(to, recipient=None, channel_name=None):
    if to == "self":
        return "self"
    return _norm(channel_name or recipient if to == "channel" else recipient)


def digest(to, text, recipient=None):
    return hashlib.sha256(("%s|%s|%s" % (to, _who(to, recipient), _norm(text))).encode("utf-8")).hexdigest()


def _number(value, default):
    ok = isinstance(value, (int, float)) and not isinstance(value, bool) and value >= 0
    return value if ok else default


def _names(box, key):
    value = box.get(key)
    if not isinstance(value, list):
        return []
    return [v for v in value if isinstance(v, str) and v.strip()]


def decide(rules, to, kind, channel_name=None):
    box = rules.get("to_" + to)
    box = box if isinstance(box, dict) else {}
    allowed = kind in _names(box, "allow")
    if to == "channel":
        named = isinstance(channel_name, str) and channel_name.strip() and channel_name in _names(box, "named")
        return "send" if allowed and named else "deny"
    return "send" if allowed else "draft"


def check(rules, log, to, kind, text, now, channel_name=None, recipient=None):
    if to == "channel" and not channel_name:
        channel_name = recipient
    verdict = decide(rules, to, kind, channel_name)
    if verdict == "deny":
        return {"verdict": "deny", "reason": "channel-not-named"}
    sends = [e for e in log if isinstance(e, dict) and e.get("verdict") == "send"
             and isinstance(e.get("at"), (int, float)) and not isinstance(e.get("at"), bool)]
    h = digest(to, text, channel_name if to == "channel" else recipient)
    if any(e.get("sha256") == h and now - e["at"] <= _number(rules.get("dedupe_s"), DEDUPE_S) for e in sends):
        return {"verdict": "skip", "reason": "duplicate"}
    if sum(1 for e in sends if now - e["at"] < 3600) >= _number(rules.get("per_hour"), PER_HOUR):
        return {"verdict": "skip", "reason": "rate"}
    return {"verdict": verdict, "reason": "allowed" if verdict == "send" else "needs-yes"}


def record(path, entry):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    line = (json.dumps(entry) + "\n").encode("utf-8")
    fd = os.open(str(path), os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
    try:
        os.write(fd, line)  # one write per line: concurrent appends do not interleave
    finally:
        os.close(fd)


def read_log(path):
    try:
        lines = Path(path).read_text().splitlines()
    except OSError:
        return []
    out = []
    for line in lines:
        try:
            entry = json.loads(line)
        except ValueError:
            continue
        if isinstance(entry, dict):
            out.append(entry)
    return out


def _live(path):
    try:
        data = json.loads(Path(path).read_text())
    except (OSError, ValueError):
        return {}
    return data if isinstance(data, dict) else {}


def load_rules(path):
    rules = json.loads(DEFAULTS.read_text())
    rules.update(_live(path))
    return rules


def notify(title, text, platform=sys.platform, path=None):
    """Show an OS notification. Return the tool name, or None when no tool works."""
    if platform == "darwin":
        name = "osascript"
        # The text goes in as an argument, never into the script, so quotes cannot break it.
        args = ["-e", "on run argv", "-e", "display notification (item 2 of argv) with title (item 1 of argv)",
                "-e", "end run", title, text]
    else:
        name, args = "notify-send", [title, text]
    tool = shutil.which(name, path=path)
    if not tool:
        return None
    try:
        r = subprocess.run([tool, *args], stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, timeout=5)
    except (OSError, subprocess.TimeoutExpired):
        return None
    return name if r.returncode == 0 else None


def _unwritable(path, e):
    print("builder cannot write %s (%s); set BUILDER_HOME to a writable folder." % (path, e.strerror or e),
          file=sys.stderr)
    return 1


def main(argv):
    p = argparse.ArgumentParser(prog="reach.py")
    sub = p.add_subparsers(dest="cmd", required=True)
    for name in ("check", "log"):
        s = sub.add_parser(name)
        s.add_argument("--to", required=True, choices=RECIPIENTS)
        s.add_argument("--kind", required=True)
        s.add_argument("--text", required=True)
        s.add_argument("--recipient", help="who gets it: a teammate's name, or the room for --to channel")
        s.add_argument("--channel-name", help="the room for --to channel (same as --recipient)")
        if name == "log":
            s.add_argument("--verdict", required=True, choices=VERDICTS)
            s.add_argument("--reason", default="")
    s = sub.add_parser("channel")
    s.add_argument("name", help='where the team talks, e.g. slack, teams; "none" for draft only')
    s = sub.add_parser("notify")
    s.add_argument("--text", required=True)
    s.add_argument("--title", default="builder")
    a = p.parse_args(argv[1:])
    if a.cmd in ("check", "log") and a.to != "self":
        if a.to == "channel":
            if a.recipient and a.channel_name and a.recipient != a.channel_name:
                p.error("--recipient and --channel-name name different rooms; give one room")
            a.recipient = a.channel_name = a.recipient or a.channel_name
        if not (a.recipient or "").strip():
            p.error("--to %s needs --recipient <%s>" % (a.to, "room name" if a.to == "channel" else "teammate's name"))

    home = ledger.home()
    rules_path, log_path = home / "reach.json", home / "reach-log.jsonl"
    if a.cmd == "check":
        print(json.dumps(check(load_rules(rules_path), read_log(log_path), a.to, a.kind, a.text,
                               time.time(), a.channel_name, a.recipient)))
        return 0
    if a.cmd == "log":
        entry = {"at": time.time(), "to": a.to, "recipient": a.recipient if a.to != "self" else "self",
                 "kind": a.kind, "verdict": a.verdict, "reason": a.reason,
                 "sha256": digest(a.to, a.text, a.recipient), "text": a.text}
        if a.channel_name:
            entry["channel_name"] = a.channel_name
        try:
            record(log_path, entry)
        except OSError as e:
            return _unwritable(log_path, e)
        print("ok")
        return 0
    if a.cmd == "channel":
        data = {}
        if rules_path.exists():
            try:
                data = json.loads(rules_path.read_text())
            except (OSError, ValueError):
                data = None
            if not isinstance(data, dict):
                print("builder cannot read %s: it is not a JSON object. Fix the file or delete it, then run this again."
                      % rules_path, file=sys.stderr)
                return 1
        data["channel"] = None if a.name.lower() in ("none", "null", "") else a.name.lower()
        try:
            ledger.save(rules_path, data)
        except OSError as e:
            return _unwritable(rules_path, e)
        print("ok")
        return 0
    tool = notify(a.title, a.text)
    if tool is None:
        print("builder found no notification tool (osascript on macOS, notify-send on Linux). "
              "Show the message in the conversation as a draft instead.", file=sys.stderr)
        return 1
    print(tool)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
