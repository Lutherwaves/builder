package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const root = "../.."

func env(t *testing.T) (map[string]string, string) {
	d := t.TempDir()
	bin := filepath.Join(d, "bin")
	os.MkdirAll(bin, 0o755)
	return map[string]string{"HOME": d, "BUILDER_HOME": filepath.Join(d, "b"), "PATH": bin}, d
}

func TestHookOnboardingNamesTheShim(t *testing.T) {
	e, home := env(t)
	text := Build(root, "claude", e, home, home, 1)
	for _, want := range []string{
		`/bin/builder" ledger next --harness claude`,
		`/bin/builder" ledger onboarded`,
		"builder onboarding: this person is new to builder.",
		"builder root: " + root + ".",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
	if strings.Contains(text, "python") {
		t.Fatal("the hook text names python")
	}
}

func TestHookAfterOnboardingShowsAtMostOneBlock(t *testing.T) {
	e, home := env(t)
	os.MkdirAll(e["BUILDER_HOME"], 0o755)
	os.WriteFile(filepath.Join(e["BUILDER_HOME"], "ledger.json"), []byte(`{"answers": {}, "onboarded": true}`), 0o644)
	text := Build(root, "claude", e, home, home, 1)
	if n := strings.Count(text, "builder recommends:"); n != 1 {
		t.Fatalf("%d blocks", n)
	}
	if strings.Contains(text, "builder onboarding") {
		t.Fatal("onboarding after onboarded")
	}
}

func TestHookShapeCursor(t *testing.T) {
	var v map[string]any
	if json.Unmarshal([]byte(Shape("cursor", "x")), &v) != nil || v["additional_context"] != "x" || len(v) != 1 {
		t.Fatal(v)
	}
}

func TestHookShapeClaude(t *testing.T) {
	var v struct {
		H map[string]string `json:"hookSpecificOutput"`
	}
	if json.Unmarshal([]byte(Shape("claude", "x<y")), &v) != nil || v.H["hookEventName"] != "SessionStart" || v.H["additionalContext"] != "x<y" {
		t.Fatal(v)
	}
}

func TestHookFailureFallsBackToSoul(t *testing.T) {
	r := t.TempDir()
	os.MkdirAll(filepath.Join(r, "skills/builder/harnesses"), 0o755)
	os.WriteFile(filepath.Join(r, "skills/builder/soul-compact.md"), []byte("SOUL\n"), 0o644)
	e, home := env(t)
	if got := Build(r, "claude", e, home, home, 1); got != "SOUL\n\nbuilder root: "+r+". Where a builder skill says ${CLAUDE_PLUGIN_ROOT}, use this path." {
		t.Fatalf("%q", got)
	}
}

func TestHookIncludesTheHarnessMapping(t *testing.T) {
	e, home := env(t)
	b, _ := os.ReadFile(filepath.Join(root, "skills/builder/harnesses/claude.md"))
	block := strings.TrimSpace(strings.SplitN(strings.SplitN(string(b), "<!-- hook -->", 2)[1], "<!-- /hook -->", 2)[0])
	if !strings.HasSuffix(Build(root, "claude", e, home, home, 1), "\n\n"+block) {
		t.Fatal("no mapping block at the end")
	}
	if strings.Contains(Build(root, "unknown", e, home, home, 1), block) {
		t.Fatal("an unknown harness gets no mapping")
	}
}

func TestHookUnderOneSecond(t *testing.T) {
	e, home := env(t)
	os.WriteFile(filepath.Join(e["PATH"], "gh"), []byte("#!/bin/bash\n/bin/sleep 5\n"), 0o755)
	start := time.Now()
	Build(root, "claude", e, home, home, 1)
	if d := time.Since(start); d >= time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestHookQuotesTheShimPathPlainly(t *testing.T) {
	e, home := env(t)
	r := t.TempDir() + `/plugin é\x`
	os.MkdirAll(filepath.Join(r, "skills/builder"), 0o755)
	os.MkdirAll(filepath.Join(r, "lib"), 0o755)
	os.WriteFile(filepath.Join(r, "skills/builder/soul-compact.md"), []byte("S\n"), 0o644)
	os.WriteFile(filepath.Join(r, "lib/recommendations.json"), []byte("[]"), 0o644)
	if text := Build(r, "claude", e, home, home, 1); !strings.Contains(text, `"`+r+`/bin/builder" ledger onboarded`) {
		t.Fatal(text)
	}
}
