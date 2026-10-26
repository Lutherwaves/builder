// Package mcpmerge adds the codebase-memory server to a .mcp.json and keeps every other key, in order, as written.
package mcpmerge

import (
	"bytes"
	"encoding/json"
	"errors"
)

const serverName = "codebase-memory"

type member struct {
	key string
	val json.RawMessage
}

// object is a JSON object whose members keep the order they were written in.
type object []member

func (o object) index(key string) int {
	for i, m := range o {
		if m.key == key {
			return i
		}
	}
	return -1
}

// set replaces a member in place, or appends it. A repeated key keeps its first place and its last value, as jq does.
func (o object) set(key string, val json.RawMessage) object {
	if i := o.index(key); i >= 0 {
		o[i].val = val
		return o
	}
	return append(o, member{key, val})
}

func parseObject(src []byte) (object, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	if tok, err := dec.Token(); err != nil {
		return nil, err
	} else if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not a JSON object")
	}
	var o object
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		o = o.set(tok.(string), val)
	}
	return o, nil
}

func (o object) encode(b *bytes.Buffer) error {
	b.WriteByte('{')
	for i, m := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		k, err := marshal(m.key)
		if err != nil {
			return err
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(m.val)
	}
	b.WriteByte('}')
	return nil
}

// marshal encodes without HTML escaping: jq does not escape <, > and &.
func marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

func Merge(src []byte, bin string) ([]byte, error) {
	if !json.Valid(src) {
		return nil, errors.New("invalid JSON")
	}
	top, err := parseObject(src)
	if err != nil {
		return nil, err
	}
	var servers object
	if i := top.index("mcpServers"); i >= 0 && len(top[i].val) > 0 && top[i].val[0] == '{' {
		if servers, err = parseObject(top[i].val); err != nil {
			return nil, err
		}
	}
	entry, err := marshal(struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Env     struct{} `json:"env"`
	}{bin, []string{"--ui=false"}, struct{}{}})
	if err != nil {
		return nil, err
	}
	var sb bytes.Buffer
	if err := servers.set(serverName, entry).encode(&sb); err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err := top.set("mcpServers", sb.Bytes()).encode(&compact); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}
