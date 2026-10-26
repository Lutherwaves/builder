// Package hook builds the SessionStart text: the builder soul and at most one next step.
package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/internal/harness"
	"github.com/Lutherwaves/builder/internal/ledger"
	"github.com/Lutherwaves/builder/internal/signals"
)

func commands(root string) (answer, onboarded, next string) {
	shim := `"` + filepath.Join(root, "bin", "builder") + `" ledger`
	return shim + " answer %s yes|later|never", shim + " onboarded", shim + " next --harness %s"
}

func block(root string, i ledger.Item) string {
	answer, _, _ := commands(root)
	str := func(k string) string { s, _ := i[k].(string); return s }
	return fmt.Sprintf("builder recommends: %s — %s. To do it: %s. After your answer, run: %s",
		str("say"), strings.TrimRight(str("why"), "."), strings.TrimRight(str("how"), "."), fmt.Sprintf(answer, str("id")))
}

func onboardingText(root string, i ledger.Item, name string) string {
	_, onboarded, next := commands(root)
	lines := []string{
		"builder onboarding: this person is new to builder.",
		"Answer the person's request first. Do the work before anything else.",
		"At the end of that reply, introduce builder with these lines, word for word.",
		"This is required, even when the request is one line. The reply is not complete without the intro.",
		"  builder: you decide. I propose, build, and end each piece of work with a report.",
		"  Nothing leaves this machine without a send rule or your yes.",
	}
	if i != nil {
		lines = append(lines,
			"  I offer a few setup steps, one at a time.",
			"Then offer this first setup step and ask: yes, not now, or never.",
			strings.Replace(block(root, i), "builder recommends:", "Step:", 1),
			"Offer one step per reply. After each answer, offer the next due step in a later reply.",
			fmt.Sprintf("To find the next due step, run: %s (it prints the step as JSON, or null when no step is due).", fmt.Sprintf(next, name)),
			"Never invent a step. Offer only the step that this command prints.",
		)
	}
	lines = append(lines, "When all setup steps are answered, run: "+onboarded)
	return strings.Join(lines, "\n")
}

func mappingBlock(root, name string) string {
	b, err := os.ReadFile(filepath.Join(root, "skills", "builder", "harnesses", name+".md"))
	if err != nil {
		return ""
	}
	_, after, ok := strings.Cut(string(b), "<!-- hook -->")
	if !ok {
		return ""
	}
	inner, _, ok := strings.Cut(after, "<!-- /hook -->")
	if !ok {
		return ""
	}
	return strings.TrimSpace(inner)
}

func body(root, name string, env map[string]string, home, cwd string, now float64) string {
	soulBytes, _ := os.ReadFile(filepath.Join(root, "skills", "builder", "soul-compact.md"))
	soul := strings.TrimRight(string(soulBytes), "\n")
	effective := name
	if !slices.Contains(harness.Names, name) {
		effective = "claude"
	}
	cat, err := ledger.LoadCatalog(root)
	if err != nil {
		return soul
	}
	cat = ledger.ForHarness(cat, effective)
	l := ledger.Load(filepath.Join(harness.BuilderHome(env, home), "ledger.json"))
	sig := signals.Collect(cwd, 500*time.Millisecond, effective, env, home)
	if !l.Onboarded {
		i := ledger.Next(l, ledger.DueCatalog(l, cat), sig, now, ledger.CooldownS, true)
		return soul + "\n\n" + onboardingText(root, i, effective)
	}
	if i := ledger.Next(l, cat, sig, now, ledger.CooldownS, false); i != nil {
		return soul + "\n\nAnswer the person's request first. Put this as one short block at the end of your reply. Never put it before the work.\n" + block(root, i)
	}
	return soul
}

// Build is the hook text before it is shaped for the harness.
func Build(root, name string, env map[string]string, home, cwd string, now float64) string {
	text := body(root, name, env, home, cwd, now) +
		fmt.Sprintf("\n\nbuilder root: %s. Where a builder skill says ${CLAUDE_PLUGIN_ROOT}, use this path.", root)
	if m := mappingBlock(root, name); m != "" {
		return text + "\n\n" + m
	}
	return text
}

// Shape wraps the text in the JSON the harness reads.
func Shape(name, text string) string {
	var v any = map[string]any{"hookSpecificOutput": map[string]string{"hookEventName": "SessionStart", "additionalContext": text}}
	if name == "cursor" {
		v = map[string]string{"additional_context": text}
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}
