package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveJSONWritesTheValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "out.json")
	if err := SaveJSON(path, map[string]int{"a": 1}, "  "); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"a\": 1\n}"; string(got) != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("folder holds %d entries, want only the saved file", len(entries))
	}
}

func TestSaveJSONLeavesTheOldFileOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(path, make(chan int), ""); err == nil {
		t.Fatal("want an error for a value that cannot be encoded")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "old" {
		t.Fatalf("file = %q, want it unchanged", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("folder holds %d entries, want 1", len(entries))
	}
}
