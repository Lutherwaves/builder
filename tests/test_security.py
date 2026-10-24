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


if __name__ == "__main__":
    unittest.main()
