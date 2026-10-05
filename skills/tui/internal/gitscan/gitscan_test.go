package gitscan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo builds a main checkout one commit behind origin/main, with four
// linked worktrees: merged+clean, merged+untracked file, merged+in use, and
// an open branch a session works in.
func newRepo(t *testing.T) (root, repo string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ = filepath.EvalSymlinks(t.TempDir())
	repo = filepath.Join(root, "app")
	run(t, root, "init", "-q", "-b", "main", repo)
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0o644)
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-qm", "a")
	run(t, repo, "commit", "-q", "--allow-empty", "-m", "b")
	run(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	run(t, repo, "reset", "-q", "--hard", "HEAD~1")
	for _, b := range []string{"done", "untracked", "busy", "open"} {
		run(t, repo, "worktree", "add", "-q", "-b", b, filepath.Join(root, "wt", b))
	}
	os.WriteFile(filepath.Join(root, "wt", "untracked", "new.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "wt", "open", "a.txt"), []byte("changed"), 0o644)
	return root, repo
}

func TestScan(t *testing.T) {
	root, repo := newRepo(t)
	merged := func(context.Context, string) map[string]bool {
		return map[string]bool{"done": true, "untracked": true, "busy": true}
	}
	in := Input{
		SessionCwds: []string{filepath.Join(root, "wt", "open", "sub"), repo},
		ProcessCwds: []string{filepath.Join(root, "wt", "busy")},
	}
	os.MkdirAll(filepath.Join(root, "wt", "open", "sub"), 0o755)
	rep := Scan(context.Background(), in, Git, merged)
	if len(rep.Errors) > 0 || len(rep.Repos) != 1 {
		t.Fatalf("report = %+v", rep)
	}
	r := rep.Repos[0]
	if r.Path != repo || r.Behind != 1 || len(r.Worktrees) != 5 {
		t.Fatalf("repo = %+v", r)
	}
	got := map[string]Worktree{}
	for _, wt := range r.Worktrees {
		got[filepath.Base(wt.Path)] = wt
	}
	if wt := got["done"]; !wt.Safe || wt.Dirty != 0 || strings.Join(wt.Remove, " ") != "git -C "+repo+" worktree remove "+wt.Path {
		t.Errorf("done = %+v", wt)
	}
	if wt := got["untracked"]; wt.Safe || wt.Dirty != 1 {
		t.Errorf("an untracked file must block removal: %+v", wt)
	}
	if wt := got["busy"]; wt.Safe || !wt.InUse {
		t.Errorf("a process inside must block removal: %+v", wt)
	}
	if wt := got["open"]; wt.Safe || wt.Merged || wt.Sessions != 1 || wt.Dirty != 1 {
		t.Errorf("open = %+v", wt)
	}
	if wt := got["app"]; wt.Sessions != 1 || wt.Dirty != 0 || wt.Branch != "main" {
		t.Errorf("main checkout = %+v", wt)
	}
}

func TestScanFallsBackToGitMerged(t *testing.T) {
	root, repo := newRepo(t)
	// No PR data: only branches git itself sees merged count. All four
	// branches point at main's commit, which origin/main contains.
	rep := Scan(context.Background(), Input{SessionCwds: []string{repo}}, Git, func(context.Context, string) map[string]bool { return nil })
	n := 0
	for _, wt := range rep.Repos[0].Worktrees {
		if wt.Merged {
			n++
		}
	}
	if n != 4 {
		t.Fatalf("want 4 merged by git, got %d (%s)", n, root)
	}
}

func TestToplevelAndMainCheckout(t *testing.T) {
	root, repo := newRepo(t)
	wt := filepath.Join(root, "wt", "open")
	os.MkdirAll(filepath.Join(wt, "deep", "er"), 0o755)
	if got := Toplevel(filepath.Join(wt, "deep", "er")); got != wt {
		t.Fatalf("toplevel = %q", got)
	}
	if got := mainCheckout(wt); got != repo {
		t.Fatalf("main checkout = %q", got)
	}
	if got := Toplevel(root); got != "" {
		t.Fatalf("no repo above %s, got %q", root, got)
	}
}

func TestScannerCachesIdleWorktrees(t *testing.T) {
	root, repo := newRepo(t)
	calls := map[string]int{}
	counting := func(ctx context.Context, dir string, args ...string) ([]byte, error) {
		if args[0] == "status" {
			calls[filepath.Base(dir)]++
		}
		return Git(ctx, dir, args...)
	}
	merged := func(context.Context, string) map[string]bool { return map[string]bool{"done": true} }
	s := &Scanner{Git: counting, Merged: merged, RecheckIdle: 30 * time.Minute}
	in := Input{SessionCwds: []string{filepath.Join(root, "wt", "open")}}
	s.Scan(context.Background(), in)
	s.Scan(context.Background(), in)
	if calls["open"] != 2 || calls["app"] != 2 {
		t.Errorf("checkouts in use must be counted every scan: %v", calls)
	}
	if calls["done"] != 1 {
		t.Errorf("a merged idle worktree must come from the cache: %v (%s)", calls, repo)
	}
}

func TestScanFindsTheRemotesDefaultBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	repo := filepath.Join(root, "app")
	run(t, root, "init", "-q", "-b", "trunk", repo)
	run(t, repo, "commit", "-q", "--allow-empty", "-m", "a")
	run(t, repo, "commit", "-q", "--allow-empty", "-m", "b")
	run(t, repo, "update-ref", "refs/remotes/origin/trunk", "HEAD")
	run(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	run(t, repo, "reset", "-q", "--hard", "HEAD~1")
	run(t, repo, "worktree", "add", "-q", "-b", "done", filepath.Join(root, "wt", "done"))

	rep := Scan(context.Background(), Input{SessionCwds: []string{repo}}, Git, func(context.Context, string) map[string]bool { return nil })
	r := rep.Repos[0]
	if r.Main != "trunk" || r.Behind != 1 {
		t.Fatalf("want trunk 1 behind, got %s %d behind", r.Main, r.Behind)
	}
	if wt := r.Worktrees[1]; !wt.Merged {
		t.Fatalf("a branch merged into trunk must count as merged: %+v", wt)
	}
}
