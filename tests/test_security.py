import json
import random
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

LIB = Path(__file__).resolve().parent.parent / "lib"
sys.path.insert(0, str(LIB))
import security  # noqa: E402

P = "CVSS:3.1/"


def finding(**over):
    f = {
        "id": "memory/CWE-409/app/upload.py/extract",
        "dimension": "memory",
        "cwe": "CWE-409",
        "location": "app/upload.py:41",
        "vector": P + "AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
        "title": "One uploaded zip fills the disk and stops the service",
    }
    f.update(over)
    return f


def doc(findings=None, mode="audit", dimensions=("memory", "osi-l7")):
    return {
        "mode": mode,
        "repo": "/tmp/repo",
        "sha": "a" * 40,
        "dimensions": list(dimensions),
        "findings": [finding()] if findings is None else findings,
    }


class CvssTest(unittest.TestCase):
    KNOWN = {
        "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": 9.8,
        "AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H": 10.0,
        "AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H": 7.5,
        "AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N": 6.1,
        "AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H": 7.8,
        "AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:L/A:N": 6.4,
        "AV:P/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N": 1.6,
    }

    def test_known_scores(self):
        for vector, score in self.KNOWN.items():
            with self.subTest(vector=vector):
                self.assertEqual(security.base_score(P + vector), score)

    def test_no_impact_is_zero(self):
        score = security.base_score(P + "AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N")
        self.assertEqual(score, 0.0)
        self.assertEqual(security.rating(score), "None")

    def test_order_does_not_matter(self):
        parts = "AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:L/A:N".split("/")
        random.Random(7).shuffle(parts)
        self.assertEqual(security.base_score(P + "/".join(parts)), 6.4)

    def test_rejects(self):
        bad = {
            "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": "CVSS:3.0",
            P + "AV:N/AV:L/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": "AV",
            P + "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H": "A",
            P + "AV:X/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": "AV:X",
            "": "CVSS:3.1",
        }
        for vector, token in bad.items():
            with self.subTest(vector=vector):
                with self.assertRaises(ValueError) as ctx:
                    security.base_score(vector)
                self.assertIn(token, str(ctx.exception))

    def test_rating_bounds(self):
        for score, want in ((0.1, "Low"), (3.9, "Low"), (4.0, "Medium"), (6.9, "Medium"),
                            (7.0, "High"), (8.9, "High"), (9.0, "Critical"), (10.0, "Critical")):
            with self.subTest(score=score):
                self.assertEqual(security.rating(score), want)


class CweTest(unittest.TestCase):
    def test_check_cwe(self):
        self.assertEqual(security.check_cwe("CWE-79"), "CWE-79")
        for bad in ("cwe-79", "79", "CWE-079", "CWE-", "CWE-0"):
            with self.subTest(cwe=bad):
                with self.assertRaises(ValueError):
                    security.check_cwe(bad)

    def test_top25(self):
        self.assertTrue(security.in_top25("CWE-79"))
        self.assertFalse(security.in_top25("CWE-409"))
        self.assertEqual(len(security.TOP25_2025), 25)
        self.assertEqual(len(set(security.TOP25_2025)), 25)


class EnrichTest(unittest.TestCase):
    def test_adds_score_rating_top25(self):
        out = security.enrich(doc())["findings"][0]
        self.assertEqual(out["score"], 7.5)
        self.assertEqual(out["rating"], "High")
        self.assertIs(out["top25"], False)

    def test_rejects_bad_findings(self):
        fid = finding()["id"]
        cases = {
            "dimension": doc([finding(dimension="vibes")]),
            "mode": doc(mode="quick"),
            "duplicate": doc([finding(), finding()]),
            "cwe": doc([finding(cwe="CWE-0x")]),
            "vector": doc([finding(vector="CVSS:3.1/AV:N")]),
        }
        for name, d in cases.items():
            with self.subTest(case=name):
                with self.assertRaises(ValueError) as ctx:
                    security.enrich(d)
                if name != "mode":
                    self.assertIn(fid, str(ctx.exception))

    def test_rejects_unknown_run_dimension(self):
        with self.assertRaises(ValueError):
            security.enrich(doc(dimensions=("memory", "vibes")))


def cli(*args):
    return subprocess.run([sys.executable, str(LIB / "security.py"), *args],
                          capture_output=True, text=True)


class CliTest(unittest.TestCase):
    def test_cvss_cli(self):
        r = cli("cvss", P + "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
        self.assertEqual((r.returncode, r.stdout), (0, "9.8 Critical\n"))
        r = cli("cvss", P + "AV:N")
        self.assertEqual(r.returncode, 2)
        self.assertTrue(r.stderr.strip())

    def test_score_cli(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "f.json"
            path.write_text(json.dumps(doc()))
            r = cli("score", str(path))
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertEqual(json.loads(r.stdout)["findings"][0]["score"], 7.5)


class BaselineTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)
        self.env = {"BUILDER_HOME": str(self.dir / "home")}
        self.repo = self.dir / "repo"
        self.repo.mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def prior(self, first_seen="2026-10-01", **over):
        return {"repo": "/tmp/repo", "sha": "b" * 40, "date": first_seen, "dimensions": ["memory"],
                "open": [dict(finding(**over), first_seen=first_seen, first_sha="b" * 40)]}

    def test_first_audit_all_new(self):
        d = security.date(2026, 10, 24)
        out = security.diff(None, security.enrich(doc()), d)
        self.assertEqual([f["first_seen"] for f in out["new"]], ["2026-10-24"])
        self.assertEqual(out["new"][0]["first_sha"], "a" * 40)
        self.assertEqual((out["still_open"], out["closed"], out["not_rechecked"]), ([], [], []))

    def test_still_open_keeps_first_seen(self):
        out = security.diff(self.prior(), security.enrich(doc()), security.date(2026, 10, 24))
        [f] = out["still_open"]
        self.assertEqual((f["first_seen"], f["days_open"], f["first_sha"]), ("2026-10-01", 23, "b" * 40))
        self.assertEqual(out["new"], [])

    def test_closed_only_when_dimension_ran(self):
        d = security.date(2026, 10, 24)
        ran = security.diff(self.prior(), security.enrich(doc([], dimensions=("memory",))), d)
        self.assertEqual([f["id"] for f in ran["closed"]], [finding()["id"]])
        skipped = security.diff(self.prior(), security.enrich(doc([], dimensions=("osi-l7",))), d)
        self.assertEqual(skipped["closed"], [])
        [f] = skipped["not_rechecked"]
        self.assertEqual((f["first_seen"], f["days_open"]), ("2026-10-01", 23))

    def test_next_baseline_drops_closed(self):
        d = security.date(2026, 10, 24)
        other = finding(id="authz/CWE-862/app/admin.py/delete", dimension="authz", cwe="CWE-862")
        prior = self.prior()
        prior["open"].append(dict(other, first_seen="2026-10-01", first_sha="b" * 40))
        nxt = security.next_baseline(prior, security.enrich(doc([], dimensions=("memory",))), d)
        self.assertEqual([f["id"] for f in nxt["open"]], [other["id"]])
        self.assertEqual((nxt["sha"], nxt["date"], nxt["dimensions"]), ("a" * 40, "2026-10-24", ["memory"]))

    def test_missing_file_is_none(self):
        self.assertIsNone(security.load_baseline(self.dir / "absent.json"))

    def test_corrupt_file_raises(self):
        for text in ("{not json", '{"open": "x"}', "[]"):
            with self.subTest(text=text):
                path = self.dir / "bad.json"
                path.write_text(text)
                with self.assertRaises(ValueError) as ctx:
                    security.load_baseline(path)
                self.assertIn(str(path), str(ctx.exception))

    def test_key_is_stable(self):
        link = self.dir / "link"
        link.symlink_to(self.repo)
        paths = {security.baseline_path(r, self.env, self.dir) for r in
                 (str(self.repo), str(self.repo) + "/", str(link))}
        self.assertEqual(len(paths), 1)
        [path] = paths
        self.assertEqual(path.parent, self.dir / "home" / "security")

    def run_cli(self, *args, mode="audit", findings=None, today="2026-10-24"):
        d = doc(findings, mode=mode)
        d["repo"] = str(self.repo)
        path = self.dir / "findings.json"
        path.write_text(json.dumps(d))
        env = dict(self.env, PATH="/usr/bin:/bin")
        return subprocess.run([sys.executable, str(LIB / "security.py"), "baseline", *args, str(path),
                               "--today", today], capture_output=True, text=True, env=env)

    def test_save_refuses_change_mode(self):
        r = self.run_cli("save", mode="change")
        self.assertEqual(r.returncode, 2)
        self.assertIn("only an audit saves the baseline", r.stderr)
        self.assertFalse((self.dir / "home" / "security").exists())

    def test_save_then_diff_round_trip(self):
        r = self.run_cli("save", today="2026-10-01")
        self.assertEqual(r.returncode, 0, r.stderr)
        r = self.run_cli("diff", today="2026-10-24")
        self.assertEqual(r.returncode, 0, r.stderr)
        out = json.loads(r.stdout)
        self.assertEqual([f["days_open"] for f in out["still_open"]], [23])
        self.assertEqual(out["new"], [])


if __name__ == "__main__":
    unittest.main()
