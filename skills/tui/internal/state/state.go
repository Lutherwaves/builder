// Package state is the snapshot file a running cockpit leaves behind so an
// agent can read the same facts without sampling again.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Lutherwaves/builder/skills/tui/internal/advice"
	"github.com/Lutherwaves/builder/skills/tui/internal/gitscan"
	"github.com/Lutherwaves/builder/skills/tui/internal/sample"
)

// Doc is everything the cockpit knows at one moment.
type Doc struct {
	sample.Snapshot
	Git             *gitscan.Report `json:"git,omitempty"`
	Recommendations []advice.Rec    `json:"recommendations"`
}

// Path is $XDG_STATE_HOME/builder/tui.json, defaulting to ~/.local/state.
func Path() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "builder", "tui.json")
}

// Write replaces the file atomically so a reader never sees half a snapshot.
func Write(path string, doc Doc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tui-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func Read(path string) (Doc, error) {
	var doc Doc
	b, err := os.ReadFile(path)
	if err != nil {
		return doc, err
	}
	return doc, json.Unmarshal(b, &doc)
}
