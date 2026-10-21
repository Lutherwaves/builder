import json
import sys
import tempfile
import threading
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))
import ledger  # noqa: E402

EMPTY = {"answers": {}, "onboarded": False}


def item(i, when=None):
    return {"id": i, "order": 0, "when": when or {}}


class LedgerTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def test_corrupt_ledger_is_empty(self):
        p = self.dir / "ledger.json"
        p.write_text("{not json")
        self.assertEqual(ledger.load(p), EMPTY)

    def test_wrong_structure_is_dropped(self):
        p = self.dir / "ledger.json"
        p.write_text(json.dumps({"answers": {"a": "yes", "b": {"verdict": "later", "at": "x"}, "c": {"verdict": "maybe", "at": 1},
                                             "d": {"verdict": "never", "at": 5}}, "onboarded": "yes"}))
        led = ledger.load(p)
        self.assertEqual(list(led["answers"]), ["d"])
        self.assertIs(led["onboarded"], True)
        self.assertIsNone(ledger.next_recommendation(led, [item("d")], {}, 10.0))

    def test_answer_unwritable_home_exits_nonzero(self):
        import os, subprocess
        f = self.dir / "file"
        f.write_text("x")
        r = subprocess.run([sys.executable, str(Path(ledger.__file__)), "answer", "a", "yes"],
                           env=dict(os.environ, BUILDER_HOME=str(f / "sub")), capture_output=True, text=True)
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("Traceback", r.stderr)
        self.assertIn("BUILDER_HOME", r.stderr)

    def test_missing_ledger_is_empty(self):
        self.assertEqual(ledger.load(self.dir / "none.json"), EMPTY)

    def test_concurrent_saves_parse(self):
        p = self.dir / "ledger.json"
        threads = [
            threading.Thread(target=ledger.save, args=(p, {"answers": {"k%d" % n: {"verdict": "yes", "at": float(n)}}, "onboarded": True}))
            for n in range(20)
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertIn("answers", json.loads(p.read_text()))

    def test_next_respects_order_and_answers(self):
        cat = [dict(item("c"), order=3), dict(item("a"), order=1), dict(item("b"), order=2)]
        led = {"answers": {"a": {"verdict": "never", "at": 1.0}}, "onboarded": False}
        self.assertEqual(ledger.next_recommendation(led, cat, {}, 100.0)["id"], "b")

    def test_yes_is_not_recommended_again(self):
        led = {"answers": {"a": {"verdict": "yes", "at": 1.0}}, "onboarded": False}
        self.assertIsNone(ledger.next_recommendation(led, [item("a")], {}, 100.0))

    def test_later_cools_down(self):
        now = 1_000_000.0
        cat = [item("a")]
        young = {"answers": {"a": {"verdict": "later", "at": now - 100}}, "onboarded": False}
        old = {"answers": {"a": {"verdict": "later", "at": now - 604801}}, "onboarded": False}
        self.assertIsNone(ledger.next_recommendation(young, cat, {}, now))
        self.assertEqual(ledger.next_recommendation(old, cat, {}, now)["id"], "a")

    def test_ge_condition(self):
        cat = [item("a", {"sessions": {">=": 3}})]
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"sessions": 2}, 1.0))
        self.assertEqual(ledger.next_recommendation(EMPTY, cat, {"sessions": 3}, 1.0)["id"], "a")

    def test_none_signal_never_satisfies(self):
        self.assertIsNone(ledger.next_recommendation(EMPTY, [item("a", {"sessions": {">=": 0}})], {"sessions": None}, 1.0))
        self.assertIsNone(ledger.next_recommendation(EMPTY, [item("a", {"sessions": 0})], {"sessions": None}, 1.0))
        self.assertIsNone(ledger.next_recommendation(EMPTY, [item("a", {"x": True})], {}, 1.0))

    def test_none_offered_once_in_onboarding(self):
        cat = [item("a", {"superpowers": False})]
        self.assertEqual(ledger.next_recommendation(EMPTY, cat, {"superpowers": None}, 1.0, onboarding=True)["id"], "a")
        led = {"answers": {"a": {"verdict": "later", "at": 1.0}}, "onboarded": False}
        self.assertIsNone(ledger.next_recommendation(led, cat, {"superpowers": None}, 2.0, onboarding=True))

    def test_none_never_offered_jit(self):
        cat = [item("a", {"superpowers": False})]
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"superpowers": None}, 1.0))
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"superpowers": None}, 1.0, onboarding=False))

    def test_none_answered_not_offered_again(self):
        cat = [item("a", {"superpowers": False})]
        for verdict in ("yes", "never"):
            led = {"answers": {"a": {"verdict": verdict, "at": 1.0}}, "onboarded": False}
            self.assertIsNone(ledger.next_recommendation(led, cat, {"superpowers": None}, 9e9, onboarding=True))

    def test_onboarding_does_not_make_unknown_numbers_or_other_conditions_hold(self):
        self.assertIsNone(ledger.next_recommendation(EMPTY, [item("a", {"sessions": {">=": 1}})], {"sessions": None}, 1.0, onboarding=True))
        cat = [item("a", {"superpowers": False, "gh": False})]
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"superpowers": None, "gh": True}, 1.0, onboarding=True))

    def test_equality_with_null_expected(self):
        cat = [item("ch", {"channel": None})]
        self.assertEqual(ledger.next_recommendation(EMPTY, cat, {"channel": None}, 1.0)["id"], "ch")
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"channel": "slack"}, 1.0))

    def test_onboarded_command_sets_flag_and_keeps_answers(self):
        import os, subprocess
        env = dict(os.environ, BUILDER_HOME=str(self.dir))
        cli = str(Path(ledger.__file__))
        subprocess.run([sys.executable, cli, "answer", "a", "never"], env=env, check=True, capture_output=True)
        r = subprocess.run([sys.executable, cli, "onboarded"], env=env, capture_output=True, text=True)
        self.assertEqual(r.returncode, 0)
        led = ledger.load(self.dir / "ledger.json")
        self.assertIs(led["onboarded"], True)
        self.assertIn("a", led["answers"])

    def test_due_catalog_holds_only_onboarding_items_until_onboarded(self):
        cat = [dict(item("a"), onboarding=True), item("b")]
        self.assertEqual([i["id"] for i in ledger.due_catalog(EMPTY, cat)], ["a"])
        done = {"answers": {}, "onboarded": True}
        self.assertEqual([i["id"] for i in ledger.due_catalog(done, cat)], ["a", "b"])

    def test_channel_item_is_for_teammates_not_for_notes_to_the_person(self):
        cat = json.loads((Path(ledger.__file__).parent / "recommendations.json").read_text())
        ch = next(i for i in cat if i["id"] == "channel")
        text = " ".join((ch["say"], ch["why"], ch["how"]))
        self.assertIn("teammates", text)
        self.assertIn("notification", text)
        self.assertNotIn("reach you", text)
        self.assertNotIn("send you", text)
        readme = (Path(ledger.__file__).parent.parent / "README.md").read_text()
        self.assertNotIn("a channel to reach you", readme)

    def test_gstack_install_goes_where_the_signal_looks(self):
        import os, subprocess
        cat = json.loads((Path(ledger.__file__).parent / "recommendations.json").read_text())
        how = next(i for i in cat if i["id"] == "gstack")["how"]
        self.assertIn("CLAUDE_CONFIG_DIR", how)
        target = '"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gstack"'
        self.assertIn(target, how)
        for cfg, want in (("/cfg", "/cfg/skills/gstack"), (None, "/h/.claude/skills/gstack")):
            env = {"HOME": "/h", "PATH": os.environ.get("PATH", "")}
            if cfg:
                env["CLAUDE_CONFIG_DIR"] = cfg
            r = subprocess.run(["bash", "-c", "printf %s " + target], env=env, capture_output=True, text=True)
            self.assertEqual(r.stdout, want)

    def test_signing_item_comes_right_after_gh(self):
        cat = sorted(ledger._catalog(), key=lambda i: i["order"])
        ids = [i["id"] for i in cat]
        self.assertEqual(ids[ids.index("gh") + 1], "signing")
        self.assertEqual(len({i["order"] for i in cat}), len(cat))
        signing = cat[ids.index("signing")]
        self.assertTrue(signing["onboarding"])
        self.assertEqual(signing["when"], {"signing": False})

    def test_parity_items_follow_matchblox_and_fire_on_their_signals(self):
        cat = sorted(ledger._catalog(), key=lambda i: i["order"])
        ids = [i["id"] for i in cat]
        by = {i["id"]: i for i in cat}
        self.assertEqual(ids[ids.index("matchblox") + 1], "matchblox-setup")
        for i in ("matchblox-setup", "code-graph", "autocompact"):
            self.assertFalse(by[i]["onboarding"])
            self.assertGreater(by[i]["order"], max(c["order"] for c in cat if c.get("onboarding")))
        self.assertEqual(by["matchblox-setup"]["when"], {"matchblox": True, "matchblox_hooks": False})
        self.assertIn("matchblox setup", by["matchblox-setup"]["how"])
        self.assertEqual(by["code-graph"]["when"], {"code_graph": False})
        self.assertIn("/builder:harness", by["code-graph"]["how"])
        self.assertEqual(by["autocompact"]["when"], {"autocompact": False})
        self.assertIn('"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"', by["autocompact"]["how"])
        self.assertIn("record", by["autocompact"]["why"])
        self.assertIn("CLAUDE_CONFIG_DIR", by["autocompact"]["how"])
        only = lambda i: [by[i]]
        sig = {"matchblox": True, "matchblox_hooks": False, "code_graph": False, "autocompact": False}
        for i in ("matchblox-setup", "code-graph", "autocompact"):
            self.assertEqual(ledger.next_recommendation(EMPTY, only(i), sig, 1.0)["id"], i)
        done = {"matchblox": True, "matchblox_hooks": True, "code_graph": True, "autocompact": True}
        for i in ("matchblox-setup", "code-graph", "autocompact"):
            self.assertIsNone(ledger.next_recommendation(EMPTY, only(i), done, 1.0))
        self.assertIsNone(ledger.next_recommendation(EMPTY, only("matchblox-setup"), dict(sig, matchblox=False), 1.0))


if __name__ == "__main__":
    unittest.main()
