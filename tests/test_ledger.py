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

    def test_equality_with_null_expected(self):
        cat = [item("ch", {"channel": None})]
        self.assertEqual(ledger.next_recommendation(EMPTY, cat, {"channel": None}, 1.0)["id"], "ch")
        self.assertIsNone(ledger.next_recommendation(EMPTY, cat, {"channel": "slack"}, 1.0))


if __name__ == "__main__":
    unittest.main()
