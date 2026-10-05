package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/NimbleMarkets/ntcharts/v2/sparkline"

	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

func (m Model) region(label string, w int) []string {
	return []string{"", m.st.label.Render(fit(" "+strings.ToUpper(label), w))}
}

// regionPath is a region label that starts with a path: paths keep their case.
func (m Model) regionPath(path, rest string, w int) []string {
	return []string{"", fit(" "+m.st.muted.Render(tilde(path))+m.st.label.Render(strings.ToUpper(rest)), w)}
}

var home, _ = os.UserHomeDir()

// tilde shortens the home directory to ~ for display.
func tilde(s string) string {
	if home == "" || home == "/" {
		return s
	}
	return strings.ReplaceAll(s, home, "~")
}

// alertLines puts what needs attention first; nothing renders when all is well.
func (m Model) alertLines(w int) []string {
	if len(m.snap.Alerts) == 0 {
		return nil
	}
	out := m.region("alerts", w)
	for _, a := range m.snap.Alerts {
		mark := m.st.warn.Render("▲ ")
		if a.Level == "crit" {
			mark = m.st.neg.Render("! ")
		}
		out = append(out, fit(" "+mark+m.st.text.Render(a.Title)+m.st.muted.Render("  "+a.Evidence), w))
	}
	return out
}

func (m Model) chart(xs []float64, w, h int, top float64) []string {
	sl := sparkline.New(w, h, sparkline.WithStyle(m.st.muted), sparkline.WithMaxValue(top))
	sl.PushAll(xs)
	sl.Draw()
	return strings.Split(sl.View(), "\n")
}

func (m Model) machine(w int) []string {
	mc, st, h := m.snap.Machine, m.st, m.history
	out := m.alertLines(w)

	out = append(out, m.region("cpu", w)...)
	out = append(out, fit(" "+st.text.Render(fmt.Sprintf("%3.0f%%", mc.CPU))+st.muted.Render(fmt.Sprintf(
		"   load %.1f / %.1f / %.1f   pressure cpu %.0f%%  io %.0f%%  memory %.0f%%",
		mc.Load[0], mc.Load[1], mc.Load[2], mc.PSI.CPU, mc.PSI.IO, mc.PSI.Memory)), w))
	if len(mc.Cores) > 0 {
		var b strings.Builder
		for _, c := range mc.Cores {
			b.WriteRune(blocks[min(int(c/100*float64(len(blocks)-1)+0.5), len(blocks)-1)])
		}
		out = append(out, fit(" "+st.label.Render("cores ")+st.muted.Render(b.String()), w))
	}
	for _, line := range m.chart(h.cpu, min(w-2, 120), 3, 100) {
		out = append(out, " "+line)
	}

	out = append(out, m.region("network and disk", w)...)
	netLine := fmt.Sprintf("↓ %s/s  ↑ %s/s   disk read %s/s  write %s/s", size(mc.NetRx), size(mc.NetTx), size(mc.DiskRead), size(mc.DiskWrite))
	switch {
	case mc.LatencyErr != "":
		netLine += "   " + st.neg.Render("! connect failed: "+mc.LatencyErr)
	case mc.Latency > 0:
		netLine += fmt.Sprintf("   connect %d ms", mc.Latency.Milliseconds())
	}
	out = append(out, fit(" "+st.text.Render(netLine), w))

	out = append(out, m.region("memory", w)...)
	out = append(out, fit(" "+st.text.Render(gib(mc.MemTotal-mc.MemAvail)+" used")+st.muted.Render(
		fmt.Sprintf(" of %s   %s available   swap %s", gib(mc.MemTotal), gib(mc.MemAvail), gib(mc.SwapUsed))), w))

	out = append(out, m.region("thermal and power", w)...)
	facts := []string{}
	if mc.TempC > 0 {
		facts = append(facts, fmt.Sprintf("package %.0f °C", mc.TempC))
	}
	for _, p := range mc.Power {
		if p.Watts > 0 {
			facts = append(facts, fmt.Sprintf("%s %.0f W", strings.ReplaceAll(p.Name, "_", " "), p.Watts))
		}
	}
	if len(facts) == 0 {
		facts = append(facts, "no package sensors readable")
	}
	out = append(out, fit(" "+st.text.Render(strings.Join(facts, st.faint.Render("  ·  "))), w))
	for _, g := range mc.GPUs {
		out = append(out, fit(" "+st.muted.Render(fmt.Sprintf("gpu %d  %3.0f%%  %.0f °C  %.0f/%.0f MiB", g.Index, g.Util, g.TempC, g.MemUsed, g.MemTot)), w))
	}

	if len(m.snap.Containers) > 0 {
		out = append(out, m.region(fmt.Sprintf("containers %d", len(m.snap.Containers)), w)...)
		for _, g := range m.snap.Groups {
			out = append(out, fit(" "+st.text.Render(fmt.Sprintf("%s  %.0f%% cpu", g.Name, g.CPU))+
				st.muted.Render(fmt.Sprintf("   %.1f%% of the machine · %d containers", g.Share, g.Containers)), w))
		}
		for _, c := range m.snap.Containers[:min(5, len(m.snap.Containers))] {
			out = append(out, fit(" "+st.muted.Render(pad(fmt.Sprintf("%3.0f%%", c.CPU), 7)+c.Name), w))
		}
	}
	return out
}

func (m Model) procs(w int) []string {
	st := m.st
	out := m.region("detached busy loops", w)
	if len(m.snap.Orphans) == 0 {
		out = append(out, st.faint.Render(" none"))
	}
	sel := m.orphanIndex()
	for i, o := range m.snap.Orphans {
		pane := "pane gone"
		if o.Pane == "" {
			pane = "no pane"
		} else if o.PaneAlive {
			pane = o.Pane + " " + o.Target
		} else {
			pane = o.Pane + " (gone)"
		}
		line := " " + st.neg.Render("! ") + st.text.Render(pad(fmt.Sprintf("%d %s", o.PID, o.Comm), 16)) +
			st.text.Render(pad(fmt.Sprintf("%3.0f%%", o.CPU), 7)) + st.muted.Render(pad("for "+sample.Human(o.HotFor), 12)) +
			st.muted.Render(pad(pane, 22)) + st.faint.Render(o.Cmdline)
		if i == sel {
			line = st.selected.Render(fit(line, w))
		}
		out = append(out, fit(line, w))
	}
	if o, ok := m.selectedOrphan(); ok {
		out = append(out, "", fit(" "+st.muted.Render(fmt.Sprintf("parent %s · age %s · cwd %s", o.Parent, sample.Human(o.Age), o.Cwd)), w))
		out = append(out, fit(" "+st.label.Render("x ")+st.muted.Render(strings.Join(o.Kill, " ")), w))
	}

	out = append(out, m.region("top processes", w)...)
	out = append(out, st.label.Render(fit(" "+pad("PID", 9)+pad("COMMAND", 18)+pad("CPU", 7)+pad("MEM", 8)+"OWNER", w)))
	names := map[string]string{}
	for _, s := range m.snap.Sessions {
		names["pane "+s.Pane] = s.Name
	}
	for _, p := range m.snap.Top {
		owner := p.Owner
		if n, ok := names[owner]; ok {
			owner += " · " + n
		}
		out = append(out, fit(" "+st.muted.Render(pad(fmt.Sprint(p.PID), 9))+st.text.Render(pad(p.Comm, 18))+
			st.text.Render(pad(fmt.Sprintf("%3.0f%%", p.CPU), 7))+st.muted.Render(pad(size(float64(p.RSS)), 8))+st.muted.Render(owner), w))
	}
	return out
}

func (m Model) orphanIndex() int {
	for i, o := range m.snap.Orphans {
		if o.PID == m.orphanPID {
			return i
		}
	}
	return 0
}

func (m Model) selectedOrphan() (sample.Orphan, bool) {
	if len(m.snap.Orphans) == 0 {
		return sample.Orphan{}, false
	}
	return m.snap.Orphans[m.orphanIndex()], true
}
