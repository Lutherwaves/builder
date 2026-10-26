package cli

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestParseFlagsAfterPositionals(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	today := fs.String("today", "", "")
	got, err := Parse([]string{"save", "f.json", "--today", "2026-10-01"}, fs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"save", "f.json"}) || *today != "2026-10-01" {
		t.Fatalf("got %v, today=%q", got, *today)
	}
}

func TestParseFlagsBetweenPositionals(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	to := fs.String("to", "", "")
	got, err := Parse([]string{"a", "--to=x", "b"}, fs)
	if err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) || *to != "x" {
		t.Fatalf("got %v %v to=%q", got, err, *to)
	}
}

func TestParseDoubleDash(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	got, err := Parse([]string{"--", "--x"}, fs)
	if err != nil || !reflect.DeepEqual(got, []string{"--x"}) {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestParseUnknownFlagErrors(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if _, err := Parse([]string{"--nope"}, fs); err == nil {
		t.Fatal("want an error")
	}
}

func TestRootNeedsBuilderRoot(t *testing.T) {
	t.Setenv("BUILDER_ROOT", "")
	if _, err := Root(); err == nil || err.Error() != "builder: BUILDER_ROOT is not set. Run builder through <plugin root>/bin/builder." {
		t.Fatalf("got %v", err)
	}
	t.Setenv("BUILDER_ROOT", "/x y")
	if r, err := Root(); err != nil || r != "/x y" {
		t.Fatalf("got %q %v", r, err)
	}
}
