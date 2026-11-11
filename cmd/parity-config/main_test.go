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

func TestNoOtherPackIsEnabledOrCopied(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude/plugins/cache/m/superpowers/1/SKILL.md", "sp")
	tr.write(".claude/plugins/installed_plugins.json", `{"plugins": {"superpowers@m": []}}`)
	tr.write(".claude/skills/gstack/browse/SKILL.md", "b")
	tr.write(".claude/settings.json", `{"enabledPlugins": {"superpowers@m": true}}`)
	tr.run()
	if _, ok := tr.json("settings.json")["enabledPlugins"]; ok {
		t.Error("settings.json enables a plugin")
	}
	for _, p := range []string{"plugins", "skills"} {
		if exists(filepath.Join(tr.cfg, p)) {
			t.Errorf("%s was written", p)
		}
	}
}

func TestRerunClearsOldCopies(t *testing.T) {
	tr := newTree(t)
	tr.write("cfg/plugins/stale", "x")
	tr.write("cfg/skills/stale/SKILL.md", "x")
	tr.run()
	if exists(filepath.Join(tr.cfg, "plugins")) || exists(filepath.Join(tr.cfg, "skills")) {
		t.Error("a run kept an old plugins or skills folder")
	}
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func TestKeepsOnlyMatchbloxHooks(t *testing.T) {
	tr := newTree(t)
	tr.write(".claude/settings.json", `{"hooks": {"Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "/x/matchblox hook Stop"}, {"type": "command", "command": "/x/other"}]}],
	  "PostToolUse": [{"hooks": [{"type": "command", "command": "/x/other"}]}]},
	  "statusLine": {"type": "command", "command": "/x/matchblox hook statusline"}, "permissions": {"allow": ["Bash(ls:*)"]}}`)
	tr.run()
	want := `{
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
		"note: codebase-memory-mcp is not set up; the builder-led run has no code graph\n",
	} {
		if !strings.Contains(stderr, n) {
			t.Errorf("stderr lacks %q: %q", n, stderr)
		}
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
