// Package budget projects the weekly usage limit, discounting the hours a person predictably does not burn:
// a sleep window (02:00 to 09:00, ×0.05) and the weekend (Saturday ×0.5, Sunday ×0.1).
package budget

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/internal/cli"
)

const (
	window        = 604800 // 7 days in seconds
	weeklyMinutes = 10080
	hour          = 3600
	Install       = "matchblox not found: install it with curl -fsSL https://matchblox.sh | sh"
)

// Mult is the burn weight of the hour that holds ts, in local time.
func Mult(ts int64) float64 {
	t := time.Unix(ts, 0).In(time.Local)
	if h := t.Hour(); h >= 2 && h < 9 {
		return 0.05
	}
	switch t.Weekday() {
	case time.Saturday:
		return 0.5
	case time.Sunday:
		return 0.1
	}
	return 1.0
}

// EffHours is the weekend-weighted hours between epochs a and b.
func EffHours(a, b int64) float64 {
	s := 0.0
	for t := a; t < b; t += hour {
		s += Mult(t) * float64(min(hour, b-t)) / hour
	}
	return s
}

func pct0(x float64) string {
	if math.IsNaN(x) {
		return "nan"
	}
	return fmt.Sprintf("%.0f", x)
}

func Project(used float64, reset, now int64) string {
	start := reset - window
	elFrac := float64(now-start) / window
	naive := math.NaN()
	if elFrac > 0 {
		naive = used / elFrac
	}
	effEl, effRem := EffHours(start, now), EffHours(now, reset)
	rate := 0.0 // % per normal hour
	if effEl > 0 {
		rate = used / effEl
	}
	return fmt.Sprintf("elapsed=%.1f%%  naive=%s%%  profile_aware=%s%%  reset_in=%.2fd",
		elFrac*100, pct0(naive), pct0(used+rate*effRem), float64(reset-now)/86400)
}

// parseISO is datetime.fromisoformat(s.replace("Z", "+00:00")).timestamp(), truncated to whole seconds.
func parseISO(s string) (int64, bool) {
	s = strings.ReplaceAll(s, "Z", "+00:00")
	for _, layout := range []string{"2006-01-02T15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02T15:04-07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix(), true
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// pyFloat is Python's float() of a JSON value.
func pyFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

// pyStr is Python's str() of a JSON value.
func pyStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return cli.Repr(v)
}

func accountLine(acc map[string]any, name string, now int64, stderr io.Writer) string {
	skip := func(cause string) string {
		fmt.Fprintf(stderr, "account %s: %s\n", name, cause)
		return ""
	}
	state, has := acc["state"]
	if !has {
		state = "measured"
	}
	if state != "measured" {
		return skip(fmt.Sprintf("state is %s, not measured; no projection", pyStr(state)))
	}
	windows, _ := acc["windows"].([]any)
	var w map[string]any
	for _, raw := range windows {
		if m, ok := raw.(map[string]any); ok && m["window_minutes"] == float64(weeklyMinutes) {
			w = m
			break
		}
	}
	if w == nil {
		return skip("no weekly window in matchblox usage")
	}
	used, okUsed := pyFloat(w["used_pct"])
	reset, okReset := int64(0), false
	if r, has := w["resets_at"]; has {
		reset, okReset = parseISO(pyStr(r))
	}
	if !okUsed || !okReset {
		return skip(fmt.Sprintf("malformed weekly window (used_pct=%s, resets_at=%s)", cli.Repr(w["used_pct"]), cli.Repr(w["resets_at"])))
	}
	return Project(used, reset, now)
}

func lookPath(name, path string) string {
	if path == "" {
		path = os.Getenv("PATH")
	}
	for _, dir := range filepath.SplitList(path) {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

// FromMatchblox prints one projection line per account in `matchblox usage --json` and returns the exit code.
func FromMatchblox(now int64, path string, stdout, stderr io.Writer) int {
	die := func(msg string, code int) int {
		fmt.Fprintln(stderr, msg)
		return code
	}
	exe := lookPath("matchblox", path)
	if exe == "" {
		return die(Install, 2)
	}
	var errBuf strings.Builder
	cmd := exec.Command(exe, "usage", "--json")
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		code := -1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return die(fmt.Sprintf("matchblox usage --json failed (exit %d): %s", code, strings.TrimSpace(errBuf.String())), 1)
	}
	var parsed any
	if json.Unmarshal(out, &parsed) != nil {
		return die("matchblox usage --json printed no JSON. Run it by hand to see why.", 1)
	}
	list, ok := parsed.([]any)
	for _, a := range list {
		if _, isMap := a.(map[string]any); !isMap {
			ok = false
		}
	}
	if !ok {
		return die("matchblox usage --json is not a list of accounts. Run it by hand to see why.", 1)
	}
	type line struct{ name, text string }
	var lines []line
	for _, a := range list {
		acc := a.(map[string]any)
		name := "?"
		for _, k := range []string{"account", "provider"} {
			if v, has := acc[k]; has && v != nil && v != "" && v != false {
				name = pyStr(v)
				break
			}
		}
		if text := accountLine(acc, name, now, stderr); text != "" {
			lines = append(lines, line{name, text})
		}
	}
	if lines == nil {
		return die("no measured weekly window in matchblox usage: run a Claude Code session so matchblox reads the plan, then try again.", 1)
	}
	for _, l := range lines {
		// Label by the number of accounts, not lines: a skipped account must not hide the label.
		if len(list) > 1 {
			fmt.Fprintf(stdout, "%s  account=%s\n", l.text, l.name)
		} else {
			fmt.Fprintln(stdout, l.text)
		}
	}
	return 0
}
