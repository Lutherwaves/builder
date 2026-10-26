package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const vec = "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"

func findingsFile(t *testing.T, dir, repo, mode string) string {
	d := map[string]any{"mode": mode, "repo": repo, "sha": strings.Repeat("a", 40), "dimensions": []any{"memory", "osi-l7"},
		"findings": []any{map[string]any{"id": "memory/CWE-409/x", "dimension": "memory", "cwe": "CWE-409", "location": "x:1", "vector": vec, "title": "t"}}}
	b, _ := json.Marshal(d)
	p := filepath.Join(dir, "findings.json")
	os.WriteFile(p, b, 0o644)
	return p
}

func TestCvssCli(t *testing.T) {
	if code, out, _ := call(t, "security", "cvss", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"); code != 0 || out != "9.8 Critical\n" {
		t.Fatal(code, out)
	}
	if code, _, errs := call(t, "security", "cvss", "CVSS:3.1/AV:N"); code != 2 || !strings.HasPrefix(errs, "security: ") {
		t.Fatal(code, errs)
	}
}

func TestScoreCli(t *testing.T) {
	d := t.TempDir()
	code, out, errs := call(t, "security", "score", findingsFile(t, d, "/tmp/repo", "audit"))
	var v map[string]any
	json.Unmarshal([]byte(out), &v)
	if code != 0 || v["findings"].([]any)[0].(map[string]any)["score"] != 7.5 || !strings.Contains(out, "\n  \"") {
		t.Fatal(code, out, errs)
	}
}

func TestSaveRefusesChangeMode(t *testing.T) {
	d := cliEnv(t)
	repo := filepath.Join(d, "repo")
	os.Mkdir(repo, 0o755)
	code, _, errs := call(t, "security", "baseline", "save", findingsFile(t, d, repo, "change"), "--today", "2026-10-24")
	if code != 2 || !strings.Contains(errs, "only an audit saves the baseline") {
		t.Fatal(code, errs)
	}
	if _, err := os.Stat(filepath.Join(d, "b", "security")); err == nil {
		t.Fatal("a baseline dir was made")
	}
}

func TestSaveThenDiffRoundTrip(t *testing.T) {
	d := cliEnv(t)
	repo := filepath.Join(d, "repo")
	os.Mkdir(repo, 0o755)
	f := findingsFile(t, d, repo, "audit")
	if code, out, errs := call(t, "security", "baseline", "save", f, "--today", "2026-10-01"); code != 0 || !strings.Contains(out, `"saved"`) {
		t.Fatal(code, errs)
	}
	code, out, errs := call(t, "security", "baseline", "diff", "--today", "2026-10-24", f)
	var v map[string][]map[string]any
	json.Unmarshal([]byte(out), &v)
	if code != 0 || len(v["new"]) != 0 || len(v["still_open"]) != 1 || v["still_open"][0]["days_open"] != 23.0 {
		t.Fatal(code, out, errs)
	}
}

func TestBadTodayExits2(t *testing.T) {
	if code, _, errs := call(t, "security", "baseline", "diff", "f.json", "--today", "soon"); code != 2 || !strings.Contains(errs, "--today") {
		t.Fatal(code, errs)
	}
}
