// Package advice ranks what to do next. Every recommendation carries the
// evidence that triggered it and an action that resolves it; a finding with
// nothing to do about it is an alert, not a recommendation.
package advice

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

type Action struct {
	Label       string     `json:"label"`
	Steps       [][]string `json:"steps"` // run in order
	Destructive bool       `json:"destructive"`
	// Guards[i], when present, must still hold right before Steps[i] runs:
	// the evidence can be minutes old. A step whose guard fails is skipped.
	Guards []Guard `json:"guards,omitempty"`
}

// Guard is a fact a step depends on, re-checked when it runs.
type Guard struct {
	PID        int    `json:"pid,omitempty"`         // this pid is still the process
	StartTicks uint64 `json:"start_ticks,omitempty"` // that started at this time
	Worktree   string `json:"worktree,omitempty"`    // has no changes and no process inside
	IdlePane   string `json:"idle_pane,omitempty"`   // the agent in this pane is still idle
}

type Rec struct {
	Level    string  `json:"level"` // crit | warn | info
	Title    string  `json:"title"`
	Evidence string  `json:"evidence"`
	Primary  *Action `json:"primary,omitempty"`   // Enter
	Second   *Action `json:"secondary,omitempty"` // x
	score    float64
}

// Thresholds for the git findings.
const (
	pollingCores    = 0.5 // agents' git use worth a look
	sharedDirty     = 200 // dirty paths in a checkout two or more sessions share
	behindWorthPull = 1
)

func one(argv ...string) [][]string { return [][]string{argv} }

func Build(snap sample.Snapshot, git *gitscan.Report) []Rec {
	var out []Rec

	for _, o := range snap.Orphans {
		where := "no pane"
		if o.Pane != "" {
			where = "from pane " + o.Pane
			if o.PaneAlive {
				where += " (" + o.Target + ")"
			} else {
				where += ", now closed"
			}
		}
		r := Rec{
			Level: "crit", score: 1000 + o.CPU,
			Title:    fmt.Sprintf("Kill detached busy loop %d (%s)", o.PID, o.Comm),
			Evidence: fmt.Sprintf("%.0f%% CPU for %s, parent %s, %s: %s", o.CPU, sample.Human(o.HotFor), o.Parent, where, o.Cmdline),
			Second: &Action{Label: "kill", Steps: one(o.Kill...), Destructive: true,
				Guards: []Guard{{PID: o.PID, StartTicks: o.Start}}},
		}
		if o.PaneAlive {
			r.Primary = &Action{Label: "jump to its pane", Steps: one("tmux", "switch-client", "-t", o.Pane)}
		}
		out = append(out, r)
	}

	for _, s := range snap.Sessions {
		if s.Do == "" || s.Pane == "" {
			continue
		}
		r := Rec{
			Level:    "warn",
			Title:    fmt.Sprintf("%s %s (%s)", capital(s.Do), s.Name, s.Pane),
			Evidence: s.Why,
			Primary:  &Action{Label: "jump to the session", Steps: one("tmux", "switch-client", "-t", s.Pane)},
			score:    500 + s.ContextPct,
		}
		if s.Do == "clear" {
			r.Level, r.score = "info", 100+s.ContextPct
		}
		// Type only into a session that says it is idle: a busy one would
		// queue the command, and an Enter at a permission prompt would answer it.
		if s.Status == "idle" {
			r.Second = &Action{
				Label:       "send /" + s.Do,
				Steps:       one("tmux", "send-keys", "-t", s.Pane, "/"+s.Do, "Enter"),
				Destructive: true,
				Guards:      []Guard{{IdlePane: s.Pane}},
			}
		}
		out = append(out, r)
	}

	out = append(out, alertRecs(snap)...)
	out = append(out, gitRecs(snap, git)...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// swapMargin is the RAM that must stay free after swap is read back in.
const swapMargin = 2 << 30

// alertRecs turns alerts that have a fix into recommendations.
func alertRecs(snap sample.Snapshot) []Rec {
	var out []Rec
	m := snap.Machine
	for _, a := range snap.Alerts {
		switch a.Key {
		case "swap-stuck":
			// Reading swap back needs room for all of it plus a margin.
			if m.MemAvail < m.SwapUsed+swapMargin {
				continue
			}
			out = append(out, Rec{
				Level: "info", score: 120,
				Title:    "Move swap back into RAM",
				Evidence: a.Evidence + "; the RAM is free, swapped pages make the next access slow",
				Second: &Action{
					Label:       "swapoff and swapon in a new window (sudo asks there)",
					Steps:       one("tmux", "new-window", "-n", "swap", "sudo", "sh", "-c", "swapoff -a && swapon -a"),
					Destructive: true,
				},
			})
		case "load1", "psi-cpu":
			if len(snap.Top) == 0 {
				continue
			}
			top := snap.Top[0]
			pane, ok := strings.CutPrefix(top.Owner, "pane ")
			if !ok {
				continue
			}
			out = append(out, Rec{
				Level: "warn", score: 450 + top.CPU/10,
				Title:    fmt.Sprintf("Look at %s in pane %s", top.Comm, pane),
				Evidence: fmt.Sprintf("%s; the biggest CPU user is %s (pid %d) at %.0f%%", a.Evidence, top.Comm, top.PID, top.CPU),
				Primary:  &Action{Label: "jump to its pane", Steps: one("tmux", "switch-client", "-t", pane)},
			})
		}
	}
	return out
}

func gitRecs(snap sample.Snapshot, git *gitscan.Report) []Rec {
	var out []Rec
	polling := map[string]float64{}
	for _, p := range snap.GitPolling {
		polling[p.Checkout] = p.Cores
	}
	if git == nil {
		for _, p := range snap.GitPolling {
			if p.Cores >= pollingCores {
				out = append(out, pollingRec(p.Checkout, p.Cores, -1, 0))
			}
		}
		return out
	}
	for _, repo := range git.Repos {
		var safe []gitscan.Worktree
		for i, wt := range repo.Worktrees {
			if wt.Safe {
				safe = append(safe, wt)
			}
			cores := polling[wt.Path]
			if cores >= pollingCores || (wt.Sessions >= 2 && wt.Dirty >= sharedDirty) {
				out = append(out, pollingRec(wt.Path, cores, wt.Dirty, wt.Sessions))
			}
			if i == 0 && wt.Branch == repo.Main && repo.Behind >= behindWorthPull && wt.Sessions > 0 {
				out = append(out, Rec{
					Level: "info", score: 200 + float64(repo.Behind),
					Title:    fmt.Sprintf("Update %s in %s", repo.Main, filepath.Base(repo.Path)),
					Evidence: fmt.Sprintf("%s is %d commits behind its remote and %d sessions work on it", repo.Main, repo.Behind, wt.Sessions),
					// It changes a checkout agents work in. The runner never lets
					// git ask for credentials and stops it after two minutes.
					Second: &Action{Label: "fast-forward", Destructive: true, Steps: one(
						"git", "-C", wt.Path, "pull", "--ff-only")},
				})
			}
		}
		if len(safe) > 0 {
			steps := make([][]string, len(safe))
			guards := make([]Guard, len(safe))
			for i, wt := range safe {
				steps[i], guards[i] = wt.Remove, Guard{Worktree: wt.Path}
			}
			out = append(out, Rec{
				Level: "info", score: 150 + float64(len(safe)),
				Title:    fmt.Sprintf("Remove %d merged worktrees of %s", len(safe), filepath.Base(repo.Path)),
				Evidence: fmt.Sprintf("their branches merged, they have no tracked or untracked changes and no process works inside (each is checked again right before its removal); ignored files such as build output go with them; branches and commits stay; %d worktrees in total", len(repo.Worktrees)),
				Second:   &Action{Label: "remove " + strconv.Itoa(len(safe)), Steps: steps, Destructive: true, Guards: guards},
			})
		}
	}
	return out
}

func pollingRec(checkout string, cores float64, dirty, sessions int) Rec {
	ev := fmt.Sprintf("agents spend about %.1f cores running git here", cores)
	if dirty >= 0 {
		ev += fmt.Sprintf("; %d dirty paths", dirty)
	}
	if sessions > 0 {
		ev += fmt.Sprintf("; %d sessions share it", sessions)
	}
	return Rec{
		Level: "warn", score: 400 + 100*cores + float64(dirty)/100,
		Title:    "Clean up " + checkout,
		Evidence: ev + ". Every agent's git status walks the whole dirty tree.",
		Primary:  &Action{Label: "open a shell there", Steps: one("tmux", "new-window", "-c", checkout)},
	}
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
