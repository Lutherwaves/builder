// Package reach holds the send rules for messages an agent writes to people: check before a send, log after it.
package reach

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	Recipients = []string{"self", "teammate", "channel"}
	Verdicts   = []string{"send", "draft", "deny", "skip"}
)

const (
	DedupeS = 3600
	PerHour = 6
	NoRoute = 3 // exit code: this route does not work here; use the next route
)

// norm: case and spacing do not make a new message. The fold is Python's str.casefold, rune by rune,
// so a message logged by the Python dedupes against the same message checked by Go.
func norm(text string) string {
	var b strings.Builder
	for _, r := range strings.Join(strings.Fields(text), " ") {
		if f, ok := casefold[r]; ok {
			b.WriteString(f)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func who(to, recipient, channelName string) string {
	if to == "self" {
		return "self"
	}
	if to == "channel" && channelName != "" {
		return norm(channelName)
	}
	return norm(recipient)
}

func Digest(to, text, recipient string) string {
	sum := sha256.Sum256([]byte(to + "|" + who(to, recipient, "") + "|" + norm(text)))
	return hex.EncodeToString(sum[:])
}

func number(v any, def float64) float64 {
	if f, ok := v.(float64); ok && f >= 0 {
		return f
	}
	return def
}

func names(box map[string]any, key string) []string {
	list, _ := box[key].([]any)
	var out []string
	for _, v := range list {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func Decide(rules map[string]any, to, kind, channelName string) string {
	box, _ := rules["to_"+to].(map[string]any)
	allowed := slices.Contains(names(box, "allow"), kind)
	if to == "channel" {
		if allowed && strings.TrimSpace(channelName) != "" && slices.Contains(names(box, "named"), channelName) {
			return "send"
		}
		return "deny"
	}
	if allowed {
		return "send"
	}
	return "draft"
}

func Check(rules map[string]any, log []map[string]any, to, kind, text string, now float64, channelName, recipient string) map[string]string {
	if to == "channel" && channelName == "" {
		channelName = recipient
	}
	verdict := Decide(rules, to, kind, channelName)
	if verdict == "deny" {
		return map[string]string{"verdict": "deny", "reason": "channel-not-named"}
	}
	key := recipient
	if to == "channel" {
		key = channelName
	}
	h := Digest(to, text, key)
	dedupe, perHour := number(rules["dedupe_s"], DedupeS), number(rules["per_hour"], PerHour)
	recent := 0.0
	for _, e := range log {
		at, ok := e["at"].(float64)
		if !ok || e["verdict"] != "send" {
			continue
		}
		if e["sha256"] == h && now-at <= dedupe {
			return map[string]string{"verdict": "skip", "reason": "duplicate"}
		}
		if now-at < 3600 {
			recent++
		}
	}
	if recent >= perHour {
		return map[string]string{"verdict": "skip", "reason": "rate"}
	}
	reason := "needs-yes"
	if verdict == "send" {
		reason = "allowed"
	}
	return map[string]string{"verdict": verdict, "reason": reason}
}

// Record appends one JSON line in one write, so concurrent appends do not interleave.
func Record(path string, entry map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func ReadLog(path string) []map[string]any {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []map[string]any
	for _, line := range strings.Split(string(b), "\n") {
		var e map[string]any
		if json.Unmarshal([]byte(line), &e) == nil && e != nil {
			out = append(out, e)
		}
	}
	return out
}

func readObject(path string) map[string]any {
	var m map[string]any
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &m) != nil || m == nil {
		return map[string]any{}
	}
	return m
}

// LoadRules is the plugin's default rules, with each key of the person's reach.json on top.
func LoadRules(root, path string) (map[string]any, error) {
	var rules map[string]any
	b, err := os.ReadFile(filepath.Join(root, "skills", "reach", "rules.default.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &rules); err != nil {
		return nil, err
	}
	for k, v := range readObject(path) {
		rules[k] = v
	}
	return rules, nil
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

func runQuiet(timeout time.Duration, tool string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, tool, args...).CombinedOutput()
}

// Notify shows an OS notification and returns the tool name, or "" when no tool works.
func Notify(title, text, goos, path string) string {
	name, args := "notify-send", []string{title, text}
	if goos == "darwin" {
		// The text goes in as an argument, never into the script, so quotes cannot break it.
		name = "osascript"
		args = []string{"-e", "on run argv", "-e", "display notification (item 2 of argv) with title (item 1 of argv)", "-e", "end run", title, text}
	}
	tool := lookPath(name, path)
	if tool == "" {
		return ""
	}
	if _, err := runQuiet(5*time.Second, tool, args...); err != nil {
		return ""
	}
	return name
}

var notifyLine = regexp.MustCompile(`(?m)^\s*(matchblox\s+)?notify(\s|$)`)

// matchbloxNotifyTool is matchblox's path only when `matchblox --help` lists a notify command.
func matchbloxNotifyTool(path string) string {
	tool := lookPath("matchblox", path)
	if tool == "" {
		return ""
	}
	out, _ := runQuiet(3*time.Second, tool, "--help")
	if !notifyLine.Match(out) {
		return ""
	}
	return tool
}

func MatchbloxNotify(text, path string) bool {
	tool := matchbloxNotifyTool(path)
	if tool == "" {
		return false
	}
	_, err := runQuiet(10*time.Second, tool, "notify", text)
	return err == nil
}
