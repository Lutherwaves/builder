package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

// series keeps the last n machine samples for the header sparklines.
type series struct {
	n                            int
	cpu, mem, net, io, lat, load []float64
}

func newSeries(n int) *series { return &series{n: n} }

func (s *series) push(m sample.Machine) {
	add := func(xs *[]float64, v float64) {
		*xs = append(*xs, v)
		if len(*xs) > s.n {
			*xs = (*xs)[len(*xs)-s.n:]
		}
	}
	add(&s.cpu, m.CPU)
	if m.MemTotal > 0 {
		add(&s.mem, float64(m.MemTotal-m.MemAvail))
	}
	add(&s.net, m.NetRx+m.NetTx)
	add(&s.io, m.DiskRead+m.DiskWrite)
	add(&s.lat, float64(m.Latency))
	add(&s.load, m.Load[0])
}

var blocks = []rune("▁▂▃▄▅▆▇█")

// spark draws the last w values scaled to their own max (or ceil if larger).
func spark(xs []float64, w int, ceil float64) string {
	if len(xs) > w {
		xs = xs[len(xs)-w:]
	}
	top := ceil
	for _, x := range xs {
		top = max(top, x)
	}
	var b strings.Builder
	for _, x := range xs {
		i := 0
		if top > 0 {
			i = min(int(x/top*float64(len(blocks)-1)+0.5), len(blocks)-1)
		}
		b.WriteRune(blocks[i])
	}
	return b.String()
}

func (m Model) render() string {
	if m.quitting {
		return ""
	}
	w := max(m.width, 60)
	var out []string
	out = append(out, m.header(w), m.tabs(w), m.st.hair.Render(strings.Repeat("─", w)))
	if !m.have {
		return strings.Join(append(out, "", m.st.faint.Render(" sampling…")), "\n")
	}
	var body []string
	switch m.tab {
	case tabSessions:
		body = m.sessions(w, m.height-len(out)-2)
	case tabMachine:
		body = m.machine(w)
	case tabProcs:
		body = m.procs(w)
	case tabGit:
		body = m.gitPanel(w)
	case tabRecs:
		body = m.recsPanel(w)
	case tabHistory:
		body = m.historyPanel(w)
	}
	if len(body) > m.height-len(out)-1 {
		body = body[:max(m.height-len(out)-1, 0)]
	}
	out = append(out, body...)
	for len(out) < m.height-1 {
		out = append(out, "")
	}
	out = append(out, m.footer(w))
	return strings.Join(out, "\n")
}

// header shows every metric with its trend when the terminal is wide, drops
// the trends when it is not, then drops metrics from the right.
func (m Model) header(w int) string {
	mc, h, st := m.snap.Machine, m.history, m.st
	sep := st.hair.Render("  │  ")
	type metric struct{ label, value, trend string }
	ms := []metric{
		{"cpu", fmt.Sprintf("%3.0f%%", mc.CPU), spark(h.cpu, 8, 100)},
		{"mem", fmt.Sprintf("%s/%s", gib(mc.MemTotal-mc.MemAvail), gib(mc.MemTotal)), spark(h.mem, 8, float64(mc.MemTotal))},
		{"load", fmt.Sprintf("%.1f", mc.Load[0]), spark(h.load, 8, 0)},
		{"net", "↓" + size(mc.NetRx) + " ↑" + size(mc.NetTx), spark(h.net, 8, 0)},
		{"io", "r " + size(mc.DiskRead) + " w " + size(mc.DiskWrite), spark(h.io, 8, 0)},
	}
	if mc.LatencyErr != "" {
		ms = append(ms, metric{"lat", st.neg.Render("! down"), ""})
	} else if mc.Latency > 0 {
		ms = append(ms, metric{"lat", fmt.Sprintf("%dms", mc.Latency.Milliseconds()), spark(h.lat, 8, 0)})
	}
	build := func(n int, trends bool) string {
		parts := []string{st.accent.Render("builder")}
		for _, x := range ms[:n] {
			p := st.label.Render(x.label) + " " + st.text.Render(x.value)
			if trends && x.trend != "" {
				p += " " + st.faint.Render(x.trend)
			}
			parts = append(parts, p)
		}
		return " " + strings.Join(parts, sep)
	}
	for _, trends := range []bool{true, false} {
		for n := len(ms); n > 0; n-- {
			if line := build(n, trends); lipgloss.Width(line) <= w {
				return fit(line, w)
			}
		}
	}
	return fit(build(1, false), w)
}

const (
	tabSessions = iota
	tabMachine
	tabProcs
	tabGit
	tabRecs
	tabHistory
)

var tabNames = []string{"sessions", "machine", "procs", "git", "recs", "history"}

func (m Model) tabs(w int) string {
	var parts []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, strings.ToUpper(name))
		switch {
		case i == m.tab:
			parts = append(parts, m.st.tabActive.Render(label))
		case i == tabProcs && len(m.snap.Orphans) > 0:
			parts = append(parts, m.st.neg.Render(label+" !"))
		case i == tabRecs && len(m.recs) > 0:
			parts = append(parts, m.st.muted.Render(fmt.Sprintf("%s %d", label, len(m.recs))))
		default:
			parts = append(parts, m.st.faint.Render(label))
		}
	}
	line := " " + strings.Join(parts, "   ")
	if n := len(m.snap.Alerts); n > 0 {
		alert := m.st.warn.Render(fmt.Sprintf("▲ %d alert", n))
		if n > 1 {
			alert += m.st.warn.Render("s")
		}
		line += strings.Repeat(" ", max(w-lipgloss.Width(line)-lipgloss.Width(alert)-1, 2)) + alert
	}
	return fit(line, w)
}

func (m Model) footer(w int) string {
	st := m.st
	if m.pending != nil {
		confirm := "⏎ run  esc cancel"
		if m.pending.destructive {
			confirm = st.neg.Render("y") + st.muted.Render(" run  any other key cancels")
		}
		return fit(" "+st.label.Render("RUN ")+st.text.Render(m.pending.String())+"   "+st.muted.Render(confirm), w)
	}
	keys := map[int]string{
		tabSessions: "↑↓ select  ⏎ jump",
		tabMachine:  "",
		tabProcs:    "↑↓ select  ⏎ jump  x kill",
		tabGit:      "↑↓ select  ⏎ shell  x remove  r rescan",
		tabRecs:     "↑↓ select  ⏎ do  x the other action",
		tabHistory:  "",
	}[m.tab] + "  1-6 panel  q quit"
	left := " " + st.muted.Render(keys)
	if m.flash != "" {
		left = " " + st.text.Render(m.flash)
	}
	right := st.faint.Render("sampled " + sample.Human(time.Since(m.snap.At)) + " ago ")
	return left + strings.Repeat(" ", max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
}

const (
	colPane = 10 // session:window.pane; widened to show the tab name when there is room
	colTab  = 12
	colName = 18
	colSt   = 6
	colIdle = 7
	colBar  = 10
	colPct  = 9
	colCPU  = 6
	colDo   = 10
)

func (m Model) sessions(w, h int) []string {
	st := m.st
	ss := m.snap.Sessions
	busy := 0
	for _, s := range ss {
		if s.Busy {
			busy++
		}
	}
	region := fmt.Sprintf(" %d AGENTS  ·  BUSY %d  ·  IDLE %d", len(ss), busy, len(ss)-busy)
	lines := []string{"", st.label.Render(region)}

	tab := 0
	if w >= 100 {
		tab = colTab
	}
	tree := min(max(w-(1+colPane+tab+colName+colSt+colIdle+colBar+1+colPct+colCPU+colDo+1), 12), 48)
	head := " " + pad("TMUX", colPane+tab) + pad("NAME", colName) + pad("STATE", colSt) + pad("IDLE", colIdle) +
		pad("CONTEXT", colBar+1+colPct) + pad("CPU", colCPU) + pad("DO", colDo+1) + "WORKTREE"
	lines = append(lines, st.label.Render(fit(head, w)))

	detail := m.detail(w)
	room := max(h-len(lines)-len(detail)-2, 3)
	sel := m.selIndex()
	first := max(min(sel-room/2, len(ss)-room), 0)
	for i := first; i < len(ss) && i < first+room; i++ {
		lines = append(lines, m.row(ss[i], i == sel, w, tab, tree))
	}
	if n := len(m.snap.IdlePanes); n > 0 {
		var ids []string
		for _, p := range m.snap.IdlePanes {
			ids = append(ids, place(p.Target, p.Pane)+" "+p.Command)
		}
		lines = append(lines, st.faint.Render(fit(fmt.Sprintf(" %d without an agent: %s", n, strings.Join(ids, ", ")), w)))
	}
	lines = append(lines, "", st.hair.Render(strings.Repeat("─", w)))
	return append(lines, detail...)
}

func (m Model) row(s sample.Session, selected bool, w, tab, tree int) string {
	st := m.st
	state, idle := st.text.Render(pad("busy", colSt)), pad("", colIdle)
	if !s.Busy {
		state = st.faint.Render(pad("idle", colSt))
		idle = st.muted.Render(pad(sample.Human(s.Idle), colIdle))
	}
	ctx := st.faint.Render(pad(s.Context, colBar+1+colPct))
	if s.Context == "known" {
		pctStyle := st.text
		switch {
		case s.ContextPct >= m.compactAt():
			pctStyle = st.neg
		case s.ContextPct >= 70:
			pctStyle = st.warn
		}
		win := ""
		if s.Window >= 1_000_000 {
			win = " 1M"
		}
		ctx = st.muted.Render(bar(s.ContextPct, colBar)) + " " + pctStyle.Render(pad(fmt.Sprintf("%3.0f%%%s", s.ContextPct, win), colPct))
	}
	cpu := st.faint.Render(pad(fmt.Sprintf("%3.0f%%", s.CPU), colCPU))
	if s.CPU >= 5 {
		cpu = st.text.Render(pad(fmt.Sprintf("%3.0f%%", s.CPU), colCPU))
	}
	do := pad("", colDo+1)
	switch s.Do {
	case "compact":
		do = st.neg.Render(pad("! compact", colDo+1))
	case "clear":
		do = st.warn.Render(pad("▲ clear", colDo+1))
	}
	name := st.text.Render(pad(s.Name, colName))
	where := place(s.Target, s.Pane)
	if selected {
		where = "▌" + where
	}
	pane := st.muted.Render(pad(where, colPane))
	if selected {
		pane = st.accent.Render(pad(where, colPane))
	}
	if tab > 0 {
		pane += st.muted.Render(pad(s.Tab, tab))
	}
	line := " " + pane + name + state + idle + ctx + cpu + do + st.muted.Render(pad(worktree(s.Cwd), tree))
	if selected {
		return st.selected.Render(fit(line, w))
	}
	return fit(line, w)
}

func (m Model) compactAt() float64 { return m.opt.CompactAt }

func (m Model) detail(w int) []string {
	st := m.st
	s, ok := m.selected()
	if !ok {
		return []string{st.faint.Render(" no agent sessions found in tmux")}
	}
	lines := []string{st.label.Render(" " + strings.ToUpper(place(s.Target, s.Pane)+" "+s.Tab+"  "+s.Name))}
	facts := []string{fmt.Sprintf("pid %d", s.PID)}
	if s.Pane != "" {
		facts = append(facts, "pane "+s.Pane)
	}
	if !s.Started.IsZero() {
		facts = append(facts, "started "+sample.Human(time.Since(s.Started))+" ago")
	}
	if s.Model != "" {
		facts = append(facts, s.Model)
	}
	if s.Tokens > 0 {
		facts = append(facts, fmt.Sprintf("%s / %s tokens", kTok(s.Tokens), kTok(s.Window)), "+"+kTok(s.Burn30m)+" in 30 min")
	}
	facts = append(facts, fmt.Sprintf("%d processes", s.Procs))
	lines = append(lines, fit(" "+st.text.Render(strings.Join(facts, st.faint.Render("  ·  "))), w))
	lines = append(lines, fit(" "+st.muted.Render(s.Cwd), w))
	if s.Why != "" {
		lines = append(lines, fit(" "+st.label.Render("WHY ")+st.text.Render(s.Do+": "+s.Why), w))
	}
	if s.Pane != "" {
		lines = append(lines, fit(" "+st.label.Render("⏎ ")+st.muted.Render("tmux switch-client -t "+s.Pane), w))
	}
	return lines
}

func bar(pct float64, w int) string {
	full := min(int(pct/100*float64(w)+0.5), w)
	return strings.Repeat("█", full) + strings.Repeat("░", w-full)
}

// worktree shortens a cwd to what tells sessions apart: the checkout name,
// or the worktree name for paths under a */worktrees/* directory.
func worktree(cwd string) string {
	if cwd == "" {
		return "?"
	}
	dir, base := filepath.Split(filepath.Clean(cwd))
	if strings.Contains(filepath.Base(dir), "worktrees") {
		return "wt/" + base
	}
	return base
}

// place names a pane the way the tmux status bar does (session:window.pane),
// falling back to its id when tmux did not list it.
func place(target, id string) string {
	if target != "" {
		return target
	}
	return id
}

func pad(s string, w int) string {
	s = ansi.Truncate(s, w-1, "…")
	return s + strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
}

func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	return s + strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
}

func gib(b uint64) string { return fmt.Sprintf("%.0fG", float64(b)/(1<<30)) }

func kTok(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	return fmt.Sprintf("%dk", (n+500)/1000)
}

// size renders bytes or bytes per second compactly: 512B, 12K, 3.4M, 1.2G.
func size(b float64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fG", b/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0fK", b/(1<<10))
	}
	return fmt.Sprintf("%.0fB", b)
}
