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

// A "standing rule" was a pre-approved send. The soul's heading "Standing rules:" is not one, so the pattern lets it pass.
var removedThing = regexp.MustCompile(`(\bbuilder (ledger|reach|budget|mcp-merge)|\bbuilder:(budget|canon|fleet|harness|intake|reach)\b|\bskills/(budget|canon|fleet|harness|intake|reach)\b|` + "`(budget|canon|fleet|harness|intake|reach)` skill" + `|\bsend rule|(?i:\bstanding rules?($|[^s:]))|recommendations\.json)`)

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
		if strings.HasPrefix(rel, "docs/specs/") || strings.HasPrefix(rel, "docs/plans/") || mayNameRemovedThings[rel] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, rel))
		if os.IsNotExist(err) {
			continue // a tracked file deleted in the working tree
		}
		if err != nil {
			t.Errorf("cannot read %s: %v", rel, err)
			continue
		}
		for i, line := range strings.Split(string(b), "\n") {
			if m := removedThing.FindString(line); m != "" {
				t.Errorf("%s:%d names a removed thing (%q): %s", rel, i+1, m, strings.TrimSpace(line))
			}
		}
	}
}

func TestRemovedThingPattern(t *testing.T) {
	for _, line := range []string{
		"run `builder reach send`",
		"load builder:fleet for parallel sessions",
		"see skills/harness/drift.sh",
		"the `harness` skill indexes the repository",
		"use a send rule",
		"Nothing leaves without the person's yes, or a standing rule that the person gave.",
		"A standing rule counts only for the write that it names.",
		"Each write had its own yes or a standing rule for that exact write.",
	} {
		if !removedThing.MatchString(line) {
			t.Errorf("must match: %s", line)
		}
	}
	for _, line := range []string{
		"internal/harness",
		"the harness you run on",
		"Standing rules:",
		"skills/builder/harnesses/claude.md",
	} {
		if m := removedThing.FindString(line); m != "" {
			t.Errorf("must not match (%q): %s", m, line)
		}
	}
}
