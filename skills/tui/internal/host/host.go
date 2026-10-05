// Package host reads the machine through gopsutil, for systems without a
// Linux /proc (macOS, Windows). It returns the procfs types so the sampler
// does not care which one it reads.
package host

import (
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// Host counts process start times in ticks since the Unix epoch, and Uptime
// is seconds since the epoch, so ages work out as they do from /proc.
type Host struct{}

var _ procfs.Host = Host{}

const tick = float64(procfs.ClockTicks)

func (Host) Procs() (map[int]procfs.Proc, error) {
	ps, err := process.Processes()
	if err != nil {
		return nil, err
	}
	out := make(map[int]procfs.Proc, len(ps))
	for _, p := range ps {
		if pr, ok := read(p); ok {
			out[pr.PID] = pr
		}
	}
	return out, nil
}

func (Host) Proc(pid int) (procfs.Proc, bool) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return procfs.Proc{}, false
	}
	return read(p)
}

// read skips a process that exits or denies access while it is read.
func read(p *process.Process) (procfs.Proc, bool) {
	name, err := p.Name()
	if err != nil {
		return procfs.Proc{}, false
	}
	created, err := p.CreateTime()
	if err != nil {
		return procfs.Proc{}, false
	}
	pr := procfs.Proc{
		PID:       int(p.Pid),
		Comm:      strings.TrimSuffix(name, ".exe"),
		StartTime: uint64(created) / (1000 / procfs.ClockTicks),
		State:     'S',
	}
	if ppid, err := p.Ppid(); err == nil {
		pr.PPID = int(ppid)
	}
	if t, err := p.Times(); err == nil {
		pr.Ticks = uint64((t.User + t.System) * tick)
	}
	if m, err := p.MemoryInfo(); err == nil {
		pr.RSSPages = int64(m.RSS) / int64(os.Getpagesize())
	}
	if st, err := p.Status(); err == nil && len(st) > 0 && st[0] == process.Running {
		pr.State = 'R'
	}
	return pr, true
}

func (Host) Environ(pid int, key string) (string, bool) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", false
	}
	env, err := p.Environ()
	if err != nil {
		return "", false
	}
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v, true
		}
	}
	return "", false
}

func (Host) Cwd(pid int) string {
	if p, err := process.NewProcess(int32(pid)); err == nil {
		s, _ := p.Cwd()
		return s
	}
	return ""
}

func (Host) Cmdline(pid int) string {
	if p, err := process.NewProcess(int32(pid)); err == nil {
		s, _ := p.Cmdline()
		return s
	}
	return ""
}

// Cgroup is Linux only: containers run inside a VM on other systems.
func (Host) Cgroup(int) string { return "" }

func (Host) UID(pid int) (int, bool) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0, false
	}
	uids, err := p.Uids()
	if err != nil || len(uids) == 0 {
		return 0, false
	}
	return int(uids[0]), true
}

func (Host) Uptime() float64 { return float64(time.Now().UnixMilli()) / 1000 }

func (Host) Stat() (procfs.CPUTimes, []procfs.CPUTimes, error) {
	all, err := cpu.Times(false)
	if err != nil || len(all) == 0 {
		return procfs.CPUTimes{}, nil, err
	}
	per, _ := cpu.Times(true)
	cores := make([]procfs.CPUTimes, len(per))
	for i, c := range per {
		cores[i] = times(c)
	}
	return times(all[0]), cores, nil
}

func times(c cpu.TimesStat) procfs.CPUTimes {
	idle := c.Idle + c.Iowait
	total := c.User + c.System + c.Nice + c.Irq + c.Softirq + c.Steal + idle
	return procfs.CPUTimes{Busy: uint64((total - idle) * tick), Total: uint64(total * tick)}
}

func (Host) Loadavg() ([3]float64, error) {
	a, err := load.Avg()
	if err != nil {
		return [3]float64{}, err
	}
	return [3]float64{a.Load1, a.Load5, a.Load15}, nil
}

func (Host) Meminfo() (procfs.Mem, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return procfs.Mem{}, err
	}
	m := procfs.Mem{Total: v.Total, Available: v.Available}
	if s, err := mem.SwapMemory(); err == nil {
		m.SwapTotal, m.SwapFree = s.Total, s.Free
	}
	return m, nil
}

// Pressure stall information is Linux only.
func (Host) Pressure() procfs.PSI { return procfs.PSI{} }

func (Host) NetBytes() (rx, tx uint64, err error) {
	ifs, err := net.IOCounters(true)
	if err != nil {
		return 0, 0, err
	}
	for _, i := range ifs {
		if loopback(i.Name) {
			continue
		}
		rx += i.BytesRecv
		tx += i.BytesSent
	}
	return rx, tx, nil
}

func loopback(name string) bool {
	n := strings.ToLower(name)
	return n == "lo" || n == "lo0" || strings.Contains(n, "loopback")
}

func (Host) DiskBytes() (read, written uint64, err error) {
	ds, err := disk.IOCounters()
	if err != nil {
		return 0, 0, err
	}
	for _, d := range ds {
		read += d.ReadBytes
		written += d.WriteBytes
	}
	return read, written, nil
}
