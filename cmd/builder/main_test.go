package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func cliEnv(t *testing.T) string {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("BUILDER_HOME", filepath.Join(d, "b"))
	t.Setenv("BUILDER_ROOT", mustAbs(t, "../.."))
	t.Setenv("BUILDER_HARNESS", "")
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	return d
}

func mustAbs(t *testing.T, p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func call(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 || out.String() != "dev\n" {
		t.Fatalf("code %d out %q", code, out.String())
	}
}

func TestUnknownCommandExits2(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"nope"}, &out, &errb); code != 2 || errb.Len() == 0 {
		t.Fatalf("code %d err %q", code, errb.String())
	}
	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("no command: code %d", code)
	}
}

func TestRemovedCommandsAreGone(t *testing.T) {
	cliEnv(t)
	for _, name := range []string{"ledger", "reach", "budget"} {
		code, _, errs := call(t, name, "x")
		first, usageLine, _ := strings.Cut(errs, "\n")
		if code != 2 || first != "builder: unknown command \""+name+"\"" || strings.Contains(usageLine, name) {
			t.Errorf("%s: code %d err %q", name, code, errs)
		}
	}
}
