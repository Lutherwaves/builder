// Package cli holds what every builder command shares: argparse-style flags and the plugin root.
package cli

import (
	"errors"
	"flag"
	"os"
)

// Parse parses fs's flags wherever they sit among the positionals, as argparse does,
// and returns the positionals. "--" ends the flags.
func Parse(args []string, fs *flag.FlagSet) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if len(args) > len(rest) && args[len(args)-len(rest)-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// Root is the plugin root that bin/builder exports, where the catalog, soul and mappings live.
func Root() (string, error) {
	if r := os.Getenv("BUILDER_ROOT"); r != "" {
		return r, nil
	}
	return "", errors.New("builder: BUILDER_ROOT is not set. Run builder through <plugin root>/bin/builder.")
}
