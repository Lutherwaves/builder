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

    def fake_ps(self, rows):
        """A fake ps that prints `pid comm args` rows, as `ps -axo pid=,comm=,args=` does."""
        self.fake("ps", "printf %s" % json.dumps("".join(r + "\n" for r in rows)).replace("%", "%%"))

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
        self.fake_ps(["11 claude claude", "22 claude claude --x", "33 vim vim claude"])
        out = signals.collect(str(self.cwd), timeout=GENEROUS)
        self.assertTrue(out["gh"])
        self.assertEqual(out["sessions_proc"], 2)

    def test_hanging_gh_is_bounded(self):
        self.fake("gh", "/bin/sleep 2")
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=0.5)
        self.assertLess(time.time() - t, 1.0)
        self.assertFalse(out["gh"])

    def test_finished_tool_whose_child_holds_stdout_is_bounded(self):
        self.fake("gh", "/bin/sleep 5 &\nexit 0")
        os.system("%s/gh >/dev/null 2>&1" % self.bin)  # warm the fake; its sleep is detached from this test
        t = time.time()
        out = signals.collect(str(self.cwd), timeout=0.5)
        self.assertLess(time.time() - t, 1.0)
        self.assertTrue(out["gh"])  # output goes to a file now, so a child that holds it open cannot block the read

    def test_finished_calls_are_read_after_a_hung_one(self):
        self.fake("matchblox", "/bin/sleep 3")
        self.fake("gh", "exit 0")
        self.fake_ps(["1 claude claude", "2 claude claude", "3 claude claude"])
        # warm the fakes so first-exec cost does not eat the budget
        for n in ("gh", "ps"):
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

    def test_has_canon_from_rules_md_that_canon_creates(self):
        (self.cwd / "RULES.md").write_text("x")
        self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["has_canon"])

    def test_canon_markers_match_the_canon_skill_and_recommendation(self):
        root = Path(signals.__file__).resolve().parent.parent
        skill = (root / "skills" / "canon" / "SKILL.md").read_text()
        for home in ("RULES.md", "DESIGN.md", "docs/rules"):
            self.assertIn(home, signals.CANON_MARKERS)
        for marker in signals.CANON_MARKERS:
            self.assertIn("`%s" % marker, skill)
        cat = json.loads((root / "lib" / "recommendations.json").read_text())
        why = next(i for i in cat if i["id"] == "canon")["why"]
        self.assertIn("RULES.md", why)
        for path in ("docs/canon", ".claude/rules"):
            self.assertNotIn(path, why)

    def test_has_canon_at_git_root_from_subdir(self):
        self.fake("git", 'echo "%s"' % self.cwd)
        (self.cwd / "docs" / "rules").mkdir(parents=True)
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

    def write_json(self, path, data):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(data if isinstance(data, str) else json.dumps(data))

    def sig(self, harness, **env):
        with mock.patch.dict(os.environ, env):
            return signals.collect(str(self.cwd), timeout=GENEROUS, harness=harness)

    def test_harness_none_uses_detect(self):
        (self.home / ".codex").mkdir()
        (self.home / ".codex" / "config.toml").write_text('[plugins."superpowers@x"]\nenabled = true\n')
        with mock.patch.dict(os.environ, {"BUILDER_HARNESS": "codex"}):
            self.assertTrue(signals.collect(str(self.cwd), timeout=GENEROUS)["superpowers"])

    def test_codex_reads_mcp_from_config_toml(self):
        cfg = self.home / ".codex"
        cfg.mkdir()
        (cfg / "config.toml").write_text('model = "x"\n\n[mcp_servers.codebase-memory-mcp]\ncommand = "x"\n')
        self.assertTrue(self.sig("codex")["code_graph"])
        (cfg / "config.toml").write_text('[mcp_servers.other]\ncommand = "x"\n')
        self.assertFalse(self.sig("codex")["code_graph"])
        (cfg / "config.toml").write_text('[mcp_servers."codebase-memory"]\ncommand = "x"\n')
        self.assertTrue(self.sig("codex")["code_graph"])

    def test_codex_toml_regex_fallback(self):
        cfg = self.home / ".codex"
        cfg.mkdir()
        (cfg / "config.toml").write_text(
            '[mcp_servers.codebase-memory-mcp]\ncommand = "x"\n[plugins."superpowers@mk"]\nenabled = true\n')
        with mock.patch.object(signals, "tomllib", None):
            out = self.sig("codex")
        self.assertTrue(out["code_graph"])
        self.assertTrue(out["superpowers"])
        (cfg / "config.toml").write_text('[mcp_servers.other]\n[plugins."other@mk"]\n')
        with mock.patch.object(signals, "tomllib", None):
            out = self.sig("codex")
        self.assertFalse(out["code_graph"])
        self.assertFalse(out["superpowers"])

    def test_codex_bad_toml_is_not_a_crash(self):
        cfg = self.home / ".codex"
        cfg.mkdir()
        (cfg / "config.toml").write_text('[mcp_servers.codebase-memory-mcp]\nbroken = = =\n')
        self.assertTrue(self.sig("codex")["code_graph"])

    def test_codex_superpowers_needs_a_plugins_table(self):
        cfg = self.home / ".codex"
        cfg.mkdir()
        (cfg / "config.toml").write_text('# superpowers@x\n[plugins."other@x"]\n')
        self.assertFalse(self.sig("codex")["superpowers"])
        (cfg / "config.toml").write_text('[plugins."superpowers@x"]\n')
        self.assertTrue(self.sig("codex", CODEX_HOME=str(cfg))["superpowers"])

    def test_cursor_mcp_json(self):
        self.fake("git", 'echo "%s"' % self.cwd)
        self.write_json(self.home / ".cursor" / "mcp.json", {"mcpServers": {"other": {}}})
        self.assertFalse(self.sig("cursor")["code_graph"])
        self.write_json(self.home / ".cursor" / "mcp.json", {"mcpServers": {"codebase-memory-mcp": {}}})
        self.assertTrue(self.sig("cursor")["code_graph"])
        (self.home / ".cursor" / "mcp.json").unlink()
        self.write_json(self.cwd / ".cursor" / "mcp.json", {"mcpServers": {"codebase-memory": {}}})
        self.assertTrue(self.sig("cursor")["code_graph"])

    def test_opencode_mcp_key(self):
        cfg = self.home / ".config" / "opencode"
        self.write_json(cfg / "opencode.json", {"mcp": {"codebase-memory-mcp": {"type": "local"}}})
        self.assertTrue(self.sig("opencode")["code_graph"])
        self.write_json(cfg / "opencode.json", {"mcp": {"other": {}}})
        self.assertFalse(self.sig("opencode")["code_graph"])

    def test_opencode_superpowers_from_plugin_list(self):
        cfg = self.home / ".config" / "opencode"
        self.write_json(cfg / "opencode.json", {"plugin": ["superpowers@git+https://x/superpowers.git"]})
        self.assertTrue(self.sig("opencode")["superpowers"])
        self.write_json(cfg / "opencode.json", {"plugin": ["other"]})
        self.assertFalse(self.sig("opencode")["superpowers"])
        self.write_json(cfg / "opencode.json", {"plugin": "superpowers"})
        self.assertFalse(self.sig("opencode")["superpowers"])

    def test_unknown_signal_is_none(self):
        out = self.sig("cursor")
        self.assertIsNone(out["superpowers"])
        self.assertIsNone(out["autocompact"])
        for h in ("codex", "cursor", "opencode"):
            self.assertIsNone(self.sig(h)["autocompact"])
        self.fake("matchblox", "exit 0")
        for h in ("codex", "opencode"):
            self.assertIsNone(self.sig(h)["matchblox_hooks"])
        self.assertFalse(self.sig("claude")["autocompact"])
        self.assertFalse(self.sig("claude")["superpowers"])

    def test_gstack_on_other_harnesses_is_any_gstack_dash_skill(self):
        for h, base in (("codex", ".codex"), ("cursor", ".cursor"), ("opencode", ".config/opencode")):
            self.assertFalse(self.sig(h)["gstack"])
            (self.home / base / "skills" / "gstack").mkdir(parents=True)
            self.assertFalse(self.sig(h)["gstack"])
            (self.home / base / "skills" / "gstack-qa").mkdir()
            self.assertTrue(self.sig(h)["gstack"])

    def test_cursor_matchblox_hooks_signal(self):
        self.fake("matchblox", "exit 0")
        hooks = self.home / ".cursor" / "hooks.json"
        self.write_json(hooks, {"version": 1, "hooks": {"stop": [{"command": "matchblox hook --cursor stop"}]}})
        self.assertTrue(self.sig("cursor")["matchblox_hooks"])
        self.write_json(hooks, {"version": 1, "hooks": {"stop": [{"command": "matchblox hook stop"}]}})
        self.assertFalse(self.sig("cursor")["matchblox_hooks"])
        self.write_json(hooks, "{bad")
        self.assertFalse(self.sig("cursor")["matchblox_hooks"])

    def test_sessions_proc_counts_all_harnesses(self):
        self.fake_ps([
            "1 claude claude", "2 /usr/bin/claude /usr/bin/claude",
            "3 codex codex", "4 opencode opencode",
            "5 node node /x/cursor-agent/versions/1/index.js",
            "6 bash bash -c echo claude",
        ])
        self.assertEqual(self.sig("claude")["sessions_proc"], 5)

    def test_cursor_worker_server_not_counted(self):
        self.fake_ps(["5 node node /x/cursor-agent/versions/1/index.js",
                      "6 node node /x/cursor-agent/versions/1/index.js worker-server"])
        self.assertEqual(self.sig("cursor")["sessions_proc"], 1)

    def test_codex_app_server_not_counted(self):
        self.fake_ps(["3 /opt/bin/codex codex", "4 /opt/bin/codex codex app-server --listen"])
        self.assertEqual(self.sig("codex")["sessions_proc"], 1)

    def test_no_claude_literal(self):
        src = Path(signals.__file__).read_text()
        self.assertNotIn("~/.claude", src)
        self.assertNotIn('".claude', src)
        self.assertFalse(hasattr(signals, "_claude_paths"))


if __name__ == "__main__":
    unittest.main()
