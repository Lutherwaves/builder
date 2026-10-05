package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/history"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

type wtRow struct {
	repo string
	wt   gitscan.Worktree
	main bool
}

// worktreeRows are the worktrees worth a line: the main checkout, the ones
// sessions work in, and merged ones. The rest are counted, not listed.
func (m Model) worktreeRows() []wtRow {
	if m.git == nil {
		return nil
	}
	var rows []wtRow
	for _, r := range m.git.Repos {
		for i, wt := range r.Worktrees {
			if i == 0 || wt.Sessions > 0 || wt.Merged {
				rows = append(rows, wtRow{r.Path, wt, i == 0})
			}
		}
	}
	return rows
}

func (m Model) selectedWorktree() (gitscan.Worktree, bool) {
	rows := m.worktreeRows()
	if len(rows) == 0 {
		return gitscan.Worktree{}, false
	}
	return rows[min(m.gitSel, len(rows)-1)].wt, true
}

func (m Model) gitPanel(w int) []string {
	st := m.st
	var out []string
	if len(m.snap.GitPolling) > 0 {
		out = append(out, m.region("git run by agents, last 10 min", w)...)
		for _, p := range m.snap.GitPolling[:min(5, len(m.snap.GitPolling))] {
			style := st.muted
			if p.Cores >= 0.5 {
				style = st.warn
			}
			out = append(out, fit(" "+style.Render(pad(fmt.Sprintf("%.2f cores", p.Cores), 12))+st.muted.Render(tilde(p.Checkout)), w))
		}
	}
	if m.git == nil {
		return append(out, "", st.faint.Render(" scanning repositories…"))
	}
	rows := m.worktreeRows()
	sel := min(m.gitSel, max(len(rows)-1, 0))
	i := 0
	for _, r := range m.git.Repos {
		safe := 0
		for _, wt := range r.Worktrees {
			if wt.Safe {
				safe++
			}
		}
		rest := fmt.Sprintf("  ·  %d worktrees", len(r.Worktrees))
		if safe > 0 {
			rest += fmt.Sprintf("  ·  %d safe to remove", safe)
		}
		out = append(out, m.regionPath(r.Path, rest, w)...)
		if r.Behind > 0 {
			out = append(out, fit(" "+st.warn.Render("▲ ")+st.text.Render(fmt.Sprintf("%s is %d commits behind its remote", r.Main, r.Behind)), w))
		}
		out = append(out, st.label.Render(fit(" "+pad("WORKTREE", 36)+pad("BRANCH", 28)+pad("AGENTS", 8)+pad("DIRTY", 8)+"STATE", w)))
		for ; i < len(rows) && rows[i].repo == r.Path; i++ {
			wt := rows[i].wt
			name := filepath.Base(wt.Path)
			if rows[i].main {
				name += " (main)"
			}
			dirty := st.faint.Render(pad("—", 8))
			if wt.Dirty >= 0 {
				dirty = st.muted.Render(pad(fmt.Sprint(wt.Dirty), 8))
				if wt.Dirty >= 200 {
					dirty = st.warn.Render(pad(fmt.Sprint(wt.Dirty), 8))
				}
			}
			state := ""
			switch {
			case wt.Safe:
				state = st.accent.Render("✓ safe to remove")
			case wt.Merged && wt.InUse:
				state = st.muted.Render("merged, in use")
			case wt.Merged && wt.Dirty != 0:
				state = st.muted.Render("merged, has changes")
			}
			agents := ""
			if wt.Sessions > 0 {
				agents = fmt.Sprint(wt.Sessions)
			}
			line := " " + st.text.Render(pad(name, 36)) + st.muted.Render(pad(wt.Branch, 28)) + st.text.Render(pad(agents, 8)) + dirty + state
			if i == sel {
				line = st.selected.Render(fit(line, w))
			}
			out = append(out, fit(line, w))
		}
	}
	for _, e := range m.git.Errors {
		out = append(out, fit(" "+st.neg.Render("! ")+st.muted.Render(e), w))
	}
	out = append(out, "", fit(" "+st.faint.Render(fmt.Sprintf("scanned %s ago in %s · r rescans",
		sample.Human(time.Since(m.git.At)), m.git.Took.Round(time.Millisecond))), w))
	if wt, ok := m.selectedWorktree(); ok {
		out = append(out, fit(" "+st.label.Render("⏎ ")+st.muted.Render("tmux new-window -c "+wt.Path), w))
		if wt.Safe {
			out = append(out, fit(" "+st.label.Render("x ")+st.muted.Render(strings.Join(wt.Remove, " ")), w))
		}
	}
	return out
}

func (m Model) recsPanel(w int) []string {
	st := m.st
	out := m.alertLines(w)
	out = append(out, m.region(fmt.Sprintf("%d recommendations", len(m.recs)), w)...)
	if len(m.recs) == 0 {
		return append(out, st.faint.Render(" nothing to do"))
	}
	sel := min(m.recSel, len(m.recs)-1)
	for i, r := range m.recs {
		mark := st.faint.Render("○ ")
		switch r.Level {
		case "crit":
			mark = st.neg.Render("! ")
		case "warn":
			mark = st.warn.Render("▲ ")
		}
		line := " " + mark + st.text.Render(tilde(r.Title))
		if i == sel {
			line = st.selected.Render(fit(line, w))
		}
		out = append(out, fit(line, w))
	}
	r := m.recs[sel]
	out = append(out, "", st.hair.Render(strings.Repeat("─", w)), fit(" "+st.text.Render(tilde(r.Title)), w))
	out = append(out, wrap(" "+tilde(r.Evidence), w, st.muted.Render)...)
	for _, a := range []struct {
		key string
		act *advice.Action
	}{{"⏎", r.Primary}, {"x", r.Second}} {
		if a.act == nil {
			continue
		}
		cmd := strings.Join(a.act.Steps[0], " ")
		if n := len(a.act.Steps); n > 1 {
			cmd += fmt.Sprintf("  (+%d more)", n-1)
		}
		label := a.act.Label
		if a.act.Destructive {
			label += ", asks y"
		}
		out = append(out, fit(" "+st.label.Render(a.key+" ")+st.text.Render(label+": ")+st.muted.Render(cmd), w))
	}
	return out
}

func wrap(s string, w int, render func(...string) string) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		if len(line)+len(word)+1 > w-2 && line != "" {
			out = append(out, render(" "+line))
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		out = append(out, render(" "+line))
	}
	return out
}

// bucket reduces points to n values keeping each bucket's peak: a spike
// matters more than an average on a history chart.
func bucket(pts []history.Point, n int, f func(history.Point) float64) []float64 {
	if len(pts) == 0 || n <= 0 {
		return nil
	}
	out := make([]float64, 0, n)
	per := max((len(pts)+n-1)/n, 1) // ceil: never more columns than n
	for i := 0; i < len(pts); i += per {
		peak := 0.0
		for _, p := range pts[i:min(i+per, len(pts))] {
			peak = max(peak, f(p))
		}
		out = append(out, peak)
	}
	return out
}

func (m Model) historyPanel(w int) []string {
	st := m.st
	pts := m.hist.Points
	if len(pts) < 2 {
		return []string{"", st.faint.Render(" collecting history: one point every 30 s")}
	}
	span := pts[len(pts)-1].At.Sub(pts[0].At)
	out := m.region(fmt.Sprintf("last %s · peak per column", sample.Human(span)), w)
	cw := min(w-28, 140)
	charts := []struct {
		name string
		unit string
		f    func(history.Point) float64
	}{
		{"cpu", "%", func(p history.Point) float64 { return p.CPU }},
		{"load1", "", func(p history.Point) float64 { return p.Load1 }},
		{"pressure cpu", "%", func(p history.Point) float64 { return p.PSICPU }},
		{"package temp", "°C", func(p history.Point) float64 { return p.TempC }},
		{"mem available", "G", func(p history.Point) float64 { return p.MemAvailGB }},
		{"swap", "G", func(p history.Point) float64 { return p.SwapGB }},
	}
	for _, c := range charts {
		xs := bucket(pts, cw, c.f)
		lo, hi := c.f(pts[0]), c.f(pts[0])
		for _, p := range pts {
			lo, hi = min(lo, c.f(p)), max(hi, c.f(p))
		}
		now := c.f(pts[len(pts)-1])
		// Scale min to max so two rows show the shape; the labels carry the
		// absolute values. The floor sits just under min so min still draws.
		floor := lo - (hi-lo)*0.05
		shifted := make([]float64, len(xs))
		for i, x := range xs {
			shifted[i] = x - floor
		}
		lines := m.chart(shifted, len(shifted), 2, hi-floor)
		out = append(out, "")
		label := []string{
			st.label.Render(pad(strings.ToUpper(c.name), 24)),
			st.text.Render(pad(fmt.Sprintf("now %.1f%s", now, c.unit), 24)),
		}
		for i, l := range lines {
			out = append(out, fit(" "+label[min(i, 1)]+l, w))
		}
		out = append(out, fit(" "+pad("", 24)+st.faint.Render(fmt.Sprintf("min %.1f%s  max %.1f%s", lo, c.unit, hi, c.unit)), w))
	}
	return out
}
