package sample

import (
	"os"
	"testing"

	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
)

// BenchmarkLiveSample measures one sample of this machine. Opt-in: LIVE=1.
func BenchmarkLiveSample(b *testing.B) {
	if os.Getenv("LIVE") == "" {
		b.Skip("set LIVE=1 to sample this machine")
	}
	home, _ := os.UserHomeDir()
	s := &Sampler{FS: procfs.FS{Root: "/proc"}, Home: home, Rules: DefaultRules}
	s.Sample()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sample()
	}
}
