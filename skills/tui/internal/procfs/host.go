package procfs

// Host is what the sampler reads about the machine and its processes. FS
// implements it from /proc on Linux and over fixtures; other systems get an
// implementation from their own APIs.
type Host interface {
	Procs() (map[int]Proc, error)
	Proc(pid int) (Proc, bool)
	Environ(pid int, key string) (string, bool)
	Cwd(pid int) string
	Cmdline(pid int) string
	Cgroup(pid int) string
	UID(pid int) (int, bool)
	// Uptime is in seconds on the same clock as Proc.StartTime, so
	// Uptime() - StartTime/ClockTicks is a process's age.
	Uptime() float64
	Stat() (all CPUTimes, cores []CPUTimes, err error)
	Loadavg() ([3]float64, error)
	Meminfo() (Mem, error)
	Pressure() PSI
	NetBytes() (rx, tx uint64, err error)
	DiskBytes() (read, written uint64, err error)
}

var _ Host = FS{}
