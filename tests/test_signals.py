import os
import stat
import sys
import tempfile
import time
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))
import signals  # noqa: E402

BASH = "/bin/bash"
GENEROUS = 10  # first exec of a new script can be slow on macOS; timing tests set their own


class SignalsTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = Path(self.tmp.name)
        self.home = root / "home"
        self.home.mkdir()
        self.bhome = root / "bhome"
        self.bhome.mkdir()
        self.bin = root / "bin"
        self.bin.mkdir()
        self.cwd = root / "proj"
        self.cwd.mkdir()
        # Only the fake bin plus the dirs holding bash, python and coreutils basics.
        env = {"HOME": str(self.home), "BUILDER_HOME": str(self.bhome), "PATH": str(self.bin)}
        p = mock.patch.dict(os.environ, env, clear=True)
        p.start()
        self.addCleanup(p.stop)
        self.addCleanup(self.tmp.cleanup)

    def fake(self, name, body):
        f = self.bin / name
        f.write_text("#!%s\n%s\n" % (BASH, body))
        f.chmod(f.stat().st_mode | stat.S_IEXEC)

    def test_matchblox_missing_and_slow(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["matchblox"])
        self.assertIsNone(signals.collect(str(self.cwd), timeout=GENEROUS)["sessions"])
        self.fake("matchblox", "/bin/sleep 2")
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=0.5)
        self.assertLess(time.time() - t, 1.0)
        self.assertTrue(out["matchblox"])
        self.assertIsNone(out["sessions"])
        self.assertIsNone(out["weekly_pct"])

    def test_weekly_pct_from_usage(self):
        self.fake("matchblox", '''
if [ "$1" = usage ]; then
  echo '[{"windows":[{"used_pct":30,"window_minutes":300},{"used_pct":82,"window_minutes":10080}]},{"windows":[{"used_pct":40,"window_minutes":10080}]}]'
elif [ "$1" = sessions ]; then
  echo '[{"id":1},{"id":2},{"id":3}]'
fi''')
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertEqual(out["weekly_pct"], 82)
        self.assertEqual(out["sessions"], 3)

    def test_matchblox_garbage_output(self):
        self.fake("matchblox", "echo not json")
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertIsNone(out["sessions"])
        self.assertIsNone(out["weekly_pct"])

    def test_plugin_and_skill_files(self):
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertFalse(out["superpowers"])
        self.assertFalse(out["gstack"])
        (self.home / ".claude" / "plugins").mkdir(parents=True)
        (self.home / ".claude" / "plugins" / "installed_plugins.json").write_text('{"plugins":{"superpowers@x":[]}}')
        (self.home / ".claude" / "skills" / "gstack").mkdir(parents=True)
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertTrue(out["superpowers"])
        self.assertTrue(out["gstack"])

    def test_gh_and_pgrep(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["gh"])
        self.assertEqual(signals.collect(str(self.cwd), timeout=GENEROUS)["sessions_proc"], 0)
        self.fake("gh", "exit 0")
        self.fake("pgrep", "printf '11\\n22\\n'")
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertTrue(out["gh"])
        self.assertEqual(out["sessions_proc"], 2)

    def test_hanging_gh_is_bounded(self):
        self.fake("gh", "/bin/sleep 2")
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=0.5)
        self.assertLess(time.time() - t, 1.0)
        self.assertFalse(out["gh"])

    def test_finished_calls_are_read_after_a_hung_one(self):
        self.fake("matchblox", "/bin/sleep 3")
        self.fake("gh", "exit 0")
        self.fake("pgrep", "printf '1\\n2\\n3\\n'")
        # warm the fakes so first-exec cost does not eat the budget
        for n in ("gh", "pgrep"):
            os.system("%s/%s >/dev/null 2>&1" % (self.bin, n))
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=1.0)
        self.assertLess(time.time() - t, 1.6)
        self.assertTrue(out["gh"])
        self.assertEqual(out["sessions_proc"], 3)
        self.assertIsNone(out["sessions"])

    def test_channel(self):
        self.assertIsNone(signals.collect(str(self.cwd), timeout=GENEROUS)["channel"])
        (self.bhome / "reach.json").write_text('{"channel": "slack"}')
        self.assertEqual(signals.collect(str(self.cwd), timeout=GENEROUS)["channel"], "slack")
        (self.bhome / "reach.json").write_text("{bad")
        self.assertIsNone(signals.collect(str(self.cwd), timeout=GENEROUS)["channel"])

    def test_has_canon(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["has_canon"])
        (self.cwd / "DESIGN.md").write_text("x")
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["has_canon"])

    def test_has_canon_at_git_root_from_subdir(self):
        self.fake("git", 'echo "%s"' % self.cwd)
        (self.cwd / "docs" / "canon").mkdir(parents=True)
        sub = self.cwd / "a" / "b"
        sub.mkdir(parents=True)
        self.assertTrue(signals.collect(str(sub), timeout=GENEROUS)["has_canon"])


if __name__ == "__main__":
    unittest.main()
