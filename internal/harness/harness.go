// Package harness knows which agent harness runs builder, and where builder keeps its files.
package harness

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var Names = []string{"claude", "codex", "cursor", "grok", "opencode"}

// markers are read in order, and the first one that is set names the harness.
// The order matters: hooks/session-start exports CLAUDE_PLUGIN_ROOT on every harness, and Grok Build and Codex
// also set it so that Claude Code plugins work. So the marker that only one harness sets comes first.
// Cursor and opencode have no marker: a Claude Code session can inherit a Cursor one from its parent shell.
var markers = []struct{ env, name string }{
	{"GROK_PLUGIN_ROOT", "grok"},
	{"CLAUDECODE", "claude"},
	{"PLUGIN_ROOT", "codex"},
	{"CLAUDE_PLUGIN_ROOT", "claude"},
}

func Detect(env map[string]string, explicit string) string {
	for _, name := range []string{explicit, env["BUILDER_HARNESS"]} {
		if slices.Contains(Names, name) {
			return name
		}
	}
	for _, m := range markers {
		if env[m.env] != "" {
			return m.name
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
