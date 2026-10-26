package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fx struct {
	t                   *testing.T
	dir, results, evals string
}

func setup(t *testing.T) *fx {
	d := t.TempDir()
	f := &fx{t, d, filepath.Join(d, "results"), filepath.Join(d, "evals")}
	for _, n := range []string{"shape", "builder"} {
		os.MkdirAll(filepath.Join(f.evals, n), 0o755)
		os.WriteFile(filepath.Join(f.evals, n, "prompt.md"), []byte("x"), 0o644)
	}
	os.MkdirAll(filepath.Join(f.evals, "results"), 0o755)
	return f
}

func (f *fx) cli(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(append([]string{"--results-dir", f.results, "--evals-dir", f.evals}, args...), &out, &errb)
	return code, out.String(), errb.String()
}

func verdicts(harness string, fixtures [][2]any, met bool) []map[string]any {
	var out []map[string]any
	for _, fx := range fixtures {
		var bullets []any
		for i := 0; i < fx[1].(int); i++ {
			var quote any = "q"
			if !met {
				quote = nil
			}
			bullets = append(bullets, map[string]any{"text": fmt.Sprintf("b%d", i), "met": met, "quote": quote})
		}
		out = append(out, map[string]any{"harness": harness, "harness_version": "codex-cli 0.163.0", "date": "2026-10-22", "fixture": fx[0], "bullets": bullets})
	}
	return out
}

func (f *fx) record(v map[string]any) int {
	b, _ := json.Marshal(v)
	p := filepath.Join(f.dir, fmt.Sprintf("%s-%s.json", v["harness"], v["fixture"]))
	os.WriteFile(p, b, 0o644)
	code, _, _ := f.cli("record", p)
	return code
}

func (f *fx) render() string {
	_, out, _ := f.cli("render")
	return out
}

func (f *fx) readme(body string) string {
	p := filepath.Join(f.dir, "README.md")
	os.WriteFile(p, []byte("# x\n\n<!-- support:start -->\n"+body+"\n<!-- support:end -->\n\nrest\n"), 0o644)
	return p
}

func bullet(v map[string]any, n int) map[string]any { return v["bullets"].([]any)[n].(map[string]any) }

func TestRecordStripsToCounts(t *testing.T) {
	f := setup(t)
	v := verdicts("codex", [][2]any{{"shape", 3}}, true)[0]
	bullet(v, 1)["met"] = false
	bullet(v, 0)["quote"] = "a transcript line from /home/someone/work"
	if f.record(v) != 0 {
		t.Fatal()
	}
	b, _ := os.ReadFile(filepath.Join(f.results, "codex.json"))
	want := "{\n  \"harness\": \"codex\",\n  \"harness_version\": \"codex-cli 0.163.0\",\n  \"date\": \"2026-10-22\",\n  \"fixtures\": {\n    \"shape\": {\n      \"met\": 2,\n      \"total\": 3\n    }\n  }\n}\n"
	if string(b) != want {
		t.Fatalf("%q", b)
	}
}

func TestRecordMergesFixturesInOrder(t *testing.T) {
	f := setup(t)
	for _, v := range verdicts("codex", [][2]any{{"shape", 3}, {"builder", 4}}, true) {
		if f.record(v) != 0 {
			t.Fatal()
		}
	}
	b, _ := os.ReadFile(filepath.Join(f.results, "codex.json"))
	if i, j := strings.Index(string(b), `"shape"`), strings.Index(string(b), `"builder"`); i < 0 || j < i {
		t.Fatalf("order lost: %s", b)
	}
	var v map[string]any
	json.Unmarshal(b, &v)
	if !reflect.DeepEqual(v["fixtures"], map[string]any{"shape": map[string]any{"met": 3.0, "total": 3.0}, "builder": map[string]any{"met": 4.0, "total": 4.0}}) {
		t.Fatal(v)
	}
}

func TestRecordRefusesUnmetNull(t *testing.T) {
	f := setup(t)
	v := verdicts("codex", [][2]any{{"shape", 3}}, true)[0]
	bullet(v, 0)["met"] = nil
	b, _ := json.Marshal(v)
	p := filepath.Join(f.dir, "v.json")
	os.WriteFile(p, b, 0o644)
	code, _, errs := f.cli("record", p)
	if _, err := os.Stat(filepath.Join(f.results, "codex.json")); code != 1 || !strings.Contains(errs, "met") || err == nil {
		t.Fatal(code, errs)
	}
}

func TestRecordRefusesPathInVersion(t *testing.T) {
	f := setup(t)
	v := verdicts("codex", [][2]any{{"shape", 3}}, true)[0]
	v["harness_version"] = "/home/me/bin/codex 1"
	if f.record(v) != 1 {
		t.Fatal()
	}
}

func TestRecordRefusesUnknownHarness(t *testing.T) {
	f := setup(t)
	if f.record(verdicts("vim", [][2]any{{"shape", 3}}, true)[0]) != 1 {
		t.Fatal()
	}
}

func lines(s string) []string { return strings.Split(strings.TrimRight(s, "\n"), "\n") }

func has(list []string, s string) bool {
	for _, l := range list {
		if l == s {
			return true
		}
	}
	return false
}

func TestRenderSupportedVsExperimental(t *testing.T) {
	f := setup(t)
	for _, v := range append(verdicts("claude", [][2]any{{"shape", 3}, {"builder", 4}}, true), verdicts("codex", [][2]any{{"shape", 3}, {"builder", 4}}, true)...) {
		f.record(v)
	}
	bad := verdicts("codex", [][2]any{{"builder", 4}}, true)[0]
	bullet(bad, 0)["met"] = false
	f.record(bad)
	out := lines(f.render())
	if out[0] != "| Harness | Version | Eval date | Status |" ||
		!has(out, "| Claude Code | codex-cli 0.163.0 | 2026-10-22 | supported |") ||
		!has(out, "| Codex CLI | codex-cli 0.163.0 | 2026-10-22 | experimental: builder 3/4 |") {
		t.Fatal(out)
	}
}

func TestOnePassingFixtureOfTheSuiteIsNotSupported(t *testing.T) {
	f := setup(t)
	f.record(verdicts("codex", [][2]any{{"shape", 3}}, true)[0])
	if out := f.render(); !strings.Contains(out, "experimental: 1 not graded") || strings.Contains(out, "supported |") {
		t.Fatal(out)
	}
	bad := verdicts("codex", [][2]any{{"builder", 4}}, true)[0]
	bullet(bad, 0)["met"] = false
	f.record(bad)
	if out := f.render(); !strings.Contains(out, "experimental: builder 3/4") {
		t.Fatal(out)
	}
}

func TestAllFixturesPassingIsSupported(t *testing.T) {
	f := setup(t)
	for _, v := range verdicts("codex", [][2]any{{"shape", 3}, {"builder", 4}}, true) {
		f.record(v)
	}
	if !strings.Contains(f.render(), "| Codex CLI | codex-cli 0.163.0 | 2026-10-22 | supported |") {
		t.Fatal(f.render())
	}
}

func TestNoResultsNoRow(t *testing.T) {
	f := setup(t)
	if code, out, _ := f.cli("render"); code != 0 || strings.TrimSpace(out) != noneNote {
		t.Fatal(code, out)
	}
	f.record(verdicts("cursor", [][2]any{{"shape", 3}}, true)[0])
	if out := f.render(); !strings.Contains(out, "Cursor CLI") || strings.Contains(out, "Codex CLI") || strings.Contains(out, "opencode") {
		t.Fatal(out)
	}
}

func TestCheckPassesWithNoResultsAndDetectsDrift(t *testing.T) {
	f := setup(t)
	readme := f.readme(noneNote)
	if code, _, _ := f.cli("check", readme); code != 0 {
		t.Fatal(code)
	}
	f.record(verdicts("codex", [][2]any{{"shape", 3}}, true)[0])
	if code, _, errs := f.cli("check", readme); code != 1 || !strings.Contains(errs, "render") {
		t.Fatal(code, errs)
	}
	b, _ := os.ReadFile(readme)
	os.WriteFile(readme, []byte(strings.Replace(string(b), noneNote, strings.TrimSpace(f.render()), 1)), 0o644)
	if code, out, _ := f.cli("check", readme); code != 0 || out != "support table ok\n" {
		t.Fatal(code, out)
	}
}

func TestCheckFailsWithoutMarkers(t *testing.T) {
	f := setup(t)
	p := filepath.Join(f.dir, "README.md")
	os.WriteFile(p, []byte("# x\n"), 0o644)
	if code, _, _ := f.cli("check", p); code != 1 {
		t.Fatal(code)
	}
}

func TestRenderMatchesREADME(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--results-dir", "../../tests/evals/results", "--evals-dir", "../../tests/evals", "check", "../../README.md"}, &out, &errb); code != 0 {
		t.Fatal(errb.String())
	}
}
