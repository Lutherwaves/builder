import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))
import harness  # noqa: E402

HOME = Path("/h")
OVERRIDES = {
    "claude": "CLAUDE_CONFIG_DIR",
    "codex": "CODEX_HOME",
    "cursor": "CURSOR_CONFIG_DIR",
    "opencode": "OPENCODE_CONFIG_DIR",
}
DEFAULTS = {
    "claude": HOME / ".claude",
    "codex": HOME / ".codex",
    "cursor": HOME / ".cursor",
    "opencode": HOME / ".config" / "opencode",
}


class DetectTest(unittest.TestCase):
    def test_explicit_wins_over_env(self):
        self.assertEqual(harness.detect({"CLAUDECODE": "1", "CURSOR_VERSION": "x"}, "codex"), "codex")

    def test_builder_harness_env(self):
        self.assertEqual(harness.detect({"BUILDER_HARNESS": "opencode", "CLAUDECODE": "1"}), "opencode")

    def test_invalid_explicit_falls_through(self):
        self.assertEqual(harness.detect({"CLAUDECODE": "1"}, "vim"), "claude")

    def test_auto_falls_through(self):
        self.assertEqual(harness.detect({"CLAUDECODE": "1"}, "auto"), "claude")

    def test_invalid_builder_harness_falls_through(self):
        self.assertEqual(harness.detect({"BUILDER_HARNESS": "vim", "CLAUDECODE": "1"}), "claude")

    def test_cursor_inside_claude_is_cursor(self):
        env = {"CLAUDECODE": "1", "CLAUDE_PROJECT_DIR": "/p", "CURSOR_VERSION": "1.0"}
        self.assertEqual(harness.detect(env), "cursor")

    def test_cursor_plugin_root_is_cursor(self):
        self.assertEqual(harness.detect({"CURSOR_PLUGIN_ROOT": "/x", "CLAUDE_PLUGIN_ROOT": "/y"}), "cursor")

    def test_claude_plugin_root_is_claude(self):
        self.assertEqual(harness.detect({"CLAUDE_PLUGIN_ROOT": "/x"}), "claude")

    def test_codex_only_via_explicit_or_builder_harness(self):
        env = {"CLAUDE_PLUGIN_ROOT": "/x"}
        self.assertEqual(harness.detect(env), "claude")
        self.assertEqual(harness.detect(env, "codex"), "codex")
        self.assertEqual(harness.detect({**env, "BUILDER_HARNESS": "codex"}), "codex")

    def test_empty_env_unknown(self):
        self.assertEqual(harness.detect({}), "unknown")


class PathsTest(unittest.TestCase):
    def test_override_dirs_with_space(self):
        for name, var in OVERRIDES.items():
            with self.subTest(name):
                p = harness.paths(name, {var: "/tmp/a b/x"}, HOME)
                self.assertEqual(p["config_dir"], Path("/tmp/a b/x"))

    def test_defaults_under_home(self):
        for name, want in DEFAULTS.items():
            with self.subTest(name):
                self.assertEqual(harness.paths(name, {}, HOME)["config_dir"], want)

    def test_claude_state_file(self):
        self.assertEqual(harness.paths("claude", {}, HOME)["mcp"], [(HOME / ".claude.json", "mcpServers")])
        p = harness.paths("claude", {"CLAUDE_CONFIG_DIR": "/c d"}, HOME)
        self.assertEqual(p["mcp"], [(Path("/c d/.claude.json"), "mcpServers")])

    def test_other_mcp_files(self):
        self.assertEqual(harness.paths("codex", {}, HOME)["mcp"], [(HOME / ".codex/config.toml", "mcp_servers")])
        self.assertEqual(harness.paths("cursor", {}, HOME)["mcp"], [(HOME / ".cursor/mcp.json", "mcpServers")])
        self.assertEqual(harness.paths("opencode", {}, HOME)["mcp"],
                         [(HOME / ".config/opencode/opencode.json", "mcp")])

    def test_project_mcp(self):
        want = {
            "claude": [(Path(".mcp.json"), "mcpServers")],
            "codex": [],
            "cursor": [(Path(".cursor/mcp.json"), "mcpServers")],
            "opencode": [],
        }
        for name, expected in want.items():
            with self.subTest(name):
                got = harness.paths(name, {}, HOME)["project_mcp"]
                self.assertEqual(got, expected)
                self.assertTrue(all(not f.is_absolute() for f, _ in got))

    def test_records_settings_skills(self):
        c = HOME / ".claude"
        p = harness.paths("claude", {}, HOME)
        self.assertEqual(p["plugins_record"], c / "plugins/installed_plugins.json")
        self.assertEqual(p["settings"], c / "settings.json")
        self.assertEqual(p["skills_dirs"], [c / "skills"])
        self.assertEqual(harness.paths("codex", {}, HOME)["plugins_record"], HOME / ".codex/config.toml")
        self.assertEqual(harness.paths("codex", {}, HOME)["skills_dirs"],
                         [HOME / ".codex/skills", HOME / ".agents/skills"])
        self.assertIsNone(harness.paths("codex", {}, HOME)["settings"])
        self.assertIsNone(harness.paths("cursor", {}, HOME)["plugins_record"])
        self.assertEqual(harness.paths("cursor", {}, HOME)["settings"], HOME / ".cursor/hooks.json")
        self.assertEqual(harness.paths("opencode", {}, HOME)["plugins_record"],
                         HOME / ".config/opencode/opencode.json")
        self.assertIsNone(harness.paths("opencode", {}, HOME)["settings"])

    def test_plugin_root_var(self):
        got = {n: harness.paths(n, {}, HOME)["plugin_root_var"] for n in harness.HARNESSES}
        self.assertEqual(got, {"claude": "CLAUDE_PLUGIN_ROOT", "codex": "CLAUDE_PLUGIN_ROOT",
                               "cursor": "CURSOR_PLUGIN_ROOT", "opencode": None})

    def test_unknown_harness_raises(self):
        with self.assertRaises(ValueError):
            harness.paths("vim", {}, HOME)


class BuilderHomeTest(unittest.TestCase):
    def test_default(self):
        self.assertEqual(harness.builder_home({}, HOME), HOME / ".config" / "builder")

    def test_override(self):
        self.assertEqual(harness.builder_home({"BUILDER_HOME": "/b c"}, HOME), Path("/b c"))

    def test_empty_override_is_default(self):
        self.assertEqual(harness.builder_home({"BUILDER_HOME": ""}, HOME), HOME / ".config" / "builder")


class ProcessMatchTest(unittest.TestCase):
    def test_union_has_all_four(self):
        names = {p for _, p, _ in harness.all_process_match()}
        self.assertTrue({"claude", "codex", "opencode"} <= names)
        self.assertEqual(len(harness.all_process_match()), 4)

    def test_cursor_match_is_argv_not_bare_agent(self):
        mode, pattern, exclude = harness.paths("cursor", {}, HOME)["process_match"][0]
        self.assertEqual(mode, "f")
        self.assertIn("cursor-agent/versions", pattern)
        self.assertNotEqual(pattern, "agent")
        self.assertEqual(exclude, "worker-server")

    def test_codex_excludes_app_server(self):
        self.assertEqual(harness.paths("codex", {}, HOME)["process_match"], (("x", "codex", "app-server"),))

    def test_claude_and_opencode(self):
        self.assertEqual(harness.paths("claude", {}, HOME)["process_match"], (("x", "claude", None),))
        self.assertEqual(harness.paths("opencode", {}, HOME)["process_match"], (("x", "opencode", None),))


if __name__ == "__main__":
    unittest.main()
