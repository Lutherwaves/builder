// Package procfs reads the Linux /proc files the cockpit needs, rooted at a
// configurable directory so tests can run against fixtures.
package procfs

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ClockTicks is USER_HZ. It is 100 on every mainstream Linux build; reading it
// properly needs cgo (sysconf), which would break the static binary.
const ClockTicks = 100

type FS struct{ Root string }

func (fs FS) path(parts ...string) string {
	return filepath.Join(append([]string{fs.Root}, parts...)...)
}

// Proc is one line of /proc/<pid>/stat, reduced to what the cockpit uses.
type Proc struct {
	PID       int
	PPID      int
	Comm      string
	State     byte
	Ticks     uint64 // utime + stime
	StartTime uint64 // clock ticks after boot
	RSSPages  int64
}

// Procs returns every process that can be read. Processes that exit between
// the directory listing and the read are skipped.
func (fs FS) Procs() (map[int]Proc, error) {
	dir, err := os.Open(fs.path())
	if err != nil {
		return nil, err
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return nil, err
	}
	// This runs every tick over every process, so it skips os.ReadFile's
	// fstat and allocation: one open, one read into a reused buffer, close.
	buf := make([]byte, 1024)
	out := make(map[int]Proc, len(names))
	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		n, ok := readSmall(fs.path(name, "stat"), buf)
		if !ok {
			continue
		}
		if p, ok := parseStat(pid, buf[:n]); ok {
			out[pid] = p
		}
	}
	return out, nil
}

// Proc reads one process.
func (fs FS) Proc(pid int) (Proc, bool) {
	buf := make([]byte, 1024)
	n, ok := readSmall(fs.path(strconv.Itoa(pid), "stat"), buf)
	if !ok {
		return Proc{}, false
	}
	return parseStat(pid, buf[:n])
}

func readSmall(path string, buf []byte) (int, bool) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return 0, false
	}
	defer syscall.Close(fd)
	n, err := syscall.Read(fd, buf)
	return n, err == nil && n > 0
}

func parseStat(pid int, b []byte) (Proc, bool) {
	// comm may contain spaces and parentheses, so split on the last ')'.
	open, end := bytes.IndexByte(b, '('), bytes.LastIndexByte(b, ')')
	if open < 0 || end < open {
		return Proc{}, false
	}
	f := strings.Fields(string(b[end+1:]))
	if len(f) < 22 {
		return Proc{}, false
	}
	ppid, _ := strconv.Atoi(f[1])
	ut, _ := strconv.ParseUint(f[11], 10, 64)
	st, _ := strconv.ParseUint(f[12], 10, 64)
	start, _ := strconv.ParseUint(f[19], 10, 64)
	rss, _ := strconv.ParseInt(f[21], 10, 64)
	return Proc{
		PID: pid, PPID: ppid, Comm: string(b[open+1 : end]), State: f[0][0],
		Ticks: ut + st, StartTime: start, RSSPages: rss,
	}, true
}

// Environ returns one variable from /proc/<pid>/environ.
func (fs FS) Environ(pid int, key string) (string, bool) {
	b, err := os.ReadFile(fs.path(strconv.Itoa(pid), "environ"))
	if err != nil {
		return "", false
	}
	prefix := []byte(key + "=")
	for _, kv := range bytes.Split(b, []byte{0}) {
		if bytes.HasPrefix(kv, prefix) {
			return string(kv[len(prefix):]), true
		}
	}
	return "", false
}

// Cwd returns the target of /proc/<pid>/cwd.
func (fs FS) Cwd(pid int) string {
	s, _ := os.Readlink(fs.path(strconv.Itoa(pid), "cwd"))
	return s
}

// Cmdline returns argv joined by spaces.
func (fs FS) Cmdline(pid int) string {
	b, _ := os.ReadFile(fs.path(strconv.Itoa(pid), "cmdline"))
	return strings.TrimSpace(string(bytes.ReplaceAll(b, []byte{0}, []byte{' '})))
}

// CPUTimes is the aggregate "cpu" line of /proc/stat.
type CPUTimes struct{ Busy, Total uint64 }

// Stat returns the aggregate line followed by one entry per core.
func (fs FS) Stat() (all CPUTimes, cores []CPUTimes, err error) {
	f, err := os.Open(fs.path("stat"))
	if err != nil {
		return all, nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu") {
			break
		}
		fields := strings.Fields(line)
		var t CPUTimes
		for i, v := range fields[1:] {
			n, _ := strconv.ParseUint(v, 10, 64)
			if i >= 8 { // guest time is already counted in user/nice
				break
			}
			t.Total += n
			if i != 3 && i != 4 { // idle, iowait
				t.Busy += n
			}
		}
		if fields[0] == "cpu" {
			all = t
		} else {
			cores = append(cores, t)
		}
	}
	return all, cores, sc.Err()
}

// Uptime returns seconds since boot.
func (fs FS) Uptime() float64 {
	b, err := os.ReadFile(fs.path("uptime"))
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}
