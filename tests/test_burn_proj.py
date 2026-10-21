import json
import os
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "skills" / "budget" / "burn-proj.py"
SHAPE = re.compile(r"^elapsed=\S+%  naive=\S+%  profile_aware=\S+%  reset_in=\S+d$")
NOW = 1782990571  # 2026-07-02T11:09:31Z
RESET = 1783504800  # 2026-07-08T10:00:00Z


def usage(*accounts):
    return [{"provider": "anthropic", "account": name, "state": "measured", "windows": windows}
            for name, windows in accounts]


WEEKLY = {"used_pct": 19, "window_minutes": 10080, "resets_at": "2026-07-08T10:00:00+00:00"}
FIVE_H = {"used_pct": 83, "window_minutes": 300, "resets_at": "2026-07-02T14:00:00+00:00"}


class BurnProjTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.bin = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def fake_matchblox(self, payload, code=0):
        data = self.bin / "usage.json"
        data.write_text(json.dumps(payload))
        fake = self.bin / "matchblox"
        fake.write_text("#!/bin/sh\n"
                        '[ "$1" = usage ] && [ "$2" = --json ] || { echo "bad args: $*" >&2; exit 9; }\n'
                        f'/bin/cat "{data}"\nexit {code}\n')
        fake.chmod(0o755)

    def run_proj(self, *args):
        env = dict(os.environ, PATH=str(self.bin), TZ="UTC")
        return subprocess.run([sys.executable, str(SCRIPT), *args], env=env,
                              capture_output=True, text=True)

    def test_positional_args_keep_the_shape(self):
        r = self.run_proj("19", str(RESET), str(NOW))
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertRegex(r.stdout.strip(), SHAPE)

    def test_from_matchblox_shape(self):
        self.fake_matchblox(usage(("me", [FIVE_H, WEEKLY])))
        r = self.run_proj("--from-matchblox", str(NOW))
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertRegex(r.stdout.strip(), SHAPE)
        # The weekly window gives the same numbers as the positional call.
        self.assertEqual(r.stdout, self.run_proj("19", str(RESET), str(NOW)).stdout)

    def test_from_matchblox_one_line_per_account(self):
        self.fake_matchblox(usage(("work", [WEEKLY]), ("home", [dict(WEEKLY, used_pct=40)])))
        r = self.run_proj("--from-matchblox", str(NOW))
        self.assertEqual(r.returncode, 0, r.stderr)
        lines = r.stdout.strip().splitlines()
        self.assertEqual(len(lines), 2)
        self.assertTrue(lines[0].endswith("account=work"), lines[0])
        self.assertTrue(lines[1].endswith("account=home"), lines[1])

    def test_from_matchblox_no_weekly_window_fails_with_cause(self):
        self.fake_matchblox(usage(("me", [FIVE_H])))
        r = self.run_proj("--from-matchblox", str(NOW))
        self.assertEqual(r.returncode, 1)
        self.assertEqual(r.stdout, "")
        self.assertIn("no weekly window", r.stderr)

    def test_from_matchblox_error_is_not_a_zero(self):
        self.fake_matchblox([], code=4)
        r = self.run_proj("--from-matchblox", str(NOW))
        self.assertEqual(r.returncode, 1)
        self.assertEqual(r.stdout, "")
        self.assertIn("matchblox usage --json failed", r.stderr)

    def test_from_matchblox_missing_exit_2(self):
        r = self.run_proj("--from-matchblox", str(NOW))
        self.assertEqual(r.returncode, 2)
        self.assertEqual(r.stdout, "")
        self.assertEqual(r.stderr.strip(),
                         "matchblox not found: install it with curl -fsSL https://matchblox.sh | sh")


if __name__ == "__main__":
    unittest.main()
