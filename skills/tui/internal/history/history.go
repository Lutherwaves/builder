// Package history keeps machine trends across restarts: an append-only CSV
// the cockpit writes, plus any other CSV with a ts column to read alongside.
package history

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

// Every is the spacing of history points; finer samples are dropped.
const Every = 30 * time.Second

// Keep bounds what the panel holds in memory.
const Keep = 24 * time.Hour

type Point struct {
	At         time.Time
	CPU        float64
	Load1      float64
	PSICPU     float64
	PSIIO      float64
	TempC      float64
	MemAvailGB float64
	SwapGB     float64
}

// Columns is the CSV header. Readers match columns by name, so a file from
// another logger that uses some of these names can be charted too.
var Columns = []string{"ts", "cpu_pct", "load1", "psi_cpu10", "psi_io10", "pkg_temp_c", "mem_avail_gb", "swap_used_gb"}

func FromSnapshot(s sample.Snapshot) Point {
	m := s.Machine
	return Point{
		At: s.At, CPU: m.CPU, Load1: m.Load[0], PSICPU: m.PSI.CPU, PSIIO: m.PSI.IO, TempC: m.TempC,
		MemAvailGB: float64(m.MemAvail) / (1 << 30), SwapGB: float64(m.SwapUsed) / (1 << 30),
	}
}

func (p Point) row() []string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
	return []string{p.At.Format(time.RFC3339), f(p.CPU), f(p.Load1), f(p.PSICPU), f(p.PSIIO), f(p.TempC), f(p.MemAvailGB), f(p.SwapGB)}
}

type Series struct{ Points []Point }

// Append adds a point if it is at least Every after the last one.
func (s *Series) Append(p Point) {
	if n := len(s.Points); n > 0 && p.At.Sub(s.Points[n-1].At) < Every {
		return
	}
	s.Points = append(s.Points, p)
	s.trim(p.At)
}

// Merge folds newer points in, keeping time order.
func (s *Series) Merge(o Series) {
	s.Points = append(s.Points, o.Points...)
	sort.SliceStable(s.Points, func(i, j int) bool { return s.Points[i].At.Before(s.Points[j].At) })
	if n := len(s.Points); n > 0 {
		s.trim(s.Points[n-1].At)
	}
}

func (s *Series) trim(now time.Time) {
	cut := 0
	for cut < len(s.Points) && now.Sub(s.Points[cut].At) > Keep {
		cut++
	}
	s.Points = s.Points[cut:]
}

// Logger appends one row every Every. It is called from one goroutine at a
// time (after each sample), so it needs no lock.
type Logger struct {
	Path string
	last time.Time
}

func (l *Logger) Log(p Point) error {
	if l.Path == "" || p.At.Sub(l.last) < Every {
		return nil
	}
	l.last = p.At
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if st, err := f.Stat(); err == nil && st.Size() == 0 {
		w.Write(Columns)
	}
	w.Write(p.row())
	w.Flush()
	return w.Error()
}

// Load reads the last Keep of every file, matching columns by name. Missing
// files are skipped; a file without a ts column is an error.
func Load(now time.Time, paths ...string) (Series, error) {
	var s Series
	for _, path := range paths {
		if path == "" {
			continue
		}
		pts, err := readCSV(path, now.Add(-Keep))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return s, fmt.Errorf("%s: %w", path, err)
		}
		s.Points = append(s.Points, pts...)
	}
	s.Merge(Series{})
	return s, nil
}

func readCSV(path string, since time.Time) ([]Point, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReader(f))
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range head {
		col[h] = i
	}
	ts, ok := col["ts"]
	if !ok {
		return nil, fmt.Errorf("no ts column")
	}
	var out []Point
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		get := func(name string) float64 {
			i, ok := col[name]
			if !ok || i >= len(rec) {
				return 0
			}
			v, _ := strconv.ParseFloat(rec[i], 64)
			return v
		}
		if ts >= len(rec) {
			continue
		}
		at, err := time.Parse(time.RFC3339, rec[ts])
		if err != nil || at.Before(since) {
			continue
		}
		out = append(out, Point{
			At: at, CPU: get("cpu_pct"), Load1: get("load1"), PSICPU: get("psi_cpu10"), PSIIO: get("psi_io10"),
			TempC: get("pkg_temp_c"), MemAvailGB: get("mem_avail_gb"), SwapGB: get("swap_used_gb"),
		})
	}
	return out, nil
}

// DefaultLog is $XDG_DATA_HOME/builder/tui-history.csv.
func DefaultLog() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "builder", "tui-history.csv")
}
