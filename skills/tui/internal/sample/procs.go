package sample

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// ProcRow is one process with the owner its CPU is attributed to.
type ProcRow struct {
	PID   int     `json:"pid"`
	Comm  string  `json:"comm"`
	CPU   float64 `json:"cpu_pct"` // 100 = one core
	RSS   uint64  `json:"rss_bytes"`
	Owner string  `json:"owner"` // "pane %3", "container ci-runner-1", "system"
}

// Orphan is a process that outlived whoever started it and keeps burning CPU.
type Orphan struct {
	PID       int           `json:"pid"`
	Start     uint64        `json:"start_ticks"` // /proc/<pid>/stat field 22: tells a reused pid apart
	Comm      string        `json:"comm"`
	Cmdline   string        `json:"cmdline"`
	Cwd       string        `json:"cwd"`
	CPU       float64       `json:"cpu_pct"`
	Age       time.Duration `json:"age_ns"`
	HotFor    time.Duration `json:"hot_for_ns"`
	Parent    string        `json:"parent"` // init | user service manager
	Pane      string        `json:"pane,omitempty"`
	PaneAlive bool          `json:"pane_alive"`
	Target    string        `json:"target,omitempty"`
	Kill      []string      `json:"kill"`
}

const topN = 15

func (s *Sampler) procCPU(p procfs.Proc, dt float64) float64 {
	prev, ok := s.prevProc[p.PID]
	if !ok || dt == 0 || prev.StartTime != p.StartTime || p.Ticks < prev.Ticks {
		return 0
	}
	return 100 * float64(p.Ticks-prev.Ticks) / procfs.ClockTicks / dt
}

func (s *Sampler) sampleProcs(snap *Snapshot, procs map[int]procfs.Proc, panes []Pane, dt float64, now time.Time) {
	cpu := make(map[int]float64, len(procs))
	for pid, p := range procs {
		cpu[pid] = s.procCPU(p, dt)
	}

	pids := make([]int, 0, len(procs))
	for pid := range procs {
		pids = append(pids, pid)
	}
	sort.Slice(pids, func(i, j int) bool { return cpu[pids[i]] > cpu[pids[j]] })
	for _, pid := range pids[:min(topN, len(pids))] {
		p := procs[pid]
		snap.Top = append(snap.Top, ProcRow{
			PID: pid, Comm: p.Comm, CPU: cpu[pid], RSS: uint64(p.RSSPages) * uint64(os.Getpagesize()),
			Owner: s.owner(p),
		})
	}

	paneByID := map[string]Pane{}
	for _, p := range panes {
		paneByID[p.ID] = p
	}
	s.orphans(snap, procs, cpu, paneByID, now)
}

// owner names what a process's CPU belongs to: its tmux pane, its container,
// or the system. Cached per process: it needs environ and cgroup reads.
func (s *Sampler) owner(p procfs.Proc) string {
	key := procKey{p.PID, p.StartTime}
	if o, ok := s.owners[key]; ok {
		return o
	}
	o := "system"
	if pane, ok := s.FS.Environ(p.PID, "TMUX_PANE"); ok {
		o = "pane " + pane
	} else if cg := s.FS.Cgroup(p.PID); strings.Contains(cg, "docker") {
		o = "container"
		for _, c := range s.containers {
			if strings.Contains(cg, c.id) {
				o = "container " + c.name
				break
			}
		}
	}
	s.owners[key] = o
	return o
}

var uid = os.Getuid()

// orphans flags processes that (1) were re-parented to init or to a user
// service manager, (2) belong to us, (3) match the watched commands, and
// (4) have burned at least CPUOver for the configured time.
func (s *Sampler) orphans(snap *Snapshot, procs map[int]procfs.Proc, cpu map[int]float64, panes map[string]Pane, now time.Time) {
	cfg := s.Cfg.Orphans
	uptime := s.FS.Uptime()
	hot := map[procKey]time.Time{}
	for pid, p := range procs {
		if cpu[pid] < cfg.CPUOver || !s.watched(p.Comm) {
			continue
		}
		parent := ""
		switch pp, ok := procs[p.PPID]; {
		case p.PPID == 1:
			parent = "init"
		case ok && pp.Comm == "systemd":
			parent = "user service manager"
		default:
			continue
		}
		if u, ok := s.FS.UID(pid); !ok || u != uid {
			continue
		}
		key := procKey{pid, p.StartTime}
		since, ok := s.hotSince[key]
		if !ok {
			since = now
		}
		hot[key] = since
		age := time.Duration((uptime - float64(p.StartTime)/procfs.ClockTicks) * float64(time.Second))
		if now.Sub(since) < cfg.For.Duration || age < cfg.MinAge.Duration {
			continue
		}
		o := Orphan{
			PID: pid, Start: p.StartTime, Comm: p.Comm, Cmdline: s.FS.Cmdline(pid), Cwd: s.FS.Cwd(pid),
			CPU: cpu[pid], Age: age.Truncate(time.Second), HotFor: now.Sub(since).Truncate(time.Second),
			Parent: parent, Kill: []string{"kill", strconv.Itoa(pid)},
		}
		if pane, ok := s.FS.Environ(pid, "TMUX_PANE"); ok {
			o.Pane = pane
			if pp, alive := panes[pane]; alive {
				o.PaneAlive, o.Target = true, pp.Target
			}
		}
		snap.Orphans = append(snap.Orphans, o)
	}
	s.hotSince = hot
	sort.Slice(snap.Orphans, func(i, j int) bool { return snap.Orphans[i].CPU > snap.Orphans[j].CPU })
}

func (s *Sampler) watched(comm string) bool {
	if len(s.Cfg.Orphans.Commands) == 0 {
		return true
	}
	for _, c := range s.Cfg.Orphans.Commands {
		if comm == c {
			return true
		}
	}
	return false
}
