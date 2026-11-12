package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/harness"
	"github.com/Lutherwaves/builder/internal/hook"
)

func init() { commands["hook"] = hookCmd }

// hookCmd ignores flags it does not know, so that a newer harness flag never breaks the hook.
func hookCmd(args []string, stdout, stderr io.Writer) int {
	name, format := "auto", ""
	for n := 0; n < len(args); n++ {
		key, value, inline := strings.Cut(args[n], "=")
		if key != "--harness" && key != "--format" {
			continue
		}
		if !inline {
			if n+1 >= len(args) {
				fmt.Fprintf(stderr, "builder hook: error: argument %s: expected one argument\n", key)
				return 2
			}
			n++
			value = args[n]
		}
		if key == "--harness" {
			name = value
		} else if value != "json" && value != "text" {
			fmt.Fprintf(stderr, "builder hook: error: argument --format: invalid choice: '%s' (choose from 'json', 'text')\n", value)
			return 2
		} else {
			format = value
		}
	}
	root, err := cli.Root()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	name = harness.Detect(harness.Env(), name)
	if format == "" {
		format = "json"
		if name == "opencode" {
			format = "text"
		}
	}
	text := hook.Build(root, name)
	if format == "json" {
		text = hook.Shape(name, text)
	}
	fmt.Fprintln(stdout, text)
	return 0
}
