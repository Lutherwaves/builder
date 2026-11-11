// Command builder runs builder's hook and security score.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

var version = "dev"

type command func(args []string, stdout, stderr io.Writer) int

var commands = map[string]command{
	"version": func(_ []string, stdout, _ io.Writer) int {
		fmt.Fprintln(stdout, version)
		return 0
	},
}

func now() float64 { return float64(time.Now().UnixNano()) / 1e9 }

func printJSON(stdout io.Writer, v any, indent string) {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", indent)
	enc.Encode(v)
}

func usage(stderr io.Writer) int {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintf(stderr, "usage: builder <%s> ...\n", strings.Join(names, "|"))
	return 2
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usage(stderr)
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "builder: unknown command %q\n", args[0])
		return usage(stderr)
	}
	return cmd(args[1:], stdout, stderr)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
