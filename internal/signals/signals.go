// Package signals reads cheap, offline machine signals. Every external call shares one time budget.
package signals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/Lutherwaves/builder/internal/harness"
)

const WeekMin = 10080

var (
	CanonMarkers = []string{"RULES.md", "DESIGN.md", "docs/rules"}    // the homes the canon skill finds or creates
	GraphServers = []string{"codebase-memory-mcp", "codebase-memory"} // user-wide name; the name harness writes to .mcp.json
)

const autocompactVar = "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"

var (
	tomlMCP         = regexp.MustCompile(`(?m)^\[mcp_servers\.("?)([^\]."]+)("?)\]`)
	tomlSuperpowers = regexp.MustCompile(`(?m)^\[plugins\."superpowers@`)
	parseTOML       = true // tests turn it off to exercise the regex fallback
)

func jsonOf(r *Result) any {
	if r == nil || r.Code != 0 {
		return nil
	}
	var v any
	if json.Unmarshal([]byte(r.Out), &v) != nil {
		return nil
	}
	return v
}

func stdout(r *Result) string {
	if r == nil || r.Code != 0 {
		return ""
	}
	return strings.TrimSpace(r.Out)
}

func weeklyPct(usage any) any {
	accts, ok := usage.([]any)
	if !ok {
		return nil
	}
	var best any
	for _, a := range accts {
		acct, _ := a.(map[string]any)
		windows, _ := acct["windows"].([]any)
		for _, w := range windows {
			win, _ := w.(map[string]any)
			pct, isNum := win["used_pct"].(float64)
			if isNum && win["window_minutes"] == float64(WeekMin) && (best == nil || pct > best.(float64)) {
				best = pct
			}
		}
	}
	return best
}

func resolve(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		p = a
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func gitRoot(cwd string, r *Result) string {
	if s := stdout(r); s != "" {
		return s
	}
	p := resolve(cwd)
	for d := p; ; d = filepath.Dir(d) {
		if exists(filepath.Join(d, ".git")) {
			return d
		}
		if filepath.Dir(d) == d {
			return p
		}
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// readText is Path.read_text(): a missing file or one that is not UTF-8 reads as nothing.
func readText(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(b) {
		return "", false
	}
	return string(b), true
}

func readJSON(path string) map[string]any {
	text, ok := readText(path)
	var m map[string]any
	if !ok || json.Unmarshal([]byte(text), &m) != nil {
		return map[string]any{}
	}
	return m
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
}

func dict(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// hookCommands reads Claude Code's {"hooks": {event: [{"hooks": [{"command": ...}]}]}}.
func hookCommands(settings map[string]any) []string {
	var out []string
	for _, groups := range dict(settings["hooks"]) {
		for _, g := range list(groups) {
			for _, h := range list(dict(g)["hooks"]) {
				if c, ok := dict(h)["command"].(string); ok {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// flatHookCommands reads Cursor's {"hooks": {event: [{"command": ...}]}}.
func flatHookCommands(settings map[string]any) []string {
	var out []string
	for _, entries := range dict(settings["hooks"]) {
		for _, h := range list(entries) {
			if c, ok := dict(h)["command"].(string); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

// readTOML returns the parsed table, or nil with the raw text when the text does not parse.
func readTOML(path string) (map[string]any, string) {
	text, ok := readText(path)
	if !ok {
		return map[string]any{}, ""
	}
	if parseTOML {
		var m map[string]any
		if _, err := toml.Decode(text, &m); err == nil {
			return m, text
		}
	}
	return nil, text
}

func serverNames(path, key string) map[string]bool {
	names := map[string]bool{}
	if filepath.Ext(path) != ".toml" {
		for k := range dict(readJSON(path)[key]) {
			names[k] = true
		}
		return names
	}
	data, text := readTOML(path)
	if data == nil {
		for _, m := range tomlMCP.FindAllStringSubmatch(text, -1) {
			if m[1] == m[3] {
				names[m[2]] = true
			}
		}
		return names
	}
	for k := range dict(data[key]) {
		names[k] = true
	}
	return names
}

func mcpServers(spec harness.Paths, name string, dirs []string) map[string]bool {
	names := map[string]bool{}
	add := func(m map[string]bool) {
		for k := range m {
			names[k] = true
		}
	}
	for _, f := range spec.MCP {
		add(serverNames(f.Path, f.Key))
		if name == "claude" {
			projects := dict(readJSON(f.Path)["projects"])
			for _, d := range dirs {
				for k := range dict(dict(projects[d])[f.Key]) {
					names[k] = true
				}
			}
		}
	}
	for _, d := range dirs {
		for _, f := range spec.ProjectMCP {
			add(serverNames(filepath.Join(d, f.Path), f.Key))
		}
	}
	return names
}

func superpowers(name string, spec harness.Paths) any {
	if spec.PluginsRecord == "" {
		return nil
	}
	switch name {
	case "codex":
		data, text := readTOML(spec.PluginsRecord)
		if data == nil {
			return tomlSuperpowers.MatchString(text)
		}
		for k := range dict(data["plugins"]) {
			if strings.HasPrefix(k, "superpowers@") {
				return true
			}
		}
		return false
	case "opencode":
		for _, e := range list(readJSON(spec.PluginsRecord)["plugin"]) {
			if s, ok := e.(string); ok && strings.Contains(s, "superpowers") {
				return true
			}
		}
		return false
	}
	b, err := os.ReadFile(spec.PluginsRecord)
	return err == nil && strings.Contains(string(b), "superpowers@")
}

func gstack(name string, spec harness.Paths) bool {
	skills := filepath.Join(spec.ConfigDir, "skills")
	if name == "claude" {
		return exists(filepath.Join(skills, "gstack"))
	}
	m, _ := filepath.Glob(filepath.Join(skills, "gstack-*"))
	return len(m) > 0
}

func matchbloxHooks(name string, spec harness.Paths) any {
	switch name {
	case "claude":
		return slices.ContainsFunc(hookCommands(readJSON(spec.Settings)), func(c string) bool { return strings.Contains(c, "matchblox hook") })
	case "cursor":
		return slices.ContainsFunc(flatHookCommands(readJSON(spec.Settings)), func(c string) bool { return strings.Contains(c, "matchblox hook --cursor") })
	}
	return nil
}

func autocompact(name string, spec harness.Paths, env map[string]string) any {
	if name != "claude" {
		return nil
	}
	_, inSettings := dict(readJSON(spec.Settings)["env"])[autocompactVar]
	_, inEnv := env[autocompactVar]
	return inSettings || inEnv
}

// countProcesses counts `ps -axo pid=,comm=,args=` rows that match a harness, minus rows whose args hold its exclude.
func countProcesses(r *Result) float64 {
	if r == nil || r.Code != 0 {
		return 0
	}
	matches := harness.AllProcessMatch()
	patterns := make([]*regexp.Regexp, len(matches))
	for n, m := range matches {
		if m.Mode != 'x' {
			patterns[n] = regexp.MustCompile(m.Pattern)
		}
	}
	total := 0.0
	for _, line := range strings.Split(r.Out, "\n") {
		parts := split3(line)
		if len(parts) < 2 {
			continue
		}
		comm := filepath.Base(parts[1])
		args := ""
		if len(parts) > 2 {
			args = parts[2]
		}
		for n, m := range matches {
			hit := comm == m.Pattern
			if patterns[n] != nil {
				hit = patterns[n].MatchString(args)
			}
			if hit && !(m.Exclude != "" && strings.Contains(args, m.Exclude)) {
				total++
				break
			}
		}
	}
	return total
}

// split3 is Python's str.split(None, 2): two fields, then the rest without its leading space.
func split3(line string) []string {
	var out []string
	rest := line
	for len(out) < 2 {
		rest = strings.TrimLeft(rest, " \t\r\v\f")
		if rest == "" {
			return out
		}
		end := strings.IndexAny(rest, " \t\r\v\f")
		if end < 0 {
			return append(out, rest)
		}
		out, rest = append(out, rest[:end]), rest[end:]
	}
	if rest = strings.TrimLeft(rest, " \t\r\v\f"); rest != "" {
		out = append(out, rest)
	}
	return out
}

// Collect reads the twelve signals the catalog's `when` conditions test.
func Collect(cwd string, timeout time.Duration, name string, env map[string]string, home string) map[string]any {
	if name == "" {
		name = harness.Detect(env, "")
	}
	if !slices.Contains(harness.Names, name) {
		name = "claude" // an unknown harness keeps the Claude Code view
	}
	spec, _ := harness.For(name, env, home)
	has := map[string]bool{}
	for _, n := range []string{"matchblox", "gh", "ps", "git"} {
		has[n] = which(n, env) != ""
	}
	cmds := map[string][]string{}
	if has["matchblox"] {
		cmds["sessions"] = []string{"matchblox", "sessions", "--json"}
		cmds["usage"] = []string{"matchblox", "usage", "--json"}
	}
	if has["gh"] {
		cmds["gh"] = []string{"gh", "auth", "token"}
	}
	if has["ps"] {
		cmds["ps"] = []string{"ps", "-axo", "pid=,comm=,args="}
	}
	if has["git"] {
		cmds["git"] = []string{"git", "rev-parse", "--show-toplevel"}
		cmds["gpgsign"] = []string{"git", "config", "--get", "commit.gpgsign"}
		cmds["signkey"] = []string{"git", "config", "--get", "user.signingkey"}
	}
	res := RunAll(cmds, cwd, timeout, env)

	root := gitRoot(cwd, res["git"])
	dirs := []string{root}
	if c := resolve(cwd); c != root {
		dirs = append(dirs, c)
	}
	servers := mcpServers(spec, name, dirs)

	var sessions any
	if l, ok := jsonOf(res["sessions"]).([]any); ok {
		sessions = float64(len(l))
	}
	var hooks any = false
	if has["matchblox"] {
		hooks = matchbloxHooks(name, spec)
	}
	return map[string]any{
		"superpowers":     superpowers(name, spec),
		"gstack":          gstack(name, spec),
		"gh":              res["gh"] != nil && res["gh"].Code == 0,
		"signing":         stdout(res["gpgsign"]) == "true" && stdout(res["signkey"]) != "",
		"matchblox":       has["matchblox"],
		"matchblox_hooks": hooks,
		"code_graph":      slices.ContainsFunc(GraphServers, func(s string) bool { return servers[s] }),
		"autocompact":     autocompact(name, spec, env),
		"sessions":        sessions,
		"weekly_pct":      weeklyPct(jsonOf(res["usage"])),
		"sessions_proc":   countProcesses(res["ps"]),
		"has_canon":       slices.ContainsFunc(CanonMarkers, func(m string) bool { return exists(filepath.Join(root, m)) }),
	}
}
