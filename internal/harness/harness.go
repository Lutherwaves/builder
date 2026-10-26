// Package harness knows which agent harness runs builder, and where that harness keeps its files.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var Names = []string{"claude", "codex", "cursor", "opencode"}

type MCPFile struct{ Path, Key string }

// Match finds a harness process: Mode 'x' compares the command name, 'f' searches the args.
type Match struct {
	Mode             byte
	Pattern, Exclude string
}

// Paths are a harness's files. An empty string means the harness has no such file.
type Paths struct {
	ConfigDir               string
	SkillsDirs              []string
	PluginsRecord, Settings string
	MCP, ProjectMCP         []MCPFile
	ProcessMatch            []Match
	PluginRootVar           string
}

type spec struct {
	override   string
	defaultDir []string
	rootVar    string
	match      []Match
}

var specs = map[string]spec{
	"claude":   {"CLAUDE_CONFIG_DIR", []string{".claude"}, "CLAUDE_PLUGIN_ROOT", []Match{{'x', "claude", ""}}},
	"codex":    {"CODEX_HOME", []string{".codex"}, "CLAUDE_PLUGIN_ROOT", []Match{{'x', "codex", "app-server"}}},
	"cursor":   {"CURSOR_CONFIG_DIR", []string{".cursor"}, "CURSOR_PLUGIN_ROOT", []Match{{'f', `/cursor-agent/versions/[^/]+/index\.js`, "worker-server"}}},
	"opencode": {"OPENCODE_CONFIG_DIR", []string{".config", "opencode"}, "", []Match{{'x', "opencode", ""}}},
}

// envMarkers: Cursor, Codex and opencode arrive via an explicit flag or BUILDER_HARNESS.
// A Cursor env marker is ignored: a Claude Code session can inherit it from its parent shell.
var envMarkers = []string{"CLAUDECODE", "CLAUDE_PLUGIN_ROOT"}

func Detect(env map[string]string, explicit string) string {
	for _, name := range []string{explicit, env["BUILDER_HARNESS"]} {
		if slices.Contains(Names, name) {
			return name
		}
	}
	for _, m := range envMarkers {
		if env[m] != "" {
			return "claude"
		}
	}
	return "unknown"
}

func BuilderHome(env map[string]string, home string) string {
	if o := env["BUILDER_HOME"]; o != "" {
		return o
	}
	return filepath.Join(home, ".config", "builder")
}

func For(name string, env map[string]string, home string) (Paths, error) {
	s, ok := specs[name]
	if !ok {
		return Paths{}, fmt.Errorf("unknown harness: %q", name)
	}
	config := filepath.Join(append([]string{home}, s.defaultDir...)...)
	overridden := env[s.override] != ""
	if overridden {
		config = env[s.override]
	}
	p := Paths{
		ConfigDir:     config,
		SkillsDirs:    []string{filepath.Join(config, "skills")},
		ProcessMatch:  s.match,
		PluginRootVar: s.rootVar,
	}
	switch name {
	case "claude":
		state := home
		if overridden {
			state = config
		}
		p.MCP = []MCPFile{{filepath.Join(state, ".claude.json"), "mcpServers"}}
		p.ProjectMCP = []MCPFile{{".mcp.json", "mcpServers"}}
		p.PluginsRecord = filepath.Join(config, "plugins", "installed_plugins.json")
		p.Settings = filepath.Join(config, "settings.json")
	case "codex":
		p.MCP = []MCPFile{{filepath.Join(config, "config.toml"), "mcp_servers"}}
		p.PluginsRecord = filepath.Join(config, "config.toml")
		p.SkillsDirs = append(p.SkillsDirs, filepath.Join(home, ".agents", "skills"))
	case "cursor":
		p.MCP = []MCPFile{{filepath.Join(config, "mcp.json"), "mcpServers"}}
		p.ProjectMCP = []MCPFile{{filepath.Join(".cursor", "mcp.json"), "mcpServers"}}
		p.Settings = filepath.Join(config, "hooks.json")
	default:
		p.MCP = []MCPFile{{filepath.Join(config, "opencode.json"), "mcp"}}
		p.PluginsRecord = filepath.Join(config, "opencode.json")
	}
	return p, nil
}

func AllProcessMatch() []Match {
	var out []Match
	for _, n := range Names {
		out = append(out, specs[n].match...)
	}
	return out
}

// Env is the process environment as a map.
func Env() map[string]string {
	m := map[string]string{}
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}
