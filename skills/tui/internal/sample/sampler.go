// Package sample turns /proc, tmux and agent session files into Snapshots.
// A Sampler keeps the previous counters so rates (CPU, network, disk) are
// deltas between two calls; it is not safe for concurrent use.
package sample

import (
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/config"
	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// Session is one agent process and the pane that owns it.
type Session struct {
	PID        int           `json:"pid"`
	Pane       string        `json:"pane"`
	Target     string        `json:"target"`
	Tab        string        `json:"tab"` // tmux window name
	Name       string        `json:"name"`
	SessionID  string        `json:"session_id"`
	Cwd        string        `json:"cwd"`
	Busy       bool          `json:"busy"`
	Status     string        `json:"status"` // as the agent reports it: busy, idle, ...
	Idle       time.Duration `json:"idle_ns"`
	Started    time.Time     `json:"started"`
	Model      string        `json:"model,omitempty"`
	Context    string        `json:"context"` // known | fresh (no turn since start or clear) | unknown (no session file)
	Tokens     int           `json:"tokens"`
	Window     int           `json:"window"`
	ContextPct float64       `json:"context_pct"`
	Burn30m    int           `json:"burn_30m_tokens"`
	CPU        float64       `json:"cpu_pct"` // whole process tree, 100 = one core
	Procs      int           `json:"procs"`
	Do         string        `json:"do,omitempty"` // compact | clear
	Why        string        `json:"why,omitempty"`
}

// IdlePane is a pane that hosts no agent.
type IdlePane struct {
	Pane    string `json:"pane"`
	Target  string `json:"target"`
	Command string `json:"command"`
	Path    string `json:"path"`
}

type Machine struct {
	CPU        float64             `json:"cpu_pct"` // 0-100 over all cores
	Cores      []float64           `json:"cores_pct"`
	Load       [3]float64          `json:"load"`
	MemTotal   uint64              `json:"mem_total"`
	MemAvail   uint64              `json:"mem_available"`
	SwapUsed   uint64              `json:"swap_used"`
	PSI        procfs.PSI          `json:"psi"`
	NetRx      float64             `json:"net_rx_bps"`
	NetTx      float64             `json:"net_tx_bps"`
	DiskRead   float64             `json:"disk_read_bps"`
	DiskWrite  float64             `json:"disk_write_bps"`
	Latency    time.Duration       `json:"latency_ns"`
	LatencyErr string              `json:"latency_error,omitempty"`
	TempC      float64             `json:"temp_c"`
	Power      []procfs.PowerLimit `json:"power_limits"`
	GPUs       []GPU               `json:"gpus"`
}

type Snapshot struct {
	At         time.Time    `json:"at"`
	Machine    Machine      `json:"machine"`
	Sessions   []Session    `json:"sessions"`
	IdlePanes  []IdlePane   `json:"idle_panes"`
	Orphans    []Orphan     `json:"orphans"`
	Top        []ProcRow    `json:"top"`
	Containers []Container  `json:"containers"`
	Groups     []GroupShare `json:"groups"`
	GitPolling []GitPolling `json:"git_polling"`
	Alerts     []Alert      `json:"alerts"`
	Errors     []string     `json:"errors,omitempty"`
}

// Rules decide the per-session suggestion.
type Rules struct {
	CompactAt   float64       // context percent at which a busy session should compact
	ClearIdle   time.Duration // idle this long with ClearMinPct context: clear
	ClearMinPct float64
	ClearStale  time.Duration // idle this long: clear regardless of context
	Windows     map[string]int
}

var DefaultRules = Rules{
	CompactAt:   85,
	ClearIdle:   30 * time.Minute,
	ClearMinPct: 20,
	ClearStale:  24 * time.Hour,
}

type Sampler struct {
	FS            procfs.Host
	Sys           procfs.Sys
	Home          string
	Tmux          func() ([]byte, error)
	GPU           func() ([]byte, error) // nil: no GPUs
	Docker        func() ([]byte, error) // nil: no containers
	Cfg           config.Config
	Agents        []string // process names that are agent sessions
	Rules         Rules
	LatencyTarget string // host:port, empty disables
	// ProcEvery is how often the process table, tmux and sessions are
	// re-read. Reading every /proc/<pid>/stat is most of a sample's cost, so
	// it runs slower than the cheap machine counters. Zero: every sample.
	ProcEvery time.Duration
	Now       func() time.Time

	agents     *agentReader
	panes      map[procKey]string // environ is read once per process
	prevAt     time.Time
	procAt     time.Time
	prevProc   map[int]procfs.Proc
	last       Snapshot // sessions and panes from the last process scan
	prevCPU    procfs.CPUTimes
	prevCore   []procfs.CPUTimes
	prevNet    [2]uint64
	prevDisk   [2]uint64
	latAt      time.Time
	lat        time.Duration
	latErr     string
	burn       map[string][]tokenPoint
	bigSession map[string]bool  // sessions seen past 200k tokens, so 1M windows
	longCtx    map[procKey]bool // processes started with a 1M model setting

	owners     map[procKey]string
	hotSince   map[procKey]time.Time
	condSince  map[string]time.Time
	powerBase  map[string]float64
	gpuAt      time.Time
	gpus       []GPU
	dockerAt   time.Time
	containers []containerRef
	prevCgroup map[string]uint64
	toplevel   map[string]string
	sensors    []string            // temperature files, found once
	power      []procfs.PowerLimit // read on the scan cadence
	polls      []pollPoint
}

// procKey tells a reused PID apart from the process that had it before.
type procKey struct {
	pid   int
	start uint64
}

type tokenPoint struct {
	at     time.Time
	tokens int
}

func (s *Sampler) init() {
	if s.agents == nil {
		s.agents = newAgentReader(s.Home)
		s.panes = map[procKey]string{}
		s.burn = map[string][]tokenPoint{}
		s.bigSession = map[string]bool{}
		s.longCtx = map[procKey]bool{}
		s.owners = map[procKey]string{}
		s.toplevel = map[string]string{}
		if s.Cfg.Interval.Duration == 0 && s.Cfg.Orphans.CPUOver == 0 {
			s.Cfg = config.Default() // an unset config, not a partly set one
		}
		if s.Rules.CompactAt == 0 {
			s.Rules = DefaultRules
		}
		if s.Now == nil {
			s.Now = time.Now
		}
		if s.Tmux == nil {
			s.Tmux = TmuxPanes
		}
		if len(s.Agents) == 0 {
			s.Agents = []string{"claude"}
		}
	}
}

func (s *Sampler) Sample() Snapshot {
	s.init()
	now := s.Now()
	snap := Snapshot{At: now}
	s.sampleMachine(&snap, seconds(s.prevAt, now))
	snap.Machine.GPUs = s.sampleGPUs(now)
	s.prevAt = now

	if s.procAt.IsZero() || now.Sub(s.procAt) >= s.ProcEvery {
		s.scan(now)
	}
	l := s.last
	snap.Sessions, snap.IdlePanes, snap.Orphans, snap.Top = l.Sessions, l.IdlePanes, l.Orphans, l.Top
	snap.Containers, snap.Groups, snap.GitPolling = l.Containers, l.Groups, l.GitPolling
	snap.Errors = append(snap.Errors, l.Errors...)
	s.alerts(&snap, now)
	return snap
}

// scan reads the process table, tmux and containers: the expensive half of a
// sample, run every ProcEvery.
func (s *Sampler) scan(now time.Time) {
	dt := seconds(s.procAt, now)
	var l Snapshot
	s.power = s.Sys.PowerLimits()
	procs, err := s.FS.Procs()
	if err != nil {
		l.Errors = append(l.Errors, "proc: "+err.Error())
	}
	out, err := s.Tmux()
	if err != nil {
		l.Errors = append(l.Errors, "tmux: "+err.Error())
	}
	panes := ParsePanes(out)
	s.sampleSessions(&l, panes, procs, dt, now)
	l.Containers, l.Groups = s.sampleContainers(now, dt, len(s.prevCore))
	s.sampleProcs(&l, procs, panes, dt, now)
	l.GitPolling = s.samplePolling(procs, now)
	for key := range s.owners {
		if p, ok := procs[key.pid]; !ok || p.StartTime != key.start {
			delete(s.owners, key)
		}
	}
	s.procAt, s.prevProc, s.last = now, procs, l
}

func seconds(prev, now time.Time) float64 {
	if prev.IsZero() || !now.After(prev) {
		return 0
	}
	return now.Sub(prev).Seconds()
}

func (s *Sampler) sampleMachine(snap *Snapshot, dt float64) {
	m := &snap.Machine
	if all, cores, err := s.FS.Stat(); err == nil {
		m.CPU = busyPct(s.prevCPU, all)
		m.Cores = make([]float64, len(cores))
		for i, c := range cores {
			if i < len(s.prevCore) {
				m.Cores[i] = busyPct(s.prevCore[i], c)
			}
		}
		s.prevCPU, s.prevCore = all, cores
	}
	m.Load, _ = s.FS.Loadavg()
	if mem, err := s.FS.Meminfo(); err == nil {
		m.MemTotal, m.MemAvail = mem.Total, mem.Available
		m.SwapUsed = mem.SwapTotal - mem.SwapFree
	}
	m.PSI = s.FS.Pressure()
	if rx, tx, err := s.FS.NetBytes(); err == nil {
		if dt > 0 {
			m.NetRx, m.NetTx = rate(s.prevNet[0], rx, dt), rate(s.prevNet[1], tx, dt)
		}
		s.prevNet = [2]uint64{rx, tx}
	}
	if r, w, err := s.FS.DiskBytes(); err == nil {
		if dt > 0 {
			m.DiskRead, m.DiskWrite = rate(s.prevDisk[0], r, dt), rate(s.prevDisk[1], w, dt)
		}
		s.prevDisk = [2]uint64{r, w}
	}
	s.sampleLatency()
	m.Latency, m.LatencyErr = s.lat, s.latErr
	if s.sensors == nil {
		s.sensors = s.Sys.TempSensors()
	}
	m.TempC = procfs.ReadTemp(s.sensors)
	m.Power = s.power
}

// sampleLatency measures a TCP connect at most every 10 s: no root and no
// ICMP needed, and it reflects what a git push or an API call will feel.
func (s *Sampler) sampleLatency() {
	if s.LatencyTarget == "" || s.Now().Sub(s.latAt) < 10*time.Second {
		return
	}
	s.latAt = s.Now()
	start := time.Now()
	c, err := net.DialTimeout("tcp", s.LatencyTarget, 2*time.Second)
	if err != nil {
		s.lat, s.latErr = 0, err.Error()
		return
	}
	c.Close()
	s.lat, s.latErr = time.Since(start), ""
}

func busyPct(prev, cur procfs.CPUTimes) float64 {
	if prev.Total == 0 || cur.Total <= prev.Total {
		return 0
	}
	return 100 * float64(cur.Busy-prev.Busy) / float64(cur.Total-prev.Total)
}

func rate(prev, cur uint64, dt float64) float64 {
	if prev == 0 || cur < prev {
		return 0
	}
	return float64(cur-prev) / dt
}

func (s *Sampler) sampleSessions(snap *Snapshot, panes []Pane, procs map[int]procfs.Proc, dt float64, now time.Time) {
	children := map[int][]int{}
	for pid, p := range procs {
		children[p.PPID] = append(children[p.PPID], pid)
	}
	byID := map[string]Pane{}
	byPID := map[int]Pane{}
	for _, p := range panes {
		byID[p.ID] = p
		byPID[p.PID] = p
	}

	hosting := map[string]bool{}
	alive := map[procKey]bool{}
	for pid, p := range procs {
		if !s.isAgent(p.Comm) {
			continue
		}
		// An agent started anywhere below another agent (helpers, a nested
		// `claude -p` in its shell) inherits its pane: count only the top one.
		if s.underAgent(p, procs) {
			continue
		}
		key := procKey{pid, p.StartTime}
		alive[key] = true
		paneID, ok := s.panes[key]
		if !ok {
			paneID = s.ownerPane(pid, procs, byPID)
			s.panes[key] = paneID
		}
		pane := byID[paneID]
		hosting[paneID] = true

		cpu, n := s.treeCPU(pid, procs, children, dt)
		sess := Session{
			PID: pid, Pane: paneID, Target: pane.Target, Tab: pane.Window, Cwd: s.FS.Cwd(pid),
			CPU: cpu, Procs: n, Window: 200_000, Context: "unknown",
		}
		if e, ok := s.agents.entry(pid); ok {
			sess.Name, sess.SessionID = e.Name, e.SessionID
			sess.Busy, sess.Status = e.Status == "busy", e.Status
			if e.StartedAt > 0 {
				sess.Started = time.UnixMilli(e.StartedAt)
			}
			if !sess.Busy && e.StatusUpdatedAt > 0 {
				sess.Idle = now.Sub(time.UnixMilli(e.StatusUpdatedAt)).Truncate(time.Second)
			}
			sess.Context = "fresh"
			if u, ok := s.agents.usageOf(e.SessionID, e.Cwd); ok {
				sess.Context = "known"
				sess.Model, sess.Tokens = u.Model, u.Tokens
				long, ok := s.longCtx[key]
				if !ok {
					env := func(k string) (string, bool) { return s.FS.Environ(pid, k) }
					long = longContext(modelSetting(s.FS.Cmdline(pid), env, e.Cwd, s.Home))
					s.longCtx[key] = long
				}
				sess.Window = s.window(e.SessionID, u.Model, u.Tokens, long)
				sess.ContextPct = 100 * float64(u.Tokens) / float64(sess.Window)
				sess.Burn30m = s.trackBurn(e.SessionID, now, u.Tokens)
			}
		}
		if sess.Name == "" {
			sess.Name = p.Comm
		}
		sess.Do, sess.Why = s.Rules.suggest(sess)
		snap.Sessions = append(snap.Sessions, sess)
	}
	for key := range s.panes {
		if !alive[key] {
			delete(s.panes, key)
		}
	}
	for key := range s.longCtx {
		if !alive[key] {
			delete(s.longCtx, key)
		}
	}
	seen := map[string]bool{}
	for _, x := range snap.Sessions {
		seen[x.SessionID] = true
	}
	for id := range s.burn {
		if !seen[id] {
			delete(s.burn, id)
		}
	}
	for id := range s.bigSession {
		if !seen[id] {
			delete(s.bigSession, id)
		}
	}
	for _, p := range panes {
		if !hosting[p.ID] {
			snap.IdlePanes = append(snap.IdlePanes, IdlePane{p.ID, p.Target, p.Command, p.Path})
		}
	}
	sort.Slice(snap.Sessions, func(i, j int) bool {
		a, b := snap.Sessions[i], snap.Sessions[j]
		if a.ContextPct != b.ContextPct {
			return a.ContextPct > b.ContextPct
		}
		return a.Pane < b.Pane
	})
}

func (s *Sampler) isAgent(comm string) bool {
	for _, a := range s.Agents {
		if comm == a {
			return true
		}
	}
	return false
}

// ownerPane prefers the pane the agent recorded in its session file, then
// TMUX_PANE from its environment (both survive re-parenting), and falls back
// to walking up to a pane's shell.
func (s *Sampler) ownerPane(pid int, procs map[int]procfs.Proc, byPID map[int]Pane) string {
	if e, ok := s.agents.entry(pid); ok && e.Tmux != "" {
		if i := strings.LastIndexByte(e.Tmux, '.'); i >= 0 && strings.HasPrefix(e.Tmux[i+1:], "%") {
			return e.Tmux[i+1:]
		}
	}
	if v, ok := s.FS.Environ(pid, "TMUX_PANE"); ok {
		return v
	}
	for cur, hops := pid, 0; cur > 1 && hops < 64; hops++ {
		if p, ok := byPID[cur]; ok {
			return p.ID
		}
		cur = procs[cur].PPID
	}
	return ""
}

func (s *Sampler) treeCPU(root int, procs map[int]procfs.Proc, children map[int][]int, dt float64) (float64, int) {
	var ticks uint64
	n := 0
	stack := []int{root}
	for len(stack) > 0 {
		pid := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		p := procs[pid]
		n++
		if prev, ok := s.prevProc[pid]; ok && prev.StartTime == p.StartTime && p.Ticks >= prev.Ticks {
			ticks += p.Ticks - prev.Ticks
		}
		stack = append(stack, children[pid]...)
	}
	if dt == 0 {
		return 0, n
	}
	return 100 * float64(ticks) / procfs.ClockTicks / dt, n
}

// window is the session's context size. A configured value for its model
// wins; otherwise a session started with a 1M model setting, or seen past
// 200k, has a 1M window. Sessions of one model can differ, so this is decided
// per session.
func (s *Sampler) window(sessionID, model string, tokens int, long bool) int {
	for prefix, w := range s.Rules.Windows {
		if strings.HasPrefix(model, prefix) {
			return w
		}
	}
	if long || tokens > 200_000 {
		s.bigSession[sessionID] = true
	}
	if s.bigSession[sessionID] {
		return 1_000_000
	}
	return 200_000
}

func (s *Sampler) underAgent(p procfs.Proc, procs map[int]procfs.Proc) bool {
	for cur, hops := p.PPID, 0; cur > 1 && hops < 64; hops++ {
		pp, ok := procs[cur]
		if !ok {
			return false
		}
		if s.isAgent(pp.Comm) {
			return true
		}
		cur = pp.PPID
	}
	return false
}

// trackBurn returns how many tokens the session's context grew in the last
// 30 minutes, keeping at most one point a minute.
func (s *Sampler) trackBurn(id string, now time.Time, tokens int) int {
	pts := s.burn[id]
	if n := len(pts); n > 0 && now.Sub(pts[n-1].at) < time.Minute {
		pts[n-1] = tokenPoint{now, tokens}
	} else {
		pts = append(pts, tokenPoint{now, tokens})
	}
	for len(pts) > 1 && now.Sub(pts[1].at) >= 30*time.Minute {
		pts = pts[1:]
	}
	s.burn[id] = pts
	return max(tokens-pts[0].tokens, 0)
}

// suggest: a session left idle while holding context should be cleared;
// one near its window should compact before its next turn hits the limit.
func (r Rules) suggest(s Session) (do, why string) {
	switch {
	case s.Tokens == 0:
		return "", ""
	case !s.Busy && s.Idle >= r.ClearStale:
		return "clear", "idle " + Human(s.Idle) + " (clear after " + Human(r.ClearStale) + ")"
	case !s.Busy && s.Idle >= r.ClearIdle && s.ContextPct >= r.ClearMinPct:
		return "clear", "idle " + Human(s.Idle) + " holding " + pct(s.ContextPct) + " context"
	case s.ContextPct >= r.CompactAt:
		return "compact", "context " + pct(s.ContextPct) + " (compact at " + pct(r.CompactAt) + ")"
	}
	return "", ""
}

func pct(f float64) string { return strconv.Itoa(int(f+0.5)) + "%" }

// Human renders a duration as its two largest units, dropping a zero
// second unit: 2h14m, 6d2h, 1d, 41m, 3s.
func Human(d time.Duration) string {
	d = d.Truncate(time.Second)
	two := func(a time.Duration, ua string, b time.Duration, ub string) string {
		out := strconv.Itoa(int(a)) + ua
		if b > 0 {
			out += strconv.Itoa(int(b)) + ub
		}
		return out
	}
	day := 24 * time.Hour
	switch {
	case d >= day:
		return two(d/day, "d", d%day/time.Hour, "h")
	case d >= time.Hour:
		return two(d/time.Hour, "h", d%time.Hour/time.Minute, "m")
	case d >= time.Minute:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	return strconv.Itoa(int(d/time.Second)) + "s"
}
