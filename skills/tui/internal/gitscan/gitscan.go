// Package gitscan reports on the checkouts agents work in: dirty paths, how
// far the default branch is behind its remote, and which worktrees are safe
// to remove. Every git call is read-only and skips optional locks, so it
// never contends with the agents' own git use.
package gitscan

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Report struct {
	At       time.Time     `json:"at"`
	Took     time.Duration `json:"took_ns"` // this scan's own wall time
	Repos    []Repo        `json:"repos"`
	Errors   []string      `json:"errors,omitempty"`
	Watching []string      `json:"watching"`
}

type Repo struct {
	Path      string     `json:"path"` // the main checkout
	Main      string     `json:"main"`
	Behind    int        `json:"main_behind"` // commits <remote>/<main> has that <main> lacks
	Worktrees []Worktree `json:"worktrees"`
}

type Worktree struct {
	Path     string   `json:"path"`
	Branch   string   `json:"branch"`
	Dirty    int      `json:"dirty"` // -1: not checked
	Sessions int      `json:"sessions"`
	Merged   bool     `json:"merged"`
	InUse    bool     `json:"in_use"` // a process has its cwd inside
	Safe     bool     `json:"safe"`   // merged, clean, unused
	Remove   []string `json:"remove,omitempty"`
}

// Input is what the live sampler knows: which directories sessions and
// other processes sit in.
type Input struct {
	SessionCwds []string
	ProcessCwds []string
	Extra       []string // configured repositories
	Main        string
	Remote      string
}

// Runner runs git with the given arguments in dir.
type Runner func(ctx context.Context, dir string, args ...string) ([]byte, error)

func Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	return cmd.Output()
}

// Merged lists branch names whose pull requests merged. Nil when unknown.
type Merged func(ctx context.Context, repo string) map[string]bool

// GHMerged asks the GitHub CLI; squash merges leave no trace in git itself.
func GHMerged(ctx context.Context, repo string) map[string]bool {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "gh", "pr", "list", "--state", "merged", "--limit", "1000",
		"--json", "headRefName", "--jq", ".[].headRefName")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	for _, b := range strings.Fields(string(out)) {
		set[b] = true
	}
	return set
}

// Toplevel walks up from dir to the directory holding .git (a directory for a
// main checkout, a file for a linked worktree) without running git.
func Toplevel(dir string) string {
	for d := filepath.Clean(dir); d != "/" && d != "."; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
	}
	return ""
}

// mainCheckout resolves a linked worktree to its main checkout by reading
// the gitdir line of its .git file.
func mainCheckout(top string) string {
	b, err := os.ReadFile(filepath.Join(top, ".git"))
	if err != nil {
		return top // .git is a directory: already the main checkout
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir: ")
	if !ok {
		return top
	}
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(top, gitdir)
	}
	// <main>/.git/worktrees/<name>
	if i := strings.Index(gitdir, string(filepath.Separator)+".git"+string(filepath.Separator)+"worktrees"+string(filepath.Separator)); i >= 0 {
		return gitdir[:i]
	}
	return top
}

func within(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

// Scanner remembers dirty counts between scans. Checkouts sessions use are
// counted every scan; a merged worktree only every RecheckIdle, because it
// can number in the hundreds and `git worktree remove` (without --force)
// refuses a dirty or untracked worktree at action time anyway.
type Scanner struct {
	Git         Runner
	Merged      Merged
	RecheckIdle time.Duration
	Now         func() time.Time
	dirty       map[string]dirtyAt
	merged      map[string]mergedAt
}

// mergedEvery bounds how often merged-branch data is fetched per repository.
const mergedEvery = 10 * time.Minute

type mergedAt struct {
	set map[string]bool
	at  time.Time
}

type dirtyAt struct {
	n  int
	at time.Time
}

// Scan reports on every repository a session works in plus the extra ones.
func Scan(ctx context.Context, in Input, git Runner, merged Merged) Report {
	return (&Scanner{Git: git, Merged: merged}).Scan(ctx, in)
}

func (s *Scanner) Scan(ctx context.Context, in Input) Report {
	if s.dirty == nil {
		s.dirty, s.merged = map[string]dirtyAt{}, map[string]mergedAt{}
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	git, merged := s.Git, s.Merged
	start := time.Now()
	rep := Report{At: start}
	if in.Remote == "" {
		in.Remote = "origin"
	}

	sessionsIn := map[string]int{}
	repos := map[string]bool{}
	for _, cwd := range in.SessionCwds {
		if top := Toplevel(cwd); top != "" {
			sessionsIn[top]++
			repos[mainCheckout(top)] = true
		}
	}
	for _, r := range in.Extra {
		if top := Toplevel(r); top != "" {
			repos[mainCheckout(top)] = true
		}
	}
	for r := range repos {
		rep.Watching = append(rep.Watching, r)
	}
	sort.Strings(rep.Watching)

	for _, path := range rep.Watching {
		r, err := s.scanRepo(ctx, path, in, sessionsIn, git, merged)
		if err != nil {
			rep.Errors = append(rep.Errors, path+": "+err.Error())
			continue
		}
		rep.Repos = append(rep.Repos, r)
	}
	rep.Took = time.Since(start)
	return rep
}

func (s *Scanner) scanRepo(ctx context.Context, path string, in Input, sessionsIn map[string]int, git Runner, merged Merged) (Repo, error) {
	def := in.Main
	if def == "" {
		def = defaultBranch(ctx, git, path, in.Remote)
	}
	r := Repo{Path: path, Main: def}
	if out, err := git(ctx, path, "rev-list", "--count", "refs/heads/"+def+"..refs/remotes/"+in.Remote+"/"+def); err == nil {
		r.Behind, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	out, err := git(ctx, path, "worktree", "list", "--porcelain")
	if err != nil {
		return r, err
	}
	r.Worktrees = parseWorktrees(out)

	var mergedSet map[string]bool
	if c, ok := s.merged[path]; ok && s.Now().Sub(c.at) < mergedEvery {
		mergedSet = c.set
	} else if merged != nil {
		mergedSet = merged(ctx, path)
		s.merged[path] = mergedAt{mergedSet, s.Now()}
	}
	if mergedSet == nil {
		mergedSet = map[string]bool{}
		if out, err := git(ctx, path, "branch", "--format=%(refname:short)", "--merged", "refs/remotes/"+in.Remote+"/"+def); err == nil {
			for _, b := range strings.Fields(string(out)) {
				mergedSet[b] = true
			}
		}
	}

	for i := range r.Worktrees {
		wt := &r.Worktrees[i]
		wt.Dirty = -1
		wt.Sessions = sessionsIn[wt.Path]
		for _, cwd := range append(in.SessionCwds, in.ProcessCwds...) {
			if within(cwd, wt.Path) {
				wt.InUse = true
				break
			}
		}
		main := i == 0
		wt.Merged = !main && wt.Branch != "" && wt.Branch != def && mergedSet[wt.Branch]
		switch {
		case wt.Sessions > 0 || main:
			wt.Dirty = s.count(ctx, git, wt.Path, 0)
		case wt.Merged:
			wt.Dirty = s.count(ctx, git, wt.Path, s.RecheckIdle)
		}
		wt.Safe = wt.Merged && wt.Dirty == 0 && !wt.InUse
		if wt.Safe {
			wt.Remove = []string{"git", "-C", path, "worktree", "remove", wt.Path}
		}
	}
	return r, nil
}

// defaultBranch is the branch the remote's HEAD points at (main, master,
// develop...), or "main" when the clone never recorded it.
func defaultBranch(ctx context.Context, git Runner, path, remote string) string {
	out, err := git(ctx, path, "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD")
	if b, ok := strings.CutPrefix(strings.TrimSpace(string(out)), remote+"/"); err == nil && ok && b != "" {
		return b
	}
	return "main"
}

// count returns a cached dirty count younger than maxAge, or counts again.
func (s *Scanner) count(ctx context.Context, git Runner, dir string, maxAge time.Duration) int {
	if c, ok := s.dirty[dir]; ok && s.Now().Sub(c.at) < maxAge {
		return c.n
	}
	n := dirty(ctx, git, dir)
	s.dirty[dir] = dirtyAt{n, s.Now()}
	return n
}

// dirty counts changed and untracked paths; -1 when git fails.
func dirty(ctx context.Context, git Runner, dir string) int {
	out, err := git(ctx, dir, "status", "--porcelain=v1", "-z", "--untracked-files=normal", "--no-renames")
	if err != nil {
		return -1
	}
	return bytes.Count(out, []byte{0})
}

func parseWorktrees(b []byte) []Worktree {
	var out []Worktree
	for _, block := range strings.Split(strings.TrimSpace(string(b)), "\n\n") {
		var wt Worktree
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "branch "):
				wt.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
			}
		}
		if wt.Path != "" {
			out = append(out, wt)
		}
	}
	return out
}

// VerifyRemovable checks, now, that a worktree has no tracked or untracked
// changes and that no process works inside it.
func VerifyRemovable(ctx context.Context, git Runner, path string, processCwds []string) error {
	for _, cwd := range processCwds {
		if within(cwd, path) {
			return fmt.Errorf("a process works in %s", cwd)
		}
	}
	switch n := dirty(ctx, git, path); {
	case n < 0:
		return fmt.Errorf("git status failed in %s", path)
	case n > 0:
		return fmt.Errorf("%s has %d changed or untracked paths", path, n)
	}
	return nil
}
