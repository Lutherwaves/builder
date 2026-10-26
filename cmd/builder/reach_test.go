package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reachEnv(t *testing.T) string {
	d := cliEnv(t)
	t.Setenv("BUILDER_HOME", d)
	return d
}

func verdictOf(t *testing.T, out string) map[string]string {
	t.Helper()
	var v map[string]string
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("%v: %q", err, out)
	}
	return v
}

func TestRecipientIsInDigestAndLog(t *testing.T) {
	d := reachEnv(t)
	text := "The API changed."
	call(t, "reach", "log", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", text, "--verdict", "send")
	_, out, _ := call(t, "reach", "check", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", text)
	if v := verdictOf(t, out); v["verdict"] != "skip" || v["reason"] != "duplicate" {
		t.Fatal(v)
	}
	_, out, _ = call(t, "reach", "check", "--to", "teammate", "--recipient", "Bob", "--kind", "fyi", "--text", text)
	if verdictOf(t, out)["verdict"] != "draft" {
		t.Fatal(out)
	}
	b, _ := os.ReadFile(filepath.Join(d, "reach-log.jsonl"))
	var e map[string]any
	json.Unmarshal([]byte(strings.SplitN(string(b), "\n", 2)[0]), &e)
	if e["recipient"] != "Ana" {
		t.Fatal(e)
	}
}

func TestTeammateAndChannelNeedARecipient(t *testing.T) {
	d := reachEnv(t)
	for _, to := range []string{"teammate", "channel"} {
		for _, cmd := range []string{"check", "log"} {
			args := []string{"reach", cmd, "--to", to, "--kind", "fyi", "--text", "x"}
			if cmd == "log" {
				args = append(args, "--verdict", "draft")
			}
			code, _, errs := call(t, args...)
			if code != 2 || !strings.Contains(errs, "--recipient") {
				t.Fatalf("%s %s: %d %q", cmd, to, code, errs)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(d, "reach-log.jsonl")); err == nil {
		t.Fatal("a log was written")
	}
}

func TestChannelRecipientIsTheChannelName(t *testing.T) {
	d := reachEnv(t)
	os.WriteFile(filepath.Join(d, "reach.json"), []byte(`{"to_channel": {"allow": ["fyi"], "named": ["releases"]}}`), 0o644)
	for room, want := range map[string]string{"releases": "send", "general": "deny"} {
		_, out, _ := call(t, "reach", "check", "--to", "channel", "--recipient", room, "--kind", "fyi", "--text", "x")
		if verdictOf(t, out)["verdict"] != want {
			t.Fatal(room, out)
		}
	}
}

func TestRecipientAndChannelNameMustAgree(t *testing.T) {
	reachEnv(t)
	code, _, errs := call(t, "reach", "check", "--to", "channel", "--recipient", "a", "--channel-name", "b", "--kind", "fyi", "--text", "x")
	if code != 2 || !strings.Contains(errs, "--recipient and --channel-name name different rooms; give one room") {
		t.Fatal(code, errs)
	}
}

func TestMalformedLiveRulesUseDefaults(t *testing.T) {
	d := reachEnv(t)
	os.WriteFile(filepath.Join(d, "reach.json"), []byte(`{"dedupe_s": "x", "per_hour": null, "to_self": [1]}`), 0o644)
	code, out, errs := call(t, "reach", "check", "--to", "self", "--kind", "waiting", "--text", "x")
	if code != 0 || verdictOf(t, out)["verdict"] != "draft" {
		t.Fatal(code, out, errs)
	}
}

func TestDuplicateTwiceIsSkippedAndLogged(t *testing.T) {
	d := reachEnv(t)
	base := []string{"--to", "self", "--kind", "waiting", "--text", "Tests are green."}
	_, out, _ := call(t, append([]string{"reach", "check"}, base...)...)
	if verdictOf(t, out)["verdict"] != "send" {
		t.Fatal(out)
	}
	call(t, append(append([]string{"reach", "log"}, base...), "--verdict", "send")...)
	_, out, _ = call(t, append([]string{"reach", "check"}, base...)...)
	if verdictOf(t, out)["reason"] != "duplicate" {
		t.Fatal(out)
	}
	call(t, append(append([]string{"reach", "log"}, base...), "--verdict", "skip", "--reason", "duplicate")...)
	b, _ := os.ReadFile(filepath.Join(d, "reach-log.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var last map[string]any
	json.Unmarshal([]byte(lines[1]), &last)
	if len(lines) != 2 || last["verdict"] != "skip" || last["reason"] != "duplicate" {
		t.Fatal(lines)
	}
}

func TestLiveRulesOverrideDefaultsAndKeepChannel(t *testing.T) {
	d := reachEnv(t)
	os.WriteFile(filepath.Join(d, "reach.json"), []byte(`{"channel": "slack", "to_teammate": {"allow": ["fyi"]}}`), 0o644)
	_, out, _ := call(t, "reach", "check", "--to", "teammate", "--recipient", "Ana", "--kind", "fyi", "--text", "The API changed.")
	if verdictOf(t, out)["verdict"] != "send" {
		t.Fatal(out)
	}
	_, out, _ = call(t, "reach", "check", "--to", "self", "--kind", "waiting", "--text", "x")
	if verdictOf(t, out)["verdict"] != "send" {
		t.Fatal(out)
	}
}

func TestChannelCommandWritesChannelKey(t *testing.T) {
	d := reachEnv(t)
	p := filepath.Join(d, "reach.json")
	os.WriteFile(p, []byte(`{"per_hour": 2}`), 0o644)
	if code, _, errs := call(t, "reach", "channel", "Slack"); code != 0 {
		t.Fatal(errs)
	}
	var v map[string]any
	b, _ := os.ReadFile(p)
	json.Unmarshal(b, &v)
	if v["channel"] != "slack" || v["per_hour"] != 2.0 {
		t.Fatal(v)
	}
	call(t, "reach", "channel", "none")
	b, _ = os.ReadFile(p)
	v = nil
	json.Unmarshal(b, &v)
	if c, ok := v["channel"]; !ok || c != nil {
		t.Fatal(v)
	}
}

func TestChannelRefusesACorruptRulesFile(t *testing.T) {
	d := reachEnv(t)
	p := filepath.Join(d, "reach.json")
	for _, body := range []string{"{not json", "[1]"} {
		os.WriteFile(p, []byte(body), 0o644)
		code, _, errs := call(t, "reach", "channel", "slack")
		b, _ := os.ReadFile(p)
		if code != 1 || !strings.Contains(errs, p) || !strings.Contains(strings.ToLower(errs), "fix the file") || string(b) != body {
			t.Fatal(code, errs)
		}
	}
}

func TestReachUnwritableHomeExitsNonzero(t *testing.T) {
	d := reachEnv(t)
	f := filepath.Join(d, "file")
	os.WriteFile(f, []byte("x"), 0o644)
	t.Setenv("BUILDER_HOME", filepath.Join(f, "sub"))
	code, _, errs := call(t, "reach", "log", "--to", "self", "--kind", "waiting", "--text", "x", "--verdict", "send")
	if code == 0 || !strings.Contains(errs, "BUILDER_HOME") {
		t.Fatal(code, errs)
	}
}

func TestNotifyWithoutAToolSaysTheFix(t *testing.T) {
	d := reachEnv(t)
	t.Setenv("PATH", filepath.Join(d, "empty"))
	code, _, errs := call(t, "reach", "notify", "--text", "Waiting for you.")
	if code == 0 || !strings.Contains(errs, "draft") {
		t.Fatal(code, errs)
	}
}

func matchbloxShim(t *testing.T, d string, cmds ...string) {
	bin := filepath.Join(d, "mbbin")
	os.MkdirAll(bin, 0o755)
	help := "Usage: matchblox [command] [flags]\n\nCommands:\n"
	for _, c := range cmds {
		help += fmt.Sprintf("  matchblox %s    does %s\n", c, c)
	}
	os.WriteFile(filepath.Join(d, "help.txt"), []byte(help), 0o644)
	os.WriteFile(filepath.Join(bin, "matchblox"), []byte(fmt.Sprintf("#!/bin/sh\nprintf \"%%s\\n\" \"$*\" >> \"%s/calls\"\n[ \"$1\" = --help ] && cat \"%s/help.txt\"\nexit 0\n", d, d)), 0o755)
	t.Setenv("PATH", bin+":/usr/bin:/bin")
}

func calls(d string) string {
	b, _ := os.ReadFile(filepath.Join(d, "calls"))
	return string(b)
}

func TestMatchbloxWithoutNotifyFallsThrough(t *testing.T) {
	d := reachEnv(t)
	matchbloxShim(t, d, "<host>", `send <who> "<text>"`, "usage")
	code, _, errs := call(t, "reach", "matchblox-notify", "--text", "Waiting for you.")
	if code != 3 || !strings.Contains(errs, "next route") || calls(d) != "--help\n" {
		t.Fatal(code, errs, calls(d))
	}
}

func TestMatchbloxWithNotifySends(t *testing.T) {
	d := reachEnv(t)
	matchbloxShim(t, d, "<host>", `notify "<text>"`, "usage")
	code, out, errs := call(t, "reach", "matchblox-notify", "--text", "Waiting for you.")
	if code != 0 || out != "matchblox\n" || calls(d) != "--help\nnotify Waiting for you.\n" {
		t.Fatal(code, out, errs, calls(d))
	}
}

func TestMatchbloxWithOnlyANotifyPrefixedCommandFallsThrough(t *testing.T) {
	d := reachEnv(t)
	matchbloxShim(t, d, "<host>", `notify-x "<text>"`, "usage")
	if code, _, _ := call(t, "reach", "matchblox-notify", "--text", "Waiting for you."); code != 3 || calls(d) != "--help\n" {
		t.Fatal(code, calls(d))
	}
}

func TestNoMatchbloxFallsThrough(t *testing.T) {
	d := reachEnv(t)
	t.Setenv("PATH", filepath.Join(d, "empty"))
	if code, _, _ := call(t, "reach", "matchblox-notify", "--text", "Waiting for you."); code != 3 {
		t.Fatal(code)
	}
}
