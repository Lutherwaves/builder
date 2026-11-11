// Package harness knows which agent harness runs builder, and where builder keeps its files.
package harness

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var Names = []string{"claude", "codex", "cursor", "opencode"}

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
