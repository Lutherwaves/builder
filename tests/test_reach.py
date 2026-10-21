import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))
import reach  # noqa: E402

ROOT = Path(__file__).resolve().parent.parent
RULES = json.loads((ROOT / "skills" / "reach" / "rules.default.json").read_text())
NOW = 1_000_000.0


def sent(to, text, at, kind="waiting", recipient="self"):
    return {"at": at, "to": to, "kind": kind, "verdict": "send", "sha256": reach.digest(to, text, recipient)}


class ReachTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def cli(self, *args, path=None):
        env = dict(os.environ, BUILDER_HOME=str(self.dir))
        if path is not None:
            env["PATH"] = path
        return subprocess.run([sys.executable, str(ROOT / "lib" / "reach.py"), *args],
                              env=env, capture_output=True, text=True)

    def test_defaults_match_the_plan(self):
        self.assertEqual(RULES, {"to_self": {"allow": ["waiting", "morning", "needs-yes"]},
                                 "to_teammate": {"allow": []}, "to_channel": {"allow": [], "named": []},
                                 "dedupe_s": 3600, "per_hour": 6})

    def test_self_waiting_sends(self):
        self.assertEqual(reach.decide(RULES, "self", "waiting"), "send")
        self.assertEqual(reach.check(RULES, [], "self", "waiting", "Waiting for you.", NOW)["verdict"], "send")
        self.assertEqual(reach.decide(RULES, "self", "chatter"), "draft")

    def test_teammate_defaults_to_draft(self):
        self.assertEqual(reach.decide(RULES, "teammate", "fyi"), "draft")
        self.assertEqual(reach.check(RULES, [], "teammate", "fyi", "The API changed.", NOW)["verdict"], "draft")

    def test_unnamed_channel_denied(self):
        self.assertEqual(reach.decide(RULES, "channel", "fyi"), "deny")
        self.assertEqual(reach.decide(RULES, "channel", "fyi", "general"), "deny")
        named = dict(RULES, to_channel={"allow": ["fyi"], "named": ["releases"]})
        self.assertEqual(reach.decide(named, "channel", "fyi", "releases"), "send")
        self.assertEqual(reach.decide(named, "channel", "fyi", "general"), "deny")
        self.assertEqual(reach.decide(named, "channel", "fyi"), "deny")
        self.assertEqual(reach.check(RULES, [], "channel", "fyi", "Hi all.", NOW)["verdict"], "deny")

    def test_duplicate_is_skipped(self):
        log = [sent("self", "Waiting for you.", NOW)]
        r = reach.check(RULES, log, "self", "waiting", "Waiting for you.", NOW + 3600)
        self.assertEqual(r, {"verdict": "skip", "reason": "duplicate"})
        self.assertEqual(reach.check(RULES, log, "self", "waiting", "Waiting for you.", NOW + 3601)["verdict"], "send")
        # Same text to another recipient is not a duplicate.
        self.assertEqual(reach.check(RULES, log, "teammate", "fyi", "Waiting for you.", NOW + 10, recipient="Ana")["verdict"], "draft")

    def test_whitespace_and_case_do_not_bypass_dedupe(self):
        log = [sent("self", "Tests are green.", NOW)]
        for again in ("Tests are green.\n", "  tests ARE   green. ", "Tests\tare green."):
            self.assertEqual(reach.check(RULES, log, "self", "waiting", again, NOW + 5),
                             {"verdict": "skip", "reason": "duplicate"}, again)

    def test_recipient_is_in_digest_and_log(self):
        text = "The API changed."
        self.cli("log", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", text, "--verdict", "send")
        r = self.cli("check", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", text)
        self.assertEqual(json.loads(r.stdout), {"verdict": "skip", "reason": "duplicate"})
        r = self.cli("check", "--to", "teammate", "--recipient", "Bob", "--kind", "fyi", "--text", text)
        self.assertEqual(json.loads(r.stdout)["verdict"], "draft")
        entry = json.loads((self.dir / "reach-log.jsonl").read_text().splitlines()[0])
        self.assertEqual(entry["recipient"], "Ana")

    def test_teammate_and_channel_need_a_recipient(self):
        for args in (("--to", "teammate"), ("--to", "channel")):
            for cmd in ("check", "log"):
                extra = ("--verdict", "draft") if cmd == "log" else ()
                r = self.cli(cmd, *args, "--kind", "fyi", "--text", "x", *extra)
                self.assertEqual(r.returncode, 2, (cmd, args))
                self.assertNotIn("Traceback", r.stderr)
                self.assertIn("--recipient", r.stderr)
        self.assertFalse((self.dir / "reach-log.jsonl").exists())

    def test_channel_recipient_is_the_channel_name(self):
        (self.dir / "reach.json").write_text(json.dumps({"to_channel": {"allow": ["fyi"], "named": ["releases"]}}))
        r = self.cli("check", "--to", "channel", "--recipient", "releases", "--kind", "fyi", "--text", "x")
        self.assertEqual(json.loads(r.stdout)["verdict"], "send")
        r = self.cli("check", "--to", "channel", "--recipient", "general", "--kind", "fyi", "--text", "x")
        self.assertEqual(json.loads(r.stdout)["verdict"], "deny")

    def test_malformed_rules_use_defaults(self):
        bad = dict(RULES, dedupe_s="x", per_hour=True)
        log = [sent("self", "note %d" % n, NOW - 60 * n) for n in range(6)]
        self.assertEqual(reach.check(bad, log, "self", "waiting", "note 7", NOW), {"verdict": "skip", "reason": "rate"})
        self.assertEqual(reach.check(bad, log, "self", "waiting", "note 0", NOW + 3000)["reason"], "duplicate")
        for v in (None, -1, "6", [6]):
            self.assertEqual(reach.check(dict(RULES, per_hour=v, dedupe_s=v), log, "self", "waiting", "note 7", NOW)["reason"], "rate")
        odd = {"to_self": "x", "to_teammate": {"allow": "fyi"}, "to_channel": None}
        self.assertEqual(reach.decide(odd, "self", "waiting"), "draft")
        self.assertEqual(reach.decide(odd, "teammate", "fyi"), "draft")
        self.assertEqual(reach.decide(odd, "channel", "fyi", "x"), "deny")
        (self.dir / "reach.json").write_text(json.dumps({"dedupe_s": "x", "per_hour": None, "to_self": [1]}))
        r = self.cli("check", "--to", "self", "--kind", "waiting", "--text", "x")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(json.loads(r.stdout)["verdict"], "draft")

    def test_empty_room_name_never_matches(self):
        rules = dict(RULES, to_channel={"allow": ["fyi"], "named": [None, "", "releases"]})
        for name in (None, "", "  "):
            self.assertEqual(reach.decide(rules, "channel", "fyi", name), "deny", repr(name))

    def test_duplicate_twice_is_skipped_and_logged(self):
        p = self.dir / "reach-log.jsonl"
        r = self.cli("check", "--to", "self", "--kind", "waiting", "--text", "Tests are green.")
        self.assertEqual(json.loads(r.stdout)["verdict"], "send")
        self.cli("log", "--to", "self", "--kind", "waiting", "--text", "Tests are green.", "--verdict", "send")
        r = self.cli("check", "--to", "self", "--kind", "waiting", "--text", "Tests are green.")
        self.assertEqual(json.loads(r.stdout), {"verdict": "skip", "reason": "duplicate"})
        self.cli("log", "--to", "self", "--kind", "waiting", "--text", "Tests are green.", "--verdict", "skip", "--reason", "duplicate")
        lines = [json.loads(x) for x in p.read_text().splitlines()]
        self.assertEqual([x["verdict"] for x in lines], ["send", "skip"])
        self.assertEqual(lines[1]["reason"], "duplicate")

    def test_rate_limit(self):
        log = [sent("self", "note %d" % n, NOW - 60 * n) for n in range(6)]
        self.assertEqual(reach.check(RULES, log, "self", "waiting", "note 7", NOW), {"verdict": "skip", "reason": "rate"})
        self.assertEqual(reach.check(RULES, log[:5], "self", "waiting", "note 7", NOW)["verdict"], "send")
        old = [sent("self", "note %d" % n, NOW - 3601 - n) for n in range(6)]
        self.assertEqual(reach.check(RULES, old, "self", "waiting", "note 7", NOW)["verdict"], "send")
        skips = [dict(e, verdict="skip") for e in log]
        self.assertEqual(reach.check(RULES, skips, "self", "waiting", "note 7", NOW)["verdict"], "send")

    def test_log_appends_jsonl(self):
        p = self.dir / "sub" / "reach-log.jsonl"
        reach.record(p, {"to": "self", "verdict": "send"})
        reach.record(p, {"to": "teammate", "verdict": "draft"})
        lines = p.read_text().splitlines()
        self.assertEqual([json.loads(x)["to"] for x in lines], ["self", "teammate"])

    def test_bad_log_lines_are_ignored(self):
        p = self.dir / "reach-log.jsonl"
        p.write_text('{not json\n[1]\n' + json.dumps(sent("self", "x", NOW)) + "\n")
        self.assertEqual(len(reach.read_log(p)), 1)
        self.assertEqual(reach.read_log(self.dir / "none.jsonl"), [])

    def test_live_rules_override_defaults_and_keep_channel(self):
        (self.dir / "reach.json").write_text(json.dumps({"channel": "slack", "to_teammate": {"allow": ["fyi"]}}))
        r = self.cli("check", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", "The API changed.")
        self.assertEqual(json.loads(r.stdout)["verdict"], "send")
        r = self.cli("check", "--to", "self", "--kind", "waiting", "--text", "x")
        self.assertEqual(json.loads(r.stdout)["verdict"], "send")

    def test_channel_command_writes_channel_key(self):
        (self.dir / "reach.json").write_text(json.dumps({"per_hour": 2}))
        r = self.cli("channel", "slack")
        self.assertEqual(r.returncode, 0, r.stderr)
        data = json.loads((self.dir / "reach.json").read_text())
        self.assertEqual(data["channel"], "slack")
        self.assertEqual(data["per_hour"], 2)
        self.cli("channel", "none")
        self.assertIsNone(json.loads((self.dir / "reach.json").read_text())["channel"])

    def test_channel_refuses_a_corrupt_rules_file(self):
        p = self.dir / "reach.json"
        for body in ("{not json", "[1]"):
            p.write_text(body)
            r = self.cli("channel", "slack")
            self.assertEqual(r.returncode, 1)
            self.assertNotIn("Traceback", r.stderr)
            self.assertIn(str(p), r.stderr)
            self.assertIn("fix the file", r.stderr.lower())
            self.assertEqual(p.read_text(), body)

    def test_unwritable_home_exits_nonzero(self):
        f = self.dir / "file"
        f.write_text("x")
        env = dict(os.environ, BUILDER_HOME=str(f / "sub"))
        r = subprocess.run([sys.executable, str(ROOT / "lib" / "reach.py"), "log", "--to", "self", "--kind", "waiting",
                            "--text", "x", "--verdict", "send"], env=env, capture_output=True, text=True)
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("Traceback", r.stderr)
        self.assertIn("BUILDER_HOME", r.stderr)

    def fake(self, name):
        bin_dir = self.dir / "bin"
        bin_dir.mkdir(exist_ok=True)
        tool = bin_dir / name
        tool.write_text('#!/bin/sh\nfor a in "$@"; do printf "%%s\\n" "$a"; done > "%s/%s.args"\n' % (self.dir, name))
        tool.chmod(0o755)
        return str(bin_dir)

    def test_notify_macos_passes_text_as_argument(self):
        path = self.fake("osascript")
        text = 'Tests "green"; $(rm -rf /)'
        self.assertEqual(reach.notify("builder", text, platform="darwin", path=path), "osascript")
        args = (self.dir / "osascript.args").read_text().splitlines()
        self.assertIn(text, args)
        self.assertIn("builder", args)
        self.assertFalse(any(text in a and a != text for a in args))

    def test_notify_linux_uses_notify_send(self):
        path = self.fake("notify-send")
        self.assertEqual(reach.notify("builder", "Waiting for you.", platform="linux", path=path), "notify-send")
        self.assertEqual((self.dir / "notify-send.args").read_text().splitlines(), ["builder", "Waiting for you."])

    def test_notify_without_a_tool_says_the_fix(self):
        empty = self.dir / "empty"
        empty.mkdir()
        r = self.cli("notify", "--text", "Waiting for you.", path=str(empty))
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("Traceback", r.stderr)
        self.assertIn("draft", r.stderr)


if __name__ == "__main__":
    unittest.main()
