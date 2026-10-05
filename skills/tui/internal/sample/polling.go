package sample

import (
	"sort"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// GitPolling estimates the CPU agents spend running git in one checkout.
// Every process scan counts the git processes on a CPU in each checkout;
// the mean over the window approximates cores spent, short-lived calls
// included, without tracing anything.
type GitPolling struct {
	Checkout string  `json:"checkout"`
	Cores    float64 `json:"cores"`
	Samples  int     `json:"samples"`
}

const pollWindow = 10 * time.Minute

type pollPoint struct {
	at     time.Time
	counts map[string]int
}

func (s *Sampler) samplePolling(procs map[int]procfs.Proc, now time.Time) []GitPolling {
	counts := map[string]int{}
	for pid, p := range procs {
		if p.Comm != "git" || (p.State != 'R' && p.State != 'D') {
			continue
		}
		cwd := s.FS.Cwd(pid)
		top, ok := s.toplevel[cwd]
		if !ok {
			top = gitscan.Toplevel(cwd)
			s.toplevel[cwd] = top
		}
		if top != "" {
			counts[top]++
		}
	}
	s.polls = append(s.polls, pollPoint{now, counts})
	for len(s.polls) > 1 && now.Sub(s.polls[0].at) > pollWindow {
		s.polls = s.polls[1:]
	}
	if len(s.toplevel) > 4096 {
		s.toplevel = map[string]string{}
	}

	sum := map[string]int{}
	for _, pt := range s.polls {
		for top, n := range pt.counts {
			sum[top] += n
		}
	}
	var out []GitPolling
	for top, n := range sum {
		out = append(out, GitPolling{Checkout: top, Cores: float64(n) / float64(len(s.polls)), Samples: len(s.polls)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cores > out[j].Cores })
	return out
}
