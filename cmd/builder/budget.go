package main

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/Lutherwaves/builder/internal/budget"
)

func init() { commands["budget"] = budgetCmd }

const budgetUsage = "usage: builder budget <used_pct> <resets_at_epoch> [now_epoch]\n       builder budget --from-matchblox [now_epoch]"

func budgetCmd(args []string, stdout, stderr io.Writer) int {
	usage := func() int { fmt.Fprintln(stderr, budgetUsage); return 2 }
	nowArg := func(n int) (int64, bool) {
		if len(args) <= n {
			return time.Now().Unix(), true
		}
		v, err := strconv.ParseInt(args[n], 10, 64)
		return v, err == nil
	}
	if len(args) > 0 && args[0] == "--from-matchblox" {
		now, ok := nowArg(1)
		if !ok {
			return usage()
		}
		return budget.FromMatchblox(now, "", stdout, stderr)
	}
	if len(args) < 2 {
		return usage()
	}
	used, err1 := strconv.ParseFloat(args[0], 64)
	reset, err2 := strconv.ParseInt(args[1], 10, 64)
	now, ok := nowArg(2)
	if err1 != nil || err2 != nil || !ok {
		return usage()
	}
	fmt.Fprintln(stdout, budget.Project(used, reset, now))
	return 0
}
