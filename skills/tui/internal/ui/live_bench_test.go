package ui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

// BenchmarkLiveRender renders this machine's snapshot. Opt-in: LIVE=1.
func BenchmarkLiveRender(b *testing.B) {
	if os.Getenv("LIVE") == "" {
		b.Skip("set LIVE=1 to render this machine")
	}
	home, _ := os.UserHomeDir()
	s := &sample.Sampler{FS: procfs.FS{Root: "/proc"}, Sys: procfs.Sys{Root: "/sys"}, Home: home}
	s.Sample()
	m := New(Options{Source: s.Sample})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	next, _ = next.Update(snapMsg(s.Sample()))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = next.(Model).render()
	}
}
