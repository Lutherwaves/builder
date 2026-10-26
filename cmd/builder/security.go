package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/harness"
	"github.com/Lutherwaves/builder/internal/ledger"
	"github.com/Lutherwaves/builder/internal/security"
)

func init() { commands["security"] = securityCmd }

func secUsage(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "usage: builder security {cvss,score,baseline} ...\nbuilder security: error: %s\n", msg)
	return 2
}

func readFindings(path string) (any, error) {
	b, err := os.ReadFile(path)
	var v any
	if err == nil {
		err = json.Unmarshal(b, &v)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", path, err)
	}
	return v, nil
}

func baseline(action, path string, today time.Time) (map[string]any, error) {
	raw, err := readFindings(path)
	if err != nil {
		return nil, err
	}
	current, err := security.Enrich(raw)
	if err != nil {
		return nil, err
	}
	if action == "save" && current["mode"] != "audit" {
		return nil, fmt.Errorf("only an audit saves the baseline (this run is mode '%s')", current["mode"])
	}
	home, _ := os.UserHomeDir()
	bpath := security.BaselinePath(current["repo"].(string), harness.Env(), home)
	prior, err := security.LoadBaseline(bpath)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range security.Diff(prior, current, today) {
		out[k] = v
	}
	if action == "save" {
		if err := ledger.SaveJSON(bpath, security.NextBaseline(prior, current, today), "  "); err != nil {
			return nil, err
		}
		out["saved"] = bpath
	}
	return out, nil
}

func securityCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return secUsage(stderr, "the following arguments are required: cmd")
	}
	fs := flag.NewFlagSet("security", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	todayFlag := fs.String("today", "", "")
	pos, err := cli.Parse(args[1:], fs)
	if err != nil {
		return secUsage(stderr, err.Error())
	}
	var result any
	switch {
	case args[0] == "cvss" && len(pos) == 1:
		score, err := security.BaseScore(pos[0])
		if err != nil {
			fmt.Fprintf(stderr, "security: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "%.1f %s\n", score, security.Rating(score))
		return 0
	case args[0] == "score" && len(pos) == 1:
		raw, err := readFindings(pos[0])
		if err == nil {
			result, err = security.Enrich(raw)
		}
		if err != nil {
			fmt.Fprintf(stderr, "security: %v\n", err)
			return 2
		}
	case args[0] == "baseline" && len(pos) == 2 && (pos[0] == "diff" || pos[0] == "save"):
		today := time.Now()
		if *todayFlag != "" {
			if today, err = time.Parse("2006-01-02", *todayFlag); err != nil {
				return secUsage(stderr, fmt.Sprintf("argument --today: invalid fromisoformat value: '%s'", *todayFlag))
			}
		}
		today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
		if result, err = baseline(pos[0], pos[1], today); err != nil {
			fmt.Fprintf(stderr, "security: %v\n", err)
			return 2
		}
	default:
		return secUsage(stderr, "expected: cvss <vector> | score <findings> | baseline diff|save <findings> [--today YYYY-MM-DD]")
	}
	printJSON(stdout, result, "  ")
	return 0
}
