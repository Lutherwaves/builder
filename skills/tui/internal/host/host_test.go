package host

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// On Linux the gopsutil reader must agree with /proc, so what it shows on
// other systems means the same thing.
func TestAgreesWithProc(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc is Linux only")
	}
	fs, h := procfs.FS{Root: "/proc"}, Host{}
	pid := os.Getpid()
	want, ok := fs.Proc(pid)
	got, ok2 := h.Proc(pid)
	if !ok || !ok2 {
		t.Fatalf("read self: proc %v host %v", ok, ok2)
	}
	if got.PPID != want.PPID || got.Comm != want.Comm {
		t.Fatalf("host %+v, proc %+v", got, want)
	}
	age := func(f procfs.Host, p procfs.Proc) float64 { return f.Uptime() - float64(p.StartTime)/procfs.ClockTicks }
	if d := math.Abs(age(h, got) - age(fs, want)); d > 2 {
		t.Fatalf("process ages differ by %.1f s", d)
	}
	if h.Cwd(pid) != fs.Cwd(pid) || h.Cmdline(pid) != fs.Cmdline(pid) {
		t.Fatalf("cwd or cmdline differ: %q %q", h.Cwd(pid), h.Cmdline(pid))
	}
	if v, ok := h.Environ(pid, "PATH"); !ok || v != os.Getenv("PATH") {
		t.Fatalf("environ PATH = %q %v", v, ok)
	}
	if u, ok := h.UID(pid); !ok || u != os.Getuid() {
		t.Fatalf("uid = %d %v", u, ok)
	}
	_, wantCores, err := fs.Stat()
	all, cores, err2 := h.Stat()
	if err != nil || err2 != nil || len(cores) != len(wantCores) || all.Total == 0 || all.Busy > all.Total {
		t.Fatalf("stat: %d cores vs %d, %+v (%v %v)", len(cores), len(wantCores), all, err, err2)
	}
	wm, _ := fs.Meminfo()
	hm, err := h.Meminfo()
	if err != nil || hm.Total != wm.Total {
		t.Fatalf("mem total %d vs %d (%v)", hm.Total, wm.Total, err)
	}
	procs, err := h.Procs()
	if err != nil || procs[pid].PID != pid {
		t.Fatalf("Procs misses self (%d processes, %v)", len(procs), err)
	}
}
