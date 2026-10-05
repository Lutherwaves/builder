package ui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"github.com/Lutherwaves/builder/skills/tui/internal/procfs"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
	"github.com/Lutherwaves/builder/skills/tui/internal/state"
)

const fixture = "../../testdata/machine"

func fixtureSource() Source {
	s := &sample.Sampler{
		FS:    procfs.FS{Root: filepath.Join(fixture, "proc")},
		Home:  filepath.Join(fixture, "home"),
		Tmux:  func() ([]byte, error) { return os.ReadFile(filepath.Join(fixture, "tmux-panes.txt")) },
		Rules: sample.DefaultRules,
	}
	return s.Sample
}

func loaded(t *testing.T, width int) Model {
	t.Helper()
	m := New(Options{Source: fixtureSource()})
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
	next, _ = next.Update(snapMsg(fixtureSource()()))
	return next.(Model)
}

func TestViewFitsWidth(t *testing.T) {
	for _, w := range []int{80, 100, 160} {
		out := loaded(t, w).render()
		for i, line := range strings.Split(out, "\n") {
			if lw := lipgloss.Width(line); lw > w {
				t.Fatalf("width %d: line %d is %d wide: %q", w, i, lw, line)
			}
		}
		for _, want := range []string{"app-feature", "app-review", "! compact", "▲ clear", "fresh", "1 without an agent: notes:1.1 zsh"} {
			if !strings.Contains(out, want) {
				t.Fatalf("width %d: view lacks %q:\n%s", w, want, out)
			}
		}
	}
}

func key(m tea.Model, k string) (tea.Model, tea.Cmd) {
	r := []rune(k)
	msg := tea.KeyPressMsg{Text: k, Code: r[0]}
	switch k {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEsc}
	}
	return m.Update(msg)
}

func TestJumpPrintsCommandThenRunsOnConfirm(t *testing.T) {
	var ran []string
	m := loaded(t, 100)
	m.opt.Run = func(argv []string) error { ran = argv; return nil }

	next, _ := key(m, "enter")
	out := next.(Model).render()
	if !strings.Contains(out, "tmux switch-client -t %1") || ran != nil {
		t.Fatalf("enter must print the command and wait; ran=%v\n%s", ran, out)
	}
	next, cmd := key(next, "enter")
	if cmd == nil {
		t.Fatal("confirm returned no command")
	}
	next.Update(cmd())
	if strings.Join(ran, " ") != "tmux switch-client -t %1" {
		t.Fatalf("ran %v", ran)
	}
}

func TestEscCancelsAction(t *testing.T) {
	m := loaded(t, 100)
	m.opt.Run = func([]string) error { t.Fatal("must not run"); return nil }
	next, _ := key(m, "enter")
	next, cmd := key(next, "esc")
	if cmd != nil || !strings.Contains(next.(Model).render(), "cancelled") {
		t.Fatal("esc must cancel without running")
	}
}

func TestDestructiveNeedsTypedYes(t *testing.T) {
	m := loaded(t, 100)
	m.opt.Run = func([]string) error { t.Fatal("enter must not confirm a destructive action"); return nil }
	m.pending = &action{steps: [][]string{{"kill", "123"}}, destructive: true}
	if _, cmd := key(m, "enter"); cmd != nil {
		t.Fatal("enter ran a destructive action")
	}
}

func TestSelectionFollowsPID(t *testing.T) {
	m := loaded(t, 100)
	next, _ := key(m, "down")
	sel, _ := next.(Model).selected()
	next, _ = next.Update(snapMsg(fixtureSource()()))
	again, _ := next.(Model).selected()
	if sel.PID != again.PID {
		t.Fatalf("selection moved from %d to %d on resample", sel.PID, again.PID)
	}
}

// syncBuffer lets the program write while the test reads.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// TestSmoke runs the real program against the fixture machine, waits for the
// first rendered sample, and quits.
func TestSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out syncBuffer
	sampled := make(chan struct{}, 1)
	m := New(Options{
		Source:   fixtureSource(),
		Interval: 50 * time.Millisecond,
		OnSnapshot: func(state.Doc) {
			select {
			case sampled <- struct{}{}:
			default:
			}
		},
	})
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(&out), tea.WithWindowSize(100, 30))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	select {
	case <-sampled:
	case <-ctx.Done():
		t.Fatal("no sample within 10s")
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "app-feature") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	p.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "app-feature") {
		t.Fatalf("rendered output lacks the fixture session:\n%q", out.String())
	}
}

func TestGuardsSkipStaleSteps(t *testing.T) {
	var ran []string
	run := func(argv []string) error { ran = append(ran, argv[len(argv)-1]); return nil }
	check := func(g advice.Guard) error {
		if g.Worktree == "/w/b" {
			return fmt.Errorf("/w/b has 1 changed or untracked paths")
		}
		return nil
	}
	a := action{
		steps:  [][]string{{"rm", "/w/a"}, {"rm", "/w/b"}, {"type", "%1"}, {"type", "%2"}},
		guards: []advice.Guard{{Worktree: "/w/a"}, {Worktree: "/w/b"}, {IdlePane: "%1"}, {IdlePane: "%2"}},
	}
	msg := runAction(a, run, check, map[string]bool{"%1": true, "%2": false})
	if strings.Join(ran, ",") != "/w/a,%1" {
		t.Fatalf("ran %v", ran)
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "ran 2, skipped 2") {
		t.Fatalf("result %v", msg.err)
	}
}

func TestRanMsgStartsNoSecondSampleLoop(t *testing.T) {
	m := loaded(t, 100)
	if _, cmd := m.Update(ranMsg{cmd: "x"}); cmd != nil {
		t.Fatal("a finished action must not start another sample; the tick loop already runs")
	}
}

// A confirmed step must never stop to ask for credentials on the cockpit's
// terminal.
func TestRunNeverPrompts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	if err := run([]string{"sh", "-c", `test "$GIT_TERMINAL_PROMPT" = 0 && test -n "$GIT_SSH_COMMAND"`}); err != nil {
		t.Fatalf("step ran with prompts allowed: %v", err)
	}
}
