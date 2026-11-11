// A removed skill or command stays out of every tracked file.
package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var removedThing = regexp.MustCompile(`(\bbuilder (ledger|reach|budget|mcp-merge)|\bskills/(budget|canon|fleet|harness|intake|reach)\b|` + "`(budget|canon|fleet|intake|reach)` skill" + `|\bsend rule|recommendations\.json)`)

// These files name a removed thing on purpose, to prove its removal.
var mayNameRemovedThings = map[string]bool{
	"CHANGELOG.md":            true,
	"tests/removed_test.go":   true,
	"tests/templates_test.go": true, // TestNoSendRuleInTheBuilderSkill holds the words "send rule"
}

func TestNoFileTellsAnyoneToUseARemovedThing(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if strings.HasPrefix(rel, "docs/") || mayNameRemovedThings[rel] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue // a tracked file deleted in the working tree
		}
		for i, line := range strings.Split(string(b), "\n") {
			if m := removedThing.FindString(line); m != "" {
				t.Errorf("%s:%d names a removed thing (%q): %s", rel, i+1, m, strings.TrimSpace(line))
			}
		}
	}
}
