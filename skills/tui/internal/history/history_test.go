package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)

func TestLoggerSpacingAndHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.csv")
	l := &Logger{Path: path}
	for i := 0; i < 5; i++ {
		l.Log(Point{At: t0.Add(time.Duration(i) * 10 * time.Second), CPU: float64(i)})
	}
	b, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if lines[0] != strings.Join(Columns, ",") || len(lines) != 3 {
		t.Fatalf("want header + 2 rows (0s, 30s), got:\n%s", b)
	}
}

func TestLoadMatchesColumnsByName(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other.csv")
	// Another logger: different order, extra columns, one row too old.
	os.WriteFile(other, []byte("ts,gpu0_pct,load1,cpu_pct\n"+
		t0.Add(-48*time.Hour).Format(time.RFC3339)+",1,9,9\n"+
		t0.Add(-time.Hour).Format(time.RFC3339)+",5,12.5,80\n"), 0o644)
	own := filepath.Join(dir, "own.csv")
	l := &Logger{Path: own}
	l.Log(Point{At: t0.Add(-2 * time.Hour), CPU: 10, TempC: 70})

	s, err := Load(t0, other, own, filepath.Join(dir, "missing.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Points) != 2 {
		t.Fatalf("points = %+v", s.Points)
	}
	if p := s.Points[0]; p.CPU != 10 || p.TempC != 70 {
		t.Errorf("own row first by time: %+v", p)
	}
	if p := s.Points[1]; p.CPU != 80 || p.Load1 != 12.5 {
		t.Errorf("foreign row: %+v", p)
	}
}

func TestLoadRejectsFileWithoutTs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.csv")
	os.WriteFile(path, []byte("cpu\n1\n"), 0o644)
	if _, err := Load(t0, path); err == nil {
		t.Fatal("want an error for a CSV without ts")
	}
}

func TestSeriesKeepsSpacingAndWindow(t *testing.T) {
	var s Series
	for i := 0; i < 100; i++ {
		s.Append(Point{At: t0.Add(time.Duration(i) * 20 * time.Minute)})
	}
	if first := s.Points[0].At; s.Points[len(s.Points)-1].At.Sub(first) > Keep {
		t.Fatalf("kept more than %s", Keep)
	}
	n := len(s.Points)
	s.Append(Point{At: s.Points[n-1].At.Add(time.Second)})
	if len(s.Points) != n {
		t.Fatal("a point closer than Every was kept")
	}
}
