// Package hook builds the SessionStart text: the builder soul, the builder root and the action map of the harness.
package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func mappingBlock(root, name string) string {
	b, err := os.ReadFile(filepath.Join(root, "skills", "builder", "harnesses", name+".md"))
	if err != nil {
		return ""
	}
	_, after, ok := strings.Cut(string(b), "<!-- hook -->")
	if !ok {
		return ""
	}
	inner, _, ok := strings.Cut(after, "<!-- /hook -->")
	if !ok {
		return ""
	}
	return strings.TrimSpace(inner)
}

func soul(root string) string {
	b, _ := os.ReadFile(filepath.Join(root, "skills", "builder", "soul-compact.md"))
	return strings.TrimRight(string(b), "\n")
}

// Build is the hook text before it is shaped for the harness.
func Build(root, name string, env map[string]string, home, cwd string, now float64) string {
	text := soul(root) +
		fmt.Sprintf("\n\nbuilder root: %s. Where a builder skill says ${CLAUDE_PLUGIN_ROOT}, use this path.", root)
	if m := mappingBlock(root, name); m != "" {
		return text + "\n\n" + m
	}
	return text
}

// Shape wraps the text in the JSON the harness reads.
func Shape(name, text string) string {
	var v any = map[string]any{"hookSpecificOutput": map[string]string{"hookEventName": "SessionStart", "additionalContext": text}}
	if name == "cursor" {
		v = map[string]string{"additional_context": text}
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}
