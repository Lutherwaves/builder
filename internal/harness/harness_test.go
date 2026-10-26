package harness

import (
	"reflect"
	"strings"
	"testing"
)

const home = "/h"

var overrides = map[string]string{"claude": "CLAUDE_CONFIG_DIR", "codex": "CODEX_HOME", "cursor": "CURSOR_CONFIG_DIR", "opencode": "OPENCODE_CONFIG_DIR"}
var defaults = map[string]string{"claude": "/h/.claude", "codex": "/h/.codex", "cursor": "/h/.cursor", "opencode": "/h/.config/opencode"}

func env(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func paths(t *testing.T, name string, e map[string]string) Paths {
	t.Helper()
	p, err := For(name, e, home)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExplicitWinsOverEnv(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CURSOR_VERSION", "x"), "codex"), "codex")
}
func TestBuilderHarnessEnv(t *testing.T) {
	eq(t, Detect(env("BUILDER_HARNESS", "opencode", "CLAUDECODE", "1"), ""), "opencode")
}
func TestInvalidExplicitFallsThrough(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1"), "vim"), "claude")
}
func TestAutoFallsThrough(t *testing.T) { eq(t, Detect(env("CLAUDECODE", "1"), "auto"), "claude") }
func TestInvalidBuilderHarnessFallsThrough(t *testing.T) {
	eq(t, Detect(env("BUILDER_HARNESS", "vim", "CLAUDECODE", "1"), ""), "claude")
}
func TestInheritedCursorMarkerDoesNotHideClaude(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CLAUDE_PROJECT_DIR", "/p", "CURSOR_VERSION", "1.0"), "auto"), "claude")
}
func TestCursorMarkerAloneIsUnknown(t *testing.T) {
	eq(t, Detect(env("CURSOR_PLUGIN_ROOT", "/x"), ""), "unknown")
}
func TestCursorArrivesOnlyExplicitly(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CURSOR_VERSION", "1"), "cursor"), "cursor")
	eq(t, Detect(env("BUILDER_HARNESS", "cursor", "CLAUDECODE", "1"), ""), "cursor")
}
func TestClaudePluginRootIsClaude(t *testing.T) {
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), ""), "claude")
}
func TestCodexOnlyViaExplicitOrBuilderHarness(t *testing.T) {
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), ""), "claude")
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), "codex"), "codex")
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x", "BUILDER_HARNESS", "codex"), ""), "codex")
}
func TestEmptyEnvUnknown(t *testing.T) { eq(t, Detect(env(), ""), "unknown") }

func TestOverrideDirsWithSpace(t *testing.T) {
	for name, v := range overrides {
		eq(t, paths(t, name, env(v, "/tmp/a b/x")).ConfigDir, "/tmp/a b/x")
	}
}
func TestDefaultsUnderHome(t *testing.T) {
	for name, want := range defaults {
		eq(t, paths(t, name, env()).ConfigDir, want)
	}
}
func TestClaudeStateFile(t *testing.T) {
	eq(t, paths(t, "claude", env()).MCP, []MCPFile{{"/h/.claude.json", "mcpServers"}})
	eq(t, paths(t, "claude", env("CLAUDE_CONFIG_DIR", "/c d")).MCP, []MCPFile{{"/c d/.claude.json", "mcpServers"}})
}
func TestOtherMCPFiles(t *testing.T) {
	eq(t, paths(t, "codex", env()).MCP, []MCPFile{{"/h/.codex/config.toml", "mcp_servers"}})
	eq(t, paths(t, "cursor", env()).MCP, []MCPFile{{"/h/.cursor/mcp.json", "mcpServers"}})
	eq(t, paths(t, "opencode", env()).MCP, []MCPFile{{"/h/.config/opencode/opencode.json", "mcp"}})
}
func TestProjectMCP(t *testing.T) {
	want := map[string][]MCPFile{
		"claude": {{".mcp.json", "mcpServers"}}, "codex": nil,
		"cursor": {{".cursor/mcp.json", "mcpServers"}}, "opencode": nil,
	}
	for name, w := range want {
		got := paths(t, name, env()).ProjectMCP
		eq(t, got, w)
		for _, f := range got {
			if strings.HasPrefix(f.Path, "/") {
				t.Fatalf("%s: %s is absolute", name, f.Path)
			}
		}
	}
}
func TestRecordsSettingsSkills(t *testing.T) {
	p := paths(t, "claude", env())
	eq(t, p.PluginsRecord, "/h/.claude/plugins/installed_plugins.json")
	eq(t, p.Settings, "/h/.claude/settings.json")
	eq(t, p.SkillsDirs, []string{"/h/.claude/skills"})
	c := paths(t, "codex", env())
	eq(t, c.PluginsRecord, "/h/.codex/config.toml")
	eq(t, c.SkillsDirs, []string{"/h/.codex/skills", "/h/.agents/skills"})
	eq(t, c.Settings, "")
	eq(t, paths(t, "cursor", env()).PluginsRecord, "")
	eq(t, paths(t, "cursor", env()).Settings, "/h/.cursor/hooks.json")
	eq(t, paths(t, "opencode", env()).PluginsRecord, "/h/.config/opencode/opencode.json")
	eq(t, paths(t, "opencode", env()).Settings, "")
}
func TestPluginRootVar(t *testing.T) {
	got := map[string]string{}
	for _, n := range Names {
		got[n] = paths(t, n, env()).PluginRootVar
	}
	eq(t, got, map[string]string{"claude": "CLAUDE_PLUGIN_ROOT", "codex": "CLAUDE_PLUGIN_ROOT", "cursor": "CURSOR_PLUGIN_ROOT", "opencode": ""})
}
func TestUnknownHarnessRaises(t *testing.T) {
	if _, err := For("vim", env(), home); err == nil {
		t.Fatal("want an error")
	}
}

func TestDefault(t *testing.T)  { eq(t, BuilderHome(env(), home), "/h/.config/builder") }
func TestOverride(t *testing.T) { eq(t, BuilderHome(env("BUILDER_HOME", "/b c"), home), "/b c") }
func TestEmptyOverrideIsDefault(t *testing.T) {
	eq(t, BuilderHome(env("BUILDER_HOME", ""), home), "/h/.config/builder")
}

func TestUnionHasAllFour(t *testing.T) {
	names := map[string]bool{}
	for _, m := range AllProcessMatch() {
		names[m.Pattern] = true
	}
	for _, n := range []string{"claude", "codex", "opencode"} {
		if !names[n] {
			t.Fatalf("missing %s", n)
		}
	}
	eq(t, len(AllProcessMatch()), 4)
}
func TestCursorMatchIsArgvNotBareAgent(t *testing.T) {
	m := paths(t, "cursor", env()).ProcessMatch[0]
	eq(t, m.Mode, byte('f'))
	if !strings.Contains(m.Pattern, "cursor-agent/versions") || m.Pattern == "agent" {
		t.Fatalf("pattern %q", m.Pattern)
	}
	eq(t, m.Exclude, "worker-server")
}
func TestCodexExcludesAppServer(t *testing.T) {
	eq(t, paths(t, "codex", env()).ProcessMatch, []Match{{'x', "codex", "app-server"}})
}
func TestClaudeAndOpencode(t *testing.T) {
	eq(t, paths(t, "claude", env()).ProcessMatch, []Match{{'x', "claude", ""}})
	eq(t, paths(t, "opencode", env()).ProcessMatch, []Match{{'x', "opencode", ""}})
}
