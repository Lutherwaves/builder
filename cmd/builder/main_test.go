package main

import (
	"bytes"
	"testing"
)

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
