package harness

import (
	"reflect"
	"testing"
)

const home = "/h"

func env(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExplicitWinsOverEnv(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CURSOR_VERSION", "x"), "codex"), "codex")
}
func TestBuilderHarnessEnv(t *testing.T) {
	eq(t, Detect(env("BUILDER_HARNESS", "opencode", "CLAUDECODE", "1"), ""), "opencode")
}
func TestInvalidExplicitFallsThrough(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1"), "vim"), "claude")
}
func TestAutoFallsThrough(t *testing.T) { eq(t, Detect(env("CLAUDECODE", "1"), "auto"), "claude") }
func TestInvalidBuilderHarnessFallsThrough(t *testing.T) {
	eq(t, Detect(env("BUILDER_HARNESS", "vim", "CLAUDECODE", "1"), ""), "claude")
}
func TestInheritedCursorMarkerDoesNotHideClaude(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CLAUDE_PROJECT_DIR", "/p", "CURSOR_VERSION", "1.0"), "auto"), "claude")
}
func TestCursorMarkerAloneIsUnknown(t *testing.T) {
	eq(t, Detect(env("CURSOR_PLUGIN_ROOT", "/x"), ""), "unknown")
}
func TestCursorArrivesOnlyExplicitly(t *testing.T) {
	eq(t, Detect(env("CLAUDECODE", "1", "CURSOR_VERSION", "1"), "cursor"), "cursor")
	eq(t, Detect(env("BUILDER_HARNESS", "cursor", "CLAUDECODE", "1"), ""), "cursor")
}
func TestClaudePluginRootIsClaude(t *testing.T) {
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), ""), "claude")
}
func TestCodexOnlyViaExplicitOrBuilderHarness(t *testing.T) {
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), ""), "claude")
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x"), "codex"), "codex")
	eq(t, Detect(env("CLAUDE_PLUGIN_ROOT", "/x", "BUILDER_HARNESS", "codex"), ""), "codex")
}
func TestEmptyEnvUnknown(t *testing.T) { eq(t, Detect(env(), ""), "unknown") }

func TestDefault(t *testing.T)  { eq(t, BuilderHome(env(), home), "/h/.config/builder") }
func TestOverride(t *testing.T) { eq(t, BuilderHome(env("BUILDER_HOME", "/b c"), home), "/b c") }
func TestEmptyOverrideIsDefault(t *testing.T) {
	eq(t, BuilderHome(env("BUILDER_HOME", ""), home), "/h/.config/builder")
}
