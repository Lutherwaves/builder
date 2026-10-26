package signals

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const generous = 10 * time.Second

type fx struct {
	t                    *testing.T
	root, home, bin, cwd string
	env                  map[string]string
}

func setup(t *testing.T) *fx {
	root := t.TempDir()
	f := &fx{t: t, root: root, home: filepath.Join(root, "home"), bin: filepath.Join(root, "bin"), cwd: filepath.Join(root, "proj")}
	for _, d := range []string{f.home, f.bin, f.cwd, filepath.Join(root, "bhome")} {
		os.MkdirAll(d, 0o755)
	}
	f.env = map[string]string{"HOME": f.home, "BUILDER_HOME": filepath.Join(root, "bhome"), "PATH": f.bin}
	return f
}

func (f *fx) fake(name, body string) {
	os.WriteFile(filepath.Join(f.bin, name), []byte("#!/bin/bash\n"+body+"\n"), 0o755)
}

func (f *fx) fakePS(rows ...string) {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r + "\n")
	}
	os.WriteFile(filepath.Join(f.bin, "ps.txt"), []byte(b.String()), 0o644)
	f.fake("ps", "/bin/cat "+filepath.Join(f.bin, "ps.txt"))
}

func (f *fx) warm(names ...string) {
	for _, n := range names {
		exec.Command(filepath.Join(f.bin, n)).Run()
	}
}

func (f *fx) collectIn(cwd string, timeout time.Duration, harness string, extra ...string) map[string]any {
	env := map[string]string{}
	for k, v := range f.env {
		env[k] = v
	}
	for n := 0; n+1 < len(extra); n += 2 {
		env[extra[n]] = extra[n+1]
	}
	return Collect(cwd, timeout, harness, env, f.home)
}

func (f *fx) collect(timeout time.Duration) map[string]any { return f.collectIn(f.cwd, timeout, "") }
func (f *fx) sig(h string, extra ...string) map[string]any {
	return f.collectIn(f.cwd, generous, h, extra...)
}

func (f *fx) write(path string, data any) {
	os.MkdirAll(filepath.Dir(path), 0o755)
	s, ok := data.(string)
	if !ok {
		b, _ := json.Marshal(data)
		s = string(b)
	}
	os.WriteFile(path, []byte(s), 0o644)
}

func want(t *testing.T, got map[string]any, key string, w any) {
	t.Helper()
	if got[key] != w {
		t.Fatalf("%s = %#v, want %#v", key, got[key], w)
	}
}

func bounded(t *testing.T, limit time.Duration, fn func()) {
	t.Helper()
	start := time.Now()
	fn()
	if d := time.Since(start); d >= limit {
		t.Fatalf("took %v, limit %v", d, limit)
	}
}

func TestMatchbloxMissingAndSlow(t *testing.T) {
	f := setup(t)
	want(t, f.collect(generous), "matchblox", false)
	want(t, f.collect(generous), "sessions", nil)
	f.fake("matchblox", "/bin/sleep 2")
	var out map[string]any
	bounded(t, time.Second, func() { out = f.collect(500 * time.Millisecond) })
	want(t, out, "matchblox", true)
	want(t, out, "sessions", nil)
	want(t, out, "weekly_pct", nil)
}

func TestWeeklyPctFromUsage(t *testing.T) {
	f := setup(t)
	f.fake("matchblox", `
if [ "$1" = usage ]; then
  echo '[{"windows":[{"used_pct":30,"window_minutes":300},{"used_pct":82,"window_minutes":10080}]},{"windows":[{"used_pct":40,"window_minutes":10080}]}]'
elif [ "$1" = sessions ]; then
  echo '[{"id":1},{"id":2},{"id":3}]'
fi`)
	out := f.collect(generous)
	want(t, out, "weekly_pct", 82.0)
	want(t, out, "sessions", 3.0)
}

func TestMatchbloxGarbageOutput(t *testing.T) {
	f := setup(t)
	f.fake("matchblox", "echo not json")
	out := f.collect(generous)
	want(t, out, "sessions", nil)
	want(t, out, "weekly_pct", nil)
}

func TestPluginAndSkillFiles(t *testing.T) {
	f := setup(t)
	out := f.collect(generous)
	want(t, out, "superpowers", false)
	want(t, out, "gstack", false)
	f.write(filepath.Join(f.home, ".claude/plugins/installed_plugins.json"), `{"plugins":{"superpowers@x":[]}}`)
	os.MkdirAll(filepath.Join(f.home, ".claude/skills/gstack"), 0o755)
	out = f.collect(generous)
	want(t, out, "superpowers", true)
	want(t, out, "gstack", true)
}

func TestGhAndPgrep(t *testing.T) {
	f := setup(t)
	want(t, f.collect(generous), "gh", false)
	want(t, f.collect(generous), "sessions_proc", 0.0)
	f.fake("gh", "exit 0")
	f.fakePS("11 claude claude", "22 claude claude --x", "33 vim vim claude")
	out := f.collect(generous)
	want(t, out, "gh", true)
	want(t, out, "sessions_proc", 2.0)
}

func TestHangingGhIsBounded(t *testing.T) {
	f := setup(t)
	f.fake("gh", "/bin/sleep 2")
	var out map[string]any
	bounded(t, time.Second, func() { out = f.collect(500 * time.Millisecond) })
	want(t, out, "gh", false)
}

func TestFinishedToolWhoseChildHoldsStdoutIsBounded(t *testing.T) {
	f := setup(t)
	f.fake("gh", "/bin/sleep 5 &\nexit 0")
	var out map[string]any
	bounded(t, time.Second, func() { out = f.collect(500 * time.Millisecond) })
	want(t, out, "gh", true)
}

func TestNonUTF8ProcessOutputDoesNotBreakCollect(t *testing.T) {
	f := setup(t)
	f.fake("ps", `printf '1 claude claude\n2 odd \377bad\n3 claude claude\n'`)
	want(t, f.collect(generous), "sessions_proc", 2.0)
	r := RunAll(map[string][]string{"a": {"/usr/bin/printf", `\377abc`}}, f.cwd, generous, f.env)
	if r["a"] == nil || r["a"].Code != 0 {
		t.Fatalf("%#v", r["a"])
	}
}

func TestFinishedCallsAreReadAfterAHungOne(t *testing.T) {
	f := setup(t)
	f.fake("matchblox", "/bin/sleep 3")
	f.fake("gh", "exit 0")
	f.fakePS("1 claude claude", "2 claude claude", "3 claude claude")
	f.warm("gh", "ps")
	var out map[string]any
	bounded(t, 1600*time.Millisecond, func() { out = f.collect(time.Second) })
	want(t, out, "gh", true)
	want(t, out, "sessions_proc", 3.0)
	want(t, out, "sessions", nil)
}

func (f *fx) fakeGitConfig(values map[string]string) {
	body := "if [ \"$1\" = config ] && [ \"$2\" = --get ]; then\n"
	for k, v := range values {
		body += "  [ \"$3\" = " + k + " ] && { echo " + v + "; exit 0; }\n"
	}
	f.fake("git", body+"  exit 1\nfi\nexit 1")
}

func TestSigningNeedsGpgsignTrueAndAKey(t *testing.T) {
	f := setup(t)
	want(t, f.collect(generous), "signing", false)
	for _, c := range []struct {
		v map[string]string
		w bool
	}{
		{map[string]string{"commit.gpgsign": "true", "user.signingkey": "/k/id.pub"}, true},
		{map[string]string{"commit.gpgsign": "true"}, false},
		{map[string]string{"commit.gpgsign": "false", "user.signingkey": "/k/id.pub"}, false},
		{map[string]string{"user.signingkey": "/k/id.pub"}, false},
	} {
		f.fakeGitConfig(c.v)
		want(t, f.collect(generous), "signing", c.w)
	}
}

func TestHangingGitConfigIsBounded(t *testing.T) {
	f := setup(t)
	f.fake("git", "/bin/sleep 2")
	var out map[string]any
	bounded(t, time.Second, func() { out = f.collect(500 * time.Millisecond) })
	want(t, out, "signing", false)
}

func TestHasCanon(t *testing.T) {
	f := setup(t)
	want(t, f.collect(generous), "has_canon", false)
	f.write(filepath.Join(f.cwd, "DESIGN.md"), "x")
	want(t, f.collect(generous), "has_canon", true)
}

func TestHasCanonFromRulesMdThatCanonCreates(t *testing.T) {
	f := setup(t)
	f.write(filepath.Join(f.cwd, "RULES.md"), "x")
	want(t, f.collect(generous), "has_canon", true)
}

func TestCanonMarkersMatchTheCanonSkillAndRecommendation(t *testing.T) {
	skill, _ := os.ReadFile("../../skills/canon/SKILL.md")
	for _, m := range CanonMarkers {
		if !strings.Contains(string(skill), "`"+m) {
			t.Fatal(m)
		}
	}
	if strings.Join(CanonMarkers, ",") != "RULES.md,DESIGN.md,docs/rules" {
		t.Fatal(CanonMarkers)
	}
	var cat []map[string]any
	b, _ := os.ReadFile("../../lib/recommendations.json")
	json.Unmarshal(b, &cat)
	for _, i := range cat {
		if i["id"] == "canon" {
			why := i["why"].(string)
			if !strings.Contains(why, "RULES.md") || strings.Contains(why, "docs/canon") || strings.Contains(why, ".claude/rules") {
				t.Fatal(why)
			}
			return
		}
	}
	t.Fatal("no canon item")
}

func TestHasCanonAtGitRootFromSubdir(t *testing.T) {
	f := setup(t)
	f.fake("git", `echo "`+f.cwd+`"`)
	os.MkdirAll(filepath.Join(f.cwd, "docs/rules"), 0o755)
	sub := filepath.Join(f.cwd, "a/b")
	os.MkdirAll(sub, 0o755)
	want(t, f.collectIn(sub, generous, ""), "has_canon", true)
}

func TestMatchbloxHooksNeedTheBinaryAndAHookCommand(t *testing.T) {
	f := setup(t)
	s := filepath.Join(f.home, ".claude/settings.json")
	f.write(s, map[string]any{"hooks": map[string]any{"Stop": []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/opt/bin/matchblox hook Stop"}}}}}})
	want(t, f.collect(generous), "matchblox_hooks", false)
	f.fake("matchblox", "exit 0")
	want(t, f.collect(generous), "matchblox_hooks", true)
	f.write(s, map[string]any{"hooks": map[string]any{"Stop": []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "echo matchblox"}}}}}})
	want(t, f.collect(generous), "matchblox_hooks", false)
	f.write(s, "{bad")
	want(t, f.collect(generous), "matchblox_hooks", false)
}

func TestSettingsComeFromClaudeConfigDirWhenSet(t *testing.T) {
	f := setup(t)
	f.fake("matchblox", "exit 0")
	cfg := filepath.Join(f.root, "cfg")
	f.write(filepath.Join(cfg, "settings.json"), map[string]any{
		"hooks": map[string]any{"Stop": []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "matchblox hook Stop"}}}}},
		"env":   map[string]any{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"},
	})
	f.write(filepath.Join(cfg, ".claude.json"), `{"mcpServers": {"codebase-memory-mcp": {"command": "x"}}}`)
	out := f.collect(generous)
	want(t, out, "matchblox_hooks", false)
	want(t, out, "autocompact", false)
	want(t, out, "code_graph", false)
	out = f.collectIn(f.cwd, generous, "", "CLAUDE_CONFIG_DIR", cfg)
	want(t, out, "matchblox_hooks", true)
	want(t, out, "autocompact", true)
	want(t, out, "code_graph", true)
}

func TestCodeGraphFromMcpServers(t *testing.T) {
	f := setup(t)
	cj := filepath.Join(f.home, ".claude.json")
	want(t, f.collect(generous), "code_graph", false)
	for _, c := range []struct {
		s string
		w bool
	}{
		{`{"mcpServers": {"other": {}}}`, false},
		{`{"mcpServers": {"codebase-memory-mcp": {"command": "x"}}}`, true},
		{`{"mcpServers": ["codebase-memory-mcp"]}`, false},
	} {
		f.write(cj, c.s)
		want(t, f.collect(generous), "code_graph", c.w)
	}
}

func TestCodeGraphFromTheRepositoryMcpJsonThatHarnessWrites(t *testing.T) {
	f := setup(t)
	f.fake("git", `echo "`+f.cwd+`"`)
	sub := filepath.Join(f.cwd, "a")
	os.MkdirAll(sub, 0o755)
	f.write(filepath.Join(f.cwd, ".mcp.json"), `{"mcpServers": {"codebase-memory": {"command": "x"}}}`)
	want(t, f.collectIn(sub, generous, ""), "code_graph", true)
	f.write(filepath.Join(f.cwd, ".mcp.json"), `{"mcpServers": {"other": {}}}`)
	want(t, f.collectIn(sub, generous, ""), "code_graph", false)
}

func TestCodeGraphFromTheLocalScopeInClaudeJson(t *testing.T) {
	f := setup(t)
	f.fake("git", `echo "`+f.cwd+`"`)
	sub := filepath.Join(f.cwd, "a")
	os.MkdirAll(sub, 0o755)
	cj := filepath.Join(f.home, ".claude.json")
	f.write(cj, map[string]any{"projects": map[string]any{f.cwd: map[string]any{"mcpServers": map[string]any{"codebase-memory-mcp": map[string]any{"command": "x"}}}}})
	want(t, f.collectIn(sub, generous, ""), "code_graph", true)
	real, _ := filepath.EvalSymlinks(sub)
	f.write(cj, map[string]any{"projects": map[string]any{real: map[string]any{"mcpServers": map[string]any{"codebase-memory": map[string]any{}}}}})
	want(t, f.collectIn(sub, generous, ""), "code_graph", true)
	f.write(cj, map[string]any{"projects": map[string]any{"/elsewhere": map[string]any{"mcpServers": map[string]any{"codebase-memory-mcp": map[string]any{}}}, f.cwd: "bad"}})
	want(t, f.collectIn(sub, generous, ""), "code_graph", false)
}

func TestPluginsAndGstackComeFromClaudeConfigDirWhenSet(t *testing.T) {
	f := setup(t)
	cfg := filepath.Join(f.root, "cfg")
	f.write(filepath.Join(cfg, "plugins/installed_plugins.json"), `{"plugins":{"superpowers@x":[]}}`)
	os.MkdirAll(filepath.Join(cfg, "skills/gstack"), 0o755)
	want(t, f.collect(generous), "superpowers", false)
	out := f.collectIn(f.cwd, generous, "", "CLAUDE_CONFIG_DIR", cfg)
	want(t, out, "superpowers", true)
	want(t, out, "gstack", true)
	os.MkdirAll(filepath.Join(f.home, ".claude/skills/gstack"), 0o755)
	want(t, f.collectIn(f.cwd, generous, "", "CLAUDE_CONFIG_DIR", filepath.Join(f.root, "empty")), "gstack", false)
}

func TestAutocompactFromSettingsEnvOrProcessEnv(t *testing.T) {
	f := setup(t)
	s := filepath.Join(f.home, ".claude/settings.json")
	want(t, f.collect(generous), "autocompact", false)
	f.write(s, map[string]any{"env": map[string]any{"OTHER": "1"}})
	want(t, f.collect(generous), "autocompact", false)
	f.write(s, map[string]any{"env": map[string]any{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"}})
	want(t, f.collect(generous), "autocompact", true)
	f.write(s, map[string]any{})
	want(t, f.collectIn(f.cwd, generous, "", "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE", "85"), "autocompact", true)
}

func TestHarnessNoneUsesDetect(t *testing.T) {
	f := setup(t)
	f.write(filepath.Join(f.home, ".codex/config.toml"), "[plugins.\"superpowers@x\"]\nenabled = true\n")
	want(t, f.collectIn(f.cwd, generous, "", "BUILDER_HARNESS", "codex"), "superpowers", true)
}

func TestNonUTF8ConfigTomlDoesNotCrash(t *testing.T) {
	f := setup(t)
	f.write(filepath.Join(f.home, ".codex/config.toml"), "[plugins.\"superpowers@x\"]\nname = \"\xff\xfe\"\n[mcp_servers.codebase-memory]\n")
	out := f.sig("codex")
	want(t, out, "superpowers", false)
	want(t, out, "code_graph", false)
}

func TestCodexReadsMcpFromConfigToml(t *testing.T) {
	f := setup(t)
	c := filepath.Join(f.home, ".codex/config.toml")
	for _, k := range []struct {
		s string
		w bool
	}{
		{"model = \"x\"\n\n[mcp_servers.codebase-memory-mcp]\ncommand = \"x\"\n", true},
		{"[mcp_servers.other]\ncommand = \"x\"\n", false},
		{"[mcp_servers.\"codebase-memory\"]\ncommand = \"x\"\n", true},
	} {
		f.write(c, k.s)
		want(t, f.sig("codex"), "code_graph", k.w)
	}
}

func TestCodexTomlRegexFallback(t *testing.T) {
	f := setup(t)
	parseTOML = false
	t.Cleanup(func() { parseTOML = true })
	c := filepath.Join(f.home, ".codex/config.toml")
	f.write(c, "[mcp_servers.codebase-memory-mcp]\ncommand = \"x\"\n[plugins.\"superpowers@mk\"]\nenabled = true\n")
	out := f.sig("codex")
	want(t, out, "code_graph", true)
	want(t, out, "superpowers", true)
	f.write(c, "[mcp_servers.other]\n[plugins.\"other@mk\"]\n")
	out = f.sig("codex")
	want(t, out, "code_graph", false)
	want(t, out, "superpowers", false)
}

func TestCodexBadTomlIsNotACrash(t *testing.T) {
	f := setup(t)
	f.write(filepath.Join(f.home, ".codex/config.toml"), "[mcp_servers.codebase-memory-mcp]\nbroken = = =\n")
	want(t, f.sig("codex"), "code_graph", true)
}

func TestCodexSuperpowersNeedsAPluginsTable(t *testing.T) {
	f := setup(t)
	cfg := filepath.Join(f.home, ".codex")
	f.write(filepath.Join(cfg, "config.toml"), "# superpowers@x\n[plugins.\"other@x\"]\n")
	want(t, f.sig("codex"), "superpowers", false)
	f.write(filepath.Join(cfg, "config.toml"), "[plugins.\"superpowers@x\"]\n")
	want(t, f.sig("codex", "CODEX_HOME", cfg), "superpowers", true)
}

func TestCursorMcpJson(t *testing.T) {
	f := setup(t)
	f.fake("git", `echo "`+f.cwd+`"`)
	m := filepath.Join(f.home, ".cursor/mcp.json")
	f.write(m, map[string]any{"mcpServers": map[string]any{"other": map[string]any{}}})
	want(t, f.sig("cursor"), "code_graph", false)
	f.write(m, map[string]any{"mcpServers": map[string]any{"codebase-memory-mcp": map[string]any{}}})
	want(t, f.sig("cursor"), "code_graph", true)
	os.Remove(m)
	f.write(filepath.Join(f.cwd, ".cursor/mcp.json"), map[string]any{"mcpServers": map[string]any{"codebase-memory": map[string]any{}}})
	want(t, f.sig("cursor"), "code_graph", true)
}

func TestOpencodeMcpKey(t *testing.T) {
	f := setup(t)
	c := filepath.Join(f.home, ".config/opencode/opencode.json")
	f.write(c, map[string]any{"mcp": map[string]any{"codebase-memory-mcp": map[string]any{"type": "local"}}})
	want(t, f.sig("opencode"), "code_graph", true)
	f.write(c, map[string]any{"mcp": map[string]any{"other": map[string]any{}}})
	want(t, f.sig("opencode"), "code_graph", false)
}

func TestOpencodeSuperpowersFromPluginList(t *testing.T) {
	f := setup(t)
	c := filepath.Join(f.home, ".config/opencode/opencode.json")
	for _, k := range []struct {
		v any
		w bool
	}{
		{[]any{"superpowers@git+https://x/superpowers.git"}, true},
		{[]any{"other"}, false},
		{"superpowers", false},
	} {
		f.write(c, map[string]any{"plugin": k.v})
		want(t, f.sig("opencode"), "superpowers", k.w)
	}
}

func TestUnknownSignalIsNone(t *testing.T) {
	f := setup(t)
	out := f.sig("cursor")
	want(t, out, "superpowers", nil)
	want(t, out, "autocompact", nil)
	for _, h := range []string{"codex", "cursor", "opencode"} {
		want(t, f.sig(h), "autocompact", nil)
	}
	f.fake("matchblox", "exit 0")
	for _, h := range []string{"codex", "opencode"} {
		want(t, f.sig(h), "matchblox_hooks", nil)
	}
	want(t, f.sig("claude"), "autocompact", false)
	want(t, f.sig("claude"), "superpowers", false)
}

func TestGstackOnOtherHarnessesIsAnyGstackDashSkill(t *testing.T) {
	f := setup(t)
	for h, base := range map[string]string{"codex": ".codex", "cursor": ".cursor", "opencode": ".config/opencode"} {
		want(t, f.sig(h), "gstack", false)
		os.MkdirAll(filepath.Join(f.home, base, "skills/gstack"), 0o755)
		want(t, f.sig(h), "gstack", false)
		os.MkdirAll(filepath.Join(f.home, base, "skills/gstack-qa"), 0o755)
		want(t, f.sig(h), "gstack", true)
	}
}

func TestCursorMatchbloxHooksSignal(t *testing.T) {
	f := setup(t)
	f.fake("matchblox", "exit 0")
	h := filepath.Join(f.home, ".cursor/hooks.json")
	f.write(h, map[string]any{"version": 1, "hooks": map[string]any{"stop": []any{map[string]any{"command": "matchblox hook --cursor stop"}}}})
	want(t, f.sig("cursor"), "matchblox_hooks", true)
	f.write(h, map[string]any{"version": 1, "hooks": map[string]any{"stop": []any{map[string]any{"command": "matchblox hook stop"}}}})
	want(t, f.sig("cursor"), "matchblox_hooks", false)
	f.write(h, "{bad")
	want(t, f.sig("cursor"), "matchblox_hooks", false)
}

func TestSessionsProcCountsAllHarnesses(t *testing.T) {
	f := setup(t)
	f.fakePS("1 claude claude", "2 /usr/bin/claude /usr/bin/claude", "3 codex codex", "4 opencode opencode",
		"5 node node /x/cursor-agent/versions/1/index.js", "6 bash bash -c echo claude")
	want(t, f.sig("claude"), "sessions_proc", 5.0)
}

func TestCursorWorkerServerNotCounted(t *testing.T) {
	f := setup(t)
	f.fakePS("5 node node /x/cursor-agent/versions/1/index.js", "6 node node /x/cursor-agent/versions/1/index.js worker-server")
	want(t, f.sig("cursor"), "sessions_proc", 1.0)
}

func TestCodexAppServerNotCounted(t *testing.T) {
	f := setup(t)
	f.fakePS("3 /opt/bin/codex codex", "4 /opt/bin/codex codex app-server --listen")
	want(t, f.sig("codex"), "sessions_proc", 1.0)
}

func TestNoClaudeLiteral(t *testing.T) {
	for _, name := range []string{"signals.go", "run.go"} {
		src, _ := os.ReadFile(name)
		if strings.Contains(string(src), "~/.claude") || strings.Contains(string(src), `".claude`) {
			t.Fatal(name)
		}
	}
}

func TestCollectHasTheTwelveKeys(t *testing.T) {
	f := setup(t)
	out := f.collect(generous)
	if len(out) != 12 {
		t.Fatalf("%d keys: %v", len(out), out)
	}
}

func TestSplit3IsPythonSplitNone2(t *testing.T) {
	for in, w := range map[string]string{"1 claude claude --x": "1|claude|claude --x", "  7  node   a  b ": "7|node|a  b ", "9 x": "9|x", "": ""} {
		if got := strings.Join(split3(in), "|"); got != w {
			t.Fatalf("%q: got %q want %q", in, got, w)
		}
	}
}
