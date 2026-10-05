// Package config loads the per-machine goals. The file is local and never
// part of the repository: thresholds describe one machine, not the tool.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
)

// Duration reads "30s", "2m", "24h" from TOML.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b))
	d.Duration = v
	return err
}

type Config struct {
	Interval      Duration `toml:"interval"`
	Agents        []string `toml:"agents"`
	LatencyTarget string   `toml:"latency_target"`
	Sessions      Sessions `toml:"sessions"`
	Alerts        Alerts   `toml:"alerts"`
	Orphans       Orphans  `toml:"orphans"`
	Groups        []Group  `toml:"groups"`
	Git           Git      `toml:"git"`
	History       History  `toml:"history"`
	Hooks         Hooks    `toml:"hooks"`
}

// Hooks run a command when something happens. The command is an argv list,
// never a shell string; the event's title and evidence are appended as two
// more arguments.
type Hooks struct {
	Alert []string `toml:"alert"` // on each new alert, e.g. ["notify-send", "builder"]
}

type Sessions struct {
	CompactAt   float64        `toml:"compact_at"`
	ClearIdle   Duration       `toml:"clear_idle"`
	ClearMinPct float64        `toml:"clear_min_pct"`
	ClearStale  Duration       `toml:"clear_stale"`
	Windows     map[string]int `toml:"windows"` // model prefix -> context tokens
}

// Alerts: a zero threshold is off. "For" durations make an alert wait until
// the condition has held that long, so a spike does not page anyone.
type Alerts struct {
	TempOverC               float64  `toml:"temp_over_c"`
	Load1Over               float64  `toml:"load1_over"`
	Load1For                Duration `toml:"load1_for"`
	PSICPUOver              float64  `toml:"psi_cpu_over"`
	PSICPUFor               Duration `toml:"psi_cpu_for"`
	MemAvailableUnderGB     float64  `toml:"mem_available_under_gb"`
	SwapOverGB              float64  `toml:"swap_over_gb"`
	SwapWhenAvailableOverGB float64  `toml:"swap_when_available_over_gb"`
	// PowerLimitsW are the expected package limits in watts, long term then
	// short term. Empty: alert when a limit changes from its value at startup.
	PowerLimitsW []float64 `toml:"power_limits_w"`
}

// Orphans are processes re-parented to init or a user service manager that
// keep burning CPU: leaked test fixtures, forgotten loops.
type Orphans struct {
	CPUOver  float64  `toml:"cpu_over"` // percent of one core
	For      Duration `toml:"for"`
	MinAge   Duration `toml:"min_age"`
	Commands []string `toml:"commands"` // empty: any command
}

// Group sums the CPU of containers whose name matches a pattern.
type Group struct {
	Name      string `toml:"name"`
	Container string `toml:"container"` // regular expression on the container name
	re        *regexp.Regexp
}

func (g Group) Match(name string) bool { return g.re != nil && g.re.MatchString(name) }

type Git struct {
	Repos    []string `toml:"repos"` // checkouts to watch besides the ones sessions use
	Interval Duration `toml:"interval"`
	Main     string   `toml:"main"`   // default branch name
	Remote   string   `toml:"remote"` // remote to compare against
}

type History struct {
	Log    string `toml:"log"`    // append-only CSV the cockpit writes; "-" disables
	Source string `toml:"source"` // extra CSV with a ts column to chart (another logger)
}

func Default() Config {
	return Config{
		Interval: Duration{2 * time.Second},
		Agents:   []string{"claude"},
		Sessions: Sessions{
			CompactAt:   85,
			ClearIdle:   Duration{30 * time.Minute},
			ClearMinPct: 20,
			ClearStale:  Duration{24 * time.Hour},
		},
		Alerts: Alerts{
			TempOverC: 95,
			Load1For:  Duration{2 * time.Minute},
			PSICPUFor: Duration{2 * time.Minute},
		},
		Orphans: Orphans{
			CPUOver:  80,
			For:      Duration{30 * time.Second},
			MinAge:   Duration{2 * time.Minute},
			Commands: []string{"sh", "bash", "zsh", "dash", "fish"},
		},
		Git: Git{Interval: Duration{5 * time.Minute}, Main: "main", Remote: "origin"},
	}
}

// Path is $XDG_CONFIG_HOME/builder/tui.toml, defaulting to ~/.config.
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "builder", "tui.toml")
}

// Load overlays the file on the defaults. A missing file is not an error.
func Load(path string) (Config, error) {
	c := Default()
	if _, err := toml.DecodeFile(path, &c); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return c, err
	}
	for i := range c.Groups {
		re, err := regexp.Compile(c.Groups[i].Container)
		if err != nil {
			return c, errors.New("groups." + c.Groups[i].Name + ": " + err.Error())
		}
		c.Groups[i].re = re
	}
	return c, nil
}
