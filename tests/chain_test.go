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
