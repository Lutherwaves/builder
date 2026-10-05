package ui

import (
	"fmt"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"strings"

	"github.com/Lutherwaves/builder/skills/tui/internal/state"
)

// Summary is the plain-text form for an agent or a log: the machine line,
// alerts, then the ranked recommendations with their exact commands.
func Summary(s state.Doc) string {
	var b strings.Builder
	mc := s.Machine
	fmt.Fprintf(&b, "at %s\n", s.At.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "machine: cpu %.0f%%  load %.1f  mem %s/%s  swap %s  psi cpu %.0f io %.0f mem %.0f\n",
		mc.CPU, mc.Load[0], gib(mc.MemTotal-mc.MemAvail), gib(mc.MemTotal), gib(mc.SwapUsed), mc.PSI.CPU, mc.PSI.IO, mc.PSI.Memory)
	busy := 0
	for _, x := range s.Sessions {
		if x.Busy {
			busy++
		}
	}
	fmt.Fprintf(&b, "sessions: %d (busy %d, idle %d), %d panes without an agent\n", len(s.Sessions), busy, len(s.Sessions)-busy, len(s.IdlePanes))
	for _, a := range s.Alerts {
		fmt.Fprintf(&b, "alert %s: %s (%s)\n", a.Level, a.Title, a.Evidence)
	}
	for _, r := range s.Recommendations {
		fmt.Fprintf(&b, "%s: %s\n  why: %s\n", r.Level, r.Title, r.Evidence)
		for _, a := range []*advice.Action{r.Primary, r.Second} {
			if a == nil {
				continue
			}
			confirm := ""
			if a.Destructive {
				confirm = " (destructive)"
			}
			for i, step := range a.Steps {
				if i == 3 {
					fmt.Fprintf(&b, "  … %d more like these (status JSON lists every step)\n", len(a.Steps)-3)
					break
				}
				fmt.Fprintf(&b, "  %s%s: %s\n", a.Label, confirm, strings.Join(step, " "))
			}
		}
	}
	for _, e := range s.Errors {
		fmt.Fprintf(&b, "error: %s\n", e)
	}
	return b.String()
}
