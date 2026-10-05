package procfs

import "testing"

const fixture = "../../testdata/machine/proc"

func TestParseStatCommWithSpacesAndParens(t *testing.T) {
	line := []byte("42 (tmux: server (1)) S 7 42 42 0 -1 0 0 0 0 0 30 12 0 0 20 0 1 0 555 1000 77")
	p, ok := parseStat(42, line)
	if !ok {
		t.Fatal("not parsed")
	}
	if p.Comm != "tmux: server (1)" || p.PPID != 7 || p.Ticks != 42 || p.StartTime != 555 || p.RSSPages != 77 || p.State != 'S' {
		t.Fatalf("got %+v", p)
	}
}

func TestProcs(t *testing.T) {
	procs, err := FS{fixture}.Procs()
	if err != nil {
		t.Fatal(err)
	}
	if got := procs[310]; got.Comm != "claude" || got.PPID != 300 {
		t.Fatalf("310 = %+v", got)
	}
	if len(procs) != 11 {
		t.Fatalf("want 11 procs, got %d", len(procs))
	}
}

func TestEnvironAndCwd(t *testing.T) {
	fs := FS{fixture}
	if v, ok := fs.Environ(400, "TMUX_PANE"); !ok || v != "%3" {
		t.Fatalf("TMUX_PANE = %q %v", v, ok)
	}
	if _, ok := fs.Environ(1, "TMUX_PANE"); ok {
		t.Fatal("pid 1 has no TMUX_PANE")
	}
	if got := fs.Cwd(200); got != "/work/app" {
		t.Fatalf("cwd = %q", got)
	}
}

func TestMachineFiles(t *testing.T) {
	fs := FS{fixture}
	all, cores, err := fs.Stat()
	if err != nil || len(cores) != 2 || all.Total != 9600 || all.Busy != 1500 {
		t.Fatalf("stat = %+v %d %v", all, len(cores), err)
	}
	m, err := fs.Meminfo()
	if err != nil || m.Available != 8192000<<10 || m.SwapTotal-m.SwapFree != 1024000<<10 {
		t.Fatalf("meminfo = %+v %v", m, err)
	}
	rx, tx, err := fs.NetBytes()
	if err != nil || rx != 1000 || tx != 2000 {
		t.Fatalf("net = %d %d %v (loopback and veth must be skipped)", rx, tx, err)
	}
	r, w, err := fs.DiskBytes()
	if err != nil || r != 200*512 || w != 400*512 {
		t.Fatalf("disk = %d %d %v (partitions must be skipped)", r, w, err)
	}
	if p := fs.Pressure(); p.CPU != 12.5 || p.IO != 12.5 {
		t.Fatalf("psi = %+v", p)
	}
	if l, _ := fs.Loadavg(); l[0] != 3.5 {
		t.Fatalf("load = %v", l)
	}
}

func TestWholeDisk(t *testing.T) {
	for name, want := range map[string]bool{
		"nvme0n1": true, "nvme0n1p2": false, "sda": true, "sda1": false,
		"vdb": true, "mmcblk0": true, "mmcblk0p1": false, "loop0": false, "dm-0": false,
	} {
		if got := wholeDisk(name); got != want {
			t.Errorf("wholeDisk(%q) = %v", name, got)
		}
	}
}

func TestPowerLimitsSkipSubzones(t *testing.T) {
	got := Sys{"../../testdata/machine/sys"}.PowerLimits()
	if len(got) != 2 || got[0] != (PowerLimit{"package-0", "long_term", 125}) || got[1].Watts != 188 {
		t.Fatalf("got %+v", got)
	}
}

func TestPackageTempIgnoresCores(t *testing.T) {
	// Core 0 reads 99 °C in the fixture; only the package sensor counts.
	if got := (Sys{"../../testdata/machine/sys"}).PackageTemp(); got != 91 {
		t.Fatalf("got %v", got)
	}
}

func TestContainerCgroup(t *testing.T) {
	s := Sys{"../../testdata/machine/sys"}
	id := "aaaabbbbccccddddeeeeffff0000111122223333444455556666777788889999"
	dir := s.ContainerCgroup(id)
	if u, ok := s.CgroupCPU(dir); !ok || u != 1000000 {
		t.Fatalf("dir %q usage %d %v", dir, u, ok)
	}
	if s.ContainerCgroup("missing") != "" {
		t.Fatal("missing container found")
	}
}
