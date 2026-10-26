package reach

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const now = 1_000_000.0

func rules(t *testing.T) map[string]any {
	t.Helper()
	r, err := LoadRules("../..", filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func with(r map[string]any, kv ...any) map[string]any {
	out := map[string]any{}
	for k, v := range r {
		out[k] = v
	}
	for n := 0; n+1 < len(kv); n += 2 {
		out[kv[n].(string)] = kv[n+1]
	}
	return out
}

func sent(to, text string, at float64) map[string]any {
	return map[string]any{"at": at, "to": to, "kind": "waiting", "verdict": "send", "sha256": Digest(to, text, "self")}
}

func verdict(t *testing.T, got map[string]string, want string) {
	t.Helper()
	if got["verdict"] != want {
		t.Fatalf("got %v, want verdict %s", got, want)
	}
}

func notes(n int, step, base float64) []map[string]any {
	var log []map[string]any
	for i := 0; i < n; i++ {
		log = append(log, sent("self", fmt.Sprintf("note %d", i), base-step*float64(i)))
	}
	return log
}

func TestDefaultsMatchThePlan(t *testing.T) {
	want := map[string]any{"to_self": map[string]any{"allow": []any{"waiting", "morning", "needs-yes"}},
		"to_teammate": map[string]any{"allow": []any{}}, "to_channel": map[string]any{"allow": []any{}, "named": []any{}},
		"dedupe_s": 3600.0, "per_hour": 6.0}
	if got := rules(t); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestSelfWaitingSends(t *testing.T) {
	r := rules(t)
	if Decide(r, "self", "waiting", "") != "send" || Decide(r, "self", "chatter", "") != "draft" {
		t.Fatal()
	}
	verdict(t, Check(r, nil, "self", "waiting", "Waiting for you.", now, "", ""), "send")
}

func TestTeammateDefaultsToDraft(t *testing.T) {
	r := rules(t)
	if Decide(r, "teammate", "fyi", "") != "draft" {
		t.Fatal()
	}
	verdict(t, Check(r, nil, "teammate", "fyi", "The API changed.", now, "", ""), "draft")
}

func TestUnnamedChannelDenied(t *testing.T) {
	r := rules(t)
	named := with(r, "to_channel", map[string]any{"allow": []any{"fyi"}, "named": []any{"releases"}})
	for _, c := range []struct {
		r          map[string]any
		room, want string
	}{{r, "", "deny"}, {r, "general", "deny"}, {named, "releases", "send"}, {named, "general", "deny"}, {named, "", "deny"}} {
		if got := Decide(c.r, "channel", "fyi", c.room); got != c.want {
			t.Fatal(c.room, got)
		}
	}
	verdict(t, Check(r, nil, "channel", "fyi", "Hi all.", now, "", ""), "deny")
}

func TestDuplicateIsSkipped(t *testing.T) {
	r := rules(t)
	log := []map[string]any{sent("self", "Waiting for you.", now)}
	if got := Check(r, log, "self", "waiting", "Waiting for you.", now+3600, "", ""); !reflect.DeepEqual(got, map[string]string{"verdict": "skip", "reason": "duplicate"}) {
		t.Fatal(got)
	}
	verdict(t, Check(r, log, "self", "waiting", "Waiting for you.", now+3601, "", ""), "send")
	verdict(t, Check(r, log, "teammate", "fyi", "Waiting for you.", now+10, "", "Ana"), "draft")
}

func TestWhitespaceAndCaseDoNotBypassDedupe(t *testing.T) {
	log := []map[string]any{sent("self", "Tests are green.", now)}
	for _, again := range []string{"Tests are green.\n", "  tests ARE   green. ", "Tests\tare green."} {
		if got := Check(rules(t), log, "self", "waiting", again, now+5, "", ""); got["reason"] != "duplicate" {
			t.Fatalf("%q: %v", again, got)
		}
	}
}

func TestNormFoldsLikePython(t *testing.T) {
	if norm("STRASSE") != "strasse" || norm("straße") != "strasse" {
		t.Fatal(norm("straße"))
	}
}

func TestDigestMatchesPython(t *testing.T) {
	// python3 -c 'import sys; sys.path.insert(0,"lib"); import reach; print(reach.digest("teammate","Hello  World","Ana"))'
	if got := Digest("teammate", "Hello  World", "Ana"); got != pythonDigest {
		t.Fatal(got)
	}
}

func TestMalformedRulesUseDefaults(t *testing.T) {
	r := rules(t)
	bad := with(r, "dedupe_s", "x", "per_hour", true)
	log := notes(6, 60, now)
	if got := Check(bad, log, "self", "waiting", "note 7", now, "", ""); got["reason"] != "rate" {
		t.Fatal(got)
	}
	if got := Check(bad, log, "self", "waiting", "note 0", now+3000, "", ""); got["reason"] != "duplicate" {
		t.Fatal(got)
	}
	for _, v := range []any{nil, -1.0, "6", []any{6.0}} {
		if got := Check(with(r, "per_hour", v, "dedupe_s", v), log, "self", "waiting", "note 7", now, "", ""); got["reason"] != "rate" {
			t.Fatal(v, got)
		}
	}
	odd := map[string]any{"to_self": "x", "to_teammate": map[string]any{"allow": "fyi"}, "to_channel": nil}
	if Decide(odd, "self", "waiting", "") != "draft" || Decide(odd, "teammate", "fyi", "") != "draft" || Decide(odd, "channel", "fyi", "x") != "deny" {
		t.Fatal()
	}
}

func TestEmptyRoomNameNeverMatches(t *testing.T) {
	r := with(rules(t), "to_channel", map[string]any{"allow": []any{"fyi"}, "named": []any{nil, "", "releases"}})
	for _, name := range []string{"", "  "} {
		if Decide(r, "channel", "fyi", name) != "deny" {
			t.Fatalf("%q", name)
		}
	}
}

func TestRateLimit(t *testing.T) {
	r := rules(t)
	log := notes(6, 60, now)
	if got := Check(r, log, "self", "waiting", "note 7", now, "", ""); got["reason"] != "rate" {
		t.Fatal(got)
	}
	verdict(t, Check(r, log[:5], "self", "waiting", "note 7", now, "", ""), "send")
	verdict(t, Check(r, notes(6, 1, now-3601), "self", "waiting", "note 7", now, "", ""), "send")
	var skips []map[string]any
	for _, e := range log {
		s := map[string]any{}
		for k, v := range e {
			s[k] = v
		}
		s["verdict"] = "skip"
		skips = append(skips, s)
	}
	verdict(t, Check(r, skips, "self", "waiting", "note 7", now, "", ""), "send")
}

func TestLogAppendsJsonl(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "reach-log.jsonl")
	Record(p, map[string]any{"to": "self", "verdict": "send"})
	Record(p, map[string]any{"to": "teammate", "verdict": "draft"})
	b, _ := os.ReadFile(p)
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var e map[string]any
		json.Unmarshal([]byte(line), &e)
		got = append(got, e["to"].(string))
	}
	if !reflect.DeepEqual(got, []string{"self", "teammate"}) {
		t.Fatal(got)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Fatal(st.Mode())
	}
}

func TestBadLogLinesAreIgnored(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "reach-log.jsonl")
	b, _ := json.Marshal(sent("self", "x", now))
	os.WriteFile(p, []byte("{not json\n[1]\n"+string(b)+"\n"), 0o644)
	if n := len(ReadLog(p)); n != 1 {
		t.Fatal(n)
	}
	if n := len(ReadLog(filepath.Join(d, "none.jsonl"))); n != 0 {
		t.Fatal(n)
	}
}

func fake(t *testing.T, dir, name string) string {
	bin := filepath.Join(dir, "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, name), []byte(fmt.Sprintf("#!/bin/sh\nfor a in \"$@\"; do printf \"%%s\\n\" \"$a\"; done > \"%s/%s.args\"\n", dir, name)), 0o755)
	return bin
}

func TestNotifyMacosPassesTextAsArgument(t *testing.T) {
	d := t.TempDir()
	path := fake(t, d, "osascript")
	text := `Tests "green"; $(rm -rf /)`
	if got := Notify("builder", text, "darwin", path); got != "osascript" {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(filepath.Join(d, "osascript.args"))
	args := strings.Split(strings.TrimSpace(string(b)), "\n")
	found := false
	for _, a := range args {
		if a == text {
			found = true
		} else if strings.Contains(a, text) {
			t.Fatal("text inside the script:", a)
		}
	}
	if !found || !strings.Contains(string(b), "builder\n") {
		t.Fatal(args)
	}
}

func TestNotifyLinuxUsesNotifySend(t *testing.T) {
	d := t.TempDir()
	path := fake(t, d, "notify-send")
	if got := Notify("builder", "Waiting for you.", "linux", path); got != "notify-send" {
		t.Fatal(got)
	}
	if b, _ := os.ReadFile(filepath.Join(d, "notify-send.args")); string(b) != "builder\nWaiting for you.\n" {
		t.Fatalf("%q", b)
	}
}

func TestNormIsPythonCasefold(t *testing.T) {
	// Each want is Python's " ".join(s.split()).casefold().
	for in, want := range map[string]string{"ΟΔΟΣ ς": "οδοσ σ", "ﬁle": "file", "ŉ": "ʼn", "ſ": "s", "ẞ": "ss", "İ": "i̇", "Ǆ": "ǆ"} {
		if got := norm(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
