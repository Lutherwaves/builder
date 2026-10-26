package budget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var shape = regexp.MustCompile(`^elapsed=\S+%  naive=\S+%  profile_aware=\S+%  reset_in=\S+d$`)

const (
	now   = 1782990571 // 2026-07-02T11:09:31Z
	reset = 1783504800 // 2026-07-08T10:00:00Z
)

var (
	weekly = map[string]any{"used_pct": 19, "window_minutes": 10080, "resets_at": "2026-07-08T10:00:00+00:00"}
	fiveH  = map[string]any{"used_pct": 83, "window_minutes": 300, "resets_at": "2026-07-02T14:00:00+00:00"}
)

func zone(t *testing.T, name string) {
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = old })
}

func with(m map[string]any, kv ...any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	for n := 0; n+1 < len(kv); n += 2 {
		out[kv[n].(string)] = kv[n+1]
	}
	return out
}

type acct struct {
	name    string
	windows []any
}

func usage(accts ...acct) []any {
	var out []any
	for _, a := range accts {
		out = append(out, map[string]any{"provider": "anthropic", "account": a.name, "state": "measured", "windows": a.windows})
	}
	return out
}

func fake(t *testing.T, payload any, code int) string {
	d := t.TempDir()
	b, _ := json.Marshal(payload)
	os.WriteFile(filepath.Join(d, "usage.json"), b, 0o644)
	os.WriteFile(filepath.Join(d, "matchblox"), []byte(fmt.Sprintf("#!/bin/sh\n[ \"$1\" = usage ] && [ \"$2\" = --json ] || { echo \"bad args: $*\" >&2; exit 9; }\n/bin/cat \"%s/usage.json\"\nexit %d\n", d, code)), 0o755)
	return d
}

func run(t *testing.T, path string) (int, string, string) {
	zone(t, "UTC")
	var out, errb bytes.Buffer
	code := FromMatchblox(now, path, &out, &errb)
	return code, out.String(), errb.String()
}

func TestPositionalArgsKeepTheShape(t *testing.T) {
	zone(t, "UTC")
	if got := Project(19, reset, now); !shape.MatchString(got) {
		t.Fatal(got)
	}
}

func TestProjectMatchesPython(t *testing.T) {
	for z, want := range map[string]string{
		"UTC":                 "elapsed=15.0%  naive=127%  profile_aware=100%  reset_in=5.95d",
		"America/Los_Angeles": "elapsed=15.0%  naive=127%  profile_aware=107%  reset_in=5.95d",
	} {
		zone(t, z)
		if got := Project(19, reset, now); got != want {
			t.Fatalf("%s: got %q", z, got)
		}
	}
}

func TestProjectEdgesMatchPython(t *testing.T) {
	zone(t, "UTC")
	for _, c := range []struct {
		used float64
		now  int64
		want string
	}{
		{0, 1782900000, "elapsed=0.0%  naive=nan%  profile_aware=0%  reset_in=7.00d"},
		{5, 1782899999, "elapsed=-0.0%  naive=nan%  profile_aware=5%  reset_in=7.00d"},
	} {
		if got := Project(c.used, reset, c.now); got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}

func TestMultLocalWeekdays(t *testing.T) {
	zone(t, "UTC")
	sat := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC).Unix()
	for ts, want := range map[int64]float64{sat: 0.5, sat + 86400: 0.1, sat + 2*86400: 1.0, sat - 9*3600: 0.05} {
		if got := Mult(ts); got != want {
			t.Fatalf("%v: %v", time.Unix(ts, 0).UTC(), got)
		}
	}
}

func TestFromMatchbloxShape(t *testing.T) {
	code, out, errs := run(t, fake(t, usage(acct{"me", []any{fiveH, weekly}}), 0))
	if code != 0 || out != Project(19, reset, now)+"\n" {
		t.Fatal(code, out, errs)
	}
}

func TestFromMatchbloxOneLinePerAccount(t *testing.T) {
	code, out, _ := run(t, fake(t, usage(acct{"work", []any{weekly}}, acct{"home", []any{with(weekly, "used_pct", 40)}}), 0))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 2 || !strings.HasSuffix(lines[0], "account=work") || !strings.HasSuffix(lines[1], "account=home") {
		t.Fatal(code, lines)
	}
}

func TestFromMatchbloxNoWeeklyWindowFailsWithCause(t *testing.T) {
	code, out, errs := run(t, fake(t, usage(acct{"me", []any{fiveH}}), 0))
	if code != 1 || out != "" || !strings.Contains(errs, "no weekly window") {
		t.Fatal(code, out, errs)
	}
}

func TestFromMatchbloxErrorIsNotAZero(t *testing.T) {
	code, out, errs := run(t, fake(t, []any{}, 4))
	if code != 1 || out != "" || !strings.Contains(errs, "matchblox usage --json failed") {
		t.Fatal(code, out, errs)
	}
}

func TestFromMatchbloxMissingExit2(t *testing.T) {
	code, out, errs := run(t, t.TempDir())
	if code != 2 || out != "" || strings.TrimSpace(errs) != "matchblox not found: install it with curl -fsSL https://matchblox.sh | sh" {
		t.Fatal(code, out, errs)
	}
}

func TestFromMatchbloxLabelCountsAccountsNotLines(t *testing.T) {
	code, out, errs := run(t, fake(t, usage(acct{"work", []any{weekly}}, acct{"home", []any{fiveH}}), 0))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 1 || !strings.HasSuffix(lines[0], "account=work") || !strings.Contains(errs, "account home: no weekly window") {
		t.Fatal(code, lines, errs)
	}
}

func TestFromMatchbloxMalformedWindowSkipsAccount(t *testing.T) {
	a := []any{with(weekly, "resets_at", nil)}
	b := []any{map[string]any{"window_minutes": 10080, "resets_at": weekly["resets_at"]}}
	code, out, errs := run(t, fake(t, usage(acct{"a", a}, acct{"b", b}, acct{"c", []any{weekly}}), 0))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 1 || !strings.HasSuffix(lines[0], "account=c") {
		t.Fatal(code, lines)
	}
	for _, want := range []string{
		"account a: malformed weekly window (used_pct=19, resets_at=None)",
		"account b: malformed weekly window (used_pct=None, resets_at='2026-07-08T10:00:00+00:00')",
	} {
		if !strings.Contains(errs, want) {
			t.Fatalf("missing %q in %q", want, errs)
		}
	}
}

func TestFromMatchbloxNonArrayFailsWithCause(t *testing.T) {
	for _, p := range []any{map[string]any{"error": "not ready"}, []any{42}, "x"} {
		if code, out, errs := run(t, fake(t, p, 0)); code != 1 || out != "" || errs == "" {
			t.Fatal(p, code, out)
		}
	}
}

func TestFromMatchbloxUnmeasuredStateIsSkippedWithCause(t *testing.T) {
	p := usage(acct{"old", []any{weekly}}, acct{"now", []any{weekly}})
	p[0].(map[string]any)["state"] = "stale"
	code, out, errs := run(t, fake(t, p, 0))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 1 || !strings.HasSuffix(lines[0], "account=now") || !strings.Contains(errs, "account old: state is stale, not measured") {
		t.Fatal(code, lines, errs)
	}
}

func TestResetsAtFormsPythonAccepts(t *testing.T) {
	zone(t, "UTC")
	for _, s := range []string{"2026-07-08T10:00:00Z", "2026-07-08T10:00:00+00:00", "2026-07-08T10:00:00.5+00:00", "2026-07-08 10:00:00+00:00", "2026-07-08T10:00:00"} {
		if got, ok := parseISO(s); !ok || got != reset {
			t.Fatalf("%s: %d %v", s, got, ok)
		}
	}
}
