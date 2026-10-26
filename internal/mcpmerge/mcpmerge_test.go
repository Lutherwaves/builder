package mcpmerge

import (
	"encoding/json"
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
