package advice

import (
	"strings"
	"testing"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

func TestBuildRanksAndActs(t *testing.T) {
	snap := sample.Snapshot{
		Orphans: []sample.Orphan{{PID: 42, Comm: "bash", CPU: 99, HotFor: time.Hour, Parent: "init",
			Pane: "%3", PaneAlive: true, Target: "w:1.1", Kill: []string{"kill", "42"}}},
		Sessions: []sample.Session{
			{Pane: "%1", Name: "busy-one", Busy: true, Do: "compact", ContextPct: 90, Why: "context 90%"},
			{Pane: "%2", Name: "idle-one", Status: "idle", Do: "clear", ContextPct: 40, Why: "idle 2h"},
			{Pane: "%4", Name: "fine"},
		},
		GitPolling: []sample.GitPolling{{Checkout: "/w/app", Cores: 2}},
	}
	git := &gitscan.Report{Repos: []gitscan.Repo{{
		Path: "/w/app", Main: "main", Behind: 3,
		Worktrees: []gitscan.Worktree{
			{Path: "/w/app", Branch: "main", Sessions: 3, Dirty: 900},
			{Path: "/w/wt/a", Safe: true, Remove: []string{"git", "-C", "/w/app", "worktree", "remove", "/w/wt/a"}},
			{Path: "/w/wt/b", Safe: true, Remove: []string{"git", "-C", "/w/app", "worktree", "remove", "/w/wt/b"}},
		},
	}}}
	recs := Build(snap, git)

	var titles []string
	for _, r := range recs {
		titles = append(titles, r.Title)
	}
	// Two cores of git polling cost the machine more than one session's
	// context, so it ranks above the compact.
	want := []string{
		"Kill detached busy loop 42 (bash)",
		"Clean up /w/app",
		"Compact busy-one (%1)",
		"Update main in app",
		"Remove 2 merged worktrees of app",
		"Clear idle-one (%2)",
	}
	if strings.Join(titles, "|") != strings.Join(want, "|") {
		t.Fatalf("order:\n got %q\nwant %q", titles, want)
	}

	kill := recs[0]
	if kill.Second == nil || !kill.Second.Destructive || strings.Join(kill.Second.Steps[0], " ") != "kill 42" || kill.Primary == nil ||
		len(kill.Second.Guards) != 1 || kill.Second.Guards[0].PID != 42 {
		t.Errorf("kill rec = %+v", kill)
	}
	if recs[2].Second != nil {
		t.Error("a busy session must not be offered typed input")
	}
	if c := recs[5].Second; c == nil || !c.Destructive || strings.Join(c.Steps[0], " ") != "tmux send-keys -t %2 /clear Enter" {
		t.Errorf("clear action = %+v", c)
	}
	if rm := recs[4].Second; rm == nil || !rm.Destructive || len(rm.Steps) != 2 || len(rm.Guards) != 2 || rm.Guards[1].Worktree != "/w/wt/b" {
		t.Errorf("bulk remove = %+v", rm)
	}
	if ff := recs[3].Second; ff == nil || !ff.Destructive || !strings.HasSuffix(strings.Join(ff.Steps[0], " "), "git -C /w/app pull --ff-only") {
		t.Errorf("fast-forward = %+v", ff)
	}
	for _, r := range recs {
		if r.Primary == nil && r.Second == nil {
			t.Errorf("%q has no action", r.Title)
		}
	}
}

func TestNeverTypesIntoANonIdleSession(t *testing.T) {
	for _, status := range []string{"busy", "shell", ""} {
		snap := sample.Snapshot{Sessions: []sample.Session{{Pane: "%1", Name: "x", Status: status, Do: "clear"}}}
		if r := Build(snap, nil); len(r) != 1 || r[0].Second != nil {
			t.Errorf("status %q: %+v", status, r)
		}
	}
}

func TestNothingToDo(t *testing.T) {
	if recs := Build(sample.Snapshot{Sessions: []sample.Session{{Pane: "%1"}}}, nil); len(recs) != 0 {
		t.Fatalf("got %+v", recs)
	}
}

func TestAlertRecs(t *testing.T) {
	const gb = 1 << 30
	snap := sample.Snapshot{
		Machine: sample.Machine{MemAvail: 40 * gb, SwapUsed: 12 * gb},
		Alerts:  []sample.Alert{{Key: "swap-stuck", Evidence: "swap 12 GB"}, {Key: "load1", Evidence: "load1 50"}},
		Top:     []sample.ProcRow{{PID: 7, Comm: "node", CPU: 300, Owner: "pane %5"}},
	}
	recs := Build(snap, nil)
	if len(recs) != 2 || recs[0].Primary == nil || strings.Join(recs[0].Primary.Steps[0], " ") != "tmux switch-client -t %5" {
		t.Fatalf("load rec: %+v", recs)
	}
	if sw := recs[1].Second; sw == nil || !sw.Destructive || sw.Steps[0][len(sw.Steps[0])-1] != "swapoff -a && swapon -a" {
		t.Fatalf("swap rec: %+v", recs[1])
	}

	// Not enough free RAM to take the swap back: no offer.
	snap.Machine.MemAvail = 13 * gb
	snap.Top[0].Owner = "system"
	if recs := Build(snap, nil); len(recs) != 0 {
		t.Fatalf("got %+v", recs)
	}
}
