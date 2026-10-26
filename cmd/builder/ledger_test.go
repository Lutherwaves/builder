package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lutherwaves/builder/internal/ledger"
)

func cliEnv(t *testing.T) string {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("BUILDER_HOME", filepath.Join(d, "b"))
	t.Setenv("BUILDER_ROOT", mustAbs(t, "../.."))
	t.Setenv("BUILDER_HARNESS", "")
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	return d
}

func mustAbs(t *testing.T, p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func call(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestAnswerUnwritableHomeExitsNonzero(t *testing.T) {
	d := cliEnv(t)
	f := filepath.Join(d, "file")
	os.WriteFile(f, []byte("x"), 0o644)
	t.Setenv("BUILDER_HOME", filepath.Join(f, "sub"))
	code, _, errs := call(t, "ledger", "answer", "a", "yes")
	if code == 0 || !strings.Contains(errs, "BUILDER_HOME") || !strings.HasPrefix(errs, "builder cannot write ") {
		t.Fatalf("code %d err %q", code, errs)
	}
}

func TestOnboardedCommandSetsFlagAndKeepsAnswers(t *testing.T) {
	d := cliEnv(t)
	if code, out, _ := call(t, "ledger", "answer", "a", "never"); code != 0 || out != "ok\n" {
		t.Fatal(code, out)
	}
	if code, _, _ := call(t, "ledger", "onboarded"); code != 0 {
		t.Fatal(code)
	}
	l := ledger.Load(filepath.Join(d, "b", "ledger.json"))
	if !l.Onboarded || l.Answers["a"].Verdict != "never" {
		t.Fatal(l)
	}
}

func TestLedgerUsageExits2(t *testing.T) {
	cliEnv(t)
	code, _, errs := call(t, "ledger", "answer", "a", "maybe")
	if code != 2 || errs != "usage: builder ledger answer <id> yes|later|never | builder ledger onboarded | builder ledger next [--harness H]\n" {
		t.Fatalf("%d %q", code, errs)
	}
}

func answerAllBut(t *testing.T, keep string) {
	cat, err := ledger.LoadCatalog(os.Getenv("BUILDER_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	call(t, "ledger", "onboarded")
	for _, i := range cat {
		if i["id"] != keep {
			call(t, "ledger", "answer", i["id"].(string), "never")
		}
	}
}

func nextFor(t *testing.T, h string) map[string]any {
	code, out, errs := call(t, "ledger", "next", "--harness", h)
	if code != 0 {
		t.Fatal(code, errs)
	}
	var v map[string]any
	json.Unmarshal([]byte(out), &v)
	return v
}

func TestNextCliHarnessFlag(t *testing.T) {
	d := cliEnv(t)
	t.Setenv("PATH", filepath.Join(d, "nobin"))
	t.Chdir(d)
	answerAllBut(t, "superpowers")
	codex, claude, unknown := nextFor(t, "codex"), nextFor(t, "claude"), nextFor(t, "unknown")
	if !strings.Contains(codex["how"].(string), "/plugins") || !strings.Contains(claude["how"].(string), "/plugin install") {
		t.Fatal(codex["how"], claude["how"])
	}
	a, _ := json.Marshal(unknown)
	b, _ := json.Marshal(claude)
	if string(a) != string(b) {
		t.Fatal("unknown differs from claude")
	}
}

func TestNextCliReadsTheSignalsOfItsHarness(t *testing.T) {
	d := cliEnv(t)
	t.Setenv("PATH", filepath.Join(d, "nobin"))
	t.Chdir(d)
	os.MkdirAll(filepath.Join(d, ".codex"), 0o755)
	os.WriteFile(filepath.Join(d, ".codex", "config.toml"), []byte("[plugins.\"superpowers@x\"]\nenabled = true\n"), 0o644)
	answerAllBut(t, "superpowers")
	if code, out, _ := call(t, "ledger", "next", "--harness", "codex"); code != 0 || out != "null\n" {
		t.Fatalf("%d %q", code, out)
	}
	if got := nextFor(t, "claude")["id"]; got != "superpowers" {
		t.Fatal(got)
	}
}

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
