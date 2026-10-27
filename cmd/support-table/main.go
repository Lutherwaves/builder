// Command support-table builds the README support table from graded eval results. It is a dev tool, never released.
//
//	support-table record <verdict.json>   fold a graded verdict into tests/evals/results/<harness>.json
//	support-table render                  print the table
//	support-table check <README.md>       exit 1 when the block between the markers differs from `render`
//
// A results file keeps counts only: no transcript, no quote, no path.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var (
	harnesses = []struct{ id, name string }{{"claude", "Claude Code"}, {"codex", "Codex CLI"}, {"cursor", "Cursor CLI"}, {"opencode", "opencode"}}
	version   = regexp.MustCompile(`^[A-Za-z0-9._ +()-]{1,60}$`)
	fixtureRe = regexp.MustCompile(`^[a-z0-9_-]+$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

const (
	noneNote   = "No harness has passed its evals yet."
	start, end = "<!-- support:start -->", "<!-- support:end -->"
)

type counts struct {
	Met   int `json:"met"`
	Total int `json:"total"`
}

type entry struct {
	name string
	c    counts
}

// fixtures keep the order of the file, as a Python dict does.
type fixtures []entry

func (f *fixtures) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return errors.New("fixtures is not an object")
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		var c counts
		if err := dec.Decode(&c); err != nil {
			return err
		}
		*f = append(*f, entry{t.(string), c})
	}
	return nil
}

func (f fixtures) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for n, e := range f {
		if n > 0 {
			b.WriteByte(',')
		}
		k, _ := json.Marshal(e.name)
		v, _ := json.Marshal(e.c)
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

type result struct {
	Harness        string   `json:"harness"`
	HarnessVersion *string  `json:"harness_version"`
	Date           string   `json:"date"`
	Fixtures       fixtures `json:"fixtures"`
}

type failure string

func fail(format string, a ...any) { panic(failure(fmt.Sprintf(format, a...))) }

func harnessIDs() []string {
	var ids []string
	for _, h := range harnesses {
		ids = append(ids, h.id)
	}
	return ids
}

func count(v map[string]any) counts {
	for _, k := range []struct {
		key string
		re  *regexp.Regexp
	}{{"harness_version", version}, {"fixture", fixtureRe}, {"date", dateRe}} {
		if s, ok := v[k.key].(string); !ok || !k.re.MatchString(s) {
			fail("%s is missing or not a plain value", k.key)
		}
	}
	if h, _ := v["harness"].(string); !slices.Contains(harnessIDs(), h) {
		fail("harness must be one of %s", strings.Join(harnessIDs(), ", "))
	}
	bullets, ok := v["bullets"].([]any)
	if !ok || len(bullets) == 0 {
		fail("bullets is empty")
	}
	c := counts{Total: len(bullets)}
	for _, raw := range bullets {
		met, ok := raw.(map[string]any)["met"].(bool)
		if !ok {
			fail("every bullet needs met true or false: the verdict is not graded yet")
		}
		if met {
			c.Met++
		}
	}
	return c
}

func record(stdout io.Writer, resultsDir, source string) {
	b, err := os.ReadFile(source)
	var v map[string]any
	if err == nil {
		err = json.Unmarshal(b, &v)
	}
	if err != nil {
		fail("cannot read %s: %v", source, err)
	}
	c := count(v)
	h, ver, fx, date := v["harness"].(string), v["harness_version"].(string), v["fixture"].(string), v["date"].(string)
	path := filepath.Join(resultsDir, h+".json")
	saved := result{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &saved); err != nil {
			fail("cannot read %s: %v", path, err)
		}
	}
	if saved.HarnessVersion != nil && *saved.HarnessVersion != ver {
		saved.Fixtures = nil // a new version starts a new result
	}
	if n := slices.IndexFunc(saved.Fixtures, func(e entry) bool { return e.name == fx }); n >= 0 {
		saved.Fixtures[n].c = c
	} else {
		saved.Fixtures = append(saved.Fixtures, entry{fx, c})
	}
	saved.Harness, saved.HarnessVersion, saved.Date = h, &ver, max(date, saved.Date)
	if saved.Fixtures == nil {
		saved.Fixtures = fixtures{}
	}
	out, _ := json.MarshalIndent(saved, "", "  ")
	os.MkdirAll(resultsDir, 0o755)
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		fail("cannot write %s: %v", path, err)
	}
	fmt.Fprintf(stdout, "recorded %s %s: %d/%d\n", h, fx, c.Met, c.Total)
}

func fixturesOf(evalsDir string) []string {
	entries, _ := os.ReadDir(evalsDir)
	var out []string
	for _, e := range entries {
		if st, err := os.Stat(filepath.Join(evalsDir, e.Name(), "prompt.md")); e.IsDir() && err == nil && st.Mode().IsRegular() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func render(resultsDir, evalsDir string) string {
	expected := fixturesOf(evalsDir)
	var rows []string
	for _, h := range harnesses {
		b, err := os.ReadFile(filepath.Join(resultsDir, h.id+".json"))
		if err != nil {
			continue
		}
		var r result
		if err := json.Unmarshal(b, &r); err != nil {
			fail("cannot read %s: %v", filepath.Join(resultsDir, h.id+".json"), err)
		}
		var short []string
		graded := 0
		for _, e := range r.Fixtures {
			if !slices.Contains(expected, e.name) {
				continue
			}
			graded++
			if e.c.Met != e.c.Total {
				short = append(short, fmt.Sprintf("%s %d/%d", e.name, e.c.Met, e.c.Total))
			}
		}
		if missing := len(expected) - graded; missing > 0 {
			short = append(short, fmt.Sprintf("%d not graded", missing))
		}
		status := "supported"
		if short != nil || len(expected) == 0 {
			status = "experimental: " + strings.Join(short, "; ")
		}
		v := ""
		if r.HarnessVersion != nil {
			v = *r.HarnessVersion
		}
		rows = append(rows, fmt.Sprintf("| %s | %s | %s | %s |", h.name, v, r.Date, status))
	}
	if rows == nil {
		return noneNote
	}
	return strings.Join(append([]string{"| Harness | Version | Eval date | Status |", "| --- | --- | --- | --- |"}, rows...), "\n")
}

func check(stdout io.Writer, resultsDir, evalsDir, readme string) {
	b, err := os.ReadFile(readme)
	if err != nil {
		fail("cannot read %s: %v", readme, err)
	}
	text := string(b)
	i, j := strings.Index(text, start), strings.Index(text, end)
	if i < 0 || j < 0 || i > j {
		fail("%s has no %s ... %s block", readme, start, end)
	}
	block := strings.TrimSpace(text[i+len(start) : j])
	if block != render(resultsDir, evalsDir) {
		fail("the support table in %s is out of date: paste the output of `go run ./cmd/support-table render` between the markers", readme)
	}
	fmt.Fprintln(stdout, "support table ok")
}

func run(args []string, stdout, stderr io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			msg, ok := r.(failure)
			if !ok {
				panic(r)
			}
			fmt.Fprintf(stderr, "support-table: %s\n", msg)
			code = 1
		}
	}()
	fs := flag.NewFlagSet("support-table", flag.ContinueOnError)
	fs.SetOutput(stderr)
	resultsDir := fs.String("results-dir", filepath.Join("tests", "evals", "results"), "")
	evalsDir := fs.String("evals-dir", "", "where the fixture dirs live; a harness is supported only when all of them pass")
	if fs.Parse(args) != nil {
		return 2
	}
	if *evalsDir == "" {
		*evalsDir = filepath.Dir(*resultsDir)
	}
	rest := fs.Args()
	switch {
	case len(rest) == 2 && rest[0] == "record":
		record(stdout, *resultsDir, rest[1])
	case len(rest) == 1 && rest[0] == "render":
		fmt.Fprintln(stdout, render(*resultsDir, *evalsDir))
	case len(rest) == 2 && rest[0] == "check":
		check(stdout, *resultsDir, *evalsDir, rest[1])
	default:
		fmt.Fprintln(stderr, "usage: support-table [--results-dir D] [--evals-dir D] {record <verdict.json>,render,check <README.md>}")
		return 2
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
