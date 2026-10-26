package mcpmerge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func parse(t *testing.T, b []byte) map[string]any {
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

var server = map[string]any{"command": "/bin/cbm", "args": []any{"--ui=false"}, "env": map[string]any{}}

func TestMergeKeepsOtherServers(t *testing.T) {
	out, err := Merge([]byte(`{"mcpServers": {"other": {"command": "x"}}, "keep": 1}`), "/bin/cbm")
	if err != nil {
		t.Fatal(err)
	}
	v := parse(t, out)
	s := v["mcpServers"].(map[string]any)
	if v["keep"] != 1.0 || s["other"] == nil || !reflect.DeepEqual(s["codebase-memory"], server) {
		t.Fatal(v)
	}
	if !strings.HasSuffix(string(out), "}\n") || !strings.Contains(string(out), "\n  \"") {
		t.Fatalf("%q", out)
	}
}

func TestMergeCreatesMcpServers(t *testing.T) {
	for _, src := range []string{`{}`, `{"mcpServers": []}`} {
		out, err := Merge([]byte(src), "/bin/cbm")
		if err != nil || !reflect.DeepEqual(parse(t, out)["mcpServers"], map[string]any{"codebase-memory": server}) {
			t.Fatal(src, string(out), err)
		}
	}
}

func TestMergeRefusesNonObject(t *testing.T) {
	for _, src := range []string{`[]`, `null`, `{bad`, `3`} {
		if _, err := Merge([]byte(src), "/bin/cbm"); err == nil {
			t.Fatal(src)
		}
	}
}

func keys(t *testing.T, raw []byte) []string {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	var out []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, k.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestMergeKeepsKeyOrder(t *testing.T) {
	out, err := Merge([]byte(`{"z":1,"mcpServers":{"b":{},"a":{}},"a":2}`), "/bin/cbm")
	if err != nil {
		t.Fatal(err)
	}
	if got := keys(t, out); !reflect.DeepEqual(got, []string{"z", "mcpServers", "a"}) {
		t.Fatal(got)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(out, &top); err != nil {
		t.Fatal(err)
	}
	if got := keys(t, top["mcpServers"]); !reflect.DeepEqual(got, []string{"b", "a", "codebase-memory"}) {
		t.Fatal(got)
	}
}

func TestMergeAppendsMcpServersLast(t *testing.T) {
	out, err := Merge([]byte(`{"z":1,"a":2}`), "/bin/cbm")
	if err != nil {
		t.Fatal(err)
	}
	if got := keys(t, out); !reflect.DeepEqual(got, []string{"z", "a", "mcpServers"}) {
		t.Fatal(got)
	}
}

func TestMergeKeepsBigNumbers(t *testing.T) {
	out, err := Merge([]byte(`{"n":12345678901234567890,"f":1.50}`), "/bin/cbm")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`12345678901234567890`, `1.50`} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("%s missing: %s", want, out)
		}
	}
}

func TestMergeReplacesInPlace(t *testing.T) {
	out, err := Merge([]byte(`{"mcpServers":{"a":{},"codebase-memory":{"command":"old"},"c":{}}}`), "/bin/cbm")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(out, &top); err != nil {
		t.Fatal(err)
	}
	if got := keys(t, top["mcpServers"]); !reflect.DeepEqual(got, []string{"a", "codebase-memory", "c"}) {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(parse(t, out)["mcpServers"].(map[string]any)["codebase-memory"], server) {
		t.Fatal(string(out))
	}
}

func TestMergeMatchesJq(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is missing")
	}
	filter := `.mcpServers = (.mcpServers // {}) | .mcpServers["codebase-memory"] = {command:$bin, args:["--ui=false"], env:{}}`
	for _, src := range []string{
		`{"z":1,"mcpServers":{"b":{},"a":{}},"a":2}`,
		`{"n":12345678901234567890,"f":1.50}`,
		`{"mcpServers":{"a":{},"codebase-memory":{"command":"old"},"c":{"args":[],"env":{}}},"x":[1,{"y":null}]}`,
		`{}`,
	} {
		file := filepath.Join(t.TempDir(), ".mcp.json")
		if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		want, err := exec.Command("jq", "--arg", "bin", "/bin/cbm", filter, file).Output()
		if err != nil {
			t.Fatal(err)
		}
		got, err := Merge([]byte(src), "/bin/cbm")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s\nwant:\n%s\ngot:\n%s", src, want, got)
		}
	}
}
