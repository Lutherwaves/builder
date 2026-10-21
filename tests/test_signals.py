import json
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

    def fake_git_config(self, values):
        """A fake git that answers `git config --get KEY` from a dict; exit 1 when unset."""
        body = 'if [ "$1" = config ] && [ "$2" = --get ]; then\n'
        for k, v in values.items():
            body += '  [ "$3" = %s ] && { echo %s; exit 0; }\n' % (k, v)
        body += '  exit 1\nfi\nexit 1'
        self.fake("git", body)

    def test_signing_needs_gpgsign_true_and_a_key(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["signing"])
        self.fake_git_config({"commit.gpgsign": "true", "user.signingkey": "/k/id.pub"})
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["signing"])
        self.fake_git_config({"commit.gpgsign": "true"})
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["signing"])
        self.fake_git_config({"commit.gpgsign": "false", "user.signingkey": "/k/id.pub"})
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["signing"])
        self.fake_git_config({"user.signingkey": "/k/id.pub"})
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["signing"])

    def test_hanging_git_config_is_bounded(self):
        self.fake("git", "/bin/sleep 2")
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=0.5)
        self.assertLess(time.time() - t, 1.0)
        self.assertFalse(out["signing"])

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

    def write_settings(self, base, data):
        (base / "settings.json").parent.mkdir(parents=True, exist_ok=True)
        (base / "settings.json").write_text(data if isinstance(data, str) else json.dumps(data))

    def test_matchblox_hooks_need_the_binary_and_a_hook_command(self):
        hooks = {"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "/opt/bin/matchblox hook Stop"}]}]}}
        self.write_settings(self.home / ".claude", hooks)
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["matchblox_hooks"])
        self.fake("matchblox", "exit 0")
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["matchblox_hooks"])
        other = {"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo matchblox"}]}]}}
        self.write_settings(self.home / ".claude", other)
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["matchblox_hooks"])
        self.write_settings(self.home / ".claude", "{bad")
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["matchblox_hooks"])

    def test_settings_come_from_claude_config_dir_when_set(self):
        self.fake("matchblox", "exit 0")
        cfg = Path(self.tmp.name) / "cfg"
        hooks = {"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "matchblox hook Stop"}]}]},
                 "env": {"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"}}
        self.write_settings(cfg, hooks)
        (cfg / ".claude.json").write_text('{"mcpServers": {"codebase-memory-mcp": {"command": "x"}}}')
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertFalse(out["matchblox_hooks"])
        self.assertFalse(out["autocompact"])
        self.assertFalse(out["code_graph"])
        with mock.patch.dict(os.environ, {"CLAUDE_CONFIG_DIR": str(cfg)}):
            out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertTrue(out["matchblox_hooks"])
        self.assertTrue(out["autocompact"])
        self.assertTrue(out["code_graph"])

    def test_code_graph_from_mcp_servers(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["code_graph"])
        (self.home / ".claude.json").write_text('{"mcpServers": {"other": {}}}')
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["code_graph"])
        (self.home / ".claude.json").write_text('{"mcpServers": {"codebase-memory-mcp": {"command": "x"}}}')
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["code_graph"])
        (self.home / ".claude.json").write_text('{"mcpServers": ["codebase-memory-mcp"]}')
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["code_graph"])

    def test_code_graph_from_the_repository_mcp_json_that_harness_writes(self):
        self.fake("git", 'echo "%s"' % self.cwd)
        (self.cwd / ".mcp.json").write_text('{"mcpServers": {"codebase-memory": {"command": "x"}}}')
        sub = self.cwd / "a"
        sub.mkdir()
        self.assertTrue(signals.collect(str(sub), timeout=GENEROUS)["code_graph"])
        (self.cwd / ".mcp.json").write_text('{"mcpServers": {"other": {}}}')
        self.assertFalse(signals.collect(str(sub), timeout=GENEROUS)["code_graph"])

    def test_code_graph_from_the_local_scope_in_claude_json(self):
        self.fake("git", 'echo "%s"' % self.cwd)
        sub = self.cwd / "a"
        sub.mkdir()
        local = {"projects": {str(self.cwd): {"mcpServers": {"codebase-memory-mcp": {"command": "x"}}}}}
        (self.home / ".claude.json").write_text(json.dumps(local))
        self.assertTrue(signals.collect(str(sub), timeout=GENEROUS)["code_graph"])
        local = {"projects": {str(sub.resolve()): {"mcpServers": {"codebase-memory": {}}}}}
        (self.home / ".claude.json").write_text(json.dumps(local))
        self.assertTrue(signals.collect(str(sub), timeout=GENEROUS)["code_graph"])
        local = {"projects": {"/elsewhere": {"mcpServers": {"codebase-memory-mcp": {}}}, str(self.cwd): "bad"}}
        (self.home / ".claude.json").write_text(json.dumps(local))
        self.assertFalse(signals.collect(str(sub), timeout=GENEROUS)["code_graph"])

    def test_plugins_and_gstack_come_from_claude_config_dir_when_set(self):
        cfg = Path(self.tmp.name) / "cfg"
        (cfg / "plugins").mkdir(parents=True)
        (cfg / "plugins" / "installed_plugins.json").write_text('{"plugins":{"superpowers@x":[]}}')
        (cfg / "skills" / "gstack").mkdir(parents=True)
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["superpowers"])
        with mock.patch.dict(os.environ, {"CLAUDE_CONFIG_DIR": str(cfg)}):
            out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertTrue(out["superpowers"])
        self.assertTrue(out["gstack"])
        (self.home / ".claude" / "skills" / "gstack").mkdir(parents=True)
        with mock.patch.dict(os.environ, {"CLAUDE_CONFIG_DIR": str(Path(self.tmp.name) / "empty")}):
            out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertFalse(out["gstack"])

    def test_autocompact_from_settings_env_or_process_env(self):
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["autocompact"])
        self.write_settings(self.home / ".claude", {"env": {"OTHER": "1"}})
        self.assertFalse(signals.collect(str(self.cwd), timeout=GENEROUS)["autocompact"])
        self.write_settings(self.home / ".claude", {"env": {"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"}})
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["autocompact"])
        self.write_settings(self.home / ".claude", {})
        with mock.patch.dict(os.environ, {"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"}):
            self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["autocompact"])


if __name__ == "__main__":
    unittest.main()
