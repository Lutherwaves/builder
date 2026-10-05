package sample

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/config"
	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

const fixture = "../../testdata/machine"

// fixtureNow is one hour after the fixture sessions started.
var fixtureNow = time.UnixMilli(1790003600000 + 10_000)

func newFixtureSampler(root string) *Sampler {
	return &Sampler{
		FS:    procfs.FS{Root: filepath.Join(root, "proc")},
		Home:  filepath.Join(root, "home"),
		Tmux:  func() ([]byte, error) { return os.ReadFile(filepath.Join(root, "tmux-panes.txt")) },
		Rules: DefaultRules,
		Cfg:   config.Default(),
		Now:   func() time.Time { return fixtureNow },
	}
}

// copyFixture copies the fixture tree, keeping the cwd symlinks that
// os.CopyFS refuses.
func copyFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	err := filepath.WalkDir(fixture, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixture, path)
		dst := filepath.Join(root, rel)
		switch {
		case d.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, dst)
		case d.IsDir():
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	// Every fixture process belongs to whoever runs the test.
	dirs, _ := filepath.Glob(filepath.Join(root, "proc", "[0-9]*"))
	for _, d := range dirs {
		status := fmt.Sprintf("Name:\tx\nUid:\t%d\t%d\t%d\t%d\n", os.Getuid(), os.Getuid(), os.Getuid(), os.Getuid())
		os.WriteFile(filepath.Join(d, "status"), []byte(status), 0o644)
	}
	return root
}

// setTicks rewrites a fixture process's utime.
func setTicks(t *testing.T, root string, pid, ticks int) {
	t.Helper()
	path := filepath.Join(root, "proc", strconv.Itoa(pid), "stat")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(string(b))
	f[13] = strconv.Itoa(ticks)
	os.WriteFile(path, []byte(strings.Join(f, " ")), 0o644)
}

func byPane(snap Snapshot) map[string]Session {
	out := map[string]Session{}
	for _, s := range snap.Sessions {
		out[s.Pane] = s
	}
	return out
}

func TestSessionsMapToPanes(t *testing.T) {
	snap := newFixtureSampler(fixture).Sample()
	if len(snap.Errors) > 0 {
		t.Fatal(snap.Errors)
	}
	got := byPane(snap)
	if len(got) != 3 {
		t.Fatalf("want 3 agent sessions, got %+v", snap.Sessions)
	}

	feature := got["%1"]
	if feature.PID != 200 || feature.Name != "app-feature" || !feature.Busy || feature.Procs != 2 {
		t.Fatalf("%%1 = %+v", feature)
	}
	// The last main-thread turn counts; the sidechain line after it does not.
	if feature.Tokens != 180000 || feature.ContextPct != 90 || feature.Do != "compact" {
		t.Fatalf("%%1 context = %d %.1f %q", feature.Tokens, feature.ContextPct, feature.Do)
	}

	review := got["%2"]
	if review.PID != 310 || review.Busy || review.Idle != time.Hour+10*time.Second || review.Do != "clear" {
		t.Fatalf("%%2 = %+v", review)
	}

	// pid 400 was re-parented to init; TMUX_PANE still names its pane.
	tools := got["%3"]
	if tools.PID != 400 || tools.Context != "fresh" || tools.Do != "" {
		t.Fatalf("%%3 = %+v", tools)
	}

	if len(snap.IdlePanes) != 1 || snap.IdlePanes[0].Pane != "%4" {
		t.Fatalf("idle panes = %+v", snap.IdlePanes)
	}
}

func TestTreeCPUAndRates(t *testing.T) {
	root := copyFixture(t)
	s := newFixtureSampler(root)
	s.Sample()

	// One second later the agent's child has burned 50 ticks: half a core.
	s.Now = func() time.Time { return fixtureNow.Add(time.Second) }
	setTicks(t, root, 210, 150)
	os.WriteFile(filepath.Join(root, "proc", "net", "dev"),
		[]byte("Inter-|\n face |\n  eth0: 3000 10 0 0 0 0 0 0 2500 20 0 0 0 0 0 0\n"), 0o644)

	snap := s.Sample()
	if cpu := byPane(snap)["%1"].CPU; cpu != 50 {
		t.Fatalf("tree cpu = %v, want 50", cpu)
	}
	if snap.Machine.NetRx != 2000 || snap.Machine.NetTx != 500 {
		t.Fatalf("net rate = %v %v", snap.Machine.NetRx, snap.Machine.NetTx)
	}
}

func TestReusedPIDIsRemapped(t *testing.T) {
	root := copyFixture(t)
	s := newFixtureSampler(root)
	s.Sample()
	// pid 400 exits and a new process with the same pid starts in pane %4.
	os.WriteFile(filepath.Join(root, "proc", "400", "stat"),
		[]byte("400 (claude) S 1 400 400 0 -1 0 0 0 0 0 1 0 0 0 20 0 1 0 9999 100000 2500"), 0o644)
	os.WriteFile(filepath.Join(root, "proc", "400", "environ"), []byte("TMUX_PANE=%4\x00"), 0o644)
	if got := byPane(s.Sample())["%4"].PID; got != 400 {
		t.Fatalf("reused pid not remapped to %%4")
	}
}

func TestSuggest(t *testing.T) {
	r := DefaultRules
	cases := []struct {
		name string
		s    Session
		want string
	}{
		{"fresh session", Session{Tokens: 0, Idle: 48 * time.Hour}, ""},
		{"busy and near the window", Session{Tokens: 1, Busy: true, ContextPct: 90}, "compact"},
		{"briefly idle and near the window", Session{Tokens: 1, Idle: 5 * time.Minute, ContextPct: 94}, "compact"},
		{"long idle beats compact", Session{Tokens: 1, Idle: 5 * time.Hour, ContextPct: 95}, "clear"},
		{"idle holding context", Session{Tokens: 1, Idle: 31 * time.Minute, ContextPct: 25}, "clear"},
		{"idle but small", Session{Tokens: 1, Idle: 2 * time.Hour, ContextPct: 10}, ""},
		{"stale", Session{Tokens: 1, Idle: 25 * time.Hour, ContextPct: 1}, "clear"},
		{"busy and fine", Session{Tokens: 1, Busy: true, ContextPct: 50}, ""},
	}
	for _, c := range cases {
		if got, _ := r.suggest(c.s); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestWindowIsPerSession(t *testing.T) {
	s := &Sampler{}
	s.init()
	if w := s.window("a", "model-a", 150_000, false); w != 200_000 {
		t.Fatalf("got %d", w)
	}
	s.window("a", "model-a", 250_000, false)
	if w := s.window("a", "model-a", 150_000, false); w != 1_000_000 {
		t.Fatalf("a session past 200k once must stay 1M, got %d", w)
	}
	if w := s.window("b", "model-a", 150_000, false); w != 200_000 {
		t.Fatalf("another session of the same model is not 1M: %d", w)
	}
	s.Rules.Windows = map[string]int{"model-b": 500_000}
	if w := s.window("c", "model-b-large", 10, false); w != 500_000 {
		t.Fatalf("configured window ignored: %d", w)
	}
	if w := s.window("d", "model-a", 150_000, true); w != 1_000_000 {
		t.Fatalf("a 1M model setting must give a 1M window below 200k: %d", w)
	}
}

func TestModelSetting(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	noEnv := func(string) (string, bool) { return "", false }
	write := func(dir, name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".claude", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if m := modelSetting("claude", noEnv, cwd, home); m != "" {
		t.Fatalf("no setting anywhere: %q", m)
	}
	write(home, "settings.json", `{"model":"big[1m]"}`)
	if m := modelSetting("claude --resume x", noEnv, cwd, home); !longContext(m) {
		t.Fatalf("user settings ignored: %q", m)
	}
	write(cwd, "settings.local.json", `{"model":"small"}`)
	if m := modelSetting("claude", noEnv, cwd, home); m != "small" {
		t.Fatalf("project local settings must win over user settings: %q", m)
	}
	env := func(k string) (string, bool) { return "env[1M]", k == "ANTHROPIC_MODEL" }
	if m := modelSetting("claude", env, cwd, home); !longContext(m) {
		t.Fatalf("ANTHROPIC_MODEL must win over settings: %q", m)
	}
	for _, cmd := range []string{"claude --model flag[1m]", "claude --model=flag[1m]"} {
		if m := modelSetting(cmd, env, cwd, home); m != "flag[1m]" {
			t.Fatalf("%s: the flag must win: %q", cmd, m)
		}
	}
}

func TestNestedAgentFoldsIntoTopSession(t *testing.T) {
	root := copyFixture(t)
	// A shell under pane %1's agent runs another agent with the same pane.
	os.MkdirAll(filepath.Join(root, "proc", "220"), 0o755)
	os.MkdirAll(filepath.Join(root, "proc", "221"), 0o755)
	os.WriteFile(filepath.Join(root, "proc", "220", "stat"), []byte("220 (bash) S 200 1 1 0 -1 0 0 0 0 0 1 0 0 0 20 0 1 0 1000 1 1"), 0o644)
	os.WriteFile(filepath.Join(root, "proc", "221", "stat"), []byte("221 (claude) S 220 1 1 0 -1 0 0 0 0 0 1 0 0 0 20 0 1 0 1000 1 1"), 0o644)
	os.WriteFile(filepath.Join(root, "proc", "221", "environ"), []byte("TMUX_PANE=%1\x00"), 0o644)
	n := 0
	for _, x := range newFixtureSampler(root).Sample().Sessions {
		if x.Pane == "%1" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("pane %%1 has %d sessions, want 1", n)
	}
}

func TestPowerZoneSeenLateIsNotDrift(t *testing.T) {
	s := &Sampler{}
	s.init()
	s.powerDrift(nil)
	if d := s.powerDrift([]procfs.PowerLimit{{Zone: "package-0", Name: "long_term", Watts: 125}}); len(d) != 0 {
		t.Fatalf("late zone flagged: %+v", d)
	}
}

func TestBurn(t *testing.T) {
	s := &Sampler{}
	s.init()
	t0 := time.Unix(0, 0)
	s.trackBurn("x", t0, 1000)
	s.trackBurn("x", t0.Add(10*time.Minute), 5000)
	if got := s.trackBurn("x", t0.Add(20*time.Minute), 9000); got != 8000 {
		t.Fatalf("burn = %d", got)
	}
	if got := s.trackBurn("x", t0.Add(45*time.Minute), 9000); got != 4000 {
		t.Fatalf("burn after window slides = %d", got)
	}
}

func TestHuman(t *testing.T) {
	for d, want := range map[time.Duration]string{
		3 * time.Second:              "3s",
		41 * time.Minute:             "41m",
		2*time.Hour + 14*time.Minute: "2h14m",
		time.Hour:                    "1h",
		24 * time.Hour:               "1d",
		6*24*time.Hour + 2*time.Hour: "6d2h",
	} {
		if got := Human(d); got != want {
			t.Errorf("Human(%v) = %q want %q", d, got, want)
		}
	}
}

func TestParsePanesSkipsJunk(t *testing.T) {
	got := ParsePanes([]byte("%1\ta:1.1\tw\t10\tzsh\t/x\n\ngarbage\n"))
	if len(got) != 1 || got[0].PID != 10 || got[0].Path != "/x" {
		t.Fatalf("got %+v", got)
	}
}

func TestOrphanFlaggedAfterSustainedCPU(t *testing.T) {
	root := copyFixture(t)
	s := newFixtureSampler(root)
	at := func(sec int) { s.Now = func() time.Time { return fixtureNow.Add(time.Duration(sec) * time.Second) } }

	s.Sample()
	// Both shells burn 90% of a core; 601 is still inside pane %2's tree.
	for i, sec := range []int{40, 80} {
		at(sec)
		setTicks(t, root, 600, 1000+3600*(i+1))
		setTicks(t, root, 601, 1000+3600*(i+1))
		snap := s.Sample()
		if i == 0 && len(snap.Orphans) != 0 {
			t.Fatalf("flagged before it stayed hot for 30s: %+v", snap.Orphans)
		}
		if i == 1 {
			if len(snap.Orphans) != 1 {
				t.Fatalf("want only pid 600, got %+v", snap.Orphans)
			}
			o := snap.Orphans[0]
			if o.PID != 600 || o.Pane != "%2" || !o.PaneAlive || o.Target != "work:2.1" ||
				o.Parent != "user service manager" || o.CPU != 90 || strings.Join(o.Kill, " ") != "kill 600" {
				t.Fatalf("orphan = %+v", o)
			}
		}
	}
}

func TestOwnerAttribution(t *testing.T) {
	root := copyFixture(t)
	s := newFixtureSampler(root)
	s.Sample()
	s.Now = func() time.Time { return fixtureNow.Add(time.Second) }
	setTicks(t, root, 210, 180)
	snap := s.Sample()
	if snap.Top[0].PID != 210 || snap.Top[0].Owner != "pane %1" {
		t.Fatalf("top = %+v", snap.Top[0])
	}
}

func TestAlerts(t *testing.T) {
	s := newFixtureSampler(fixture)
	s.Cfg.Alerts.PowerLimitsW = []float64{125, 188}
	s.Cfg.Alerts.TempOverC = 85
	s.Cfg.Alerts.Load1Over = 3
	s.Cfg.Alerts.Load1For.Duration = time.Minute
	s.Cfg.Alerts.SwapOverGB = 0.5
	s.Cfg.Alerts.SwapWhenAvailableOverGB = 4
	s.Cfg.Alerts.MemAvailableUnderGB = 10
	s.Sys = procfs.Sys{Root: filepath.Join(fixture, "sys")}

	keys := func(snap Snapshot) string {
		var k []string
		for _, a := range snap.Alerts {
			k = append(k, a.Key)
		}
		return strings.Join(k, ",")
	}
	// load1 3.5 > 3 has not held for a minute yet; the others fire at once.
	if got := keys(s.Sample()); got != "temp,mem-low,swap-stuck" {
		t.Fatalf("first sample alerts = %q", got)
	}
	s.Now = func() time.Time { return fixtureNow.Add(61 * time.Second) }
	if got := keys(s.Sample()); got != "temp,load1,mem-low,swap-stuck" {
		t.Fatalf("after a minute alerts = %q", got)
	}
}

func TestPowerDrift(t *testing.T) {
	s := &Sampler{}
	s.init()
	limits := []procfs.PowerLimit{{Zone: "package-0", Name: "long_term", Watts: 125}, {Zone: "package-0", Name: "short_term", Watts: 188}}
	if d := s.powerDrift(limits); len(d) != 0 {
		t.Fatalf("baseline drifted: %+v", d)
	}
	limits[0].Watts = 253
	if d := s.powerDrift(limits); len(d) != 1 || !strings.Contains(d[0].evidence, "is 253 W, at startup 125 W") {
		t.Fatalf("change from startup not caught: %+v", d)
	}
	s.Cfg.Alerts.PowerLimitsW = []float64{253, 188}
	if d := s.powerDrift(limits); len(d) != 0 {
		t.Fatalf("expected value must win over startup: %+v", d)
	}
}

func TestContainerGroups(t *testing.T) {
	root := copyFixture(t)
	id := "aaaabbbbccccddddeeeeffff0000111122223333444455556666777788889999"
	s := newFixtureSampler(root)
	s.Sys = procfs.Sys{Root: filepath.Join(root, "sys")}
	s.Docker = func() ([]byte, error) { return []byte(id + "\tci-runner-1\n"), nil }
	cfg, err := config.Load(writeTOML(t, "[[groups]]\nname = \"CI\"\ncontainer = \"runner\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	s.Cfg = cfg
	s.Sample()
	s.Now = func() time.Time { return fixtureNow.Add(2 * time.Second) }
	os.WriteFile(filepath.Join(root, "sys/fs/cgroup/system.slice/docker-"+id+".scope/cpu.stat"), []byte("usage_usec 4000000\n"), 0o644)
	snap := s.Sample()
	if len(snap.Containers) != 1 || snap.Containers[0].CPU != 150 || snap.Containers[0].Name != "ci-runner-1" {
		t.Fatalf("containers = %+v", snap.Containers)
	}
	// 150% of a core on a two-core fixture is 75% of the machine.
	if g := snap.Groups; len(g) != 1 || g[0].Containers != 1 || g[0].Share != 75 {
		t.Fatalf("groups = %+v", g)
	}
}

func writeTOML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tui.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The session file's pane wins over the environment: it is the only source
// on systems where another process's environment cannot be read.
func TestPaneFromSessionFile(t *testing.T) {
	root := copyFixture(t)
	f := filepath.Join(root, "home", ".claude", "sessions", "200.json")
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), `"pid":200,`, `"pid":200,"tmux":"notes:@4.%4",`, 1))
	if err := os.WriteFile(f, b, 0o644); err != nil {
		t.Fatal(err)
	}
	snap := newFixtureSampler(root).Sample()
	for _, s := range snap.Sessions {
		if s.PID == 200 {
			if s.Pane != "%4" || s.Target != "notes:1.1" {
				t.Fatalf("want %%4 notes:1.1 from the session file, got %s %s", s.Pane, s.Target)
			}
			return
		}
	}
	t.Fatal("session 200 not found")
}
