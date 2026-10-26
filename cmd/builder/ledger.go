package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/harness"
	"github.com/Lutherwaves/builder/internal/ledger"
	"github.com/Lutherwaves/builder/internal/signals"
)

func init() { commands["ledger"] = ledgerCmd }

func now() float64 { return float64(time.Now().UnixNano()) / 1e9 }

func unwritable(stderr io.Writer, path string, err error) int {
	fmt.Fprintf(stderr, "builder cannot write %s (%s); set BUILDER_HOME to a writable folder.\n", path, cli.Strerror(err))
	return 1
}

func printJSON(stdout io.Writer, v any, indent string) {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", indent)
	enc.Encode(v)
}

func ledgerCmd(args []string, stdout, stderr io.Writer) int {
	path := filepath.Join(ledger.Home(), "ledger.json")
	switch {
	case len(args) == 3 && args[0] == "answer" && slices.Contains(ledger.Verdicts, args[2]):
		l := ledger.Load(path)
		l.Answers[args[1]] = ledger.Answer{Verdict: args[2], At: now()}
		if err := ledger.Save(path, l); err != nil {
			return unwritable(stderr, path, err)
		}
		fmt.Fprintln(stdout, "ok")
		return 0
	case len(args) == 1 && args[0] == "onboarded":
		l := ledger.Load(path)
		l.Onboarded = true
		if err := ledger.Save(path, l); err != nil {
			return unwritable(stderr, path, err)
		}
		fmt.Fprintln(stdout, "ok")
		return 0
	case (len(args) == 1 || len(args) == 3 && args[1] == "--harness") && args[0] == "next":
		root, err := cli.Root()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		env := harness.Env()
		name := harness.Detect(env, "")
		if len(args) == 3 {
			name = args[2]
		}
		if !slices.Contains(harness.Names, name) {
			name = "claude"
		}
		cat, err := ledger.LoadCatalog(root)
		if err != nil {
			fmt.Fprintf(stderr, "builder cannot read the catalog: %v\n", err)
			return 1
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		l := ledger.Load(path)
		sig := signals.Collect(cwd, 500*time.Millisecond, name, env, home)
		item := ledger.Next(l, ledger.DueCatalog(l, ledger.ForHarness(cat, name)), sig, now(), ledger.CooldownS, !l.Onboarded)
		if item == nil {
			fmt.Fprintln(stdout, "null")
			return 0
		}
		printJSON(stdout, item, "")
		return 0
	}
	fmt.Fprintln(stderr, "usage: builder ledger answer <id> yes|later|never | builder ledger onboarded | builder ledger next [--harness H]")
	return 2
}
