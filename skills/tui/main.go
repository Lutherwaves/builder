// Command builder-tui is a terminal cockpit for a machine that runs many AI
// coding agents: sessions by tmux pane, machine load, git, and what to do.
//
//	builder-tui                 open the cockpit
//	builder-tui status          print what it knows as JSON (for agents)
//	builder-tui status --text   the same, as a short summary
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"github.com/Lutherwaves/builder/skills/tui/internal/config"
	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/history"
	"github.com/Lutherwaves/builder/skills/tui/internal/host"
	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
	"github.com/Lutherwaves/builder/skills/tui/internal/state"
	"github.com/Lutherwaves/builder/skills/tui/internal/ui"
)

var version = "dev"

func main() {
	// BUILDER_TUI_CPUPROFILE=file profiles the cockpit itself, to keep its
	// own cost honest.
	if path := os.Getenv("BUILDER_TUI_CPUPROFILE"); path != "" {
		if f, err := os.Create(path); err == nil {
			_ = pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}
	if err := run(os.Args[1:]); err != nil {
		pprof.StopCPUProfile()
		fmt.Fprintln(os.Stderr, "builder-tui:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "tui"
	if len(args) > 0 && (args[0] == "status" || args[0] == "version") {
		cmd, args = args[0], args[1:]
	}
	fl := flag.NewFlagSet("builder-tui", flag.ContinueOnError)
	cfgPath := fl.String("config", config.Path(), "machine goals (TOML); a missing file means defaults")
	interval := fl.Duration("interval", 0, "sampling interval (default from config, else 2s)")
	root := fl.String("fixtures", "", "read proc/, sys/, home/ and tmux-panes.txt from this directory instead of the live machine")
	text := fl.Bool("text", false, "status: print a summary instead of JSON")
	if err := fl.Parse(args); err != nil {
		return err
	}
	if cmd == "version" {
		fmt.Println(version)
		return nil
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("%s: %w", *cfgPath, err)
	}
	if *interval > 0 {
		cfg.Interval.Duration = *interval
	}

	// The cockpit must not add load: lowest CPU priority for us and every
	// command we start.
	lowestPriority()

	fixtures := *root != ""
	smp := newSampler(cfg, *root)
	var scanGit ui.GitSource
	if !fixtures {
		scanGit = gitSource(cfg)
	}
	if cmd == "status" {
		return status(smp, scanGit, cfg.Interval.Duration, fixtures, *text)
	}

	opt := ui.Options{
		Check:     checkGuard,
		Source:    smp.Sample,
		Interval:  cfg.Interval.Duration,
		Git:       scanGit,
		GitEvery:  cfg.Git.Interval.Duration,
		CompactAt: cfg.Sessions.CompactAt,
	}
	if !fixtures {
		logPath := cfg.History.Log
		if logPath == "" {
			logPath = history.DefaultLog()
		} else if logPath == "-" {
			logPath = ""
		}
		opt.OnSnapshot = recorder(state.Path(), logPath, cfg.Hooks.Alert)
		opt.History = func() history.Series {
			s, _ := history.Load(time.Now(), cfg.History.Source, logPath)
			return s
		}
	}
	// The screen changes on a sample or a key press, so 15 frames a second
	// is still instant to the eye and wakes the process 4x less than 60.
	_, err = tea.NewProgram(ui.New(opt), tea.WithFPS(15)).Run()
	return err
}

// recorder persists each sample: the state file for `status`, one history
// row every 30 s, and the alert hook for alerts that were not firing before.
func recorder(statePath, logPath string, hook []string) func(state.Doc) {
	logger := &history.Logger{Path: logPath}
	firing := map[string]bool{}
	first := true
	var wrote time.Time
	return func(d state.Doc) {
		// The file is for agents asking now and then; every few seconds is fresh.
		if d.At.Sub(wrote) >= stateEvery {
			_ = state.Write(statePath, d)
			wrote = d.At
		}
		if !first { // the first sample has no rates yet
			_ = logger.Log(history.FromSnapshot(d.Snapshot))
		}
		first = false
		now := map[string]bool{}
		for _, a := range d.Alerts {
			now[a.Key] = true
			if !firing[a.Key] && len(hook) > 0 {
				c := exec.Command(hook[0], append(hook[1:], a.Title, a.Evidence)...)
				if c.Start() == nil {
					go c.Wait()
				}
			}
		}
		firing = now
	}
}

func gitSource(cfg config.Config) ui.GitSource {
	scanner := &gitscan.Scanner{Git: gitscan.Git, Merged: gitscan.GHMerged, RecheckIdle: 2 * time.Hour}
	return func(sessionCwds []string) gitscan.Report {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		in := gitscan.Input{
			SessionCwds: sessionCwds,
			ProcessCwds: processCwds(),
			Extra:       cfg.Git.Repos,
			Main:        cfg.Git.Main,
			Remote:      cfg.Git.Remote,
		}
		return scanner.Scan(ctx, in)
	}
}

// checkGuard re-verifies, right before a step runs, what its evidence said.
func checkGuard(g advice.Guard) error {
	if g.PID > 0 {
		p, ok := newHost().Proc(g.PID)
		if !ok || p.StartTime != g.StartTicks {
			return fmt.Errorf("pid %d is no longer the process that was flagged", g.PID)
		}
	}
	if g.Worktree != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		return gitscan.VerifyRemovable(ctx, gitscan.Git, g.Worktree, processCwds())
	}
	return nil
}

// processCwds is where every process of ours sits, so a worktree someone
// still works in is never offered for removal.
func processCwds() []string {
	fs := newHost()
	procs, _ := fs.Procs()
	out := make([]string, 0, len(procs))
	for pid := range procs {
		if cwd := fs.Cwd(pid); cwd != "" {
			out = append(out, cwd)
		}
	}
	return out
}

// newHost reads /proc on Linux and the system's own APIs elsewhere.
func newHost() procfs.Host {
	if runtime.GOOS == "linux" {
		return procfs.FS{Root: "/proc"}
	}
	return host.Host{}
}

// stateEvery is how often the state file is rewritten; status treats a file
// older than staleAfter as no cockpit running.
const (
	stateEvery = 5 * time.Second
	staleAfter = 15 * time.Second
)

func newSampler(cfg config.Config, root string) *sample.Sampler {
	smp := &sample.Sampler{
		Cfg:    cfg,
		Agents: cfg.Agents,
		Rules: sample.Rules{
			CompactAt:   cfg.Sessions.CompactAt,
			ClearIdle:   cfg.Sessions.ClearIdle.Duration,
			ClearMinPct: cfg.Sessions.ClearMinPct,
			ClearStale:  cfg.Sessions.ClearStale.Duration,
			Windows:     cfg.Sessions.Windows,
		},
	}
	if root != "" {
		smp.FS = procfs.FS{Root: filepath.Join(root, "proc")}
		smp.Sys = procfs.Sys{Root: filepath.Join(root, "sys")}
		smp.Home = filepath.Join(root, "home")
		smp.Tmux = func() ([]byte, error) { return os.ReadFile(filepath.Join(root, "tmux-panes.txt")) }
		return smp
	}
	smp.FS, smp.Sys = newHost(), procfs.Sys{Root: "/sys"}
	smp.Home, _ = os.UserHomeDir()
	smp.GPU, smp.Docker = sample.NvidiaSMI, sample.DockerPS
	smp.LatencyTarget = cfg.LatencyTarget
	smp.ProcEvery = 5 * cfg.Interval.Duration
	return smp
}

// status prints the running cockpit's state when it is fresh. Otherwise it
// samples on its own: two samples a second apart so rates are real, and one
// git scan.
func status(smp *sample.Sampler, scanGit ui.GitSource, interval time.Duration, fixtures, text bool) error {
	doc, err := state.Read(state.Path())
	if fixtures || err != nil || time.Since(doc.At) > max(staleAfter, 3*interval) {
		smp.ProcEvery = 0
		smp.Sample()
		time.Sleep(time.Second)
		doc = state.Doc{Snapshot: smp.Sample()}
		if scanGit != nil {
			var cwds []string
			for _, s := range doc.Sessions {
				cwds = append(cwds, s.Cwd)
			}
			r := scanGit(cwds)
			doc.Git = &r
		}
		doc.Recommendations = advice.Build(doc.Snapshot, doc.Git)
	}
	if text {
		fmt.Print(ui.Summary(doc))
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
