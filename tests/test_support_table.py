"""Test scripts/support-table.py: record, render and check."""
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent.parent / "scripts" / "support-table.py"
NONE_NOTE = "No harness has passed its evals yet."


def verdict(harness="codex", fixtures=None, met=True):
    fixtures = fixtures or {"shape": 3}
    out = []
    for name, n in fixtures.items():
        out.append({"harness": harness, "harness_version": "codex-cli 0.163.0", "date": "2026-10-22",
                    "fixture": name,
                    "bullets": [{"text": f"b{i}", "met": met, "quote": "q" if met else None} for i in range(n)]})
    return out


class SupportTable(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.dir = Path(self._tmp.name)
        self.results = self.dir / "results"
        self.evals = self.dir / "evals"
        for name in ("shape", "builder"):
            (self.evals / name).mkdir(parents=True)
            (self.evals / name / "prompt.md").write_text("x")
        (self.evals / "results").mkdir()

    def run_cli(self, *args):
        return subprocess.run([sys.executable, str(SCRIPT), "--results-dir", str(self.results),
                               "--evals-dir", str(self.evals), *args],
                              capture_output=True, text=True)

    def record(self, v):
        path = self.dir / f"{v['harness']}-{v['fixture']}.json"
        path.write_text(json.dumps(v))
        return self.run_cli("record", str(path))

    def readme(self, body):
        path = self.dir / "README.md"
        path.write_text(f"# x\n\n<!-- support:start -->\n{body}\n<!-- support:end -->\n\nrest\n")
        return path

    def test_record_strips_to_counts(self):
        v = verdict(fixtures={"shape": 3})[0]
        v["bullets"][1]["met"] = False
        v["bullets"][0]["quote"] = "a transcript line from /home/someone/work"
        r = self.record(v)
        self.assertEqual(r.returncode, 0, r.stderr)
        saved = json.loads((self.results / "codex.json").read_text())
        self.assertEqual(saved, {"harness": "codex", "harness_version": "codex-cli 0.163.0",
                                 "date": "2026-10-22", "fixtures": {"shape": {"met": 2, "total": 3}}})
        self.assertNotIn("/home", (self.results / "codex.json").read_text())

    def test_record_merges_fixtures(self):
        for v in verdict(fixtures={"shape": 3, "builder": 4}):
            self.assertEqual(self.record(v).returncode, 0)
        saved = json.loads((self.results / "codex.json").read_text())
        self.assertEqual(saved["fixtures"], {"shape": {"met": 3, "total": 3}, "builder": {"met": 4, "total": 4}})

    def test_record_refuses_unmet_null(self):
        v = verdict()[0]
        v["bullets"][0]["met"] = None
        r = self.record(v)
        self.assertEqual(r.returncode, 1)
        self.assertIn("met", r.stderr)
        self.assertFalse((self.results / "codex.json").exists())

    def test_record_refuses_path_in_version(self):
        v = verdict()[0]
        v["harness_version"] = "/home/me/bin/codex 1"
        self.assertEqual(self.record(v).returncode, 1)

    def test_record_refuses_unknown_harness(self):
        self.assertEqual(self.record(verdict(harness="vim")[0]).returncode, 1)

    def test_render_supported_vs_experimental(self):
        for v in verdict("claude", {"shape": 3, "builder": 4}) + verdict("codex", {"shape": 3, "builder": 4}):
            self.record(v)
        bad = verdict("codex", {"builder": 4})[0]
        bad["harness_version"] = "codex-cli 0.163.0"
        bad["bullets"][0]["met"] = False
        self.record(bad)
        out = self.run_cli("render").stdout.splitlines()
        self.assertEqual(out[0], "| Harness | Version | Eval date | Status |")
        self.assertIn("| Claude Code | codex-cli 0.163.0 | 2026-10-22 | supported |", out)
        self.assertIn("| Codex CLI | codex-cli 0.163.0 | 2026-10-22 | experimental: builder 3/4 |", out)

    def test_one_passing_fixture_of_the_suite_is_not_supported(self):
        self.record(verdict("codex", {"shape": 3})[0])
        out = self.run_cli("render").stdout
        self.assertIn("experimental: 1 not graded", out)
        self.assertNotIn("supported |", out)
        bad = verdict("codex", {"builder": 4})[0]
        bad["bullets"][0]["met"] = False
        self.record(bad)
        self.assertIn("experimental: builder 3/4", self.run_cli("render").stdout)

    def test_all_fixtures_passing_is_supported(self):
        for v in verdict("codex", {"shape": 3, "builder": 4}):
            self.record(v)
        self.assertIn("| Codex CLI | codex-cli 0.163.0 | 2026-10-22 | supported |", self.run_cli("render").stdout)

    def test_no_results_no_row(self):
        r = self.run_cli("render")
        self.assertEqual(r.returncode, 0)
        self.assertEqual(r.stdout.strip(), NONE_NOTE)
        self.record(verdict("cursor")[0])
        out = self.run_cli("render").stdout
        self.assertIn("Cursor CLI", out)
        self.assertNotIn("Codex CLI", out)
        self.assertNotIn("opencode", out)

    def test_check_passes_with_no_results_and_detects_drift(self):
        readme = self.readme(NONE_NOTE)
        self.assertEqual(self.run_cli("check", str(readme)).returncode, 0)
        self.record(verdict()[0])
        r = self.run_cli("check", str(readme))
        self.assertEqual(r.returncode, 1)
        self.assertIn("render", r.stderr)
        readme.write_text(readme.read_text().replace(NONE_NOTE, self.run_cli("render").stdout.strip()))
        self.assertEqual(self.run_cli("check", str(readme)).returncode, 0)

    def test_check_fails_without_markers(self):
        path = self.dir / "README.md"
        path.write_text("# x\n")
        self.assertEqual(self.run_cli("check", str(path)).returncode, 1)


if __name__ == "__main__":
    unittest.main()
