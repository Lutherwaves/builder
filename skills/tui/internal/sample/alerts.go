package sample

import (
	"fmt"
	"math"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

type Alert struct {
	Key      string    `json:"key"`
	Level    string    `json:"level"` // warn | crit
	Title    string    `json:"title"`
	Evidence string    `json:"evidence"`
	Since    time.Time `json:"since"`
}

// alerts evaluates the configured thresholds. A threshold with a "for"
// duration fires only once its condition has held that long.
func (s *Sampler) alerts(snap *Snapshot, now time.Time) {
	a, m := s.Cfg.Alerts, snap.Machine
	held := map[string]time.Time{}
	add := func(key, level, title, evidence string, cond bool, wait time.Duration) {
		if !cond {
			return
		}
		since, ok := s.condSince[key]
		if !ok {
			since = now
		}
		held[key] = since
		if now.Sub(since) >= wait {
			snap.Alerts = append(snap.Alerts, Alert{key, level, title, evidence, since})
		}
	}
	const gb = 1 << 30

	add("temp", "crit", "CPU package hot",
		fmt.Sprintf("package %.0f °C, limit %.0f °C", m.TempC, a.TempOverC),
		a.TempOverC > 0 && m.TempC > a.TempOverC, 0)
	add("load1", "warn", "Load stays high",
		fmt.Sprintf("load1 %.1f over %.0f for %s", m.Load[0], a.Load1Over, Human(a.Load1For.Duration)),
		a.Load1Over > 0 && m.Load[0] > a.Load1Over, a.Load1For.Duration)
	add("psi-cpu", "warn", "CPU pressure stays high",
		fmt.Sprintf("PSI cpu some avg10 %.0f%% over %.0f%% for %s", m.PSI.CPU, a.PSICPUOver, Human(a.PSICPUFor.Duration)),
		a.PSICPUOver > 0 && m.PSI.CPU > a.PSICPUOver, a.PSICPUFor.Duration)
	avail, swap := float64(m.MemAvail)/gb, float64(m.SwapUsed)/gb
	add("mem-low", "crit", "Memory almost gone",
		fmt.Sprintf("available %.1f GB, under %.0f GB", avail, a.MemAvailableUnderGB),
		a.MemAvailableUnderGB > 0 && m.MemTotal > 0 && avail < a.MemAvailableUnderGB, 0)
	add("swap-stuck", "warn", "Swap held while RAM is free",
		fmt.Sprintf("swap %.1f GB over %.0f GB while %.1f GB is available", swap, a.SwapOverGB, avail),
		a.SwapOverGB > 0 && swap > a.SwapOverGB && avail > a.SwapWhenAvailableOverGB, 0)

	for _, d := range s.powerDrift(m.Power) {
		add("power-"+d.key, "crit", "CPU power limit changed", d.evidence, true, 0)
	}
	s.condSince = held
}

type drift struct{ key, evidence string }

// powerDrift compares each limit with the configured value (long term, short
// term, by constraint order) or, when none is configured, with its value at
// the first sample.
func (s *Sampler) powerDrift(limits []procfs.PowerLimit) []drift {
	if s.powerBase == nil {
		s.powerBase = map[string]float64{}
	}
	for _, l := range limits {
		// A zone first seen now (a driver loaded late) starts its baseline now.
		if _, ok := s.powerBase[l.Zone+"/"+l.Name]; !ok {
			s.powerBase[l.Zone+"/"+l.Name] = l.Watts
		}
	}
	var out []drift
	idx := map[string]int{}
	for _, l := range limits {
		key := l.Zone + "/" + l.Name
		i := idx[l.Zone]
		idx[l.Zone]++
		want, from := s.powerBase[key], "at startup"
		if exp := s.Cfg.Alerts.PowerLimitsW; i < len(exp) {
			want, from = exp[i], "expected"
		}
		if math.Abs(l.Watts-want) >= 1 {
			out = append(out, drift{key, fmt.Sprintf("%s %s is %.0f W, %s %.0f W", l.Zone, l.Name, l.Watts, from, want)})
		}
	}
	return out
}
