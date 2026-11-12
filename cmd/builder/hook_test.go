package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookCommandPrintsJSONForClaudeAndTextForOpencode(t *testing.T) {
	d := cliEnv(t)
	t.Setenv("PATH", filepath.Join(d, "nobin"))
	t.Chdir(d)
	code, out, _ := call(t, "hook", "--harness", "claude", "--extra-ignored")
	var v map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &v) != nil || v["hookSpecificOutput"] == nil {
		t.Fatalf("%d %q", code, out)
	}
	if code, out, _ = call(t, "hook", "--harness", "opencode"); code != 0 || strings.HasPrefix(out, "{") {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, _ = call(t, "hook", "--format", "yaml"); code != 2 {
		t.Fatal(code)
	}
}
