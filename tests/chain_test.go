// The five stages chain by themselves: each one names the next, and both sides of a handoff name the same record.
package tests

import (
	"strings"
	"testing"
)

const reviewPath = "${BUILDER_HOME:-$HOME/.config/builder}/reviews/<repo>/<branch>.md"

func TestEachStageLoadsTheNext(t *testing.T) {
	for stage, next := range map[string]string{"spec": "plan", "plan": "build", "build": "review", "review": "ship"} {
		if !strings.Contains(read(t, stage, "SKILL.md"), "oad `builder:"+next+"`") && !strings.Contains(read(t, stage, "SKILL.md"), "start `builder:"+next+"`") {
			t.Errorf("skills/%s/SKILL.md does not say to load builder:%s", stage, next)
		}
	}
	// A failed review gate goes back to build, never forward.
	if !strings.Contains(read(t, "review", "SKILL.md"), "go back to `builder:build`") {
		t.Error("skills/review/SKILL.md does not send a failed gate back to builder:build")
	}
}

func TestTheSoulNamesEachStageSkill(t *testing.T) {
	soul := read(t, "builder", "soul-compact.md")
	for _, stage := range []string{"spec", "plan", "build", "review", "ship"} {
		if !strings.Contains(soul, "`builder:"+stage+"`") {
			t.Errorf("the soul does not name builder:%s", stage)
		}
	}
	if !strings.Contains(soul, "When work enters a stage, load its skill first") {
		t.Error("the soul does not say to load the skill of a stage")
	}
}

func TestBothSidesOfAHandoffNameTheSameRecord(t *testing.T) {
	plan, build := read(t, "plan", "SKILL.md"), read(t, "build", "SKILL.md")
	for _, line := range []string{"Spec:", "Test command:", "Plan approved:"} {
		if !strings.Contains(plan, "\n"+line+" <") {
			t.Errorf("the plan header has no line %q", line)
		}
		if !strings.Contains(build, "`"+strings.TrimSuffix(line, ":")) {
			t.Errorf("skills/build/SKILL.md does not read the header line %q", line)
		}
	}
	if !strings.Contains(build, "git log --oneline") {
		t.Error("skills/build/SKILL.md does not resume from the task commits")
	}
	for _, stage := range []string{"review", "ship"} {
		s := read(t, stage, "SKILL.md")
		if !strings.Contains(s, reviewPath) || !strings.Contains(s, "`Head:") {
			t.Errorf("skills/%s/SKILL.md does not name the review path and its Head line", stage)
		}
	}
}

func TestNoStageIsStillNamedShape(t *testing.T) {
	for _, stage := range []string{"builder", "spec", "plan", "build", "review", "ship"} {
		if strings.Contains(read(t, stage, "SKILL.md"), "`shape`") {
			t.Errorf("skills/%s/SKILL.md still names `shape`", stage)
		}
	}
}

// debug is outside the five stages: build loads it when a test fails, and it hands a bug fix to review.
func TestAFailureGoesToDebugAndComesBack(t *testing.T) {
	mustContain(t, read(t, "build", "SKILL.md"), "Load `builder:debug`")
	debug := read(t, "debug", "SKILL.md")
	for _, want := range []string{"load `builder:review`", "step up to `builder:spec`", "After the third fix that did not hold, stop"} {
		mustContain(t, debug, want)
	}
	for _, f := range []string{"SKILL.md", "soul-compact.md"} {
		mustContain(t, read(t, "builder", f), "`builder:debug`")
	}
}

// One list of slop, one owner: build runs the pass over its own diff, and review reads the same file.
func TestBuildAndReviewShareOneSlopList(t *testing.T) {
	mustContain(t, read(t, "build", "SKILL.md"), "Load `deslop.md`, next to this file")
	mustContain(t, read(t, "build", "SKILL.md"), "Remove each item under its Slop heading")
	mustContain(t, read(t, "review", "SKILL.md"), "`../build/deslop.md`")
	mustContain(t, read(t, "review", "SKILL.md"), "Declined to judge · Slop · Verdict")
	mustContain(t, read(t, "review", "bar.md"), "Slop is not a finding")
	deslop := read(t, "build", "deslop.md")
	for _, want := range []string{"## The pass", "## The list", "## What is never slop", "Remove, do not rewrite"} {
		mustContain(t, deslop, want)
	}
}

// A handoff says `builder:<name>`. Only Claude Code and Codex show that name, so each other map says what it means.
func TestEachHarnessMapSaysHowToLoadAStage(t *testing.T) {
	for _, h := range []string{"claude", "codex", "cursor", "grok", "opencode"} {
		m := read(t, "builder", "harnesses", h+".md")
		hook, _, _ := strings.Cut(m, "<!-- /hook -->")
		if !strings.Contains(hook, "builder:<name>") {
			t.Errorf("the session-start block of harnesses/%s.md does not say what builder:<name> means", h)
		}
		mustContain(t, m, "| load a skill (and its syntax) |")
	}
}

// A skill that runs the binary still finds it when no session-start hook named the builder root.
func TestTheBinaryPathHasAFallbackWithoutTheHook(t *testing.T) {
	for _, f := range [][]string{{"security", "SKILL.md"}, {"review", "security-pass.md"}} {
		mustContain(t, read(t, f...), "With no \"builder root\" line from the session start")
	}
}

// The architecture lens has one owner: plan runs it, review checks against the same file, and the project's own rules win.
func TestPlanAndReviewShareTheArchitectureLens(t *testing.T) {
	mustContain(t, read(t, "plan", "SKILL.md"), "Load `architecture.md` and run its lens")
	mustContain(t, read(t, "review", "SKILL.md"), "`../plan/architecture.md`")
	lens := read(t, "plan", "architecture.md")
	for _, want := range []string{"The project's own rules win", "## The contract", "## The boundaries", "## More than one customer", "## Tools for agents", "## Guards", "## Questions with no default"} {
		mustContain(t, lens, want)
	}
	for _, f := range []string{"SKILL.md", "soul-compact.md"} {
		mustContain(t, read(t, "builder", f), "fails when it checks nothing")
	}
}
