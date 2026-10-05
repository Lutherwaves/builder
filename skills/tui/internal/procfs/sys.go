package procfs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Sys reads /sys, rooted like FS so tests can use fixtures.
type Sys struct{ Root string }

func (s Sys) path(parts ...string) string {
	return filepath.Join(append([]string{s.Root}, parts...)...)
}

func readTrim(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

func readInt(path string) (int64, bool) {
	s, ok := readTrim(path)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	return v, err == nil
}

// PowerLimit is one RAPL constraint of a package zone.
type PowerLimit struct {
	Zone  string  `json:"zone"` // package-0
	Name  string  `json:"name"` // long_term, short_term
	Watts float64 `json:"watts"`
}

// PowerLimits lists the constraints of every top-level powercap zone
// (packages, not their core/uncore subzones). Readable without root.
func (s Sys) PowerLimits() []PowerLimit {
	zones, _ := filepath.Glob(s.path("class", "powercap", "*-rapl:*"))
	var out []PowerLimit
	for _, z := range zones {
		if strings.Count(filepath.Base(z), ":") != 1 {
			continue
		}
		zone, _ := readTrim(filepath.Join(z, "name"))
		for i := 0; ; i++ {
			c := filepath.Join(z, "constraint_"+strconv.Itoa(i)+"_")
			uw, ok := readInt(c + "power_limit_uw")
			if !ok {
				break
			}
			name, _ := readTrim(c + "name")
			out = append(out, PowerLimit{Zone: zone, Name: name, Watts: float64(uw) / 1e6})
		}
	}
	return out
}

// TempSensors lists the files that report a CPU package temperature:
// coretemp/k10temp package sensors, else the x86_pkg_temp thermal zone.
// Finding them walks /sys, so callers keep the list and re-read only these.
func (s Sys) TempSensors() []string {
	var out []string
	mons, _ := filepath.Glob(s.path("class", "hwmon", "hwmon*"))
	for _, h := range mons {
		name, _ := readTrim(filepath.Join(h, "name"))
		if name != "coretemp" && name != "k10temp" && name != "zenpower" {
			continue
		}
		labels, _ := filepath.Glob(filepath.Join(h, "temp*_label"))
		for _, l := range labels {
			label, _ := readTrim(l)
			if strings.HasPrefix(label, "Package id") || label == "Tctl" || label == "Tdie" {
				out = append(out, strings.TrimSuffix(l, "_label")+"_input")
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	zones, _ := filepath.Glob(s.path("class", "thermal", "thermal_zone*"))
	for _, z := range zones {
		if t, _ := readTrim(filepath.Join(z, "type")); t == "x86_pkg_temp" {
			out = append(out, filepath.Join(z, "temp"))
		}
	}
	return out
}

// ReadTemp returns the hottest of the sensor files in °C, 0 when none reads.
func ReadTemp(sensors []string) float64 {
	best := 0.0
	for _, f := range sensors {
		if mc, ok := readInt(f); ok {
			best = max(best, float64(mc)/1000)
		}
	}
	return best
}

// PackageTemp is the hottest CPU package in °C, or 0 when no sensor is found.
func (s Sys) PackageTemp() float64 { return ReadTemp(s.TempSensors()) }

// CgroupCPU returns usage_usec from a cgroup v2 directory's cpu.stat.
func (s Sys) CgroupCPU(dir string) (uint64, bool) {
	b, err := os.ReadFile(s.path("fs", "cgroup", dir, "cpu.stat"))
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "usage_usec "); ok {
			n, err := strconv.ParseUint(v, 10, 64)
			return n, err == nil
		}
	}
	return 0, false
}

// ContainerCgroup finds a container's cgroup under the systemd or the
// cgroupfs driver layout. Empty when neither exists.
func (s Sys) ContainerCgroup(id string) string {
	for _, dir := range []string{
		filepath.Join("system.slice", "docker-"+id+".scope"),
		filepath.Join("docker", id),
	} {
		if _, err := os.Stat(s.path("fs", "cgroup", dir, "cpu.stat")); err == nil {
			return dir
		}
	}
	return ""
}

// UID returns the real uid from /proc/<pid>/status.
func (fs FS) UID(pid int) (int, bool) {
	b, err := os.ReadFile(fs.path(strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "Uid:"); ok {
			f := strings.Fields(v)
			if len(f) == 0 {
				return 0, false
			}
			uid, err := strconv.Atoi(f[0])
			return uid, err == nil
		}
	}
	return 0, false
}

// Cgroup returns the cgroup v2 path of a process ("/system.slice/x.scope").
func (fs FS) Cgroup(pid int) string {
	b, err := os.ReadFile(fs.path(strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "0::"); ok {
			return v
		}
	}
	return ""
}
