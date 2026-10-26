package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/harness"
	"github.com/Lutherwaves/builder/internal/hook"
)

func init() { commands["hook"] = hookCmd }

// hookCmd ignores flags it does not know, as the Python hook's parse_known_args does.
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
	env := harness.Env()
	name = harness.Detect(env, name)
	if format == "" {
		format = "json"
		if name == "opencode" {
			format = "text"
		}
	}
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	text := hook.Build(root, name, env, home, cwd, now())
	if format == "json" {
		text = hook.Shape(name, text)
	}
	fmt.Fprintln(stdout, text)
	return 0
}
