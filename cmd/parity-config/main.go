// Command parity-config writes the builder-led Claude Code config dir for tests/evals/parity.sh. It is a dev tool, never released.
//
//	parity-config <src> <src_state> <cfg>
//
// src is the person's config dir, src_state their .claude.json, cfg the folder parity owns. It rebuilds cfg:
//   - plugins and skills: empty, so no other skill pack is in the builder-led run;
//   - settings.json and .claude.json: only what builder recommends.
//
// It never copies credentials or account data. It writes JSON as Python's json.dumps(indent=2) did, so a kept cfg does not churn.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: parity-config <src> <src_state> <cfg>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2], os.Args[3], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "parity-config:", err)
		os.Exit(1)
	}
}

func run(src, srcState, cfg string, stderr io.Writer) error {
	note := func(format string, a ...any) { fmt.Fprintf(stderr, "note: "+format+"\n", a...) }

	for _, d := range []string{"plugins", "skills"} {
		if err := clear(join(cfg, d)); err != nil {
			return err
		}
	}

	// The matchblox hooks and status line that `matchblox setup` wrote to the person's settings.
	settings := load(join(src, "settings.json"))
	hooks := &object{}
	if all, _ := settings.get("hooks"); truthy(all) {
		events, ok := all.(*object)
		if !ok {
			return errors.New("settings.json: hooks is not an object")
		}
		for _, ev := range events.members {
			groups, _ := ev.val.([]any)
			for _, g := range groups {
				group, ok := g.(*object)
				if !ok {
					return fmt.Errorf("settings.json: a %s hook group is not an object", ev.key)
				}
				keep, err := matchbloxHooks(group)
				if err != nil {
					return err
				}
				if len(keep) > 0 {
					kept := group.clone()
					kept.set("hooks", keep)
					list, _ := hooks.get(ev.key)
					hooks.set(ev.key, append(asList(list), kept))
				}
			}
		}
	}
	var status any
	if sl, _ := settings.get("statusLine"); truthy(sl) {
		so, ok := sl.(*object)
		if !ok {
			return errors.New("settings.json: statusLine is not an object")
		}
		if cmd, _ := so.get("command"); strings.Contains(pyStr(cmd), "matchblox hook") {
			status = so
		}
	}
	if len(hooks.members) == 0 {
		note("your settings have no matchblox hooks, so the builder-led run has none. Run `matchblox setup` first for a fair run.")
	}
	env := &object{}
	env.set("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE", "85")
	out := &object{}
	out.set("env", env)
	if len(hooks.members) > 0 {
		out.set("hooks", hooks)
	}
	if status != nil {
		out.set("statusLine", status)
	}
	if err := writeJSON(join(cfg, "settings.json"), out); err != nil {
		return err
	}

	// The code graph server, merged into cfg's own .claude.json so a kept sign-in survives.
	var server any
	if all, _ := load(srcState).get("mcpServers"); truthy(all) {
		servers, ok := all.(*object)
		if !ok {
			return errors.New(".claude.json: mcpServers is not an object")
		}
		server, _ = servers.get("codebase-memory-mcp")
	}
	if server == nil {
		note("codebase-memory-mcp is not set up; the builder-led run has no code graph")
		return nil
	}
	state := load(join(cfg, ".claude.json"))
	all, ok := state.get("mcpServers")
	if !ok {
		all = &object{}
		state.set("mcpServers", all)
	}
	servers, ok := all.(*object)
	if !ok {
		return errors.New("the kept .claude.json: mcpServers is not an object")
	}
	servers.set("codebase-memory-mcp", server)
	return writeJSON(join(cfg, ".claude.json"), state)
}

// matchbloxHooks returns the hooks of one group whose command runs `matchblox hook`.
func matchbloxHooks(group *object) ([]any, error) {
	v, _ := group.get("hooks")
	switch hs := v.(type) {
	case []any:
		var keep []any
		for _, h := range hs {
			if ho, ok := h.(*object); ok {
				if cmd, _ := ho.get("command"); strings.Contains(pyStr(cmd), "matchblox hook") {
					keep = append(keep, ho)
				}
			}
		}
		return keep, nil
	case json.Number, bool:
		if truthy(hs) {
			return nil, errors.New("settings.json: a hook list is not a list")
		}
	}
	return nil, nil
}

func asList(v any) []any { l, _ := v.([]any); return l }

// pyStr is the text Python's str() gave a command: a string as is, any other value as JSON. A missing command is "".
func pyStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	var b bytes.Buffer
	encode(&b, v, 0)
	return b.String()
}

// truthy is Python's bool(): null, false, 0, "", [] and {} are false.
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case json.Number:
		f, err := strconv.ParseFloat(string(t), 64)
		return err != nil || f != 0
	case []any:
		return len(t) > 0
	case *object:
		return len(t.members) > 0
	}
	return true
}

// Paths, as Python's pathlib joins and compares them: no symlink is resolved and ".." stays.

func cleanPath(p string) string {
	var parts []string
	for _, s := range strings.Split(p, "/") {
		if s != "" && s != "." {
			parts = append(parts, s)
		}
	}
	joined := strings.Join(parts, "/")
	switch {
	case strings.HasPrefix(p, "/"):
		return "/" + joined
	case joined == "":
		return "."
	}
	return joined
}

func join(base string, more ...string) string {
	return cleanPath(base + "/" + strings.Join(more, "/"))
}

// clear removes a link without following it, or a folder parity owns.
func clear(p string) error {
	fi, err := os.Lstat(p)
	switch {
	case err != nil:
		return nil
	case fi.Mode()&fs.ModeSymlink != 0 || fi.Mode().IsRegular():
		return os.Remove(p)
	case fi.IsDir():
		return os.RemoveAll(p)
	}
	return nil
}

// A JSON value: nil, bool, string, json.Number, []any or *object. An object keeps its members in the order read.

type member struct {
	key string
	val any
}

type object struct{ members []member }

func (o *object) get(key string) (any, bool) {
	for _, m := range o.members {
		if m.key == key {
			return m.val, true
		}
	}
	return nil, false
}

// set replaces a member in place, or appends it. A repeated key keeps its first place and its last value, as Python does.
func (o *object) set(key string, val any) {
	for i := range o.members {
		if o.members[i].key == key {
			o.members[i].val = val
			return
		}
	}
	o.members = append(o.members, member{key, val})
}

func (o *object) clone() *object { return &object{members: append([]member(nil), o.members...)} }

// load reads a JSON object. A missing or unreadable file, bad JSON or a value that is not an object gives an empty object.
func load(p string) *object {
	b, err := os.ReadFile(p)
	if err != nil || !utf8.Valid(b) {
		return &object{}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	v, err := decode(dec)
	if err != nil {
		return &object{}
	}
	if _, err := dec.Token(); err != io.EOF {
		return &object{}
	}
	o, ok := v.(*object)
	if !ok {
		return &object{}
	}
	return o
}

func decode(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch d {
	case '{':
		o := &object{}
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				return nil, err
			}
			v, err := decode(dec)
			if err != nil {
				return nil, err
			}
			o.set(k.(string), v)
		}
		_, err := dec.Token()
		return o, err
	case '[':
		l := []any{}
		for dec.More() {
			v, err := decode(dec)
			if err != nil {
				return nil, err
			}
			l = append(l, v)
		}
		_, err := dec.Token()
		return l, err
	}
	return nil, fmt.Errorf("unexpected %v", d)
}

// writeJSON writes v as Python's json.dumps(v, indent=2) did: ASCII only, keys in order, no final newline.
func writeJSON(p string, v any) error {
	var b bytes.Buffer
	encode(&b, v, 0)
	return os.WriteFile(p, b.Bytes(), 0o666)
}

func encode(b *bytes.Buffer, v any, depth int) {
	indent := func(d int) { b.WriteByte('\n'); b.WriteString(strings.Repeat("  ", d)) }
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(t))
	case string:
		quote(b, t)
	case json.Number:
		b.WriteString(pyNumber(string(t)))
	case []any:
		if len(t) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, x := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			indent(depth + 1)
			encode(b, x, depth+1)
		}
		indent(depth)
		b.WriteByte(']')
	case *object:
		if len(t.members) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteByte('{')
		for i, m := range t.members {
			if i > 0 {
				b.WriteByte(',')
			}
			indent(depth + 1)
			quote(b, m.key)
			b.WriteString(": ")
			encode(b, m.val, depth+1)
		}
		indent(depth)
		b.WriteByte('}')
	}
}

// quote escapes as Python's ensure_ascii does: every char outside space to "~" becomes an escape, in lower-case hex.
func quote(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r >= ' ' && r <= '~':
			b.WriteRune(r)
		case r > 0xffff:
			r -= 0x10000
			fmt.Fprintf(b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
		default:
			fmt.Fprintf(b, `\u%04x`, r)
		}
	}
	b.WriteByte('"')
}

// pyNumber prints a JSON number as Python reads and prints it: an integer as written ("-0" as "0"), a float as repr().
func pyNumber(s string) string {
	if !strings.ContainsAny(s, ".eE") {
		if s == "-0" {
			return "0"
		}
		return s
	}
	f, _ := strconv.ParseFloat(s, 64)
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	sign := ""
	if math.Signbit(f) {
		sign, f = "-", -f
	}
	// The shortest digits that read back as f, and where the decimal point goes: what repr() starts from.
	mant, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	digits := strings.Replace(mant, ".", "", 1)
	x, _ := strconv.Atoi(exp)
	point := x + 1
	switch {
	case point <= -4 || point > 16:
		out := digits[:1]
		if len(digits) > 1 {
			out += "." + digits[1:]
		}
		ex, es := point-1, "+"
		if ex < 0 {
			ex, es = -ex, "-"
		}
		return fmt.Sprintf("%s%se%s%02d", sign, out, es, ex)
	case point <= 0:
		return sign + "0." + strings.Repeat("0", -point) + digits
	case point < len(digits):
		return sign + digits[:point] + "." + digits[point:]
	}
	return sign + digits + strings.Repeat("0", point-len(digits)) + ".0"
}
