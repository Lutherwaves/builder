package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SaveJSON writes v as JSON through a temp file in the same folder, then renames it into place.
func SaveJSON(path string, v any, indent string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if indent != "" {
		b, err = json.MarshalIndent(v, "", indent)
	}
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
