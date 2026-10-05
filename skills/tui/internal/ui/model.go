// Package ui is the Bubble Tea program. Sampling and git scans run off the
// update loop in commands; the model only renders their latest results.
package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/history"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
	"github.com/Lutherwaves/builder/skills/tui/internal/state"
)

// Source produces one snapshot per call. The sampler is never called again
// before the previous call returned, so it needs no locking.
type Source func() sample.Snapshot

// GitSource scans the repositories the given session directories belong to.
type GitSource func(sessionCwds []string) gitscan.Report

type Options struct {
	Source   Source
	Interval time.Duration
	Git      GitSource // nil: no git panel
	GitEvery time.Duration
	// CompactAt colours context at the compact threshold.
	CompactAt float64
	// OnSnapshot runs after each sample, off the update loop: the state file
	// for `status` and the history log.
	OnSnapshot func(state.Doc)
	// History loads past samples for the history panel. Nil: none.
	History func() history.Series
	// Run executes one confirmed step. Nil means exec.Command.
	Run func(argv []string) error
	// Check re-verifies a guard right before its step runs (same process,
	// worktree still clean and unused). Nil checks nothing.
	Check func(advice.Guard) error
}

type snapMsg sample.Snapshot
type gitMsg gitscan.Report
type historyMsg history.Series
type tickMsg struct{}
type gitTickMsg struct{}
type ranMsg struct {
	cmd string
	err error
}

// action is a command waiting for the user's confirm.
type action struct {
	label       string
	steps       [][]string
	destructive bool
	guards      []advice.Guard // guards[i] holds for steps[i]; may be empty
}

func (a action) String() string {
	s := strings.Join(a.steps[0], " ")
	if n := len(a.steps); n > 1 {
		s += fmt.Sprintf("  (+%d more)", n-1)
	}
	return s
}

func fromAdvice(a *advice.Action) *action {
	if a == nil || len(a.Steps) == 0 {
		return nil
	}
	return &action{label: a.Label, steps: a.Steps, destructive: a.Destructive, guards: a.Guards}
}

type Model struct {
	opt        Options
	st         styles
	width      int
	height     int
	snap       sample.Snapshot
	have       bool
	git        *gitscan.Report
	gitRunning bool
	recs       []advice.Rec
	hist       history.Series
	selPID     int
	orphanPID  int
	recSel     int
	gitSel     int
	tab        int
	pending    *action
	flash      string
	history    *series
	quitting   bool
}

func New(opt Options) Model {
	if opt.Interval <= 0 {
		opt.Interval = 2 * time.Second
	}
	if opt.GitEvery <= 0 {
		opt.GitEvery = 5 * time.Minute
	}
	if opt.CompactAt <= 0 {
		opt.CompactAt = sample.DefaultRules.CompactAt
	}
	if opt.Run == nil {
		opt.Run = func(argv []string) error { return exec.Command(argv[0], argv[1:]...).Run() }
	}
	return Model{opt: opt, st: newStyles(true), width: 100, height: 30, history: newSeries(60)}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestBackgroundColor, m.sampleCmd()}
	if m.opt.History != nil {
		load := m.opt.History
		cmds = append(cmds, func() tea.Msg { return historyMsg(load()) })
	}
	return tea.Batch(cmds...)
}

func (m Model) doc() state.Doc {
	return state.Doc{Snapshot: m.snap, Git: m.git, Recommendations: m.recs}
}

func (m Model) sampleCmd() tea.Cmd {
	src, on, git := m.opt.Source, m.opt.OnSnapshot, m.git
	return func() tea.Msg {
		s := src()
		if on != nil {
			on(state.Doc{Snapshot: s, Git: git, Recommendations: advice.Build(s, git)})
		}
		return snapMsg(s)
	}
}

func (m *Model) gitCmd() tea.Cmd {
	if m.opt.Git == nil || m.gitRunning {
		return nil
	}
	m.gitRunning = true
	var cwds []string
	for _, s := range m.snap.Sessions {
		cwds = append(cwds, s.Cwd)
	}
	scan := m.opt.Git
	return func() tea.Msg { return gitMsg(scan(cwds)) }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.st = newStyles(msg.IsDark())
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case snapMsg:
		first := !m.have
		m.snap, m.have = sample.Snapshot(msg), true
		m.history.push(m.snap.Machine)
		if !first { // the first sample has no rates yet
			m.hist.Append(history.FromSnapshot(m.snap))
		}
		m.keepSelection()
		m.recs = advice.Build(m.snap, m.git)
		cmds := []tea.Cmd{tea.Tick(m.opt.Interval, func(time.Time) tea.Msg { return tickMsg{} })}
		if first {
			cmds = append(cmds, m.gitCmd())
		}
		return m, tea.Batch(cmds...)
	case tickMsg:
		return m, m.sampleCmd()
	case gitMsg:
		r := gitscan.Report(msg)
		m.git, m.gitRunning = &r, false
		m.recs = advice.Build(m.snap, m.git)
		return m, tea.Tick(m.opt.GitEvery, func(time.Time) tea.Msg { return gitTickMsg{} })
	case gitTickMsg:
		return m, m.gitCmd()
	case historyMsg:
		loaded := history.Series(msg)
		loaded.Merge(m.hist)
		m.hist = loaded
	case ranMsg:
		if msg.err != nil {
			m.flash = "failed: " + msg.cmd + ": " + msg.err.Error()
		} else {
			m.flash = "ran: " + msg.cmd
		}
		// The running tick loop picks up the result; starting another sample
		// here would add a second loop and run two samples at once.
	case tea.KeyPressMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	if m.pending != nil {
		a := *m.pending
		m.pending = nil
		// A destructive action needs a typed y; Enter alone is not consent.
		if k == "y" || (k == "enter" && !a.destructive) {
			idle := map[string]bool{}
			for _, s := range m.snap.Sessions {
				idle[s.Pane] = s.Status == "idle"
			}
			run, check := m.opt.Run, m.opt.Check
			return m, func() tea.Msg { return runAction(a, run, check, idle) }
		}
		m.flash = "cancelled: " + a.String()
		return m, nil
	}
	m.flash = ""
	switch k {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "tab":
		m.tab = (m.tab + 1) % len(tabNames)
	case "shift+tab":
		m.tab = (m.tab + len(tabNames) - 1) % len(tabNames)
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "enter":
		m.pending = m.primary()
	case "x":
		m.pending = m.secondary()
	case "r":
		if m.tab == tabGit {
			if cmd := m.gitCmd(); cmd != nil {
				m.flash = "scanning git…"
				return m, cmd
			}
		}
	default:
		if len(k) == 1 && k[0] >= '1' && int(k[0]-'1') < len(tabNames) {
			m.tab = int(k[0] - '1')
		}
	}
	return m, nil
}

func jump(pane string) *action {
	return &action{label: "jump", steps: [][]string{{"tmux", "switch-client", "-t", pane}}}
}

// primary is what Enter does on the current panel: never destructive.
func (m Model) primary() *action {
	switch m.tab {
	case tabSessions:
		if s, ok := m.selected(); ok && s.Pane != "" {
			return jump(s.Pane)
		}
	case tabProcs:
		if o, ok := m.selectedOrphan(); ok && o.PaneAlive {
			return jump(o.Pane)
		}
	case tabGit:
		if wt, ok := m.selectedWorktree(); ok {
			return &action{label: "shell", steps: [][]string{{"tmux", "new-window", "-c", wt.Path}}}
		}
	case tabRecs:
		if r, ok := m.selectedRec(); ok {
			return fromAdvice(r.Primary)
		}
	}
	return nil
}

// secondary is what x does: the destructive action, behind a typed y.
func (m Model) secondary() *action {
	switch m.tab {
	case tabProcs:
		if o, ok := m.selectedOrphan(); ok {
			return &action{label: "kill", steps: [][]string{o.Kill}, destructive: true,
				guards: []advice.Guard{{PID: o.PID, StartTicks: o.Start}}}
		}
	case tabGit:
		if wt, ok := m.selectedWorktree(); ok && wt.Safe {
			return &action{label: "remove", steps: [][]string{wt.Remove}, destructive: true,
				guards: []advice.Guard{{Worktree: wt.Path}}}
		}
	case tabRecs:
		if r, ok := m.selectedRec(); ok {
			return fromAdvice(r.Second)
		}
	}
	return nil
}

func clampMove(i, d, n int) int { return min(max(i+d, 0), max(n-1, 0)) }

func (m *Model) move(d int) {
	switch m.tab {
	case tabProcs:
		if o := m.snap.Orphans; len(o) > 0 {
			m.orphanPID = o[clampMove(m.orphanIndex(), d, len(o))].PID
		}
	case tabGit:
		m.gitSel = clampMove(m.gitSel, d, len(m.worktreeRows()))
	case tabRecs:
		m.recSel = clampMove(m.recSel, d, len(m.recs))
	case tabSessions:
		if ss := m.snap.Sessions; len(ss) > 0 {
			m.selPID = ss[clampMove(m.selIndex(), d, len(ss))].PID
		}
	}
}

func (m Model) selIndex() int {
	for i, s := range m.snap.Sessions {
		if s.PID == m.selPID {
			return i
		}
	}
	return 0
}

func (m Model) selected() (sample.Session, bool) {
	if len(m.snap.Sessions) == 0 {
		return sample.Session{}, false
	}
	return m.snap.Sessions[m.selIndex()], true
}

func (m Model) selectedRec() (advice.Rec, bool) {
	if len(m.recs) == 0 {
		return advice.Rec{}, false
	}
	return m.recs[min(m.recSel, len(m.recs)-1)], true
}

// keepSelection follows the selected session by PID across re-sorts.
func (m *Model) keepSelection() {
	if _, ok := m.selected(); ok {
		m.selPID = m.snap.Sessions[m.selIndex()].PID
	}
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// runAction runs each step whose guard still holds, skipping the others.
func runAction(a action, run func([]string) error, check func(advice.Guard) error, idle map[string]bool) ranMsg {
	ran, skipped, why := 0, 0, ""
	for i, step := range a.steps {
		if i < len(a.guards) {
			g := a.guards[i]
			err := error(nil)
			if g.IdlePane != "" && !idle[g.IdlePane] {
				err = fmt.Errorf("the session in %s is no longer idle", g.IdlePane)
			} else if check != nil {
				err = check(g)
			}
			if err != nil {
				skipped++
				why = err.Error()
				continue
			}
		}
		if err := run(step); err != nil {
			return ranMsg{strings.Join(step, " "), err}
		}
		ran++
	}
	if skipped > 0 {
		return ranMsg{a.String(), fmt.Errorf("ran %d, skipped %d: %s", ran, skipped, why)}
	}
	return ranMsg{a.String(), nil}
}
