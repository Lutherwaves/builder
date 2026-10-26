package security

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const P = "CVSS:3.1/"

func finding(over ...any) map[string]any {
	f := map[string]any{
		"id": "memory/CWE-409/app/upload.py/extract", "dimension": "memory", "cwe": "CWE-409",
		"location": "app/upload.py:41", "vector": P + "AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
		"title": "One uploaded zip fills the disk and stops the service",
	}
	for n := 0; n+1 < len(over); n += 2 {
		if over[n+1] == del {
			delete(f, over[n].(string))
		} else {
			f[over[n].(string)] = over[n+1]
		}
	}
	return f
}

type deleted struct{}

var del = deleted{}

func doc(findings []any, mode string, dims ...any) map[string]any {
	if findings == nil {
		findings = []any{finding()}
	}
	if dims == nil {
		dims = []any{"memory", "osi-l7"}
	}
	return map[string]any{"mode": mode, "repo": "/tmp/repo", "sha": strings.Repeat("a", 40), "dimensions": dims, "findings": findings}
}

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func ids(list []map[string]any) []any {
	out := []any{}
	for _, f := range list {
		out = append(out, f["id"])
	}
	return out
}

var known = map[string]float64{
	"AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": 9.8, "AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H": 10.0,
	"AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H": 7.5, "AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N": 6.1,
	"AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H": 7.8, "AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:L/A:N": 6.4,
	"AV:P/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N": 1.6,
}

func TestKnownScores(t *testing.T) {
	for v, want := range known {
		if got, err := BaseScore(P + v); err != nil || got != want {
			t.Fatalf("%s: %v %v", v, got, err)
		}
	}
}

func TestNoImpactIsZero(t *testing.T) {
	s, _ := BaseScore(P + "AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N")
	if s != 0 || Rating(s) != "None" {
		t.Fatal(s)
	}
}

func TestOrderDoesNotMatter(t *testing.T) {
	parts := strings.Split("AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:L/A:N", "/")
	rand.New(rand.NewSource(7)).Shuffle(len(parts), func(a, b int) { parts[a], parts[b] = parts[b], parts[a] })
	if s, _ := BaseScore(P + strings.Join(parts, "/")); s != 6.4 {
		t.Fatal(s)
	}
}

func TestRejects(t *testing.T) {
	for v, token := range map[string]string{
		"CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": "CVSS:3.0",
		P + "AV:N/AV:L/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": "AV",
		P + "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H":          "A",
		P + "AV:X/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H":      "AV:X",
		"": "CVSS:3.1",
	} {
		if _, err := BaseScore(v); err == nil || !strings.Contains(err.Error(), token) {
			t.Fatalf("%q: %v", v, err)
		}
	}
}

func TestRejectMessagesMatchPython(t *testing.T) {
	for v, want := range map[any]string{
		nil:                                   "vector must be a string like CVSS:3.1/..., got None",
		"CVSS:3.0/AV:N":                       "vector must start with CVSS:3.1/, got 'CVSS:3.0'",
		P + "AV:N/AV:L":                       "duplicate metric AV in 'CVSS:3.1/AV:N/AV:L'",
		P + "AV:X":                            "bad metric AV:X in 'CVSS:3.1/AV:X'",
		P + "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H": "missing metric A in 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H'",
	} {
		if _, err := BaseScore(v); err == nil || err.Error() != want {
			t.Fatalf("%v: got %v", v, err)
		}
	}
}

func TestRatingBounds(t *testing.T) {
	for s, want := range map[float64]string{0.1: "Low", 3.9: "Low", 4.0: "Medium", 6.9: "Medium", 7.0: "High", 8.9: "High", 9.0: "Critical", 10.0: "Critical"} {
		if Rating(s) != want {
			t.Fatal(s)
		}
	}
}

func TestCheckCwe(t *testing.T) {
	if c, err := CheckCWE("CWE-79"); err != nil || c != "CWE-79" {
		t.Fatal(c, err)
	}
	for _, bad := range []any{"cwe-79", "79", "CWE-079", "CWE-", "CWE-0", nil, 79.0} {
		if _, err := CheckCWE(bad); err == nil {
			t.Fatal(bad)
		}
	}
}

func TestTop25(t *testing.T) {
	if in, _ := InTop25("CWE-79"); !in {
		t.Fatal()
	}
	if in, _ := InTop25("CWE-409"); in {
		t.Fatal()
	}
	seen := map[string]bool{}
	for _, c := range Top25_2025 {
		seen[c] = true
	}
	if len(Top25_2025) != 25 || len(seen) != 25 {
		t.Fatal(len(seen))
	}
}

func TestAddsScoreRatingTop25(t *testing.T) {
	out, err := Enrich(doc(nil, "audit"))
	if err != nil {
		t.Fatal(err)
	}
	f := out["findings"].([]map[string]any)[0]
	if f["score"] != 7.5 || f["rating"] != "High" || f["top25"] != false || f["title"] == nil {
		t.Fatal(f)
	}
}

func TestRejectsBadFindings(t *testing.T) {
	fid := finding()["id"].(string)
	for name, d := range map[string]map[string]any{
		"dimension": doc([]any{finding("dimension", "vibes")}, "audit"),
		"mode":      doc(nil, "quick"),
		"duplicate": doc([]any{finding(), finding()}, "audit"),
		"cwe":       doc([]any{finding("cwe", "CWE-0x")}, "audit"),
		"vector":    doc([]any{finding("vector", "CVSS:3.1/AV:N")}, "audit"),
	} {
		_, err := Enrich(d)
		if err == nil || name != "mode" && !strings.Contains(err.Error(), fid) {
			t.Fatal(name, err)
		}
	}
}

func TestRejectsMissingOrMalformedFields(t *testing.T) {
	without := func(k string) map[string]any { d := doc(nil, "audit"); delete(d, k); return d }
	withKey := func(k string, v any) map[string]any { d := doc(nil, "audit"); d[k] = v; return d }
	for name, d := range map[string]any{
		"no findings key": without("findings"), "no dimensions key": without("dimensions"),
		"no sha": without("sha"), "no repo": without("repo"), "relative repo": withKey("repo", "."),
		"findings not a list": withKey("findings", "oops"), "finding not a dict": doc([]any{"oops"}, "audit"),
		"vector null": doc([]any{finding("vector", nil)}, "audit"), "no id": doc([]any{finding("id", del)}, "audit"),
		"empty title": doc([]any{finding("title", "")}, "audit"), "no location": doc([]any{finding("location", del)}, "audit"),
		"not a dict": []any{"oops"},
	} {
		if _, err := Enrich(d); err == nil {
			t.Fatal(name)
		}
	}
}

func TestEmptyFindingsListIsFine(t *testing.T) {
	out, err := Enrich(doc([]any{}, "audit"))
	if err != nil || len(out["findings"].([]map[string]any)) != 0 {
		t.Fatal(out, err)
	}
}

func TestRejectsUnknownRunDimension(t *testing.T) {
	if _, err := Enrich(doc(nil, "audit", "memory", "vibes")); err == nil || err.Error() != "unknown dimension 'vibes' in dimensions" {
		t.Fatal(err)
	}
}

func enriched(t *testing.T, d map[string]any) map[string]any {
	t.Helper()
	out, err := Enrich(d)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func prior(firstSeen string, over ...any) map[string]any {
	f := finding(over...)
	f["first_seen"], f["first_sha"] = firstSeen, strings.Repeat("b", 40)
	return map[string]any{"repo": "/tmp/repo", "sha": strings.Repeat("b", 40), "date": firstSeen, "dimensions": []any{"memory"}, "open": []any{f}}
}

func TestFirstAuditAllNew(t *testing.T) {
	out := Diff(nil, enriched(t, doc(nil, "audit")), day("2026-10-24"))
	if len(out["new"]) != 1 || out["new"][0]["first_seen"] != "2026-10-24" || out["new"][0]["first_sha"] != strings.Repeat("a", 40) {
		t.Fatal(out)
	}
	if len(out["still_open"])+len(out["closed"])+len(out["not_rechecked"]) != 0 {
		t.Fatal(out)
	}
}

func TestStillOpenKeepsFirstSeen(t *testing.T) {
	out := Diff(prior("2026-10-01"), enriched(t, doc(nil, "audit")), day("2026-10-24"))
	f := out["still_open"][0]
	if f["first_seen"] != "2026-10-01" || f["days_open"] != 23 || f["first_sha"] != strings.Repeat("b", 40) || len(out["new"]) != 0 {
		t.Fatal(f)
	}
}

func TestClosedOnlyWhenDimensionRan(t *testing.T) {
	d := day("2026-10-24")
	ran := Diff(prior("2026-10-01"), enriched(t, doc([]any{}, "audit", "memory")), d)
	if !reflect.DeepEqual(ids(ran["closed"]), []any{finding()["id"]}) {
		t.Fatal(ran)
	}
	skipped := Diff(prior("2026-10-01"), enriched(t, doc([]any{}, "audit", "osi-l7")), d)
	if len(skipped["closed"]) != 0 || skipped["not_rechecked"][0]["first_seen"] != "2026-10-01" || skipped["not_rechecked"][0]["days_open"] != 23 {
		t.Fatal(skipped)
	}
}

func TestChangeModeNeverCloses(t *testing.T) {
	out := Diff(prior("2026-10-01"), enriched(t, doc([]any{}, "change", "memory")), day("2026-10-24"))
	if len(out["closed"]) != 0 || !reflect.DeepEqual(ids(out["not_rechecked"]), []any{finding()["id"]}) {
		t.Fatal(out)
	}
}

func TestNextBaselineDropsClosed(t *testing.T) {
	other := finding("id", "authz/CWE-862/app/admin.py/delete", "dimension", "authz", "cwe", "CWE-862")
	other["first_seen"], other["first_sha"] = "2026-10-01", strings.Repeat("b", 40)
	p := prior("2026-10-01")
	p["open"] = append(p["open"].([]any), other)
	nxt := NextBaseline(p, enriched(t, doc([]any{}, "audit", "memory")), day("2026-10-24"))
	open := nxt["open"].([]map[string]any)
	if !reflect.DeepEqual(ids(open), []any{other["id"]}) || nxt["sha"] != strings.Repeat("a", 40) || nxt["date"] != "2026-10-24" || !reflect.DeepEqual(nxt["dimensions"], []any{"memory"}) {
		t.Fatal(nxt)
	}
	if _, has := open[0]["days_open"]; has {
		t.Fatal("days_open kept")
	}
}

func TestMissingFileIsNone(t *testing.T) {
	if b, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json")); b != nil || err != nil {
		t.Fatal(b, err)
	}
}

func TestCorruptFileRaises(t *testing.T) {
	bad, _ := json.Marshal(map[string]any{"open": []any{finding("first_seen", "bogus")}})
	intDate, _ := json.Marshal(map[string]any{"open": []any{finding("first_seen", 7)}})
	p := filepath.Join(t.TempDir(), "bad.json")
	for _, text := range []string{"{not json", `{"open": "x"}`, "[]", string(bad), string(intDate)} {
		os.WriteFile(p, []byte(text), 0o644)
		if _, err := LoadBaseline(p); err == nil || !strings.Contains(err.Error(), p) {
			t.Fatal(text, err)
		}
	}
}

func TestKeyIsStable(t *testing.T) {
	d := t.TempDir()
	repo := filepath.Join(d, "repo")
	os.Mkdir(repo, 0o755)
	os.Symlink(repo, filepath.Join(d, "link"))
	env := map[string]string{"BUILDER_HOME": filepath.Join(d, "home")}
	paths := map[string]bool{}
	for _, r := range []string{repo, repo + "/", filepath.Join(d, "link")} {
		paths[BaselinePath(r, env, d)] = true
	}
	if len(paths) != 1 {
		t.Fatal(paths)
	}
	for p := range paths {
		if filepath.Dir(p) != filepath.Join(d, "home", "security") || len(filepath.Base(p)) != 16+len(".json") {
			t.Fatal(p)
		}
	}
}

func TestBaselinePathMatchesPython(t *testing.T) {
	// python3 -c 'import sys; sys.path.insert(0,"lib"); import security; print(security.baseline_path("/tmp/no-such-repo", {"BUILDER_HOME": "/b"}, "/h"))'
	if got := BaselinePath("/tmp/no-such-repo", map[string]string{"BUILDER_HOME": "/b"}, "/h"); got != pythonBaselinePath {
		t.Fatal(got)
	}
}
