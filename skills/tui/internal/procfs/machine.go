package procfs

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Mem is /proc/meminfo in bytes.
type Mem struct{ Total, Available, SwapTotal, SwapFree uint64 }

func (fs FS) Meminfo() (Mem, error) {
	f, err := os.Open(fs.path("meminfo"))
	if err != nil {
		return Mem{}, err
	}
	defer f.Close()
	var m Mem
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		kb, _ := strconv.ParseUint(fields[0], 10, 64)
		switch k {
		case "MemTotal":
			m.Total = kb << 10
		case "MemAvailable":
			m.Available = kb << 10
		case "SwapTotal":
			m.SwapTotal = kb << 10
		case "SwapFree":
			m.SwapFree = kb << 10
		}
	}
	return m, sc.Err()
}

// Loadavg returns load1, load5, load15.
func (fs FS) Loadavg() (l [3]float64, err error) {
	b, err := os.ReadFile(fs.path("loadavg"))
	if err != nil {
		return l, err
	}
	f := strings.Fields(string(b))
	for i := 0; i < 3 && i < len(f); i++ {
		l[i], _ = strconv.ParseFloat(f[i], 64)
	}
	return l, nil
}

// NetBytes sums rx/tx bytes over physical-looking interfaces. Loopback and
// virtual links (containers, bridges, tunnels) are skipped so container
// traffic is not counted twice.
func (fs FS) NetBytes() (rx, tx uint64, err error) {
	f, err := os.Open(fs.path("net", "dev"))
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if virtualIface(name) {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		t, _ := strconv.ParseUint(fields[8], 10, 64)
		rx += r
		tx += t
	}
	return rx, tx, sc.Err()
}

func virtualIface(name string) bool {
	for _, p := range []string{"lo", "veth", "docker", "br-", "virbr", "tailscale", "tun", "wg", "cni", "flannel"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// DiskBytes sums sectors read/written on whole disks (partitions and virtual
// devices are skipped so nothing is counted twice).
func (fs FS) DiskBytes() (read, written uint64, err error) {
	f, err := os.Open(fs.path("diskstats"))
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 10 || !wholeDisk(fields[2]) {
			continue
		}
		r, _ := strconv.ParseUint(fields[5], 10, 64)
		w, _ := strconv.ParseUint(fields[9], 10, 64)
		read += r * 512
		written += w * 512
	}
	return read, written, sc.Err()
}

func wholeDisk(name string) bool {
	switch {
	case strings.HasPrefix(name, "nvme"):
		return !strings.Contains(name, "p") // nvme0n1, not nvme0n1p1
	case strings.HasPrefix(name, "sd"), strings.HasPrefix(name, "vd"), strings.HasPrefix(name, "xvd"):
		last := name[len(name)-1]
		return last < '0' || last > '9'
	case strings.HasPrefix(name, "mmcblk"):
		return !strings.Contains(name, "p")
	}
	return false
}

// PSI is the "some avg10" pressure for cpu, io and memory, in percent.
type PSI struct{ CPU, IO, Memory float64 }

func (fs FS) Pressure() PSI {
	return PSI{
		CPU:    fs.psiSome("cpu"),
		IO:     fs.psiSome("io"),
		Memory: fs.psiSome("memory"),
	}
}

func (fs FS) psiSome(res string) float64 {
	b, err := os.ReadFile(fs.path("pressure", res))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "some ") {
			continue
		}
		for _, kv := range strings.Fields(line)[1:] {
			if v, ok := strings.CutPrefix(kv, "avg10="); ok {
				f, _ := strconv.ParseFloat(v, 64)
				return f
			}
		}
	}
	return 0
}
