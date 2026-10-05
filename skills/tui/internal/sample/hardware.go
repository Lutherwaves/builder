package sample

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type GPU struct {
	Index   int     `json:"index"`
	Util    float64 `json:"util_pct"`
	TempC   float64 `json:"temp_c"`
	MemUsed float64 `json:"mem_used_mib"`
	MemTot  float64 `json:"mem_total_mib"`
}

type Container struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu_pct"` // 100 = one core
}

// GroupShare is the CPU of the containers a configured group matches.
type GroupShare struct {
	Name       string  `json:"name"`
	Containers int     `json:"containers"`
	CPU        float64 `json:"cpu_pct"`   // 100 = one core
	Share      float64 `json:"share_pct"` // of the whole machine
}

// slowEvery is the cadence of the external commands (nvidia-smi, docker):
// each costs tens of milliseconds of CPU, far more than reading /proc.
const slowEvery = 60 * time.Second

func NvidiaSMI() ([]byte, error) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil, nil
	}
	return exec.Command("nvidia-smi", "--query-gpu=index,utilization.gpu,temperature.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
}

func DockerPS() ([]byte, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, nil
	}
	return exec.Command("docker", "ps", "--no-trunc", "--format", "{{.ID}}\t{{.Names}}").Output()
}

func parseGPUs(b []byte) []GPU {
	var out []GPU
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(line, ",")
		if len(f) < 5 {
			continue
		}
		num := func(i int) float64 { v, _ := strconv.ParseFloat(strings.TrimSpace(f[i]), 64); return v }
		out = append(out, GPU{Index: int(num(0)), Util: num(1), TempC: num(2), MemUsed: num(3), MemTot: num(4)})
	}
	return out
}

type containerRef struct{ id, name, cgroup string }

func (s *Sampler) sampleGPUs(now time.Time) []GPU {
	if s.GPU == nil || now.Sub(s.gpuAt) < slowEvery {
		return s.gpus
	}
	s.gpuAt = now
	b, err := s.GPU()
	if err != nil {
		return nil
	}
	s.gpus = parseGPUs(b)
	return s.gpus
}

// sampleContainers lists containers on the slow cadence and reads their
// cgroup CPU counters on every process scan.
func (s *Sampler) sampleContainers(now time.Time, dt float64, cores int) ([]Container, []GroupShare) {
	if s.Docker != nil && now.Sub(s.dockerAt) >= slowEvery {
		s.dockerAt = now
		if b, err := s.Docker(); err == nil {
			s.containers = s.containers[:0]
			for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
				id, name, ok := strings.Cut(line, "\t")
				if !ok {
					continue
				}
				s.containers = append(s.containers, containerRef{id, name, s.Sys.ContainerCgroup(id)})
			}
		}
	}
	var out []Container
	usage := map[string]uint64{}
	for _, c := range s.containers {
		u, ok := s.Sys.CgroupCPU(c.cgroup)
		if c.cgroup == "" || !ok {
			continue
		}
		usage[c.id] = u
		cpu := 0.0
		if prev, ok := s.prevCgroup[c.id]; ok && dt > 0 && u >= prev {
			cpu = 100 * float64(u-prev) / 1e6 / dt
		}
		out = append(out, Container{ID: c.id[:min(12, len(c.id))], Name: c.name, CPU: cpu})
	}
	s.prevCgroup = usage
	sort.Slice(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })

	var groups []GroupShare
	for _, g := range s.Cfg.Groups {
		gs := GroupShare{Name: g.Name}
		for _, c := range out {
			if g.Match(c.Name) {
				gs.Containers++
				gs.CPU += c.CPU
			}
		}
		if cores > 0 {
			gs.Share = gs.CPU / float64(cores)
		}
		groups = append(groups, gs)
	}
	return out, groups
}
