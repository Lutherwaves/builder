package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tree struct {
	t                     *testing.T
	home, src, state, cfg string
}

func newTree(t *testing.T) *tree {
	t.Helper()
	home := t.TempDir()
	tr := &tree{t: t, home: home, src: filepath.Join(home, ".claude"), state: filepath.Join(home, ".claude.json"), cfg: filepath.Join(home, "cfg")}
	if err := os.MkdirAll(tr.cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	return tr
}

func (tr *tree) write(rel, body string) string {
	tr.t.Helper()
	p := filepath.Join(tr.home, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		tr.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		tr.t.Fatal(err)
	}
	return p
}

func (tr *tree) run() string {
	tr.t.Helper()
	var stderr bytes.Buffer
	if err := run(tr.src, tr.state, tr.cfg, &stderr); err != nil {
		tr.t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	return stderr.String()
}

func (tr *tree) read(rel string) string {
	tr.t.Helper()
	b, err := os.ReadFile(filepath.Join(tr.cfg, rel))
	if err != nil {
		tr.t.Fatal(err)
	}
	return string(b)
}

func (tr *tree) json(rel string) map[string]any {
	tr.t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(tr.read(rel)), &v); err != nil {
		tr.t.Fatalf("%s: %v", rel, err)
	}
	return v
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func (tr *tree) plugins(extraPath string) {
	tr.write(".claude/plugins/cache/m/superpowers/1/SKILL.md", "sp")
	tr.write(".claude/plugins/cache/m/other/1/SKILL.md", "other")
	tr.write(".claude/plugins/marketplaces/m/x", "mk")
	tr.write(".claude/plugins/marketplaces/x/y", "x")
	sp := filepath.Join(tr.src, "plugins/cache/m/superpowers/1")
	if extraPath != "" {
		sp = extraPath
	}
	b, _ := json.Marshal(map[string]any{"version": 2, "plugins": map[string]any{
		"superpowers@m": []any{map[string]any{"installPath": sp}},
		"other@x":       []any{map[string]any{"installPath": filepath.Join(tr.src, "plugins/cache/m/other/1")}},
	}})
	tr.write(".claude/plugins/installed_plugins.json", string(b))
	b, _ = json.Marshal(map[string]any{
		"m": map[string]any{"installLocation": filepath.Join(tr.src, "plugins/marketplaces/m")},
		"x": map[string]any{"installLocation": filepath.Join(tr.src, "plugins/marketplaces/x")},
	})
	tr.write(".claude/plugins/known_marketplaces.json", string(b))
	tr.write(".claude/settings.json", `{"enabledPlugins": {"superpowers@m": true, "other@x": true}}`)
}

func TestKeepsOnlySuperpowers(t *testing.T) {
	tr := newTree(t)
	tr.plugins("")
	tr.run()
	inst := tr.json("plugins/installed_plugins.json")
	if ps := inst["plugins"].(map[string]any); len(ps) != 1 || ps["superpowers@m"] == nil {
		t.Errorf("installed plugins = %v, want only superpowers@m", ps)
	}
	if inst["version"] != float64(2) {
		t.Errorf("version = %v, want the other keys kept", inst["version"])
	}
	if en := tr.json("settings.json")["enabledPlugins"].(map[string]any); len(en) != 1 || en["superpowers@m"] != true {
		t.Errorf("enabledPlugins = %v", en)
	}
	if km := tr.json("plugins/known_marketplaces.json"); len(km) != 1 || km["m"] == nil {
		t.Errorf("known marketplaces = %v, want only m", km)
	}
	if exists(filepath.Join(tr.cfg, "plugins/cache/m/other")) || exists(filepath.Join(tr.cfg, "plugins/marketplaces/x")) {
		t.Error("other@x was copied")
	}
}

func TestRelinksPluginPaths(t *testing.T) {
	tr := newTree(t)
	tr.plugins("")
	tr.run()
	want := filepath.Join(tr.cfg, "plugins/cache/m/superpowers/1")
	got := tr.json("plugins/installed_plugins.json")["plugins"].(map[string]any)["superpowers@m"].([]any)[0].(map[string]any)["installPath"]
	if got != want {
		t.Errorf("installPath = %v, want %s", got, want)
	}
	if loc := tr.json("plugins/known_marketplaces.json")["m"].(map[string]any)["installLocation"]; loc != filepath.Join(tr.cfg, "plugins/marketplaces/m") {
		t.Errorf("installLocation = %v", loc)
	}
	if tr.read("plugins/cache/m/superpowers/1/SKILL.md") != "sp" || tr.read("plugins/marketplaces/m/x") != "mk" {
		t.Error("the plugin folders were not copied")
	}
}

func TestOutsidePluginsIsNotCopied(t *testing.T) {
	tr := newTree(t)
	outside := filepath.Join(tr.home, "elsewhere/sp")
	tr.write("elsewhere/sp/SKILL.md", "sp")
	tr.plugins(outside)
	stderr := tr.run()
	if !strings.Contains(stderr, "note: "+outside+" is outside your plugins folder; parity does not copy it\n") {
		t.Errorf("stderr = %q", stderr)
	}
	got := tr.json("plugins/installed_plugins.json")["plugins"].(map[string]any)["superpowers@m"].([]any)[0].(map[string]any)["installPath"]
	if got != outside {
		t.Errorf("installPath = %v, want it unchanged", got)
	}
	if exists(filepath.Join(tr.cfg, "plugins/elsewhere")) || exists(filepath.Join(tr.cfg, "plugins/sp")) {
		t.Error("an outside folder was copied")
	}
}

func TestKeepsOnlyMatchbloxHooks(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude/settings.json", `{"hooks": {"Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "/x/matchblox hook Stop"}, {"type": "command", "command": "/x/other"}]}],
	  "PostToolUse": [{"hooks": [{"type": "command", "command": "/x/other"}]}]},
	  "statusLine": {"type": "command", "command": "/x/matchblox hook statusline"}, "permissions": {"allow": ["Bash(ls:*)"]}}`)
	tr.run()
	want := `{
  "enabledPlugins": {},
  "env": {
    "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"
  },
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/x/matchblox hook Stop"
          }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "/x/matchblox hook statusline"
  }
}`
	if got := tr.read("settings.json"); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}

	tr.write(".claude/settings.json", `{"statusLine": {"type": "command", "command": "ccstatus"}}`)
	stderr := tr.run()
	if s := tr.json("settings.json"); s["statusLine"] != nil || s["hooks"] != nil {
		t.Errorf("settings.json = %v, want no hooks and no statusLine", s)
	}
	if !strings.Contains(stderr, "note: your settings have no matchblox hooks, so the builder-led run has none. Run `matchblox setup` first for a fair run.\n") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestCodeGraphMergesIntoExistingState(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude.json", `{"mcpServers": {"codebase-memory-mcp": {"command": "/x/cbm"}, "other": {}}, "oauthAccount": {"secret": 1}}`)
	tr.write("cfg/.claude.json", `{"oauthAccount":{}}`)
	tr.run()
	want := `{
  "oauthAccount": {},
  "mcpServers": {
    "codebase-memory-mcp": {
      "command": "/x/cbm"
    }
  }
}`
	if got := tr.read(".claude.json"); got != want {
		t.Errorf(".claude.json =\n%s\nwant\n%s", got, want)
	}

	tr2 := newTree(t)
	stderr := tr2.run()
	if exists(filepath.Join(tr2.cfg, ".claude.json")) {
		t.Error("wrote .claude.json without a code graph server")
	}
	for _, n := range []string{
		"note: superpowers is not installed; the builder-led run has no superpowers\n",
		"note: gstack is not installed; the builder-led run has no gstack\n",
		"note: codebase-memory-mcp is not set up; the builder-led run has no code graph\n",
	} {
		if !strings.Contains(stderr, n) {
			t.Errorf("stderr lacks %q: %q", n, stderr)
		}
	}
}

func TestGstackLinkAndOwnedCopies(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude/skills/gstack/browse/SKILL.md", "b")
	tr.write(".claude/skills/browse/.gstack-owned", "own")
	tr.write(".claude/skills/personal/SKILL.md", "p")
	link := filepath.Join(tr.src, "skills/browse/SKILL.md")
	if err := os.Symlink(filepath.Join(tr.src, "skills/gstack/browse/SKILL.md"), link); err != nil {
		t.Fatal(err)
	}
	tr.run()
	if got, err := os.Readlink(filepath.Join(tr.cfg, "skills/gstack")); err != nil || got != filepath.Join(tr.src, "skills/gstack") {
		t.Errorf("skills/gstack -> %q, %v", got, err)
	}
	if tr.read("skills/browse/.gstack-owned") != "own" {
		t.Error("the gstack-owned folder was not copied")
	}
	if got, err := os.Readlink(filepath.Join(tr.cfg, "skills/browse/SKILL.md")); err != nil || got != filepath.Join(tr.src, "skills/gstack/browse/SKILL.md") {
		t.Errorf("the copied symlink -> %q, %v; want it kept as a link", got, err)
	}
	if exists(filepath.Join(tr.cfg, "skills/personal")) {
		t.Error("a personal skill was copied")
	}
}

func TestRerunClearsOldCopies(t *testing.T) {
	tr := newTree(t)
	tr.plugins("")
	tr.write(".claude/skills/gstack/SKILL.md", "g")
	tr.write(".claude/skills/browse/.gstack-owned", "own")
	tr.run()
	if err := os.RemoveAll(filepath.Join(tr.src, "skills/browse")); err != nil {
		t.Fatal(err)
	}
	tr.write("cfg/plugins/stale", "x")
	tr.run()
	if exists(filepath.Join(tr.cfg, "skills/browse")) || exists(filepath.Join(tr.cfg, "plugins/stale")) {
		t.Error("a second run kept an old copy")
	}
	if tr.read("plugins/cache/m/superpowers/1/SKILL.md") != "sp" || !exists(filepath.Join(tr.cfg, "skills/gstack")) {
		t.Error("a second run did not rebuild the copies")
	}
}

// The files match what Python's json.dumps(indent=2) wrote: key order as read, non-ASCII escaped, numbers as Python prints them.
func TestWritesJSONAsPythonDid(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude.json", `{"mcpServers": {"codebase-memory-mcp": {"z": 1, "a": "é <&> 😀\u007f", "n": [1.50, 1e16, 1E-5, 0.0001, -0, 12345678901234567890, 2.5e3], "e": [], "o": {}, "t": true, "f": false, "u": null}}}`)
	tr.run()
	want := `{
  "mcpServers": {
    "codebase-memory-mcp": {
      "z": 1,
      "a": "\u00e9 <&> \ud83d\ude00\u007f",
      "n": [
        1.5,
        1e+16,
        1e-05,
        0.0001,
        0,
        12345678901234567890,
        2500.0
      ],
      "e": [],
      "o": {},
      "t": true,
      "f": false,
      "u": null
    }
  }
}`
	if got := tr.read(".claude.json"); got != want {
		t.Errorf(".claude.json =\n%s\nwant\n%s", got, want)
	}
}
