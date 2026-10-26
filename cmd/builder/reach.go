package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/ledger"
	"github.com/Lutherwaves/builder/internal/reach"
)

func init() { commands["reach"] = reachCmd }

func quoted(list []string) string {
	q := make([]string, len(list))
	for n, s := range list {
		q[n] = "'" + s + "'"
	}
	return strings.Join(q, ", ")
}

func reachErr(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "usage: builder reach {check,log,channel,matchblox-notify,notify} ...\nbuilder reach: error: %s\n", msg)
	return 2
}

type flagSpec struct {
	name     string
	required bool
	choices  []string
	def      string
}

// parseFlags is argparse for one subcommand: required flags and choices give argparse's messages.
func parseFlags(args []string, specs []flagSpec) (map[string]string, map[string]bool, []string, string) {
	fs := flag.NewFlagSet("reach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	vals := map[string]*string{}
	for _, s := range specs {
		vals[s.name] = fs.String(s.name, s.def, "")
	}
	pos, err := cli.Parse(args, fs)
	if err != nil {
		return nil, nil, nil, err.Error()
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var missing []string
	out := map[string]string{}
	for _, s := range specs {
		out[s.name] = *vals[s.name]
		if s.required && !set[s.name] {
			missing = append(missing, "--"+s.name)
		}
		if set[s.name] && s.choices != nil && !slices.Contains(s.choices, out[s.name]) {
			return nil, nil, nil, fmt.Sprintf("argument --%s: invalid choice: '%s' (choose from %s)", s.name, out[s.name], quoted(s.choices))
		}
	}
	if missing != nil {
		return nil, nil, nil, "the following arguments are required: " + strings.Join(missing, ", ")
	}
	return out, set, pos, ""
}

func reachCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return reachErr(stderr, "the following arguments are required: cmd")
	}
	cmd, rest := args[0], args[1:]
	specs := map[string][]flagSpec{
		"check":            {{"to", true, reach.Recipients, ""}, {"kind", true, nil, ""}, {"text", true, nil, ""}, {"recipient", false, nil, ""}, {"channel-name", false, nil, ""}},
		"log":              {{"to", true, reach.Recipients, ""}, {"kind", true, nil, ""}, {"text", true, nil, ""}, {"recipient", false, nil, ""}, {"channel-name", false, nil, ""}, {"verdict", true, reach.Verdicts, ""}, {"reason", false, nil, ""}},
		"channel":          {},
		"matchblox-notify": {{"text", true, nil, ""}},
		"notify":           {{"text", true, nil, ""}, {"title", false, nil, "builder"}},
	}
	spec, ok := specs[cmd]
	if !ok {
		return reachErr(stderr, fmt.Sprintf("argument cmd: invalid choice: '%s' (choose from 'check', 'log', 'channel', 'matchblox-notify', 'notify')", cmd))
	}
	a, _, pos, msg := parseFlags(rest, spec)
	if msg != "" {
		return reachErr(stderr, msg)
	}
	if cmd == "channel" && len(pos) != 1 {
		return reachErr(stderr, "the following arguments are required: name")
	}
	if cmd != "channel" && len(pos) > 0 {
		return reachErr(stderr, "unrecognized arguments: "+strings.Join(pos, " "))
	}
	if (cmd == "check" || cmd == "log") && a["to"] != "self" {
		if a["to"] == "channel" {
			if a["recipient"] != "" && a["channel-name"] != "" && a["recipient"] != a["channel-name"] {
				return reachErr(stderr, "--recipient and --channel-name name different rooms; give one room")
			}
			if a["recipient"] == "" {
				a["recipient"] = a["channel-name"]
			}
			a["channel-name"] = a["recipient"]
		}
		if strings.TrimSpace(a["recipient"]) == "" {
			what := "teammate's name"
			if a["to"] == "channel" {
				what = "room name"
			}
			return reachErr(stderr, fmt.Sprintf("--to %s needs --recipient <%s>", a["to"], what))
		}
	}

	home := ledger.Home()
	rulesPath, logPath := filepath.Join(home, "reach.json"), filepath.Join(home, "reach-log.jsonl")
	switch cmd {
	case "check":
		root, err := cli.Root()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		rules, err := reach.LoadRules(root, rulesPath)
		if err != nil {
			fmt.Fprintf(stderr, "builder cannot read the default send rules: %v\n", err)
			return 1
		}
		printJSON(stdout, reach.Check(rules, reach.ReadLog(logPath), a["to"], a["kind"], a["text"], now(), a["channel-name"], a["recipient"]), "")
		return 0
	case "log":
		recipient := a["recipient"]
		if a["to"] == "self" {
			recipient = "self"
		}
		entry := map[string]any{"at": now(), "to": a["to"], "recipient": recipient, "kind": a["kind"], "verdict": a["verdict"],
			"reason": a["reason"], "sha256": reach.Digest(a["to"], a["text"], a["recipient"]), "text": a["text"]}
		if a["channel-name"] != "" {
			entry["channel_name"] = a["channel-name"]
		}
		if err := reach.Record(logPath, entry); err != nil {
			return unwritable(stderr, logPath, err)
		}
		fmt.Fprintln(stdout, "ok")
		return 0
	case "channel":
		data := map[string]any{}
		if b, err := os.ReadFile(rulesPath); err == nil {
			if json.Unmarshal(b, &data) != nil || data == nil {
				fmt.Fprintf(stderr, "builder cannot read %s: it is not a JSON object. Fix the file or delete it, then run this again.\n", rulesPath)
				return 1
			}
		}
		name := strings.ToLower(pos[0])
		if name == "none" || name == "null" || name == "" {
			data["channel"] = nil
		} else {
			data["channel"] = name
		}
		if err := ledger.SaveJSON(rulesPath, data, ""); err != nil {
			return unwritable(stderr, rulesPath, err)
		}
		fmt.Fprintln(stdout, "ok")
		return 0
	case "matchblox-notify":
		if reach.MatchbloxNotify(a["text"], "") {
			fmt.Fprintln(stdout, "matchblox")
			return 0
		}
		fmt.Fprintln(stderr, "matchblox has no notify command here; use the next route.")
		return reach.NoRoute
	}
	tool := reach.Notify(a["title"], a["text"], runtime.GOOS, "")
	if tool == "" {
		fmt.Fprintln(stderr, "builder found no notification tool (osascript on macOS, notify-send on Linux). Show the message in the conversation as a draft instead.")
		return 1
	}
	fmt.Fprintln(stdout, tool)
	return 0
}
