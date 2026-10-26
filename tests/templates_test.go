// The shape and canon templates keep the section order their SKILL.md lists.
//
// Eval agents cannot read the templates (they sit outside the eval's working
// directory), so the order written in each SKILL.md must match the template.
package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const skillsDir = "../skills"

func read(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{skillsDir}, parts...)...))
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(parts...), err)
	}
	return string(b)
}

func readme(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	return string(b)
}

func mustContain(t *testing.T, text, part string) {
	t.Helper()
	if !strings.Contains(text, part) {
		t.Errorf("missing %q", part)
	}
}

func mustNotContain(t *testing.T, text, part string) {
	t.Helper()
	if strings.Contains(text, part) {
		t.Errorf("must not contain %q", part)
	}
}

// listedOrder returns the " · " list that follows marker on one line of SKILL.md.
func listedOrder(t *testing.T, skillMD, marker string) []string {
	t.Helper()
	m := regexp.MustCompile(`(?m)` + regexp.QuoteMeta(marker) + `\s*(.+?)\.\s*(?:$|[A-Z])`).FindStringSubmatch(skillMD)
	if m == nil {
		t.Fatalf("SKILL.md has no line with %q", marker)
	}
	parts := strings.Split(m[1], "·")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func submatches(re, text string) []string {
	var out []string
	for _, m := range regexp.MustCompile(re).FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

func TestIssueTemplateMatchesShapeSkill(t *testing.T) {
	headings := submatches(`(?m)^## (.+?)(?: \(.*\))?$`, read(t, "shape", "issue-template.md"))
	want := listedOrder(t, read(t, "shape", "SKILL.md"), "Its sections, in order:")
	if !reflect.DeepEqual(headings, want) {
		t.Fatalf("issue-template headings %q, SKILL.md lists %q", headings, want)
	}
}

func TestRuleTemplateMatchesCanonSkill(t *testing.T) {
	lines := submatches(`(?m)^\*\*(\w+)\.\*\*`, read(t, "canon", "rule-template.md"))
	want := listedOrder(t, read(t, "canon", "SKILL.md"), "Its lines, in order:")
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("rule-template lines %q, SKILL.md lists %q", lines, want)
	}
}

var (
	reportOrder  = []string{"state", "▶ You", "⚠ Not proven", "✓ Proven", "↗ Outside", "Calls", "↻ Next"}
	reportStates = []string{"✓ Done", "▶ Needs you", "✗ Blocked"}
)

// reportLine returns the reportOrder name of a bold report label, or "".
func reportLine(label string) string {
	trimmed := strings.TrimLeft(label, "<")
	for _, s := range reportStates {
		if strings.HasPrefix(trimmed, s) {
			return "state"
		}
	}
	for _, name := range reportOrder[1:] {
		if strings.HasPrefix(label, name) {
			return name
		}
	}
	return ""
}

func reportTemplate(t *testing.T) string {
	t.Helper()
	skill := read(t, "builder", "SKILL.md")
	start := strings.Index(skill, "```markdown")
	if start < 0 {
		t.Fatal("builder SKILL.md has no markdown report template")
	}
	end := strings.Index(skill[start+3:], "```")
	if end < 0 {
		t.Fatal("builder SKILL.md report template has no closing fence")
	}
	return skill[start : start+3+end]
}

func TestReportTemplateHasEveryLineInOrder(t *testing.T) {
	var got []string
	for _, label := range submatches(`(?m)^\*\*(.+?):?\*\*`, reportTemplate(t)) {
		got = append(got, reportLine(label))
	}
	if !reflect.DeepEqual(got, reportOrder) {
		t.Fatalf("report lines %q, want %q", got, reportOrder)
	}
}

func TestTemplateLinesAreApart(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(reportTemplate(t), "\n"), "\n")[1:]
	for i := 1; i < len(lines); i += 2 {
		if lines[i] != "" {
			t.Fatalf("each report line needs a blank line after it; line %d is %q", i+1, lines[i])
		}
	}
}

func TestSoulListsTheLinesInOrder(t *testing.T) {
	soul := read(t, "builder", "soul-compact.md")
	m := regexp.MustCompile(`in this order: (.+?)\.\n`).FindStringSubmatch(soul)
	if m == nil {
		t.Fatal(`soul-compact.md has no "in this order:" list`)
	}
	listed := regexp.MustCompile(` \(.*?\)`).ReplaceAllString(m[1], "")
	want := append([]string{"the state"}, reportOrder[1:]...)
	if got := strings.Split(listed, ", "); !reflect.DeepEqual(got, want) {
		t.Fatalf("soul lists %q, want %q", got, want)
	}
	mustContain(t, soul, "not in a code block")
}

func TestReadmeExampleFollowsTheOrder(t *testing.T) {
	labels := submatches(`(?m)^> \*\*(.+?):?\*\*`, readme(t))
	if len(labels) == 0 {
		t.Fatal("README has no report example")
	}
	order := make([]int, len(labels))
	for i, l := range labels {
		name := reportLine(l)
		idx := -1
		for j, o := range reportOrder {
			if o == name {
				idx = j
			}
		}
		if idx < 0 {
			t.Fatalf("README label %q is not a report line", l)
		}
		order[i] = idx
	}
	if !sort.IntsAreSorted(order) {
		t.Fatalf("README report lines out of order: %q", labels)
	}
}

var (
	oldReportLine = regexp.MustCompile(`^(For|Evidence|Outside):\s{2,}|\bunder Evidence\b|\b(For|Evidence) line\b`)
	outsideLine   = regexp.MustCompile(`\bOutside(:| line)`)
)

// hasOldReportLine reports whether line uses a report line the template dropped.
// Python's (?<!↗ )\bOutside lookbehind is not in RE2, so the prefix is checked here.
func hasOldReportLine(line string) bool {
	if oldReportLine.MatchString(line) {
		return true
	}
	for _, loc := range outsideLine.FindAllStringIndex(line, -1) {
		if !strings.HasSuffix(line[:loc[0]], "↗ ") {
			return true
		}
	}
	return false
}

func TestOldReportLineAllowsOutsideArrow(t *testing.T) {
	cases := []struct {
		line string
		old  bool
	}{
		{"**↗ Outside:** nothing", false},
		{"Outside:   x", true},
		{"The Outside line is gone", true},
		{"Outside: x", true},
		{"under Evidence", true},
	}
	for _, c := range cases {
		if got := hasOldReportLine(c.line); got != c.old {
			t.Errorf("hasOldReportLine(%q) = %v, want %v", c.line, got, c.old)
		}
	}
}

func TestNoSkillUsesAnOldReportLine(t *testing.T) {
	// A review or security finding keeps its own "Evidence:" field; only the old report's aligned labels and names fail.
	err := filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for n, line := range strings.SplitAfter(string(b), "\n") {
			if hasOldReportLine(line) {
				t.Errorf("%s:%d: %s", d.Name(), n+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReportAsksWithAnActionEveryHarnessMaps(t *testing.T) {
	mustContain(t, read(t, "builder", "SKILL.md"), `the "ask the person to choose" action of your harness`)
	for _, harness := range []string{"claude", "codex", "cursor", "opencode"} {
		mustContain(t, read(t, "builder", "harnesses", harness+".md"), "| ask the person to choose |")
	}
}

// builder reach lets a send rule allow a message. Every text that names the gate says so.

func TestReachTeammateRowNamesTheSendRule(t *testing.T) {
	var row string
	for _, l := range strings.Split(read(t, "reach", "SKILL.md"), "\n") {
		if strings.HasPrefix(l, "| **teammate**") {
			row = l
			break
		}
	}
	if row == "" {
		t.Fatal("reach SKILL.md has no teammate row")
	}
	mustContain(t, row, "unless a send rule allows the kind")
	mustContain(t, row, "the person's yes")
}

func TestBuilderGateTextsNameTheSendRule(t *testing.T) {
	skill := read(t, "builder", "SKILL.md")
	mustContain(t, skill, "without a send rule or the person's yes in this session")
	mustContain(t, skill, "- [ ] Nothing left the repository without a send rule or the person's yes.")
	mustContain(t, skill, "sent by rule: <n>, logged.")
	mustNotContain(t, skill, "Nothing left the repository without a yes.")
}

func TestQuestionsFollowTheDecisionBrief(t *testing.T) {
	_, after, ok := strings.Cut(read(t, "builder", "SKILL.md"), "**Questions.**")
	if !ok {
		t.Fatal(`builder SKILL.md has no "**Questions.**" section`)
	}
	section, _, _ := strings.Cut(after, "\n\n**")
	for _, part := range []string{"What is at stake", "Recommendation", "(recommended)", "pro", "con", "Net", "typed word"} {
		mustContain(t, section, part)
	}
	mustNotContain(t, section, "Completeness")
}

func whyBuilderSection(t *testing.T) string {
	t.Helper()
	_, after, ok := strings.Cut(readme(t), "## Why builder")
	if !ok {
		t.Fatal(`README has no "## Why builder" section`)
	}
	section, _, _ := strings.Cut(after, "\n## ")
	return section
}

func TestReadmeLinksResolve(t *testing.T) {
	// Python used a (?!https?://|#) lookahead; RE2 has none, so skip those targets in code.
	for _, target := range submatches(`\]\(([^)\s]+)\)`, readme(t)) {
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "#") {
			continue
		}
		path, _, _ := strings.Cut(target, "#")
		if _, err := os.Stat(filepath.Join("..", path)); err != nil {
			t.Errorf("README link %q does not resolve", target)
		}
	}
}

func TestReadmeSaysWhyBuilder(t *testing.T) {
	section := whyBuilderSection(t)
	for _, name := range []string{"superpowers", "gstack", "CLAUDE.md"} {
		mustContain(t, section, name)
	}
}

func TestWhyBuilderClaimsHold(t *testing.T) {
	section := whyBuilderSection(t)
	// superpowers also injects itself at every session start, and the report comes once per piece of work.
	mustNotContain(t, section, "when that\nskill runs")
	mustNotContain(t, section, "every reply")
	// A hand-written CLAUDE.md can ask for a reason and for proof too.
	mustNotContain(t, section, "| no |\n| Unproven")
}

func TestBudgetFindsTheOldWatcher(t *testing.T) {
	// Split so that validate.sh, which forbids the old tool name outside the budget skill, passes this file.
	mustContain(t, read(t, "budget", "SKILL.md"), "`builder budget` or the old `burn-proj"+".py`")
}
